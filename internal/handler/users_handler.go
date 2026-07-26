package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/H4fizWasabie/pims/internal/db"
)

func (h *Handler) HandleUsersList(w http.ResponseWriter, r *http.Request) {
	users, err := db.ListUsers(h.DB)
	if err != nil {
		h.Error(w, 500, "Server Error: "+err.Error())
		return
	}
	h.JSON(w, 200, users)
}

func (h *Handler) HandleUsersCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.Error(w, 400, "Invalid request")
		return
	}
	if req.Email == "" || req.Password == "" {
		h.Error(w, 400, "Email and password required")
		return
	}
	if req.Role == "" {
		req.Role = "user"
	}
	_, err := db.CreateUser(h.DB, req.Email, req.Password, req.Role)
	if err != nil {
		h.Error(w, 500, "Server Error: "+err.Error())
		return
	}
	h.Success(w, "User created")
}

func (h *Handler) HandleUsersDelete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	if id <= 0 {
		h.Error(w, 400, "Invalid user ID")
		return
	}
	if err := db.DeleteUser(h.DB, id); err != nil {
		h.Error(w, 500, "Server Error: "+err.Error())
		return
	}
	h.Success(w, "User deleted")
}
