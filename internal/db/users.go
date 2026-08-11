package db

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID           int
	Email        string
	PasswordHash string
	Role         string
	IsDemo       bool
}

func CreateUser(d *sql.DB, email, password, role string) (*User, error) {
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
		`SELECT id, email, password_hash, role FROM users WHERE email = $1`,
		email,
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
		userID, token, time.Now().Add(24*time.Hour),
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
		randomToken(32), time.Now().Add(24*time.Hour),
	).Scan(&token)
	return token, err
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

func ChangePassword(d *sql.DB, email, oldPass, newPass string) error {
	u, err := GetUserByEmail(d, email)
	if err != nil {
		return err
	}
	if !u.CheckPassword(oldPass) {
		return sql.ErrNoRows // reusing this to signal wrong password
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPass), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = d.Exec(`UPDATE users SET password_hash = $1 WHERE email = $2`, string(hash), email)
	return err
}
