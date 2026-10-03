package db

import (
	"database/sql"
	"fmt"
	"math"
	"strings"
	"time"
)

type ExpiryItem struct {
	RowIndex  int     `json:"rowIndex"`
	StockID   string  `json:"stockId"`
	ItemName  string  `json:"itemName"`
	Batch     string  `json:"batch"`
	Expiry    string  `json:"expiry"`
	UOM       string  `json:"uom"`
	Remarks   string  `json:"remarks"`
	LatestQty float64 `json:"latestQty"`
	Level     string  `json:"level"`
	Label     string  `json:"label"`
	DaysLeft  int     `json:"daysLeft"`
}

// expiryBands is the one definition of the expiry levels: the labels the UI
// shows and the day ranges the SQL filter uses both come from here.
var expiryBands = []struct {
	Key, Level, Label string
	MaxDays           int // inclusive upper bound, in days from today
}{
	{"expired", "level-expired", "EXPIRED", 0},
	{"critical", "level-critical", "Critical", 30},
	{"action", "level-action", "Action", 90},
	{"warning", "level-warning", "Warning", 180},
	{"alert", "level-alert", "Short Exp", 365},
}

// expiryWindowDays is how far ahead batches are tracked at all.
var expiryWindowDays = expiryBands[len(expiryBands)-1].MaxDays

// ExpiryPage is one page of results plus counts per band for the active search
// (ignoring the band filter), so the UI can show how many batches each band holds.
type ExpiryPage struct {
	Items  []ExpiryItem
	Total  int
	Counts map[string]int
}

// likeEscape makes user text literal inside a LIKE pattern.
var likeEscape = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// GetExpiryList pages through batches expiring within a year (earliest first).
// band ("" for all) and q (matches item name, stock ID or batch) narrow the list.
// Filtering happens in SQL so pages are never short.
func GetExpiryList(d *sql.DB, page, pageSize int, band, q string) (*ExpiryPage, error) {
	days := `(e.expiry_date - CURRENT_DATE)`
	where := fmt.Sprintf(`e.expiry_date <= CURRENT_DATE + %d`, expiryWindowDays)
	var args []any
	if q = strings.TrimSpace(q); q != "" {
		args = append(args, "%"+strings.ToLower(likeEscape.Replace(q))+"%")
		where += ` AND (LOWER(e.item_name) LIKE $1 OR LOWER(e.stock_id) LIKE $1 OR LOWER(e.batch_no) LIKE $1)`
	}

	// Counts per band (search applied, band filter not).
	cases := ""
	for _, b := range expiryBands {
		cases += fmt.Sprintf(` WHEN %s <= %d THEN '%s'`, days, b.MaxDays, b.Key)
	}
	page0 := &ExpiryPage{Counts: map[string]int{}}
	rows, err := d.Query(`SELECT CASE`+cases+` END, COUNT(*) FROM expiry_tracking e WHERE `+where+` GROUP BY 1`, args...)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var key sql.NullString
		var n int
		if err := rows.Scan(&key, &n); err != nil {
			rows.Close()
			return nil, err
		}
		page0.Counts[key.String] = n
		page0.Counts["all"] += n
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Band filter: lo is just past the previous band's upper bound.
	lo := math.MinInt32
	for _, b := range expiryBands {
		if b.Key == band {
			args = append(args, lo, b.MaxDays)
			where += fmt.Sprintf(` AND %s BETWEEN $%d AND $%d`, days, len(args)-1, len(args))
			break
		}
		lo = b.MaxDays + 1
	}
	if band != "" && band != "all" {
		page0.Total = page0.Counts[band]
	} else {
		page0.Total = page0.Counts["all"]
	}

	args = append(args, pageSize, page*pageSize)
	rows, err = d.Query(fmt.Sprintf(
		`SELECT e.id, e.stock_id, e.item_name, e.batch_no, e.expiry_date, e.uom, e.remarks,
		        COALESCE((SELECT qty FROM expiry_monthly_qty WHERE expiry_tracking_id = e.id ORDER BY month_key DESC LIMIT 1), 0),
		        %s
		 FROM expiry_tracking e
		 WHERE %s
		 ORDER BY e.expiry_date ASC, e.id LIMIT $%d OFFSET $%d`, days, where, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	page0.Items = make([]ExpiryItem, 0)
	for rows.Next() {
		var it ExpiryItem
		var expDate time.Time
		if err := rows.Scan(&it.RowIndex, &it.StockID, &it.ItemName, &it.Batch, &expDate, &it.UOM, &it.Remarks, &it.LatestQty, &it.DaysLeft); err != nil {
			return nil, err
		}
		it.Expiry = expDate.Format("02/01/2006")
		it.Level, it.Label = expiryLevel(it.DaysLeft)
		page0.Items = append(page0.Items, it)
	}
	return page0, rows.Err()
}

func UpsertExpiryTracking(d *sql.DB, stockID, itemName, batch, expiryStr, uom string, qty float64) error {
	expDate, err := time.Parse("02/01/2006", expiryStr)
	if err != nil {
		expDate, err = time.Parse("2006-01-02", expiryStr)
		if err != nil {
			return fmt.Errorf("invalid date: %s", expiryStr)
		}
	}
	// Batches expiring beyond a year are deliberately not tracked (the list only shows <= 365 days).
	if time.Until(expDate).Hours() > 24*365 {
		return nil
	}

	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var trackingID int
	err = tx.QueryRow(
		`INSERT INTO expiry_tracking (stock_id, item_name, batch_no, expiry_date, uom)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (stock_id, batch_no) DO UPDATE SET item_name = $2, expiry_date = $4, uom = $5
		 RETURNING id`,
		stockID, itemName, batch, expDate, uom,
	).Scan(&trackingID)
	if err != nil {
		return err
	}

	monthKey := expDate.Format("Jan-2006")
	_, err = tx.Exec(
		`INSERT INTO expiry_monthly_qty (expiry_tracking_id, month_key, qty)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (expiry_tracking_id, month_key) DO UPDATE SET qty = $3`,
		trackingID, monthKey, qty,
	)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func UpdateExpiryRemark(d *sql.DB, rowID int, remark string) error {
	_, err := d.Exec(`UPDATE expiry_tracking SET remarks = $1 WHERE id = $2`, remark, rowID)
	return err
}

func expiryLevel(days int) (string, string) {
	for _, b := range expiryBands {
		if days <= b.MaxDays {
			return b.Level, b.Label
		}
	}
	return "", ""
}
