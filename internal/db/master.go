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

func ReplaceMasterData(d *sql.DB, items [][]string) error {
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

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

func parseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}
