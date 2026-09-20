package db

import (
	"database/sql"
	"fmt"
	"log"
	"net/url"
)

// demoSchema holds a separate copy of the data tables (seeded with
// fabricated rows) so that demo sessions can browse a realistic-looking
// app without ever touching real inventory, suppliers or costs.
//
// Only DATA tables live in this schema. Auth tables (users, sessions)
// stay in the public schema on purpose: demo sessions are real session
// rows, and login/logout/session validation must keep working for them.
const demoSchema = "demo"

// EnsureDemoSchema creates the demo schema and its tables (if missing)
// by replaying migration.sql inside it, then seeds fabricated data.
// Safe to call on every startup: everything is IF NOT EXISTS / count-gated.
func EnsureDemoSchema(d *sql.DB) error {
	if _, err := d.Exec(fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %s`, demoSchema)); err != nil {
		return fmt.Errorf("create demo schema: %w", err)
	}

	// Replay the migration with the demo schema as the creation target.
	// SET LOCAL scopes the search_path to this transaction only, so the
	// shared pool is never affected.
	tx, err := d.Begin()
	if err != nil {
		return fmt.Errorf("demo tx begin: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(fmt.Sprintf(`SET LOCAL search_path = %s`, demoSchema)); err != nil {
		return fmt.Errorf("demo search_path: %w", err)
	}
	if _, err := tx.Exec(migrationSQL); err != nil {
		return fmt.Errorf("demo migrate: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("demo tx commit: %w", err)
	}

	return seedDemoData(d)
}

// seedDemoData fills demo.master_items with fabricated items if empty.
// No real stock IDs, item names, suppliers or costs are ever used here.
func seedDemoData(d *sql.DB) error {
	var count int
	if err := d.QueryRow(fmt.Sprintf(`SELECT COUNT(*) FROM %s.master_items`, demoSchema)).Scan(&count); err != nil {
		return fmt.Errorf("demo seed count: %w", err)
	}
	if count > 0 {
		return nil
	}

	items := []struct {
		stockID, name, uom, group string
		cost                      float64
		supplier, status          string
	}{
		{"DEMO-001", "Sample Canine Dry Food Chicken 2kg", "pack", "Pet Food | Pet food", 45.00, "Demo Pet Supplies Sdn Bhd", "Available"},
		{"DEMO-002", "Sample Canine Dry Food Salmon 2kg", "pack", "Pet Food | Pet food", 48.50, "Demo Pet Supplies Sdn Bhd", "Available"},
		{"DEMO-003", "Sample Feline Dry Food Chicken 1.5kg", "pack", "Pet Food | Pet food", 39.20, "Demo Pet Supplies Sdn Bhd", "Available"},
		{"DEMO-004", "Sample Feline Wet Food Tuna 85g", "can", "Pet Food | Pet food", 4.80, "Demo Pet Supplies Sdn Bhd", "Available"},
		{"DEMO-005", "Sample Multivitamin Syrup 100ml", "bottle", "Supplements | Vitamins", 22.10, "Demo Veterinary Pharma Ltd", "Available"},
		{"DEMO-006", "Sample Calcium Tablet 100s", "bottle", "Supplements | Vitamins", 18.60, "Demo Veterinary Pharma Ltd", "Available"},
		{"DEMO-007", "Sample Dewormer Suspension 50ml", "bottle", "Medication | Anthelmintics", 16.40, "Demo Veterinary Pharma Ltd", "Available"},
		{"DEMO-008", "Sample Antibiotic Tablet 500mg 100s", "box", "Medication | Antibiotics", 55.00, "Demo Veterinary Pharma Ltd", "Available"},
		{"DEMO-009", "Sample Anti-inflammatory Injection 20ml", "vial", "Medication | Anti-inflammatory", 27.30, "Demo Veterinary Pharma Ltd", "Available"},
		{"DEMO-010", "Sample Tick & Flea Spot-on 3-Pack", "box", "Medication | Parasiticides", 62.90, "Demo Animal Health Co", "Available"},
		{"DEMO-011", "Sample Vaccine Vial 1-Dose", "vial", "Medication | Vaccines", 35.75, "Demo Animal Health Co", "Available"},
		{"DEMO-012", "Sample Syringe 5ml (Box of 100)", "box", "Consumables | Syringes", 24.00, "Demo Medical Supply Sdn Bhd", "Available"},
		{"DEMO-013", "Sample Needle 21G (Box of 100)", "box", "Consumables | Syringes", 12.50, "Demo Medical Supply Sdn Bhd", "Available"},
		{"DEMO-014", "Sample Surgical Gloves M (Box of 100)", "box", "Consumables | Gloves", 31.20, "Demo Medical Supply Sdn Bhd", "Available"},
		{"DEMO-015", "Sample Cotton Roll 500g", "roll", "Consumables | Cotton", 14.90, "Demo Medical Supply Sdn Bhd", "Available"},
		{"DEMO-016", "Sample Gauze Roll 10m", "roll", "Consumables | Gauze", 8.40, "Demo Medical Supply Sdn Bhd", "Available"},
		{"DEMO-017", "Sample Elastic Bandage 4in", "roll", "Consumables | Bandages", 6.75, "Demo Medical Supply Sdn Bhd", "Available"},
		{"DEMO-018", "Sample Surgical Blade #22 (Box of 100)", "box", "Consumables | Surgical", 19.30, "Demo Medical Supply Sdn Bhd", "Available"},
		{"DEMO-019", "Sample Absorbable Suture 3-0", "pack", "Consumables | Sutures", 21.60, "Demo Medical Supply Sdn Bhd", "Available"},
		{"DEMO-020", "Sample Shampoo Medicated 250ml", "bottle", "Grooming | Shampoo", 17.25, "Demo Pet Supplies Sdn Bhd", "Available"},
		{"DEMO-021", "Sample Ear Cleaner Solution 100ml", "bottle", "Grooming | Ear Care", 15.80, "Demo Pet Supplies Sdn Bhd", "Available"},
		{"DEMO-022", "Sample Dental Chew Stick Medium", "pack", "Pet Food | Treats", 11.90, "Demo Pet Supplies Sdn Bhd", "Available"},
		{"DEMO-023", "Sample IV Fluid Bag 500ml", "bag", "Consumables | IV Fluids", 13.45, "Demo Veterinary Pharma Ltd", "Available"},
		{"DEMO-024", "Sample Anaesthetic Bottle 250ml", "bottle", "Medication | Anaesthetics", 88.00, "Demo Veterinary Pharma Ltd", "Available"},
	}

	tx, err := d.Begin()
	if err != nil {
		return fmt.Errorf("demo seed tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(fmt.Sprintf(
		`INSERT INTO %s.master_items (stock_id, item_name, uom, item_group, cost, last_supplier, product_status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (stock_id) DO NOTHING`, demoSchema))
	if err != nil {
		return fmt.Errorf("demo seed stmt: %w", err)
	}
	defer stmt.Close()

	for _, it := range items {
		if _, err := stmt.Exec(it.stockID, it.name, it.uom, it.group, it.cost, it.supplier, it.status); err != nil {
			return fmt.Errorf("demo seed insert: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("demo seed commit: %w", err)
	}
	log.Printf("demo schema seeded with %d fabricated items", len(items))
	return nil
}

// ConnectDemo opens a connection pool pinned to the demo schema via the
// `options` startup parameter, so every unqualified query made through it
// resolves to demo tables. Returns nil-safe error; caller decides whether
// demo isolation is mandatory.
func ConnectDemo(databaseURL string) (*sql.DB, error) {
	u, err := url.Parse(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("demo db url parse: %w", err)
	}
	q := u.Query()
	q.Set("options", fmt.Sprintf("-csearch_path=%s", demoSchema))
	u.RawQuery = q.Encode()

	db, err := sql.Open("postgres", u.String())
	if err != nil {
		return nil, fmt.Errorf("demo db open: %w", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("demo db ping: %w", err)
	}
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	return db, nil
}
