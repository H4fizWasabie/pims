package handler

import (
	"net/http"

	"github.com/H4fizWasabie/pims/internal/db"
)

func (h *Handler) HandleDashboardSummary(w http.ResponseWriter, r *http.Request) {
	dbConn, stockDB := h.databases(r.Context())
	summary, err := db.GetDashboardSummary(dbConn, stockDB)
	if err != nil {
		h.Error(w, 500, "Server Error: "+err.Error())
		return
	}
	h.JSON(w, 200, summary)
}
