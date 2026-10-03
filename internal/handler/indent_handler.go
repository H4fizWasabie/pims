package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
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
		h.businessError(w, r, err)
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
		h.businessError(w, r, err)
		return
	}
	h.Success(w, "Request Rejected.")
}

type bulkResult struct {
	ID      int    `json:"id"`
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
}

const maxBulk = 200

// HandleIndentBulk approves or rejects many indent lines in one call. Each line
// is its own transaction (same rules as the single endpoints), so one
// out-of-stock line never blocks the rest; the response says which failed.
func (h *Handler) HandleIndentBulk(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r.Context())
	if user == nil || !containsFold(h.Cfg.IndentApprovers, user.Email) {
		h.Error(w, 403, "Access Denied.")
		return
	}
	var req struct {
		Action string `json:"action"`
		IDs    []int  `json:"ids"`
	}
	if !h.decode(w, r, &req) {
		return
	}
	if req.Action != "approve" && req.Action != "reject" {
		h.Error(w, 400, "Action must be approve or reject.")
		return
	}
	if len(req.IDs) == 0 || len(req.IDs) > maxBulk {
		h.Error(w, 400, fmt.Sprintf("Select between 1 and %d lines.", maxBulk))
		return
	}
	results := make([]bulkResult, 0, len(req.IDs))
	done := 0
	for _, id := range req.IDs {
		var err error
		if req.Action == "approve" {
			err = db.ApproveIndent(h.DB, h.StockDB, id, user.Email)
		} else {
			err = db.RejectIndent(h.DB, id, user.Email)
		}
		res := bulkResult{ID: id, OK: err == nil}
		if err != nil {
			var ve db.ValidationError
			if errors.As(err, &ve) {
				res.Message = ve.Error()
			} else {
				log.Printf("bulk %s indent %d: %v", req.Action, id, err)
				res.Message = "server error"
			}
		} else {
			done++
		}
		results = append(results, res)
	}
	h.JSON(w, 200, map[string]any{"success": true, "done": done, "failed": len(results) - done, "results": results})
}
