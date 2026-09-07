package db

import (
	"database/sql"
	"fmt"
	"strings"
)

func currentStocks(pimsDB, procuraDB *sql.DB, stockIDs []string) (map[string]float64, error) {
	stocks := make(map[string]float64, len(stockIDs))
	if len(stockIDs) == 0 {
		return stocks, nil
	}

	db := procuraDB
	query := "SELECT stock_id, COALESCE(current_stock, 0) FROM items WHERE stock_id IN ("
	if db == nil {
		db = pimsDB
		query = "SELECT stock_id, COALESCE(current_stock, 0) FROM inventory WHERE stock_id IN ("
	}
	placeholders := make([]string, len(stockIDs))
	args := make([]any, len(stockIDs))
	for i, stockID := range stockIDs {
		args[i] = stockID
		if procuraDB == nil {
			placeholders[i] = fmt.Sprintf("$%d", i+1)
		} else {
			placeholders[i] = "?"
		}
	}
	query += strings.Join(placeholders, ",") + ")"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var stockID string
		var qty float64
		if err := rows.Scan(&stockID, &qty); err != nil {
			return nil, err
		}
		stocks[stockID] = qty
	}
	return stocks, rows.Err()
}
