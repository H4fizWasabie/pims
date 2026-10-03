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

// GetExpiryList pages through batches expiring within a year (earliest first)
// and returns the total so callers can paginate. The window is filtered in
// SQL so pages are never short.
func GetExpiryList(d *sql.DB, page, pageSize int) ([]ExpiryItem, int, error) {
	var total int
	if err := d.QueryRow(`SELECT COUNT(*) FROM expiry_tracking WHERE expiry_date <= CURRENT_DATE + 365`).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := d.Query(
		`SELECT e.id, e.stock_id, e.item_name, e.batch_no, e.expiry_date, e.uom, e.remarks,
		        COALESCE((SELECT qty FROM expiry_monthly_qty WHERE expiry_tracking_id = e.id ORDER BY month_key DESC LIMIT 1), 0),
		        (e.expiry_date - CURRENT_DATE)
		 FROM expiry_tracking e
		 WHERE e.expiry_date <= CURRENT_DATE + 365
		 ORDER BY e.expiry_date ASC, e.id LIMIT $1 OFFSET $2`,
		pageSize, page*pageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]ExpiryItem, 0)
	for rows.Next() {
		var it ExpiryItem
		var expDate time.Time
		if err := rows.Scan(&it.RowIndex, &it.StockID, &it.ItemName, &it.Batch, &expDate, &it.UOM, &it.Remarks, &it.LatestQty, &it.DaysLeft); err != nil {
			return nil, 0, err
		}
		it.Expiry = expDate.Format("02/01/2006")
		it.Level, it.Label = expiryLevel(it.DaysLeft)
		items = append(items, it)
	}
	return items, total, rows.Err()
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
