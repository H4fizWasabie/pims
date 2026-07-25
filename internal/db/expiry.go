package db

import (
	"database/sql"
	"fmt"
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

func GetExpiryList(d *sql.DB, page, pageSize int) ([]ExpiryItem, error) {
	offset := page * pageSize
	rows, err := d.Query(
		`SELECT e.id, e.stock_id, e.item_name, e.batch_no, e.expiry_date, e.uom, e.remarks,
		        COALESCE((SELECT qty FROM expiry_monthly_qty WHERE expiry_tracking_id = e.id ORDER BY month_key DESC LIMIT 1), 0)
		 FROM expiry_tracking e
		 ORDER BY e.expiry_date ASC LIMIT $1 OFFSET $2`,
		pageSize, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []ExpiryItem
	today := time.Now()
	for rows.Next() {
		var it ExpiryItem
		var expDate time.Time
		if err := rows.Scan(&it.RowIndex, &it.StockID, &it.ItemName, &it.Batch, &expDate, &it.UOM, &it.Remarks, &it.LatestQty); err != nil {
			return nil, err
		}
		it.Expiry = expDate.Format("02/01/2006")
		daysDiff := int(expDate.Sub(today).Hours() / 24)
		it.DaysLeft = daysDiff
		it.Level, it.Label = expiryLevel(daysDiff)
		if it.Level == "" {
			continue
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

func UpsertExpiryTracking(d *sql.DB, stockID, itemName, batch, expiryStr, uom string, qty float64) error {
	expDate, err := time.Parse("02/01/2006", expiryStr)
	if err != nil {
		expDate, err = time.Parse("2006-01-02", expiryStr)
		if err != nil {
			return fmt.Errorf("invalid date: %s", expiryStr)
		}
	}
	if time.Until(expDate).Hours() > 8760 {
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
	switch {
	case days <= 0:
		return "level-expired", "EXPIRED"
	case days <= 30:
		return "level-critical", "Critical"
	case days <= 90:
		return "level-action", "Action"
	case days <= 180:
		return "level-warning", "Warning"
	case days <= 365:
		return "level-alert", "Short Exp"
	default:
		return "", ""
	}
}
