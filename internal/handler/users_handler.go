package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/H4fizWasabie/pims/internal/db"
)

func (h *Handler) HandleUsersList(w http.ResponseWriter, r *http.Request) {
	users, err := db.ListUsers(h.DB)
	if err != nil {
		h.ServerError(w, r, err)
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
	if !strings.Contains(req.Email, "@") {
		h.Error(w, 400, "A valid email is required")
		return
	}
	if !validPassword(req.Password) {
		h.Error(w, 400, passwordRule)
		return
	}
	if req.Role == "" {
		req.Role = "user"
	}
	if req.Role != "user" && req.Role != "admin" {
		h.Error(w, 400, "Role must be user or admin")
		return
	}
	if _, err := db.CreateUser(h.DB, req.Email, req.Password, req.Role); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			h.Error(w, 409, err.Error())
		} else {
			h.ServerError(w, r, err)
		}
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
	if me := userFromContext(r.Context()); me != nil && me.ID == id {
		h.Error(w, 400, "You cannot delete your own account")
		return
	}
	if err := db.DeleteUser(h.DB, id); err != nil {
		h.ServerError(w, r, err)
		return
	}
	h.Success(w, "User deleted")
}
