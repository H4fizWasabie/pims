package db

import (
	"database/sql"
	"time"
)

type AnalysisItem struct {
	StockID       string  `json:"stockId"`
	ItemName      string  `json:"name"`
	Locations     string  `json:"loc"`
	UOM           string  `json:"uom"`
	Cost          float64 `json:"cost,omitempty"`
	SystemQty     float64 `json:"sysQty"`
	PhysicalQty   float64 `json:"phyQty"`
	Difference    float64 `json:"diff"`
	VarianceValue float64 `json:"val"`
	Status        string  `json:"status"`
}

type AnalysisResult struct {
	Date     string         `json:"date"`
	Surplus  float64        `json:"surplus"`
	Shortage float64        `json:"shortage"`
	Items    []AnalysisItem `json:"items"`
}

func RunStockAnalysis(d *sql.DB) (*AnalysisResult, error) {
	result := &AnalysisResult{Date: time.Now().Format("2006-01-02")}

	takeMap := map[string]*struct {
		qty       float64
		locations map[string]bool
	}{}
	rows, err := d.Query(`SELECT stock_id, physical_qty, location FROM stock_takes WHERE take_date = CURRENT_DATE`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id, loc string
		var qty float64
		if err := rows.Scan(&id, &qty, &loc); err != nil {
			return nil, err
		}
		if _, ok := takeMap[id]; !ok {
			takeMap[id] = &struct {
				qty       float64
				locations map[string]bool
			}{locations: map[string]bool{}}
		}
		takeMap[id].qty += qty
		takeMap[id].locations[loc] = true
	}
	rows.Close()

	for stockID, take := range takeMap {
		item := AnalysisItem{StockID: stockID, PhysicalQty: take.qty}

		d.QueryRow(`SELECT COALESCE(current_stock, 0) FROM inventory WHERE stock_id = $1`, stockID).Scan(&item.SystemQty)
		d.QueryRow(`SELECT item_name, uom, cost FROM master_items WHERE stock_id = $1`, stockID).Scan(&item.ItemName, &item.UOM, &item.Cost)

		locs := ""
		for l := range take.locations {
			if locs != "" {
				locs += ", "
			}
			locs += l
		}
		item.Locations = locs

		item.Difference = item.PhysicalQty - item.SystemQty
		item.VarianceValue = item.Difference * item.Cost

		switch {
		case item.Difference > 0:
			item.Status = "Surplus"
			result.Surplus += item.VarianceValue
		case item.Difference < 0:
			item.Status = "Shortage"
			result.Shortage += -item.VarianceValue
		default:
			item.Status = "Tally"
		}
		result.Items = append(result.Items, item)
	}
	return result, nil
}
