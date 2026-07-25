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
	items, err := db.GetExpiryList(h.DB, page-1, pageSize)
	if err != nil {
		h.Error(w, 500, "Server Error: "+err.Error())
		return
	}
	h.JSON(w, 200, map[string]any{"items": items, "currentPage": page})
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
	if err := db.UpdateExpiryRemark(h.DB, req.RowIndex, req.Remark); err != nil {
		h.Error(w, 500, "Server Error: "+err.Error())
		return
	}
	h.Success(w, "Remark updated")
}
