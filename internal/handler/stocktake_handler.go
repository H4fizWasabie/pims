package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/H4fizWasabie/pims/internal/db"
	"github.com/H4fizWasabie/pims/internal/ocr"
)

func (h *Handler) HandleStockTakeSubmit(w http.ResponseWriter, r *http.Request) {
	var data db.StockTakeSubmit
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.Error(w, 400, "Invalid request")
		return
	}
	if data.StockID == "" {
		h.Error(w, 400, "Stock ID is required.")
		return
	}
	if data.Qty < 0 {
		h.Error(w, 400, "Quantity cannot be negative.")
		return
	}
	if data.Location == "" {
		h.Error(w, 400, "Location is required.")
		return
	}
	user := userFromContext(r.Context())
	email := ""
	if user != nil {
		email = user.Email
	}
	if err := db.SubmitStockTake(h.DB, &data, email); err != nil {
		h.ServerError(w, r, err)
		return
	}
	if data.Batch != "" && data.Expiry != "" {
		if err := db.UpsertExpiryTracking(h.DB, data.StockID, data.ItemName, data.Batch, data.Expiry, data.UOM, data.Qty); err != nil {
			// The scan itself is saved; tell the user the expiry list was not updated.
			log.Printf("expiry upsert %s/%s: %v", data.StockID, data.Batch, err)
			h.Success(w, "Saved, but the expiry date could not be tracked (check the date format).")
			return
		}
	}
	h.Success(w, "Saved")
}

func (h *Handler) HandleStockTakeToday(w http.ResponseWriter, r *http.Request) {
	dbConn, _ := h.databases(r.Context())
	items, err := db.GetTodayStockTake(dbConn)
	if err != nil {
		h.ServerError(w, r, err)
		return
	}
	h.JSON(w, 200, items)
}

type ocrRequest struct {
	Images []string `json:"images"`
}

func (h *Handler) HandleStockTakeHistory(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	dbConn, _ := h.databases(r.Context())
	items, err := db.GetStockTakeHistory(dbConn, q.Get("group"), q.Get("dateFrom"), q.Get("dateTo"))
	if err != nil {
		h.ServerError(w, r, err)
		return
	}
	h.JSON(w, 200, items)
}

func (h *Handler) HandleStockTakeAnalyzeImage(w http.ResponseWriter, r *http.Request) {
	var req ocrRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.Error(w, 400, "Invalid request")
		return
	}
	if len(req.Images) == 0 || len(req.Images) > 6 {
		h.Error(w, 400, "Send between 1 and 6 images.")
		return
	}
	result := ocr.AnalyzeImages(req.Images, h.Cfg.OpenRouterAPIKey, h.Cfg.OpenRouterModel, h.Cfg.GeminiAPIKey)
	h.JSON(w, 200, result)
}
