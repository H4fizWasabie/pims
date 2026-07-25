package handler

import (
	"net/http"

	"github.com/H4fizWasabie/pims/internal/db"
)

func (h *Handler) HandleOrderPRFNumber(w http.ResponseWriter, r *http.Request) {
	prfNo, err := db.NextPRFNumber(h.DB)
	if err != nil {
		h.Error(w, 500, "Server Error: "+err.Error())
		return
	}
	h.JSON(w, 200, map[string]any{"prfNo": prfNo})
}
