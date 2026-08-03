package db

import (
	"database/sql"
	"time"
)

type DisposalItem struct {
	StockID  string  `json:"stockId"`
	ItemName string  `json:"itemName"`
	Batch    string  `json:"batch"`
	Expiry   string  `json:"expiry"`
	UOM      string  `json:"uom"`
	Cost     float64 `json:"cost"`
}

type DisposalSubmit struct {
	StockID  string  `json:"stockId"`
	ItemName string  `json:"itemName"`
	Qty      float64 `json:"qty"`
	Reason   string  `json:"reason"`
	Remarks  string  `json:"remarks"`
	Batch    string  `json:"batch"`
	Cost     float64 `json:"cost"`
}

func SearchDisposalBatches(d *sql.DB, query string) ([]DisposalItem, error) {
	q := "%" + query + "%"
	rows, err := d.Query(
		`SELECT e.stock_id, e.item_name, e.batch_no, e.expiry_date, e.uom, COALESCE(m.cost, 0)
		 FROM expiry_tracking e LEFT JOIN master_items m ON e.stock_id = m.stock_id
		 WHERE LOWER(e.batch_no) LIKE LOWER($1) OR LOWER(e.item_name) LIKE LOWER($1)
		 LIMIT 15`, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []DisposalItem
	for rows.Next() {
		var it DisposalItem
		var expDate time.Time
		if err := rows.Scan(&it.StockID, &it.ItemName, &it.Batch, &expDate, &it.UOM, &it.Cost); err != nil {
			return nil, err
		}
		it.Expiry = expDate.Format("02/01/2006")
		items = append(items, it)
	}
	return items, rows.Err()
}

func SubmitDisposal(d *sql.DB, data *DisposalSubmit, userEmail string) error {
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`UPDATE inventory SET current_stock = current_stock - $1, updated_at = NOW() WHERE stock_id = $2`,
		data.Qty, data.StockID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(
		`INSERT INTO disposal_logs (stock_id, item_name, batch_no, qty_disposed, unit_cost, total_loss, reason, remarks, user_email)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		data.StockID, data.ItemName, data.Batch, data.Qty, data.Cost, data.Qty*data.Cost, data.Reason, data.Remarks, userEmail,
	)
	if err != nil {
		return err
	}

	// Update expiry tracking remarks — non-fatal
	tx.Exec(
		`UPDATE expiry_tracking SET remarks = CONCAT('DISPOSED ', $3::text, ' (', $4::text, ') | ', COALESCE(remarks, ''))
		 WHERE stock_id = $1 AND batch_no = $2`,
		data.StockID, data.Batch, data.Qty, data.Reason)

	return tx.Commit()
}
