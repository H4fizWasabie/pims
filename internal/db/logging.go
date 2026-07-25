package db

import "database/sql"

func LogEvent(d *sql.DB, logType, message, userEmail string) {
	d.Exec(
		`INSERT INTO system_logs (log_type, message, user_email) VALUES ($1, $2, $3)`,
		logType, message, userEmail,
	)
}

func LogError(d *sql.DB, context string, err error, userEmail string) {
	d.Exec(
		`INSERT INTO system_logs (log_type, message, user_email, stack_trace) VALUES ('ERROR', $1, $2, $3)`,
		"["+context+"] "+err.Error(), userEmail, "",
	)
}
