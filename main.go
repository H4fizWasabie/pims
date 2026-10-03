package main

import (
	"context"
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/H4fizWasabie/pims/internal/config"
	"github.com/H4fizWasabie/pims/internal/db"
	"github.com/H4fizWasabie/pims/internal/handler"
)

//go:embed static
var staticFiles embed.FS

func main() {
	cfg := config.Load()

	database, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer database.Close()

	stockDatabase, err := db.ConnectProcura(cfg.ProcuraDBPath)
	if err != nil {
		log.Fatalf("procura db: %v", err)
	}
	defer stockDatabase.Close()

	if err := db.Migrate(database); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	if cfg.AdminEmail != "" && cfg.AdminPassword != "" {
		if err := db.EnsureAdmin(database, cfg.AdminEmail, cfg.AdminPassword); err != nil {
			log.Fatalf("bootstrap admin: %v", err)
		}
	}
	if db.HasDefaultAdmin(database) {
		log.Printf("WARNING: admin@pims.local still has the default password. Change it or delete the user.")
	}
	go func() {
		for {
			if err := db.PurgeExpiredSessions(database); err != nil {
				log.Printf("purge sessions: %v", err)
			}
			time.Sleep(10 * time.Minute)
		}
	}()

	// Demo sessions read fabricated data from an isolated schema, never
	// the real item master / Procura stock.
	if err := db.EnsureDemoSchema(database); err != nil {
		log.Fatalf("demo schema: %v", err)
	}
	demoDatabase, err := db.ConnectDemo(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("demo db: %v", err)
	}
	defer demoDatabase.Close()

	staticFS, _ := fs.Sub(staticFiles, "static")
	h := &handler.Handler{DB: database, StockDB: stockDatabase, DemoDB: demoDatabase, Cfg: cfg, StaticFS: staticFS}

	mux := h.Routes()

	addr := ":" + cfg.Port
	log.Printf("PIMS starting on %s", addr)
	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second, // photos/spreadsheets upload as JSON
		WriteTimeout:      90 * time.Second, // OCR calls can take ~45s
		IdleTimeout:       120 * time.Second,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx) // let in-flight requests finish on restart
	}()
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server: %v", err)
	}
}
