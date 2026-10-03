package db

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
)

type OrderItem struct {
	StockID  string  `json:"stockId"`
	ItemName string  `json:"itemName"`
	UOM      string  `json:"uom"`
	Cost     float64 `json:"cost"`
	Supplier string  `json:"supplier"`
	Qty      float64 `json:"qty"`
	Reason   string  `json:"reason"`
	Total    float64 `json:"total"`
}

type OrderRow struct {
	ID             int     `json:"id"`
	PRFNo          string  `json:"prfNo"`
	Department     string  `json:"department"`
	ItemName       string  `json:"itemName"`
	StockID        string  `json:"stockId"`
	UOM            string  `json:"uom"`
	Qty            float64 `json:"qty"`
	UnitCost       float64 `json:"unitCost"`
	TotalCost      float64 `json:"totalCost"`
	Reason         string  `json:"reason"`
	OrderedAt      string  `json:"orderedAt"`
	OrderTickAt    *string `json:"orderTickAt"`
	PaymentTickAt  *string `json:"paymentTickAt"`
	ReceivedTickAt *string `json:"receivedTickAt"`
}

// ErrTokenReused means a submission token was replayed with a different order.
var ErrTokenReused = errors.New("submission token reused with a different order")

// SaveOrders stores the lines under one new PRF number and returns it.
// Line totals are recomputed here; the client's total is never trusted.
// With a token, a retry of the same order returns the original PRF number
// (replayed=true) instead of creating a duplicate.
func SaveOrders(d *sql.DB, department string, items []OrderItem, token string) (prfNo string, replayed bool, err error) {
	tx, err := d.Begin()
	if err != nil {
		return "", false, err
	}
	defer tx.Rollback()

	if token != "" {
		raw, _ := json.Marshal([]any{department, items})
		sum := sha256.Sum256(raw)
		hash := hex.EncodeToString(sum[:])
		// The unique key makes a concurrent duplicate wait for the first commit.
		res, err := tx.Exec(`INSERT INTO order_submissions (token, payload_hash) VALUES ($1, $2) ON CONFLICT DO NOTHING`, token, hash)
		if err != nil {
			return "", false, err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			var prev string
			var prf sql.NullString
			if err := tx.QueryRow(`SELECT payload_hash, prf_no FROM order_submissions WHERE token = $1`, token).Scan(&prev, &prf); err != nil {
				return "", false, err
			}
			if prev != hash {
				return "", false, ErrTokenReused
			}
			return prf.String, true, nil
		}
	}

	prfNo, err = nextNumber(tx, "PRF")
	if err != nil {
		return "", false, err
	}
	for _, item := range items {
		_, err := tx.Exec(
			`INSERT INTO orders (prf_no, department, item_name, stock_id, uom, qty, unit_cost, total_cost, reason)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			prfNo, department, item.ItemName, item.StockID, item.UOM, item.Qty, item.Cost, item.Qty*item.Cost, item.Reason,
		)
		if err != nil {
			return "", false, err
		}
	}
	if token != "" {
		if _, err := tx.Exec(`UPDATE order_submissions SET prf_no = $1 WHERE token = $2`, prfNo, token); err != nil {
			return "", false, err
		}
	}
	return prfNo, false, tx.Commit()
}

func GetOrders(d *sql.DB, department, dateFrom, dateTo string) ([]OrderRow, error) {
	query := `SELECT id, prf_no, department, item_name, stock_id, uom, qty, unit_cost, total_cost, reason,
		COALESCE(to_char(ordered_at, 'YYYY-MM-DD HH24:MI:SS'), ''),
		COALESCE(to_char(order_tick_at, 'YYYY-MM-DD HH24:MI:SS'), ''),
		COALESCE(to_char(payment_tick_at, 'YYYY-MM-DD HH24:MI:SS'), ''),
		COALESCE(to_char(received_tick_at, 'YYYY-MM-DD HH24:MI:SS'), '')
	 FROM orders WHERE 1=1`
	args := []interface{}{}
	argIdx := 1

	if department != "" {
		query += " AND department = $" + itoa(argIdx)
		args = append(args, department)
		argIdx++
	}
	if dateFrom != "" {
		query += " AND ordered_at::date >= $" + itoa(argIdx)
		args = append(args, dateFrom)
		argIdx++
	}
	if dateTo != "" {
		query += " AND ordered_at::date <= $" + itoa(argIdx)
		args = append(args, dateTo)
		argIdx++
	}
	query += " ORDER BY ordered_at DESC"

	rows, err := d.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]OrderRow, 0)
	for rows.Next() {
		var r OrderRow
		var ot, pt, rt sql.NullString
		if err := rows.Scan(&r.ID, &r.PRFNo, &r.Department, &r.ItemName, &r.StockID, &r.UOM,
			&r.Qty, &r.UnitCost, &r.TotalCost, &r.Reason, &r.OrderedAt, &ot, &pt, &rt); err != nil {
			return nil, err
		}
		if ot.Valid {
			r.OrderTickAt = &ot.String
		}
		if pt.Valid {
			r.PaymentTickAt = &pt.String
		}
		if rt.Valid {
			r.ReceivedTickAt = &rt.String
		}
		items = append(items, r)
	}
	return items, rows.Err()
}

func UpdateOrderTick(d *sql.DB, id int, tickField string, force bool) error {
	var col string
	switch tickField {
	case "order":
		col = "order_tick_at"
	case "payment":
		col = "payment_tick_at"
	case "received":
		col = "received_tick_at"
	default:
		return fmt.Errorf("unknown tick field %q", tickField)
	}
	if force {
		_, err := d.Exec("UPDATE orders SET "+col+" = NOW() WHERE id = $1", id)
		return err
	}
	// Only set if not already ticked
	_, err := d.Exec("UPDATE orders SET "+col+" = NOW() WHERE id = $1 AND "+col+" IS NULL", id)
	return err
}

func itoa(n int) string { return strconv.Itoa(n) }
