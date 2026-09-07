package db

import "database/sql"

type InventoryItem struct {
	StockID      string  `json:"stockId"`
	ItemName     string  `json:"itemName"`
	CurrentStock float64 `json:"currentStock"`
}

func GetInventoryChunk(d, stockDB *sql.DB, page, pageSize int) ([]InventoryItem, error) {
	offset := page * pageSize
	rows, err := d.Query(
		`SELECT stock_id, item_name FROM inventory ORDER BY stock_id LIMIT $1 OFFSET $2`,
		pageSize, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []InventoryItem
	var stockIDs []string
	for rows.Next() {
		var it InventoryItem
		if err := rows.Scan(&it.StockID, &it.ItemName); err != nil {
			return nil, err
		}
		items = append(items, it)
		stockIDs = append(stockIDs, it.StockID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	stocks, err := currentStocks(d, stockDB, stockIDs)
	if err != nil {
		return nil, err
	}
	for i := range items {
		items[i].CurrentStock = stocks[items[i].StockID]
	}
	return items, nil
}
