package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/H4fizWasabie/pims/internal/db"
)

func (h *Handler) HandleInventoryChunk(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	if pageSize <= 0 {
		pageSize = 50
	}
	items, err := db.GetInventoryChunk(h.DB, page, pageSize)
	if err != nil {
		h.Error(w, 500, "Server Error: "+err.Error())
		return
	}
	h.JSON(w, 200, items)
}

func (h *Handler) HandleInventoryReplace(w http.ResponseWriter, r *http.Request) {
	var data [][]string
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.Error(w, 400, "Invalid data format")
		return
	}
	if err := db.ReplaceInventoryData(h.DB, data); err != nil {
		h.Error(w, 500, "Server Error: "+err.Error())
		return
	}
	h.Success(w, "Successfully updated Inventory.")
}
