package handler

import (
	"net/http"

	"github.com/H4fizWasabie/pims/internal/db"
)

func (h *Handler) HandleDashboardSummary(w http.ResponseWriter, r *http.Request) {
	dbConn, stockDB := h.databases(r.Context())
	summary, err := db.GetDashboardSummary(dbConn, stockDB)
	if err != nil {
		h.ServerError(w, r, err)
		return
	}
	h.JSON(w, 200, summary)
}
