package handler

import (
	"net/http"

	"github.com/H4fizWasabie/pims/internal/db"
)

func (h *Handler) HandleDashboardSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := db.GetDashboardSummary(h.DB)
	if err != nil {
		h.Error(w, 500, "Server Error: "+err.Error())
		return
	}
	h.JSON(w, 200, summary)
}
