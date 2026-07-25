package auth

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/H4fizWasabie/pims/internal/db"
)

func Login(database *sql.DB, email, password string) (string, error) {
	user, err := db.GetUserByEmail(database, email)
	if err != nil {
		return "", fmt.Errorf("invalid email or password")
	}
	if !user.CheckPassword(password) {
		return "", fmt.Errorf("invalid email or password")
	}
	return db.CreateSession(database, user.ID)
}

func DeleteSession(d *sql.DB, token string) error {
	return db.DeleteSession(d, token)
}

func ValidateSession(d *sql.DB, token string) (*db.User, error) {
	return db.ValidateSession(d, token)
}

func SetSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "pims_session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400,
	})
}

func ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:   "pims_session",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
}
