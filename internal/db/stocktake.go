package db

import (
	"database/sql"
	"time"
)

type StockTakeRow struct {
	Timestamp string  `json:"timestamp"`
	Location  string  `json:"location"`
	StockID   string  `json:"stockId"`
	ItemName  string  `json:"itemName"`
	Qty       float64 `json:"qty"`
	Batch     string  `json:"batch"`
	Expiry    string  `json:"expiry"`
}

type StockTakeSubmit struct {
	Location string  `json:"location"`
	StockID  string  `json:"stockId"`
	ItemName string  `json:"itemName"`
	UOM      string  `json:"uom"`
	Group    string  `json:"group"`
	Cost     float64 `json:"cost"`
	Qty      float64 `json:"qty"`
	Batch    string  `json:"batch"`
	Expiry   string  `json:"expiry"`
}

func SubmitStockTake(d *sql.DB, data *StockTakeSubmit, userEmail string) error {
	_, err := d.Exec(
		`INSERT INTO stock_takes (take_date, location, stock_id, item_name, uom, item_group, cost, physical_qty, batch_no, expiry_date, user_email)
		 VALUES (CURRENT_DATE, $1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		data.Location, data.StockID, data.ItemName, data.UOM, data.Group, data.Cost, data.Qty, data.Batch, data.Expiry, userEmail,
	)
	return err
}

func GetTodayStockTake(d *sql.DB) ([]StockTakeRow, error) {
	rows, err := d.Query(
		`SELECT timestamp, location, stock_id, item_name, physical_qty, batch_no, expiry_date
		 FROM stock_takes WHERE take_date = CURRENT_DATE ORDER BY timestamp DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []StockTakeRow
	for rows.Next() {
		var it StockTakeRow
		var ts time.Time
		if err := rows.Scan(&ts, &it.Location, &it.StockID, &it.ItemName, &it.Qty, &it.Batch, &it.Expiry); err != nil {
			return nil, err
		}
		it.Timestamp = ts.Format("15:04:05")
		items = append(items, it)
	}
	return items, rows.Err()
}
