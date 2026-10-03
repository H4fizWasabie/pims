package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/H4fizWasabie/pims/internal/db"
)

func (h *Handler) HandleExpiryList(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	if pageSize <= 0 {
		pageSize = 20
	}
	if page <= 0 {
		page = 1
	}
	dbConn, _ := h.databases(r.Context())
	if pageSize > 200 {
		pageSize = 200
	}
	items, total, err := db.GetExpiryList(dbConn, page-1, pageSize)
	if err != nil {
		h.ServerError(w, r, err)
		return
	}
	totalPages := max(1, (total+pageSize-1)/pageSize)
	h.JSON(w, 200, map[string]any{
		"items": items, "currentPage": page, "totalPages": totalPages, "totalItems": total,
		"hasPrev": page > 1, "hasNext": page < totalPages,
	})
}

type updateRemarkReq struct {
	RowIndex int    `json:"rowIndex"`
	Remark   string `json:"remark"`
}

func (h *Handler) HandleExpiryUpdateRemark(w http.ResponseWriter, r *http.Request) {
	var req updateRemarkReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.Error(w, 400, "Invalid request")
		return
	}
	if len(req.Remark) > 500 {
		h.Error(w, 400, "Remark too long (max 500 characters)")
		return
	}
	if err := db.UpdateExpiryRemark(h.DB, req.RowIndex, req.Remark); err != nil {
		h.ServerError(w, r, err)
		return
	}
	h.Success(w, "Remark updated")
}
