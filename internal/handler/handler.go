package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"reflect"
	"strings"

	"github.com/H4fizWasabie/pims/internal/auth"
	"github.com/H4fizWasabie/pims/internal/config"
	"github.com/H4fizWasabie/pims/internal/db"
)

type Handler struct {
	DB      *sql.DB
	StockDB *sql.DB
	// DemoDB, when set, is the only database demo sessions may read from.
	// It points at the fabricated-data demo schema. nil keeps the legacy
	// behaviour (demo sessions read the real DB) for local/test setups.
	DemoDB   *sql.DB
	Cfg      *config.Config
	StaticFS fs.FS
}

// databases returns the (data, stock) DB pair a request should read from.
// Demo sessions never see real data: they are routed to the demo schema
// (with StockDB nil, so stock lookups fall back to the demo inventory
// table instead of the real Procura stock source).
func (h *Handler) databases(ctx context.Context) (*sql.DB, *sql.DB) {
	if h.DemoDB != nil {
		if u := userFromContext(ctx); u != nil && u.IsDemo {
			return h.DemoDB, nil
		}
	}
	return h.DB, h.StockDB
}

func (h *Handler) JSON(w http.ResponseWriter, status int, data any) {
	// A nil slice encodes as null, which the UI cannot iterate; send [] instead.
	if v := reflect.ValueOf(data); v.Kind() == reflect.Slice && v.IsNil() {
		data = []any{}
	}
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

// ServerError logs the real error (and records it in system_logs) and sends
// the client a generic message, so SQL/driver text never leaks.
func (h *Handler) ServerError(w http.ResponseWriter, r *http.Request, err error) {
	email := ""
	if u := userFromContext(r.Context()); u != nil {
		email = u.Email
	}
	log.Printf("ERROR %s %s: %v", r.Method, r.URL.Path, err)
	db.LogError(h.DB, r.Method+" "+r.URL.Path, err, email)
	h.Error(w, 500, "Server error. Please try again.")
}

// businessError sends db.ValidationError messages as a 400 (they are written for
// users) and treats anything else as a server fault with a generic message.
func (h *Handler) businessError(w http.ResponseWriter, r *http.Request, err error) {
	var ve db.ValidationError
	if errors.As(err, &ve) {
		h.Error(w, 400, ve.Error())
		return
	}
	h.ServerError(w, r, err)
}

const (
	maxBody    = 1 << 20
	maxBigBody = 25 << 20
)

// bigBody paths carry spreadsheets or base64 photos.
var bigBody = map[string]bool{"/api/master/replace": true, "/api/stocktake/analyze-image": true}

// decode reads a JSON body into v; false means a 400 was already sent.
func (h *Handler) decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		h.Error(w, 400, "Invalid request")
		return false
	}
	return true
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
		auth.SetSessionCookie(w, token)
		limit := int64(maxBody)
		if bigBody[r.URL.Path] {
			limit = maxBigBody
		}
		r.Body = http.MaxBytesReader(w, r.Body, limit)
		// CSRF: a cross-site form cannot send a JSON content type without a
		// CORS preflight, so requiring it on writes blocks forged requests.
		if r.Method != http.MethodGet && r.Method != http.MethodHead &&
			!strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			h.Error(w, 415, "Content-Type must be application/json")
			return
		}
		// All write routes are POST; demo sessions may only read.
		if user.IsDemo && r.Method != http.MethodGet {
			h.Error(w, 403, "Demo mode: read-only browsing")
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

func containsFold(list []string, s string) bool { return auth.Contains(list, s) }
