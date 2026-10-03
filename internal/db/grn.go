package db

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
)

var ErrDuplicateGRN = errors.New("duplicate GRN")

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

// SubmitGRN saves the GRN and returns its number. A repeated submission
// token is reported as ErrDuplicateGRN (the token column is UNIQUE, so this
// holds even for concurrent double-submits).
func SubmitGRN(d *sql.DB, createdBy string, data *GRNSubmitData) (string, error) {
	tx, err := d.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	grnNo, err := nextNumber(tx, "GRN")
	if err != nil {
		return "", err
	}

	var logID int
	err = tx.QueryRow(
		`INSERT INTO grn_logs (grn_no, supplier, do_date, invoice_no, po_no, created_by, submission_token)
		 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`,
		grnNo, data.Supplier, data.DODate, data.InvNo, data.PONo, createdBy, data.SubmissionToken,
	).Scan(&logID)
	if err != nil {
		var pe *pq.Error
		if errors.As(err, &pe) && pe.Code == "23505" && pe.Constraint == "grn_logs_submission_token_key" {
			return "", ErrDuplicateGRN
		}
		return "", err
	}

	for _, item := range data.Items {
		_, err = tx.Exec(
			`INSERT INTO grn_items (grn_log_id, item_name, qty_po, qty_do, qty_inv, uom, batch, status, remarks)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			logID, item.ItemName, item.QtyPO, item.QtyDO, item.QtyInv, item.UOM, item.Batch, item.Status, item.Remarks)
		if err != nil {
			return "", err
		}
	}
	return grnNo, tx.Commit()
}

type rowQueryer interface {
	QueryRow(query string, args ...any) *sql.Row
}

// nextNumber returns PREFIX-YYYYMMDD-NNN from a per-day counter. Pass the
// caller's tx so a failed submit rolls the counter back (no gaps, no dupes).
func nextNumber(q rowQueryer, prefix string) (string, error) {
	day := time.Now().Format("20060102")
	var n int
	err := q.QueryRow(
		`INSERT INTO id_counters (key, counter) VALUES ($1, 1)
		 ON CONFLICT (key) DO UPDATE SET counter = id_counters.counter + 1
		 RETURNING counter`, prefix+"_"+day,
	).Scan(&n)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%s-%03d", prefix, day, n), nil
}
