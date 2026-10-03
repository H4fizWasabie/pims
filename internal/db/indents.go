package db

import (
	"database/sql"
	"fmt"
	"time"
)

type IndentItem struct {
	ItemName string  `json:"itemName"`
	StockID  string  `json:"stockId"`
	UOM      string  `json:"uom"`
	Qty      float64 `json:"reqQty"`
}

func GetIndentMasterData(d, stockDB *sql.DB) ([]map[string]any, error) {
	rows, err := d.Query(
		`SELECT m.stock_id, m.item_name, m.uom
		 FROM master_items m
		 WHERE LOWER(m.product_status) NOT IN ('unavailable', 'not-available')
		 ORDER BY m.stock_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []map[string]any
	var stockIDs []string
	for rows.Next() {
		var stockID, itemName, uom string
		if err := rows.Scan(&stockID, &itemName, &uom); err != nil {
			return nil, err
		}
		items = append(items, map[string]any{
			"stockId": stockID, "itemName": itemName, "uom": uom,
		})
		stockIDs = append(stockIDs, stockID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	stocks, err := currentStocks(d, stockDB, stockIDs)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		item["currentStock"] = stocks[item["stockId"].(string)]
	}
	return items, nil
}

// SubmitIndent saves all lines under one new REQ-YYYYMMDD-NNN id and returns it.
func SubmitIndent(d *sql.DB, requester string, items []IndentItem) (string, error) {
	tx, err := d.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	indentID, err := nextNumber(tx, "REQ")
	if err != nil {
		return "", err
	}
	now := time.Now()
	for _, item := range items {
		_, err := tx.Exec(
			`INSERT INTO indents (indent_id, request_date, requester, status, item_name, stock_id, uom, requested_qty)
			 VALUES ($1, $2, $3, 'Pending', $4, $5, $6, $7)`,
			indentID, now, requester, item.ItemName, item.StockID, item.UOM, item.Qty)
		if err != nil {
			return "", err
		}
	}
	return indentID, tx.Commit()
}

// ApproveIndent checks stock against the quantity stored on the row, never a client-supplied one.
func ApproveIndent(d, stockDB *sql.DB, indentRowID int, approverEmail string) error {
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var status, rowStockID string
	var reqQty float64
	err = tx.QueryRow(`SELECT status, stock_id, requested_qty FROM indents WHERE id = $1 FOR UPDATE`, indentRowID).Scan(&status, &rowStockID, &reqQty)
	if err != nil {
		return ValidationError("indent not found")
	}
	if status != "Pending" {
		return ValidationError("item was already processed")
	}

	stocks, err := currentStocks(d, stockDB, []string{rowStockID})
	if err != nil {
		return err
	}
	currentStock, ok := stocks[rowStockID]
	if !ok {
		return ValidationError(fmt.Sprintf("stock ID %s not found in Procura", rowStockID))
	}
	if currentStock < reqQty {
		return ValidationError(fmt.Sprintf("insufficient stock! Current: %.0f, Req: %.0f", currentStock, reqQty))
	}

	_, err = tx.Exec(`UPDATE indents SET status = 'Approved', action_log = $1 WHERE id = $2`,
		"Approved by: "+approverEmail, indentRowID)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func RejectIndent(d *sql.DB, indentRowID int, approverEmail string) error {
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var status string
	err = tx.QueryRow(`SELECT status FROM indents WHERE id = $1 FOR UPDATE`, indentRowID).Scan(&status)
	if err != nil {
		return ValidationError("indent not found")
	}
	if status != "Pending" {
		return ValidationError("status is not Pending")
	}
	_, err = tx.Exec(`UPDATE indents SET status = 'Rejected', action_log = $1 WHERE id = $2`,
		"Rejected by: "+approverEmail, indentRowID)
	if err != nil {
		return err
	}
	return tx.Commit()
}
