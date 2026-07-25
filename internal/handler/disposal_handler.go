package handler

import (
	"encoding/json"
	"net/http"

	"github.com/H4fizWasabie/pims/internal/db"
)

func (h *Handler) HandleDisposalSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		h.JSON(w, 200, []any{})
		return
	}
	items, err := db.SearchDisposalBatches(h.DB, q)
	if err != nil {
		h.Error(w, 500, "Server Error: "+err.Error())
		return
	}
	h.JSON(w, 200, items)
}

func (h *Handler) HandleDisposalSubmit(w http.ResponseWriter, r *http.Request) {
	var data db.DisposalSubmit
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.Error(w, 400, "Invalid request")
		return
	}
	if data.StockID == "" {
		h.Error(w, 400, "Stock ID is required.")
		return
	}
	if data.Qty <= 0 {
		h.Error(w, 400, "Quantity must be greater than 0.")
		return
	}
	if data.Reason == "" {
		h.Error(w, 400, "Reason is required.")
		return
	}
	user := userFromContext(r.Context())
	email := ""
	if user != nil {
		email = user.Email
	}
	if err := db.SubmitDisposal(h.DB, &data, email); err != nil {
		h.Error(w, 500, "Server Error: "+err.Error())
		return
	}
	h.Success(w, "Disposal Logged & Inventory Updated.")
}
