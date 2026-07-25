package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/H4fizWasabie/pims/internal/db"
)

func (h *Handler) HandleMasterChunk(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	if pageSize <= 0 {
		pageSize = 50
	}
	items, err := db.GetMasterChunk(h.DB, page, pageSize)
	if err != nil {
		h.Error(w, 500, "Server Error: "+err.Error())
		return
	}
	h.JSON(w, 200, items)
}

func (h *Handler) HandleMasterSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		h.JSON(w, 200, []any{})
		return
	}
	items, err := db.SearchMaster(h.DB, q)
	if err != nil {
		h.Error(w, 500, "Server Error: "+err.Error())
		return
	}
	h.JSON(w, 200, items)
}

func (h *Handler) HandleMasterReplace(w http.ResponseWriter, r *http.Request) {
	var data [][]string
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.Error(w, 400, "Invalid data format")
		return
	}
	if err := db.ReplaceMasterData(h.DB, data); err != nil {
		h.Error(w, 500, "Server Error: "+err.Error())
		return
	}
	h.Success(w, "Successfully replaced database.")
}

func (h *Handler) HandleMasterAll(w http.ResponseWriter, r *http.Request) {
	items, err := db.GetAllMasterItems(h.DB)
	if err != nil {
		h.Error(w, 500, "Server Error: "+err.Error())
		return
	}
	h.JSON(w, 200, items)
}
