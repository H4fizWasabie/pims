package db

import "database/sql"

type DashboardSummary struct {
	Stats struct {
		TotalItems      int     `json:"totalItems"`
		LowStock        int     `json:"lowStock"`
		PendingRequests int     `json:"pendingRequests"`
		ExpiryCritical  int     `json:"expiryCritical"`
		VarianceValue   float64 `json:"varianceValue"`
		PendingSpecs    int     `json:"pendingSpecs"`
	} `json:"stats"`
	PendingIndents []PendingIndent `json:"pendingIndents"`
	PendingSpecs   []PendingSpec   `json:"pendingSpecs"`
}

type PendingIndent struct {
	RowParams int     `json:"rowParams"`
	Date      string  `json:"date"`
	Requester string  `json:"requester"`
	ItemName  string  `json:"itemName"`
	StockID   string  `json:"stockId"`
	UOM       string  `json:"uom"`
	Qty       float64 `json:"qty"`
}

type PendingSpec struct {
	RowIndex      int     `json:"rowIndex"`
	ReqID         string  `json:"reqId"`
	Requester     string  `json:"requester"`
	ItemName      string  `json:"itemName"`
	Cost          float64 `json:"cost"`
	UOM           string  `json:"uom"`
	Justification string  `json:"justification"`
}

func GetDashboardSummary(d *sql.DB) (*DashboardSummary, error) {
	s := &DashboardSummary{
		PendingIndents: []PendingIndent{},
		PendingSpecs:   []PendingSpec{},
	}

	d.QueryRow(`SELECT COUNT(*) FROM inventory`).Scan(&s.Stats.TotalItems)
	d.QueryRow(`SELECT COUNT(*) FROM inventory WHERE current_stock < 10`).Scan(&s.Stats.LowStock)
	d.QueryRow(`SELECT COUNT(*) FROM indents WHERE status = 'Pending'`).Scan(&s.Stats.PendingRequests)
	d.QueryRow(`SELECT COUNT(*) FROM expiry_tracking WHERE expiry_date - CURRENT_DATE <= 90`).Scan(&s.Stats.ExpiryCritical)
	d.QueryRow(`SELECT COUNT(*) FROM new_item_requests WHERE status = 'Pending Review'`).Scan(&s.Stats.PendingSpecs)

	rows, err := d.Query(
		`SELECT id, request_date, requester, item_name, stock_id, uom, requested_qty
		 FROM indents WHERE status = 'Pending' ORDER BY request_date DESC`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var p PendingIndent
			var dt sql.NullTime
			rows.Scan(&p.RowParams, &dt, &p.Requester, &p.ItemName, &p.StockID, &p.UOM, &p.Qty)
			if dt.Valid {
				p.Date = dt.Time.Format("02/01/2006")
			}
			s.PendingIndents = append(s.PendingIndents, p)
		}
	}

	rows2, err := d.Query(
		`SELECT id, req_id, requester, item_name, cost, uom, justification
		 FROM new_item_requests WHERE status = 'Pending Review' ORDER BY request_date DESC`)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var p PendingSpec
			rows2.Scan(&p.RowIndex, &p.ReqID, &p.Requester, &p.ItemName, &p.Cost, &p.UOM, &p.Justification)
			s.PendingSpecs = append(s.PendingSpecs, p)
		}
	}

	return s, nil
}
