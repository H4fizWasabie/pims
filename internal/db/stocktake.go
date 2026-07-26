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

var locationGroups = map[string][]string{
	"pharmacy":    {"Pharmacy Level 1", "Mini Pharmacy Level 3"},
	"lab":         {"Lab Level 1", "Lab Level 2"},
	"medicalward": {"Store Level 2", "Physiotherapy room"},
}

type StockTakeHistoryRow struct {
	ItemName   string  `json:"itemName"`
	ExpiryDate string  `json:"expiryDate"`
	Qty        float64 `json:"qty"`
	Location   string  `json:"location"`
	ScannedAt  string  `json:"scannedAt"`
}

func GetStockTakeHistory(d *sql.DB, group, dateFrom, dateTo string) ([]StockTakeHistoryRow, error) {
	query := `SELECT item_name, expiry_date, physical_qty, location,
		COALESCE(to_char(timestamp, 'YYYY-MM-DD HH24:MI:SS'), '')
	 FROM stock_takes WHERE 1=1`
	args := []interface{}{}
	argIdx := 1

	if locs, ok := locationGroups[group]; ok && len(locs) > 0 {
		query += " AND location IN ("
		for i, loc := range locs {
			if i > 0 {
				query += ", "
			}
			query += "$" + itoa(argIdx)
			args = append(args, loc)
			argIdx++
		}
		query += ")"
	}
	if dateFrom != "" {
		query += " AND take_date >= $" + itoa(argIdx)
		args = append(args, dateFrom)
		argIdx++
	}
	if dateTo != "" {
		query += " AND take_date <= $" + itoa(argIdx)
		args = append(args, dateTo)
		argIdx++
	}
	query += " ORDER BY timestamp DESC LIMIT 500"

	rows, err := d.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]StockTakeHistoryRow, 0)
	for rows.Next() {
		var r StockTakeHistoryRow
		if err := rows.Scan(&r.ItemName, &r.ExpiryDate, &r.Qty, &r.Location, &r.ScannedAt); err != nil {
			return nil, err
		}
		items = append(items, r)
	}
	return items, rows.Err()
}
