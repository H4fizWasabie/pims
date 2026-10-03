package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/H4fizWasabie/pims/internal/db"
)

type orderRequest struct {
	Department      string         `json:"department"`
	SubmissionToken string         `json:"submissionToken"`
	Items           []db.OrderItem `json:"items"`
}

func (h *Handler) HandleOrderGenerate(w http.ResponseWriter, r *http.Request) {
	var req orderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.Error(w, 400, "Invalid request")
		return
	}
	if strings.TrimSpace(req.Department) == "" {
		h.Error(w, 400, "Department is required")
		return
	}
	if len(req.Items) == 0 {
		h.Error(w, 400, "No items in order")
		return
	}
	for _, it := range req.Items {
		if it.ItemName == "" || it.Qty <= 0 || it.Cost < 0 {
			h.Error(w, 400, "Each item needs a name, a quantity above 0 and a non-negative cost")
			return
		}
	}
	prfNo, replayed, err := db.SaveOrders(h.DB, req.Department, req.Items, req.SubmissionToken)
	if errors.Is(err, db.ErrTokenReused) {
		h.Error(w, 409, "This submission was already used for a different order. Please resubmit.")
		return
	}
	if err != nil {
		h.ServerError(w, r, err)
		return
	}

	h.JSON(w, 200, map[string]any{
		"success":  true,
		"message":  "Order submitted successfully.",
		"prfNo":    prfNo,
		"replayed": replayed,
	})
}

func (h *Handler) HandleOrderList(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	dbConn, _ := h.databases(r.Context())
	items, err := db.GetOrders(dbConn, q.Get("department"), q.Get("dateFrom"), q.Get("dateTo"))
	if err != nil {
		h.ServerError(w, r, err)
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

	if req.Field != "order" && req.Field != "payment" && req.Field != "received" {
		h.Error(w, 400, "Unknown tick field")
		return
	}
	user := userFromContext(r.Context())
	isAdmin := user != nil && containsFold(h.Cfg.MasterAdmins, user.Email)

	if err := db.UpdateOrderTick(h.DB, req.ID, req.Field, isAdmin); err != nil {
		h.ServerError(w, r, err)
		return
	}
	h.Success(w, "Tick updated")
}
