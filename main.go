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
	mux.HandleFunc("/api/auth/login", handler.Recover(h.HandleLogin))
	mux.HandleFunc("/api/auth/logout", handler.Recover(h.HandleLogout))
	mux.HandleFunc("/api/auth/me", handler.Recover(h.HandleMe))
	mux.HandleFunc("/api/auth/change-password", handler.Recover(h.AuthMiddleware(h.HandleChangePassword)))

	// Master
	mux.HandleFunc("/api/master/chunk", handler.Recover(h.HandleMasterChunk))
	mux.HandleFunc("/api/master/search", handler.Recover(h.HandleMasterSearch))
	mux.HandleFunc("/api/master/replace", handler.Recover(h.AdminMiddleware(h.HandleMasterReplace)))
	mux.HandleFunc("/api/master/all", handler.Recover(h.HandleMasterAll))

	// Inventory
	mux.HandleFunc("/api/inventory/chunk", handler.Recover(h.HandleInventoryChunk))
	mux.HandleFunc("/api/inventory/replace", handler.Recover(h.AdminMiddleware(h.HandleInventoryReplace)))

	// Indents
	mux.HandleFunc("/api/indent/master-data", handler.Recover(h.HandleIndentMasterData))
	mux.HandleFunc("/api/indent/submit", handler.Recover(h.AuthMiddleware(h.HandleIndentSubmit)))
	mux.HandleFunc("/api/indent/approve", handler.Recover(h.AuthMiddleware(h.HandleIndentApprove)))
	mux.HandleFunc("/api/indent/reject", handler.Recover(h.AuthMiddleware(h.HandleIndentReject)))

	// GRN
	mux.HandleFunc("/api/grn/master-data", handler.Recover(h.HandleGRNMasterData))
	mux.HandleFunc("/api/grn/submit", handler.Recover(h.AuthMiddleware(h.HandleGRNSubmit)))

	// Stock Take
	mux.HandleFunc("/api/stocktake/submit", handler.Recover(h.AuthMiddleware(h.HandleStockTakeSubmit)))
	mux.HandleFunc("/api/stocktake/today", handler.Recover(h.HandleStockTakeToday))
	mux.HandleFunc("/api/stocktake/history", handler.Recover(h.HandleStockTakeHistory))
	mux.HandleFunc("/api/stocktake/analyze-image", handler.Recover(h.AuthMiddleware(h.HandleStockTakeAnalyzeImage)))

	// Disposal
	mux.HandleFunc("/api/disposal/search", handler.Recover(h.HandleDisposalSearch))
	mux.HandleFunc("/api/disposal/submit", handler.Recover(h.AuthMiddleware(h.HandleDisposalSubmit)))

	// Analysis
	mux.HandleFunc("/api/analysis/run", handler.Recover(h.HandleAnalysisRun))
	mux.HandleFunc("/api/analysis/today", handler.Recover(h.HandleAnalysisToday))

	// Expiry
	mux.HandleFunc("/api/expiry/list", handler.Recover(h.HandleExpiryList))
	mux.HandleFunc("/api/expiry/update-remark", handler.Recover(h.AuthMiddleware(h.HandleExpiryUpdateRemark)))

	// Specs
	mux.HandleFunc("/api/spec/submit", handler.Recover(h.AuthMiddleware(h.HandleSpecSubmit)))
	mux.HandleFunc("/api/spec/approve", handler.Recover(h.AuthMiddleware(h.HandleSpecApprove)))
	mux.HandleFunc("/api/spec/reject", handler.Recover(h.AuthMiddleware(h.HandleSpecReject)))

	// Dashboard
	mux.HandleFunc("/api/dashboard/summary", handler.Recover(h.AuthMiddleware(h.HandleDashboardSummary)))

	// Users (admin only)
	mux.HandleFunc("/api/users", handler.Recover(h.AdminMiddleware(h.HandleUsersList)))
	mux.HandleFunc("/api/users/create", handler.Recover(h.AdminMiddleware(h.HandleUsersCreate)))
	mux.HandleFunc("/api/users/delete", handler.Recover(h.AdminMiddleware(h.HandleUsersDelete)))

	// Order
	mux.HandleFunc("/api/order/prf-number", handler.Recover(h.AuthMiddleware(h.HandleOrderPRFNumber)))
	mux.HandleFunc("/api/order/generate", handler.Recover(h.AuthMiddleware(h.HandleOrderGenerate)))
	mux.HandleFunc("/api/order/list", handler.Recover(h.HandleOrderList))
	mux.HandleFunc("/api/order/tick", handler.Recover(h.AuthMiddleware(h.HandleOrderTick)))

	// SPA
	mux.HandleFunc("/", handler.Recover(h.HandleSPA))

	addr := ":" + cfg.Port
	log.Printf("PIMS starting on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server: %v", err)
	}
}
