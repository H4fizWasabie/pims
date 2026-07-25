package handler

import (
	"encoding/json"
	"net/http"

	"github.com/H4fizWasabie/pims/internal/auth"
)

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.Error(w, 405, "Method not allowed")
		return
	}
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.Error(w, 400, "Invalid request body")
		return
	}
	token, err := auth.Login(h.DB, req.Email, req.Password)
	if err != nil {
		h.Error(w, 401, err.Error())
		return
	}
	auth.SetSessionCookie(w, token)
	h.Success(w, "Logged in")
}

func (h *Handler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("pims_session"); err == nil {
		auth.DeleteSession(h.DB, cookie.Value)
	}
	auth.ClearSessionCookie(w)
	h.Success(w, "Logged out")
}

func (h *Handler) HandleMe(w http.ResponseWriter, r *http.Request) {
	token, ok := getSessionToken(r)
	if !ok {
		h.Error(w, 401, "Not authenticated")
		return
	}
	user, err := auth.ValidateSession(h.DB, token)
	if err != nil {
		h.Error(w, 401, "Invalid session")
		return
	}
	h.JSON(w, 200, map[string]any{
		"email": user.Email,
		"role":  user.Role,
	})
}
