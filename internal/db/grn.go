package db

import (
	"database/sql"
	"fmt"
	"time"
)

type GRNMasterData struct {
	Items     []GRNItem `json:"items"`
	Suppliers []string  `json:"suppliers"`
}

type GRNItem struct {
	StockID  string `json:"stockId"`
	ItemName string `json:"itemName"`
	UOM      string `json:"uom"`
}

func GetGRNMasterData(d *sql.DB) (*GRNMasterData, error) {
	rows, err := d.Query(
		`SELECT stock_id, item_name, uom, last_supplier
		 FROM master_items WHERE LOWER(product_status) NOT IN ('unavailable', 'not-available')
		 ORDER BY stock_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []GRNItem
	supplierSet := map[string]bool{}

	for rows.Next() {
		var stockID, itemName, uom, supplier string
		if err := rows.Scan(&stockID, &itemName, &uom, &supplier); err != nil {
			return nil, err
		}
		if stockID != "" && itemName != "" {
			items = append(items, GRNItem{StockID: stockID, ItemName: itemName, UOM: uom})
		}
		if supplier != "" {
			supplierSet[supplier] = true
		}
	}

	var suppliers []string
	for s := range supplierSet {
		suppliers = append(suppliers, s)
	}
	return &GRNMasterData{Items: items, Suppliers: suppliers}, rows.Err()
}

type GRNSubmitData struct {
	Supplier        string        `json:"supplier"`
	DODate          string        `json:"doDate"`
	InvNo           string        `json:"invNo"`
	PONo            string        `json:"poNo"`
	SubmissionToken string        `json:"submissionToken"`
	Items           []GRNLineItem `json:"items"`
}

type GRNLineItem struct {
	ItemName string  `json:"itemName"`
	QtyPO    float64 `json:"qtyPo"`
	QtyDO    float64 `json:"qtyDo"`
	QtyInv   float64 `json:"qtyInv"`
	UOM      string  `json:"uom"`
	Batch    string  `json:"batch"`
	Status   string  `json:"status"`
	Remarks  string  `json:"remarks"`
}

func CheckGRNDoubleEntry(d *sql.DB, token string) (bool, error) {
	var exists bool
	err := d.QueryRow(`SELECT EXISTS(SELECT 1 FROM grn_logs WHERE submission_token = $1)`, token).Scan(&exists)
	return exists, err
}

func SubmitGRN(d *sql.DB, grnNo, createdBy string, data *GRNSubmitData) error {
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var logID int
	err = tx.QueryRow(
		`INSERT INTO grn_logs (grn_no, supplier, do_date, invoice_no, po_no, created_by, submission_token)
		 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`,
		grnNo, data.Supplier, data.DODate, data.InvNo, data.PONo, createdBy, data.SubmissionToken,
	).Scan(&logID)
	if err != nil {
		return err
	}

	for _, item := range data.Items {
		_, err = tx.Exec(
			`INSERT INTO grn_items (grn_log_id, item_name, qty_po, qty_do, qty_inv, uom, batch, status, remarks)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			logID, item.ItemName, item.QtyPO, item.QtyDO, item.QtyInv, item.UOM, item.Batch, item.Status, item.Remarks)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func NextGRNNumber(d *sql.DB) (string, error) {
	todayStr := time.Now().Format("20060102")
	key := "GRN_" + todayStr

	var count int
	err := d.QueryRow(
		`INSERT INTO id_counters (key, counter) VALUES ($1, 1)
		 ON CONFLICT (key) DO UPDATE SET counter = id_counters.counter + 1
		 RETURNING counter`, key,
	).Scan(&count)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("GRN-%s-%03d", todayStr, count), nil
}

func NextPRFNumber(d *sql.DB) (string, error) {
	todayStr := time.Now().Format("20060102")
	key := "PRF_" + todayStr

	var count int
	err := d.QueryRow(
		`INSERT INTO id_counters (key, counter) VALUES ($1, 1)
		 ON CONFLICT (key) DO UPDATE SET counter = id_counters.counter + 1
		 RETURNING counter`, key,
	).Scan(&count)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("PRF-%s-%03d", todayStr, count), nil
}
