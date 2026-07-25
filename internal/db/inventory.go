package db

import "database/sql"

type InventoryItem struct {
	StockID      string  `json:"stockId"`
	ItemName     string  `json:"itemName"`
	CurrentStock float64 `json:"currentStock"`
}

func GetInventoryChunk(d *sql.DB, page, pageSize int) ([]InventoryItem, error) {
	offset := page * pageSize
	rows, err := d.Query(
		`SELECT stock_id, item_name, current_stock FROM inventory ORDER BY stock_id LIMIT $1 OFFSET $2`,
		pageSize, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []InventoryItem
	for rows.Next() {
		var it InventoryItem
		if err := rows.Scan(&it.StockID, &it.ItemName, &it.CurrentStock); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

func ReplaceInventoryData(d *sql.DB, data [][]string) error {
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM inventory`); err != nil {
		return err
	}
	if len(data) == 0 {
		return tx.Commit()
	}
	stmt, err := tx.Prepare(`INSERT INTO inventory (stock_id, item_name, current_stock) VALUES ($1, $2, $3)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, row := range data {
		if len(row) < 3 {
			continue
		}
		qty := parseFloat(row[2])
		_, err = stmt.Exec(row[0], row[1], qty)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}
