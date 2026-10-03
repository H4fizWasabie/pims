package handler

import (
	"errors"
	"log"
	"net/http"

	"github.com/H4fizWasabie/pims/internal/auth"
	"github.com/H4fizWasabie/pims/internal/db"
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

	ip := getClientIP(r)
	if !loginLimiter.allow(ip) {
		h.Error(w, 429, "Too many login attempts. Please wait 1 minute.")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBody)
	var req loginRequest
	if !h.decode(w, r, &req) {
		return
	}
	token, err := auth.Login(h.DB, req.Email, req.Password)
	if err != nil {
		loginLimiter.record(ip)
		h.Error(w, 401, err.Error())
		return
	}
	auth.SetSessionCookie(w, token)
	h.Success(w, "Logged in")
}

func (h *Handler) HandleDemoLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.Error(w, 405, "Method not allowed")
		return
	}
	ip := getClientIP(r)
	if !demoLimiter.allow(ip) {
		h.Error(w, 429, "Too many demo sessions. Please wait 1 minute.")
		return
	}
	demoLimiter.record(ip)
	token, err := db.CreateDemoSession(h.DB)
	if err != nil {
		log.Printf("demo session: %v", err)
		h.Error(w, 500, "Demo unavailable, try again")
		return
	}
	auth.SetSessionCookie(w, token)
	h.Success(w, "Demo session started")
}

func (h *Handler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("pims_session"); err == nil {
		auth.DeleteSession(h.DB, cookie.Value)
	}
	auth.ClearSessionCookie(w)
	h.Success(w, "Logged out")
}

func (h *Handler) HandleChangePassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OldPassword string `json:"oldPassword"`
		NewPassword string `json:"newPassword"`
	}
	if !h.decode(w, r, &req) {
		return
	}
	user := userFromContext(r.Context())
	if user == nil {
		h.Error(w, 401, "Authentication required")
		return
	}
	if !validPassword(req.NewPassword) {
		h.Error(w, 400, passwordRule)
		return
	}
	token, _ := getSessionToken(r)
	if err := db.ChangePassword(h.DB, user.Email, req.OldPassword, req.NewPassword, token); err != nil {
		if errors.Is(err, db.ErrWrongPassword) {
			h.Error(w, 400, "Current password is incorrect")
		} else {
			h.ServerError(w, r, err)
		}
		return
	}
	h.Success(w, "Password changed")
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
	auth.SetSessionCookie(w, token)
	h.JSON(w, 200, map[string]any{
		"email": user.Email,
		"role":  user.Role,
		"demo":  user.IsDemo,
		// Lets the UI hide controls the user could not use anyway.
		"canApproveIndent": containsFold(h.Cfg.IndentApprovers, user.Email),
		"canApproveSpec":   containsFold(h.Cfg.SpecApprovers, user.Email),
	})
}

const passwordRule = "Password must be 8-72 characters"

// bcrypt ignores everything past 72 bytes.
func validPassword(p string) bool { return len(p) >= 8 && len(p) <= 72 }
