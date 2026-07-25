package db

import (
	"database/sql"
	"strconv"
)

type MasterItem struct {
	StockID       string  `json:"stockId"`
	ItemName      string  `json:"itemName"`
	UOM           string  `json:"uom"`
	Group         string  `json:"group"`
	Cost          float64 `json:"cost"`
	LastSupplier  string  `json:"lastSupplier"`
	ProductStatus string  `json:"productStatus"`
}

func GetMasterChunk(d *sql.DB, page, pageSize int) ([]MasterItem, error) {
	offset := page * pageSize
	rows, err := d.Query(
		`SELECT stock_id, item_name, uom, item_group, cost, last_supplier, product_status
		 FROM master_items ORDER BY stock_id LIMIT $1 OFFSET $2`, pageSize, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMasterItems(rows)
}

func SearchMaster(d *sql.DB, query string) ([]MasterItem, error) {
	q := "%" + query + "%"
	rows, err := d.Query(
		`SELECT stock_id, item_name, uom, item_group, cost, last_supplier, product_status
		 FROM master_items WHERE LOWER(stock_id) LIKE LOWER($1) OR LOWER(item_name) LIKE LOWER($1)
		 LIMIT 50`, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMasterItems(rows)
}

func ReplaceMasterData(d *sql.DB, items [][]string) error {
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM inventory`); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM master_items`); err != nil {
		return err
	}
	if len(items) == 0 {
		return tx.Commit()
	}
	stmt, err := tx.Prepare(
		`INSERT INTO master_items (stock_id, item_name, uom, item_group, cost, last_supplier, product_status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, row := range items {
		if len(row) < 7 {
			continue
		}
		cost := parseFloat(row[4])
		_, err = stmt.Exec(row[0], row[1], row[2], row[3], cost, row[5], row[6])
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func GetAllMasterItems(d *sql.DB) ([]MasterItem, error) {
	rows, err := d.Query(
		`SELECT stock_id, item_name, uom, item_group, cost, last_supplier, product_status
		 FROM master_items WHERE LOWER(product_status) NOT IN ('unavailable', 'not-available')
		 ORDER BY stock_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMasterItems(rows)
}

func scanMasterItems(rows *sql.Rows) ([]MasterItem, error) {
	var items []MasterItem
	for rows.Next() {
		var m MasterItem
		if err := rows.Scan(&m.StockID, &m.ItemName, &m.UOM, &m.Group, &m.Cost, &m.LastSupplier, &m.ProductStatus); err != nil {
			return nil, err
		}
		items = append(items, m)
	}
	return items, rows.Err()
}

func parseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}
