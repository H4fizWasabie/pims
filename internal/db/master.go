package db

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
)

type MasterItem struct {
	StockID       string  `json:"stockId"`
	ItemName      string  `json:"itemName"`
	UOM           string  `json:"uom"`
	Group         string  `json:"group"`
	Cost          float64 `json:"cost"`
	LastSupplier  string  `json:"lastSupplier"`
	ProductStatus string  `json:"productStatus"`
	CurrentStock  float64 `json:"currentStock"`
}

func GetMasterChunk(d, stockDB *sql.DB, page, pageSize int) ([]MasterItem, error) {
	offset := page * pageSize
	rows, err := d.Query(
		`SELECT m.stock_id, m.item_name, m.uom, m.item_group, m.cost, m.last_supplier, m.product_status
		 FROM master_items m
		 ORDER BY m.stock_id LIMIT $1 OFFSET $2`, pageSize, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMasterItems(d, stockDB, rows)
}

func SearchMaster(d, stockDB *sql.DB, query string) ([]MasterItem, error) {
	q := "%" + query + "%"
	rows, err := d.Query(
		`SELECT m.stock_id, m.item_name, m.uom, m.item_group, m.cost, m.last_supplier, m.product_status
		 FROM master_items m
		 WHERE LOWER(m.product_status) NOT IN ('unavailable', 'not-available')
		   AND (LOWER(m.stock_id) LIKE LOWER($1) OR LOWER(m.item_name) LIKE LOWER($1))
		 ORDER BY m.stock_id LIMIT 50`, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMasterItems(d, stockDB, rows)
}

// ValidationError marks bad caller input (as opposed to a DB failure).
type ValidationError string

func (e ValidationError) Error() string { return string(e) }

// ReplaceMasterData swaps the whole item master. Every row is validated
// first so a bad upload can never leave the master half-wiped or emptied.
func ReplaceMasterData(d *sql.DB, items [][]string) error {
	if len(items) == 0 {
		return ValidationError("Upload is empty; refusing to wipe the item master.")
	}
	costs := make([]float64, len(items))
	seen := make(map[string]bool, len(items))
	for i, row := range items {
		n := i + 1
		if len(row) < 7 {
			return ValidationError(fmt.Sprintf("Row %d has %d columns, expected 7.", n, len(row)))
		}
		if row[0] == "" || row[1] == "" {
			return ValidationError(fmt.Sprintf("Row %d is missing the stock ID or item name.", n))
		}
		if seen[row[0]] {
			return ValidationError(fmt.Sprintf("Row %d repeats stock ID %q.", n, row[0]))
		}
		seen[row[0]] = true
		c, err := strconv.ParseFloat(strings.TrimSpace(row[4]), 64)
		if err != nil && strings.TrimSpace(row[4]) != "" {
			return ValidationError(fmt.Sprintf("Row %d has an invalid cost %q.", n, row[4]))
		}
		costs[i] = c
	}

	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM master_items`); err != nil {
		return err
	}
	stmt, err := tx.Prepare(
		`INSERT INTO master_items (stock_id, item_name, uom, item_group, cost, last_supplier, product_status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for i, row := range items {
		if _, err := stmt.Exec(row[0], row[1], row[2], row[3], costs[i], row[5], row[6]); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func GetAllMasterItems(d, stockDB *sql.DB) ([]MasterItem, error) {
	rows, err := d.Query(
		`SELECT m.stock_id, m.item_name, m.uom, m.item_group, m.cost, m.last_supplier, m.product_status
		 FROM master_items m
		 WHERE LOWER(m.product_status) NOT IN ('unavailable', 'not-available')
		 ORDER BY m.stock_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMasterItems(d, stockDB, rows)
}

func scanMasterItems(pimsDB, procuraDB *sql.DB, rows *sql.Rows) ([]MasterItem, error) {
	var items []MasterItem
	var stockIDs []string
	for rows.Next() {
		var m MasterItem
		if err := rows.Scan(&m.StockID, &m.ItemName, &m.UOM, &m.Group, &m.Cost, &m.LastSupplier, &m.ProductStatus); err != nil {
			return nil, err
		}
		items = append(items, m)
		stockIDs = append(stockIDs, m.StockID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	stocks, err := currentStocks(pimsDB, procuraDB, stockIDs)
	if err != nil {
		return nil, err
	}
	for i := range items {
		items[i].CurrentStock = stocks[items[i].StockID]
	}
	return items, nil
}
