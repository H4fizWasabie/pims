package db

import (
	"database/sql"
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
	ID             int        `json:"id"`
	PRFNo          string     `json:"prfNo"`
	Department     string     `json:"department"`
	ItemName       string     `json:"itemName"`
	StockID        string     `json:"stockId"`
	UOM            string     `json:"uom"`
	Qty            float64    `json:"qty"`
	UnitCost       float64    `json:"unitCost"`
	TotalCost      float64    `json:"totalCost"`
	Reason         string     `json:"reason"`
	OrderedAt      string     `json:"orderedAt"`
	OrderTickAt    *string    `json:"orderTickAt"`
	PaymentTickAt  *string    `json:"paymentTickAt"`
	ReceivedTickAt *string    `json:"receivedTickAt"`
}

func SaveOrders(d *sql.DB, prfNo, department string, items []OrderItem) error {
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, item := range items {
		_, err := tx.Exec(
			`INSERT INTO orders (prf_no, department, item_name, stock_id, uom, qty, unit_cost, total_cost, reason)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			prfNo, department, item.ItemName, item.StockID, item.UOM, item.Qty, item.Cost, item.Total, item.Reason,
		)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
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
		if ot.Valid { r.OrderTickAt = &ot.String }
		if pt.Valid { r.PaymentTickAt = &pt.String }
		if rt.Valid { r.ReceivedTickAt = &rt.String }
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
		return nil
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
