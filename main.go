package main

import (
	"log"
	"net/http"

	"github.com/H4fizWasabie/pims/internal/config"
	"github.com/H4fizWasabie/pims/internal/db"
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

	mux := http.NewServeMux()
	mux.HandleFunc("/pims/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("PIMS API - coming soon"))
	})

	addr := ":" + cfg.Port
	log.Printf("PIMS starting on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server: %v", err)
	}
}
