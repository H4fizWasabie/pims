package handler

import (
	"encoding/json"
	"net/http"

	"github.com/H4fizWasabie/pims/internal/db"
)

func (h *Handler) HandleSpecSubmit(w http.ResponseWriter, r *http.Request) {
	var req db.SpecRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.Error(w, 400, "Invalid request")
		return
	}
	if req.ItemName == "" {
		h.Error(w, 400, "Item name is required.")
		return
	}
	user := userFromContext(r.Context())
	email := ""
	if user != nil {
		email = user.Email
	}
	reqID, err := db.SubmitSpecRequest(h.DB, &req, email)
	if err != nil {
		h.Error(w, 500, "Server Error: "+err.Error())
		return
	}
	h.Success(w, "New item request submitted: "+reqID)
}

type specActionReq struct {
	RowIndex int `json:"rowIndex"`
}

func (h *Handler) HandleSpecApprove(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r.Context())
	if user == nil || !containsFold(h.Cfg.SpecApprovers, user.Email) {
		h.Error(w, 403, "Access Denied: Only authorized users can approve.")
		return
	}
	var req specActionReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.Error(w, 400, "Invalid request")
		return
	}
	newStockID, err := db.ApproveSpecRequest(h.DB, req.RowIndex)
	if err != nil {
		h.Error(w, 400, err.Error())
		return
	}
	h.Success(w, "Item added to Master DB (ID: "+newStockID+")")
}

func (h *Handler) HandleSpecReject(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r.Context())
	if user == nil || !containsFold(h.Cfg.SpecApprovers, user.Email) {
		h.Error(w, 403, "Access Denied.")
		return
	}
	var req specActionReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.Error(w, 400, "Invalid request")
		return
	}
	if err := db.RejectSpecRequest(h.DB, req.RowIndex); err != nil {
		h.Error(w, 400, err.Error())
		return
	}
	h.Success(w, "Request Rejected")
}
