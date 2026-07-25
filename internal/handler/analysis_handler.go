package handler

import (
	"net/http"

	"github.com/H4fizWasabie/pims/internal/db"
)

func (h *Handler) HandleAnalysisRun(w http.ResponseWriter, r *http.Request) {
	result, err := db.RunStockAnalysis(h.DB)
	if err != nil {
		h.Error(w, 500, "Analysis failed: "+err.Error())
		return
	}
	h.JSON(w, 200, map[string]any{
		"success":  true,
		"message":  "Analysis Updated!",
		"date":     result.Date,
		"surplus":  result.Surplus,
		"shortage": result.Shortage,
	})
}

func (h *Handler) HandleAnalysisToday(w http.ResponseWriter, r *http.Request) {
	result, err := db.RunStockAnalysis(h.DB)
	if err != nil {
		h.Error(w, 500, "Server Error: "+err.Error())
		return
	}
	h.JSON(w, 200, result)
}
