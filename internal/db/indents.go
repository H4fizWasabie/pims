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

func GetIndentMasterData(d *sql.DB) ([]map[string]any, error) {
	rows, err := d.Query(
		`SELECT m.stock_id, m.item_name, m.uom, COALESCE(i.current_stock, 0)
		 FROM master_items m LEFT JOIN inventory i ON m.stock_id = i.stock_id
		 WHERE LOWER(m.product_status) NOT IN ('unavailable', 'not-available')
		 ORDER BY m.stock_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []map[string]any
	for rows.Next() {
		var stockID, itemName, uom string
		var currentStock float64
		if err := rows.Scan(&stockID, &itemName, &uom, &currentStock); err != nil {
			return nil, err
		}
		items = append(items, map[string]any{
			"stockId": stockID, "itemName": itemName, "uom": uom, "currentStock": currentStock,
		})
	}
	return items, rows.Err()
}

func SubmitIndent(d *sql.DB, requester string, items []IndentItem, indentID string) error {
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := time.Now()
	for _, item := range items {
		_, err := tx.Exec(
			`INSERT INTO indents (indent_id, request_date, requester, status, item_name, stock_id, uom, requested_qty)
			 VALUES ($1, $2, $3, 'Pending', $4, $5, $6, $7)`,
			indentID, now, requester, item.ItemName, item.StockID, item.UOM, item.Qty)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func NextIndentID() string {
	now := time.Now()
	return fmt.Sprintf("REQ-%s-%s", now.Format("0201"), now.Format("1504"))
}

func ApproveIndent(d *sql.DB, indentRowID int, reqQty float64, approverEmail string) error {
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var status, rowStockID string
	err = tx.QueryRow(`SELECT status, stock_id FROM indents WHERE id = $1 FOR UPDATE`, indentRowID).Scan(&status, &rowStockID)
	if err != nil {
		return fmt.Errorf("indent not found")
	}
	if status != "Pending" {
		return fmt.Errorf("item was already processed")
	}

	var currentStock float64
	err = tx.QueryRow(`SELECT current_stock FROM inventory WHERE stock_id = $1 FOR UPDATE`, rowStockID).Scan(&currentStock)
	if err != nil {
		return fmt.Errorf("stock ID %s not found in inventory", rowStockID)
	}
	if currentStock < reqQty {
		return fmt.Errorf("insufficient stock! Current: %.0f, Req: %.0f", currentStock, reqQty)
	}

	_, err = tx.Exec(`UPDATE inventory SET current_stock = current_stock - $1, updated_at = NOW() WHERE stock_id = $2`, reqQty, rowStockID)
	if err != nil {
		return err
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
		return fmt.Errorf("indent not found")
	}
	if status != "Pending" {
		return fmt.Errorf("status is not Pending")
	}
	_, err = tx.Exec(`UPDATE indents SET status = 'Rejected', action_log = $1 WHERE id = $2`,
		"Rejected by: "+approverEmail, indentRowID)
	if err != nil {
		return err
	}
	return tx.Commit()
}
