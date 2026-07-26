package handler

import (
	"encoding/json"
	"net/http"

	"github.com/H4fizWasabie/pims/internal/db"
)

type orderRequest struct {
	Department string         `json:"department"`
	Items      []db.OrderItem `json:"items"`
}

func (h *Handler) HandleOrderPRFNumber(w http.ResponseWriter, r *http.Request) {
	prfNo, err := db.NextPRFNumber(h.DB)
	if err != nil {
		h.Error(w, 500, "Server Error: "+err.Error())
		return
	}
	h.JSON(w, 200, map[string]any{"prfNo": prfNo})
}

func (h *Handler) HandleOrderGenerate(w http.ResponseWriter, r *http.Request) {
	var req orderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.Error(w, 400, "Invalid request")
		return
	}
	if len(req.Items) == 0 {
		h.Error(w, 400, "No items in order")
		return
	}

	prfNo, err := db.NextPRFNumber(h.DB)
	if err != nil {
		h.Error(w, 500, "Server Error: "+err.Error())
		return
	}

	if err := db.SaveOrders(h.DB, prfNo, req.Department, req.Items); err != nil {
		h.Error(w, 500, "Server Error: "+err.Error())
		return
	}

	h.JSON(w, 200, map[string]any{
		"success": true,
		"message": "Order submitted successfully.",
		"prfNo":   prfNo,
	})
}

func (h *Handler) HandleOrderList(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	items, err := db.GetOrders(h.DB, q.Get("department"), q.Get("dateFrom"), q.Get("dateTo"))
	if err != nil {
		h.Error(w, 500, "Server Error: "+err.Error())
		return
	}
	h.JSON(w, 200, items)
}

func (h *Handler) HandleOrderTick(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID    int    `json:"id"`
		Field string `json:"field"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.Error(w, 400, "Invalid request")
		return
	}

	user := userFromContext(r.Context())
	isAdmin := user != nil && containsFold(h.Cfg.MasterAdmins, user.Email)

	if err := db.UpdateOrderTick(h.DB, req.ID, req.Field, isAdmin); err != nil {
		h.Error(w, 500, "Server Error: "+err.Error())
		return
	}
	h.Success(w, "Tick updated")
}
