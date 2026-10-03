package handler

import (
	"encoding/json"
	"net/http"

	"github.com/H4fizWasabie/pims/internal/db"
)

func (h *Handler) HandleIndentMasterData(w http.ResponseWriter, r *http.Request) {
	dbConn, stockDB := h.databases(r.Context())
	items, err := db.GetIndentMasterData(dbConn, stockDB)
	if err != nil {
		h.ServerError(w, r, err)
		return
	}
	h.JSON(w, 200, items)
}

type indentSubmitReq struct {
	Requester string          `json:"requester"`
	Items     []db.IndentItem `json:"items"`
}

func (h *Handler) HandleIndentSubmit(w http.ResponseWriter, r *http.Request) {
	var req indentSubmitReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.Error(w, 400, "Invalid request")
		return
	}
	if req.Requester == "" {
		h.Error(w, 400, "Requester department is required.")
		return
	}
	if len(req.Items) == 0 {
		h.Error(w, 400, "No valid items to submit.")
		return
	}
	for _, it := range req.Items {
		if it.StockID == "" || it.ItemName == "" || it.Qty <= 0 {
			h.Error(w, 400, "Each item needs a stock ID, name and a quantity above 0.")
			return
		}
	}
	indentID, err := db.SubmitIndent(h.DB, req.Requester, req.Items)
	if err != nil {
		h.ServerError(w, r, err)
		return
	}
	h.Success(w, "Request "+indentID+" Submitted!")
}

type indentActionReq struct {
	IndentRowIndex int `json:"indentRowIndex"`
}

func (h *Handler) HandleIndentApprove(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r.Context())
	if user == nil || !containsFold(h.Cfg.IndentApprovers, user.Email) {
		h.Error(w, 403, "Access Denied.")
		return
	}
	var req indentActionReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.Error(w, 400, "Invalid request")
		return
	}
	if err := db.ApproveIndent(h.DB, h.StockDB, req.IndentRowIndex, user.Email); err != nil {
		h.Error(w, 400, err.Error())
		return
	}
	h.Success(w, "Approved.")
}

func (h *Handler) HandleIndentReject(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r.Context())
	if user == nil || !containsFold(h.Cfg.IndentApprovers, user.Email) {
		h.Error(w, 403, "Access Denied.")
		return
	}
	var req indentActionReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.Error(w, 400, "Invalid request")
		return
	}
	if err := db.RejectIndent(h.DB, req.IndentRowIndex, user.Email); err != nil {
		h.Error(w, 400, err.Error())
		return
	}
	h.Success(w, "Request Rejected.")
}
