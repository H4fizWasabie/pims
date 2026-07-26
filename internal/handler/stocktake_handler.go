package handler

import (
	"encoding/json"
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
		h.Error(w, 500, "Server Error: "+err.Error())
		return
	}
	if data.Batch != "" && data.Expiry != "" {
		_ = db.UpsertExpiryTracking(h.DB, data.StockID, data.ItemName, data.Batch, data.Expiry, data.UOM, data.Qty)
	}
	h.Success(w, "Saved")
}

func (h *Handler) HandleStockTakeToday(w http.ResponseWriter, r *http.Request) {
	items, err := db.GetTodayStockTake(h.DB)
	if err != nil {
		h.Error(w, 500, "Server Error: "+err.Error())
		return
	}
	h.JSON(w, 200, items)
}

type ocrRequest struct {
	Images []string `json:"images"`
}

func (h *Handler) HandleStockTakeHistory(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	items, err := db.GetStockTakeHistory(h.DB, q.Get("group"), q.Get("dateFrom"), q.Get("dateTo"))
	if err != nil {
		h.Error(w, 500, "Server Error: "+err.Error())
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
	result := ocr.AnalyzeImages(req.Images, h.Cfg.OpenRouterAPIKey, h.Cfg.OpenRouterModel, h.Cfg.GeminiAPIKey)
	h.JSON(w, 200, result)
}
