package handler

import (
	"encoding/json"
	"net/http"

	"github.com/H4fizWasabie/pims/internal/db"
)

func (h *Handler) HandleGRNMasterData(w http.ResponseWriter, r *http.Request) {
	data, err := db.GetGRNMasterData(h.DB)
	if err != nil {
		h.Error(w, 500, "Server Error: "+err.Error())
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
	if data.SubmissionToken == "" {
		h.Error(w, 400, "Security Error: Missing Transaction Token.")
		return
	}
	dup, err := db.CheckGRNDoubleEntry(h.DB, data.SubmissionToken)
	if err != nil {
		h.Error(w, 500, "Server Error: "+err.Error())
		return
	}
	if dup {
		h.Error(w, 409, "Double Entry Detected: This GRN has already been saved.")
		return
	}
	grnNo, err := db.NextGRNNumber(h.DB)
	if err != nil {
		h.Error(w, 500, "Server Error: "+err.Error())
		return
	}
	user := userFromContext(r.Context())
	email := ""
	if user != nil {
		email = user.Email
	}
	if err := db.SubmitGRN(h.DB, grnNo, email, &data); err != nil {
		h.Error(w, 500, "Server Error: "+err.Error())
		return
	}
	h.JSON(w, 200, map[string]any{
		"success": true,
		"message": "GRN Saved",
		"grnNo":   grnNo,
	})
}
