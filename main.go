package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"

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

	if err := db.Migrate(database); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	staticFS, _ := fs.Sub(staticFiles, "static")
	h := &handler.Handler{DB: database, Cfg: cfg, StaticFS: staticFS}

	mux := http.NewServeMux()

	// Auth
	mux.HandleFunc("/pims/api/auth/login", handler.Recover(h.HandleLogin))
	mux.HandleFunc("/pims/api/auth/logout", handler.Recover(h.HandleLogout))
	mux.HandleFunc("/pims/api/auth/me", handler.Recover(h.HandleMe))

	// Master
	mux.HandleFunc("/pims/api/master/chunk", handler.Recover(h.HandleMasterChunk))
	mux.HandleFunc("/pims/api/master/search", handler.Recover(h.HandleMasterSearch))
	mux.HandleFunc("/pims/api/master/replace", handler.Recover(h.AdminMiddleware(h.HandleMasterReplace)))
	mux.HandleFunc("/pims/api/master/all", handler.Recover(h.HandleMasterAll))

	// Inventory
	mux.HandleFunc("/pims/api/inventory/chunk", handler.Recover(h.HandleInventoryChunk))
	mux.HandleFunc("/pims/api/inventory/replace", handler.Recover(h.AdminMiddleware(h.HandleInventoryReplace)))

	// Indents
	mux.HandleFunc("/pims/api/indent/master-data", handler.Recover(h.HandleIndentMasterData))
	mux.HandleFunc("/pims/api/indent/submit", handler.Recover(h.AuthMiddleware(h.HandleIndentSubmit)))
	mux.HandleFunc("/pims/api/indent/approve", handler.Recover(h.AuthMiddleware(h.HandleIndentApprove)))
	mux.HandleFunc("/pims/api/indent/reject", handler.Recover(h.AuthMiddleware(h.HandleIndentReject)))

	// GRN
	mux.HandleFunc("/pims/api/grn/master-data", handler.Recover(h.HandleGRNMasterData))
	mux.HandleFunc("/pims/api/grn/submit", handler.Recover(h.AuthMiddleware(h.HandleGRNSubmit)))

	// Stock Take
	mux.HandleFunc("/pims/api/stocktake/submit", handler.Recover(h.AuthMiddleware(h.HandleStockTakeSubmit)))
	mux.HandleFunc("/pims/api/stocktake/today", handler.Recover(h.HandleStockTakeToday))
	mux.HandleFunc("/pims/api/stocktake/analyze-image", handler.Recover(h.AuthMiddleware(h.HandleStockTakeAnalyzeImage)))

	// Disposal
	mux.HandleFunc("/pims/api/disposal/search", handler.Recover(h.HandleDisposalSearch))
	mux.HandleFunc("/pims/api/disposal/submit", handler.Recover(h.AuthMiddleware(h.HandleDisposalSubmit)))

	// Analysis
	mux.HandleFunc("/pims/api/analysis/run", handler.Recover(h.HandleAnalysisRun))
	mux.HandleFunc("/pims/api/analysis/today", handler.Recover(h.HandleAnalysisToday))

	// Expiry
	mux.HandleFunc("/pims/api/expiry/list", handler.Recover(h.HandleExpiryList))
	mux.HandleFunc("/pims/api/expiry/update-remark", handler.Recover(h.AuthMiddleware(h.HandleExpiryUpdateRemark)))

	// Specs
	mux.HandleFunc("/pims/api/spec/submit", handler.Recover(h.AuthMiddleware(h.HandleSpecSubmit)))
	mux.HandleFunc("/pims/api/spec/approve", handler.Recover(h.AuthMiddleware(h.HandleSpecApprove)))
	mux.HandleFunc("/pims/api/spec/reject", handler.Recover(h.AuthMiddleware(h.HandleSpecReject)))

	// Dashboard
	mux.HandleFunc("/pims/api/dashboard/summary", handler.Recover(h.HandleDashboardSummary))

	// Order
	mux.HandleFunc("/pims/api/order/prf-number", handler.Recover(h.AuthMiddleware(h.HandleOrderPRFNumber)))
	mux.HandleFunc("/pims/api/order/generate", handler.Recover(h.AuthMiddleware(h.HandleOrderGenerate)))

	// SPA
	mux.HandleFunc("/pims/", handler.Recover(h.HandleSPA))

	addr := ":" + cfg.Port
	log.Printf("PIMS starting on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server: %v", err)
	}
}
