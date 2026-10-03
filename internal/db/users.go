package db

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const sessionIdleTimeout = 10 * time.Minute

type User struct {
	ID           int
	Email        string
	PasswordHash string
	Role         string
	IsDemo       bool
}

var ErrWrongPassword = errors.New("wrong password")

// defaultAdminEmail/Password were seeded by early migrations; EnsureAdmin
// replaces that seed and WarnDefaultAdmin flags installs that still have it.
const (
	defaultAdminEmail    = "admin@pims.local"
	defaultAdminPassword = "admin123"
)

func normEmail(email string) string { return strings.ToLower(strings.TrimSpace(email)) }

func CreateUser(d *sql.DB, email, password, role string) (*User, error) {
	email = normEmail(email)
	var taken bool
	if err := d.QueryRow(`SELECT EXISTS(SELECT 1 FROM users WHERE LOWER(email) = $1)`, email).Scan(&taken); err != nil {
		return nil, err
	}
	if taken {
		return nil, fmt.Errorf("user already exists")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	var u User
	err = d.QueryRow(
		`INSERT INTO users (email, password_hash, role) VALUES ($1, $2, $3)
		 RETURNING id, email, password_hash, role`,
		email, string(hash), role,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role)
	return &u, err
}

func GetUserByEmail(d *sql.DB, email string) (*User, error) {
	var u User
	err := d.QueryRow(
		`SELECT id, email, password_hash, role FROM users WHERE LOWER(email) = $1`,
		normEmail(email),
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (u *User) CheckPassword(password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) == nil
}

func CreateSession(d *sql.DB, userID int) (string, error) {
	token := randomToken(32)
	_, err := d.Exec(
		`INSERT INTO sessions (user_id, token, expires_at) VALUES ($1, $2, $3)`,
		userID, token, time.Now().Add(sessionIdleTimeout),
	)
	return token, err
}

func ValidateSession(d *sql.DB, token string) (*User, error) {
	var u User
	err := d.QueryRow(
		`SELECT u.id, u.email, u.password_hash, u.role, COALESCE(s.is_demo, FALSE)
		 FROM sessions s JOIN users u ON s.user_id = u.id
		 WHERE s.token = $1 AND s.expires_at > NOW()`,
		token,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.IsDemo)
	if err != nil {
		return nil, err
	}
	// Slide the expiry at most once a minute instead of writing on every request.
	if _, err := d.Exec(`UPDATE sessions SET expires_at = $1::timestamptz WHERE token = $2 AND expires_at < $1::timestamptz - INTERVAL '1 minute'`, time.Now().Add(sessionIdleTimeout), token); err != nil {
		return nil, err
	}
	return &u, nil
}

// CreateDemoSession seeds a demo user (random password, unusable via login form)
// and returns a session flagged read-only.
func CreateDemoSession(d *sql.DB) (string, error) {
	randomPass := make([]byte, 16)
	rand.Read(randomPass)
	hash, err := bcrypt.GenerateFromPassword(randomPass, bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	if _, err := d.Exec(
		`INSERT INTO users (email, password_hash, role)
		 SELECT 'demo@pims.local', $1, 'user'
		 WHERE NOT EXISTS (SELECT 1 FROM users WHERE email = 'demo@pims.local')`,
		string(hash),
	); err != nil {
		return "", err
	}
	var token string
	err = d.QueryRow(
		`INSERT INTO sessions (user_id, token, expires_at, is_demo)
		 SELECT id, $1, $2, TRUE FROM users WHERE email = 'demo@pims.local'
		 RETURNING token`,
		randomToken(32), time.Now().Add(sessionIdleTimeout),
	).Scan(&token)
	return token, err
}

// PurgeExpiredSessions drops dead sessions (including abandoned demo ones).
func PurgeExpiredSessions(d *sql.DB) error {
	_, err := d.Exec(`DELETE FROM sessions WHERE expires_at < NOW()`)
	return err
}

// EnsureAdmin creates the bootstrap admin if no user with that email exists.
func EnsureAdmin(d *sql.DB, email, password string) error {
	email = normEmail(email)
	var exists bool
	if err := d.QueryRow(`SELECT EXISTS(SELECT 1 FROM users WHERE LOWER(email) = $1)`, email).Scan(&exists); err != nil || exists {
		return err
	}
	_, err := CreateUser(d, email, password, "admin")
	return err
}

// HasDefaultAdmin reports whether the old seeded admin/admin123 login still works.
func HasDefaultAdmin(d *sql.DB) bool {
	u, err := GetUserByEmail(d, defaultAdminEmail)
	return err == nil && u.CheckPassword(defaultAdminPassword)
}

func DeleteSession(d *sql.DB, token string) error {
	_, err := d.Exec(`DELETE FROM sessions WHERE token = $1`, token)
	return err
}

func randomToken(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

type UserRow struct {
	ID        int    `json:"id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	CreatedAt string `json:"createdAt"`
}

func ListUsers(d *sql.DB) ([]UserRow, error) {
	rows, err := d.Query(`SELECT id, email, role, COALESCE(to_char(created_at, 'YYYY-MM-DD HH24:MI'), '') FROM users ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	users := make([]UserRow, 0)
	for rows.Next() {
		var u UserRow
		if err := rows.Scan(&u.ID, &u.Email, &u.Role, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func DeleteUser(d *sql.DB, id int) error {
	_, err := d.Exec(`DELETE FROM users WHERE id = $1`, id)
	return err
}

// ChangePassword sets a new password and signs out every other session of the user.
func ChangePassword(d *sql.DB, email, oldPass, newPass, keepToken string) error {
	u, err := GetUserByEmail(d, email)
	if err != nil {
		return err
	}
	if !u.CheckPassword(oldPass) {
		return ErrWrongPassword
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPass), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`UPDATE users SET password_hash = $1 WHERE id = $2`, string(hash), u.ID); err != nil {
		return err
	}
	if _, err = tx.Exec(`DELETE FROM sessions WHERE user_id = $1 AND token <> $2`, u.ID, keepToken); err != nil {
		return err
	}
	return tx.Commit()
}
