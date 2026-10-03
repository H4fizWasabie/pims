package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/H4fizWasabie/pims/internal/db"
)

func (h *Handler) HandleGRNMasterData(w http.ResponseWriter, r *http.Request) {
	dbConn, _ := h.databases(r.Context())
	data, err := db.GetGRNMasterData(dbConn)
	if err != nil {
		h.ServerError(w, r, err)
		return
	}
	h.JSON(w, 200, data)
}

func (h *Handler) HandleGRNSubmit(w http.ResponseWriter, r *http.Request) {
	var data db.GRNSubmitData
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.Error(w, 400, "Invalid request")
		return
	}
	if strings.TrimSpace(data.Supplier) == "" || len(data.Items) == 0 {
		h.Error(w, 400, "Supplier and at least one item are required.")
		return
	}
	for _, it := range data.Items {
		if it.ItemName == "" || it.QtyPO < 0 || it.QtyDO < 0 || it.QtyInv < 0 {
			h.Error(w, 400, "Each item needs a name and non-negative quantities.")
			return
		}
	}
	if data.SubmissionToken == "" {
		h.Error(w, 400, "Security Error: Missing Transaction Token.")
		return
	}
	dup, err := db.CheckGRNDoubleEntry(h.DB, data.SubmissionToken)
	if err != nil {
		h.ServerError(w, r, err)
		return
	}
	if dup {
		h.Error(w, 409, "Double Entry Detected: This GRN has already been saved.")
		return
	}
	user := userFromContext(r.Context())
	email := ""
	if user != nil {
		email = user.Email
	}
	grnNo, err := db.SubmitGRN(h.DB, email, &data)
	if errors.Is(err, db.ErrDuplicateGRN) {
		h.Error(w, 409, "Double Entry Detected: This GRN has already been saved.")
		return
	}
	if err != nil {
		h.ServerError(w, r, err)
		return
	}
	h.JSON(w, 200, map[string]any{
		"success": true,
		"message": "GRN Saved",
		"grnNo":   grnNo,
	})
}
