package db

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"
)

type SpecRequest struct {
	ReqID         string  `json:"reqId"`
	Requester     string  `json:"requester"`
	ItemName      string  `json:"itemName"`
	ItemGroup     string  `json:"category"`
	UOM           string  `json:"uom"`
	Cost          float64 `json:"estCost"`
	Justification string  `json:"justification"`
}

func SubmitSpecRequest(d *sql.DB, req *SpecRequest, requesterEmail string) (string, error) {
	reqID, err := nextNumber(d, "SPEC")
	if err != nil {
		return "", err
	}
	_, err = d.Exec(
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
		return "", ValidationError("request not found")
	}
	if status != "Pending Review" {
		return "", ValidationError("this request has already been processed")
	}

	newStockID := fmt.Sprintf("NEW-%d", 1000+rowID)

	// No ON CONFLICT: a clash must fail the approval, not report a master item that was never added.
	_, err = tx.Exec(
		`INSERT INTO master_items (stock_id, item_name, uom, item_group, cost, last_supplier, product_status)
		 VALUES ($1, $2, $3, $4, $5, 'Pending Vendor', 'Available')`,
		newStockID, itemName, uom, itemGroup, cost,
	)
	if err != nil {
		var pe *pq.Error
		if errors.As(err, &pe) && pe.Code == "23505" {
			return "", ValidationError("stock ID " + newStockID + " already exists in the item master; an admin needs to resolve the clash")
		}
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
		return ValidationError("this request has already been processed")
	}
	return nil
}
