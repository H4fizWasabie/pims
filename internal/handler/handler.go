package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"strings"

	"github.com/H4fizWasabie/pims/internal/config"
	"github.com/H4fizWasabie/pims/internal/db"
)

type Handler struct {
	DB       *sql.DB
	Cfg      *config.Config
	StaticFS fs.FS
}

func (h *Handler) JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *Handler) Error(w http.ResponseWriter, status int, msg string) {
	h.JSON(w, status, map[string]any{"success": false, "message": msg})
}

func (h *Handler) Success(w http.ResponseWriter, msg string) {
	h.JSON(w, 200, map[string]any{"success": true, "message": msg})
}

func Recover(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("PANIC: %v", err)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(500)
				json.NewEncoder(w).Encode(map[string]any{
					"success": false,
					"message": "Internal server error",
				})
			}
		}()
		next(w, r)
	}
}

func (h *Handler) AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return Recover(func(w http.ResponseWriter, r *http.Request) {
		token, ok := getSessionToken(r)
		if !ok {
			h.Error(w, 401, "Authentication required")
			return
		}
		user, err := db.ValidateSession(h.DB, token)
		if err != nil {
			h.Error(w, 401, "Invalid or expired session")
			return
		}
		ctx := contextWithUser(r.Context(), user)
		next(w, r.WithContext(ctx))
	})
}

func (h *Handler) AdminMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return h.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		user := userFromContext(r.Context())
		if user == nil || !containsFold(h.Cfg.MasterAdmins, user.Email) {
			h.Error(w, 403, "Access Denied: Admin only")
			return
		}
		next(w, r)
	})
}

type ctxKey int

const userCtxKey ctxKey = iota

func contextWithUser(ctx context.Context, u *db.User) context.Context {
	return context.WithValue(ctx, userCtxKey, u)
}

func userFromContext(ctx context.Context) *db.User {
	u, _ := ctx.Value(userCtxKey).(*db.User)
	return u
}

func getSessionToken(r *http.Request) (string, bool) {
	cookie, err := r.Cookie("pims_session")
	if err == nil {
		return cookie.Value, true
	}
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return auth[7:], true
	}
	return "", false
}

func containsFold(list []string, s string) bool {
	for _, v := range list {
		if equalsFold(v, s) {
			return true
		}
	}
	return false
}

func equalsFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 32
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 32
		}
		if ca != cb {
			return false
		}
	}
	return true
}
