package main

import (
	"log"
	"net/http"

	"github.com/H4fizWasabie/pims/internal/config"
	"github.com/H4fizWasabie/pims/internal/db"
	"github.com/H4fizWasabie/pims/internal/handler"
)

func main() {
	cfg := config.Load()

	database, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer database.Close()

	if err := db.Migrate(database); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	h := &handler.Handler{DB: database, Cfg: cfg}

	mux := http.NewServeMux()

	// Auth routes
	mux.HandleFunc("/pims/api/auth/login", handler.Recover(h.HandleLogin))
	mux.HandleFunc("/pims/api/auth/logout", handler.Recover(h.HandleLogout))
	mux.HandleFunc("/pims/api/auth/me", handler.Recover(h.HandleMe))

	mux.HandleFunc("/pims/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("PIMS API - coming soon"))
	})

	addr := ":" + cfg.Port
	log.Printf("PIMS starting on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server: %v", err)
	}
}
