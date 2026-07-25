package db

import (
	"database/sql"
	"fmt"
	"time"
)

type SpecRequest struct {
	ReqID         string  `json:"reqId"`
	Requester     string  `json:"requester"`
	ItemName      string  `json:"itemName"`
	ItemGroup     string  `json:"itemGroup"`
	UOM           string  `json:"uom"`
	Cost          float64 `json:"cost"`
	Justification string  `json:"justification"`
}

func SubmitSpecRequest(d *sql.DB, req *SpecRequest, requesterEmail string) (string, error) {
	reqID := fmt.Sprintf("SPEC-%s", time.Now().Format("020106-150405"))
	_, err := d.Exec(
		`INSERT INTO new_item_requests (req_id, requester, item_name, item_group, uom, cost, justification, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, 'Pending Review')`,
		reqID, requesterEmail, req.ItemName, req.ItemGroup, req.UOM, req.Cost, req.Justification,
	)
	return reqID, err
}

func ApproveSpecRequest(d *sql.DB, rowID int) (string, error) {
	tx, err := d.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	var itemName, itemGroup, uom, status string
	var cost float64
	err = tx.QueryRow(
		`SELECT item_name, item_group, uom, cost, status FROM new_item_requests WHERE id = $1 FOR UPDATE`,
		rowID,
	).Scan(&itemName, &itemGroup, &uom, &cost, &status)
	if err != nil {
		return "", fmt.Errorf("request not found")
	}
	if status != "Pending Review" {
		return "", fmt.Errorf("this request has already been processed")
	}

	newStockID := fmt.Sprintf("NEW-%d", 1000+rowID)

	_, err = tx.Exec(
		`INSERT INTO master_items (stock_id, item_name, uom, item_group, cost, last_supplier, product_status)
		 VALUES ($1, $2, $3, $4, $5, 'Pending Vendor', 'Available')
		 ON CONFLICT (stock_id) DO NOTHING`,
		newStockID, itemName, uom, itemGroup, cost,
	)
	if err != nil {
		return "", err
	}

	_, err = tx.Exec(`UPDATE new_item_requests SET status = 'Approved' WHERE id = $1`, rowID)
	if err != nil {
		return "", err
	}
	return newStockID, tx.Commit()
}

func RejectSpecRequest(d *sql.DB, rowID int) error {
	res, err := d.Exec(`UPDATE new_item_requests SET status = 'Rejected' WHERE id = $1 AND status = 'Pending Review'`, rowID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("this request has already been processed")
	}
	return nil
}
