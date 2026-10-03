package handler

import "net/http"

// Routes is the single route table, shared by main and the tests.
func (h *Handler) Routes() *http.ServeMux {
	mux := http.NewServeMux()

	// Auth
	mux.HandleFunc("/api/auth/login", Recover(h.HandleLogin))
	mux.HandleFunc("/api/auth/demo", Recover(h.HandleDemoLogin))
	mux.HandleFunc("/api/auth/logout", Recover(h.HandleLogout))
	mux.HandleFunc("/api/auth/me", Recover(h.HandleMe))
	mux.HandleFunc("/api/auth/change-password", Recover(h.AuthMiddleware(h.HandleChangePassword)))

	// Master
	mux.HandleFunc("/api/master/chunk", Recover(h.AuthMiddleware(h.HandleMasterChunk)))
	mux.HandleFunc("/api/master/search", Recover(h.AuthMiddleware(h.HandleMasterSearch)))
	mux.HandleFunc("/api/master/replace", Recover(h.AdminMiddleware(h.HandleMasterReplace)))
	mux.HandleFunc("/api/master/all", Recover(h.AuthMiddleware(h.HandleMasterAll)))

	// Inventory
	mux.HandleFunc("/api/inventory/chunk", Recover(h.AuthMiddleware(h.HandleInventoryChunk)))
	mux.HandleFunc("/api/inventory/replace", Recover(h.AdminMiddleware(h.HandleInventoryReplace)))

	// Indents
	mux.HandleFunc("/api/indent/master-data", Recover(h.AuthMiddleware(h.HandleIndentMasterData)))
	mux.HandleFunc("/api/indent/submit", Recover(h.AuthMiddleware(h.HandleIndentSubmit)))
	mux.HandleFunc("/api/indent/approve", Recover(h.AuthMiddleware(h.HandleIndentApprove)))
	mux.HandleFunc("/api/indent/bulk", Recover(h.AuthMiddleware(h.HandleIndentBulk)))
	mux.HandleFunc("/api/indent/reject", Recover(h.AuthMiddleware(h.HandleIndentReject)))

	// GRN
	mux.HandleFunc("/api/grn/master-data", Recover(h.AuthMiddleware(h.HandleGRNMasterData)))
	mux.HandleFunc("/api/grn/submit", Recover(h.AuthMiddleware(h.HandleGRNSubmit)))

	// Stock Take
	mux.HandleFunc("/api/stocktake/submit", Recover(h.AuthMiddleware(h.HandleStockTakeSubmit)))
	mux.HandleFunc("/api/stocktake/today", Recover(h.AuthMiddleware(h.HandleStockTakeToday)))
	mux.HandleFunc("/api/stocktake/history", Recover(h.AuthMiddleware(h.HandleStockTakeHistory)))
	mux.HandleFunc("/api/stocktake/analyze-image", Recover(h.AuthMiddleware(h.HandleStockTakeAnalyzeImage)))

	// Disposal
	mux.HandleFunc("/api/disposal/search", Recover(h.AuthMiddleware(h.HandleDisposalSearch)))
	mux.HandleFunc("/api/disposal/submit", Recover(h.AuthMiddleware(h.HandleDisposalSubmit)))

	// Analysis
	mux.HandleFunc("/api/analysis/run", Recover(h.AuthMiddleware(h.HandleAnalysisRun)))
	mux.HandleFunc("/api/analysis/today", Recover(h.AuthMiddleware(h.HandleAnalysisToday)))

	// Expiry
	mux.HandleFunc("/api/expiry/list", Recover(h.AuthMiddleware(h.HandleExpiryList)))
	mux.HandleFunc("/api/expiry/update-remark", Recover(h.AuthMiddleware(h.HandleExpiryUpdateRemark)))

	// Specs
	mux.HandleFunc("/api/spec/submit", Recover(h.AuthMiddleware(h.HandleSpecSubmit)))
	mux.HandleFunc("/api/spec/approve", Recover(h.AuthMiddleware(h.HandleSpecApprove)))
	mux.HandleFunc("/api/spec/reject", Recover(h.AuthMiddleware(h.HandleSpecReject)))

	// Dashboard
	mux.HandleFunc("/api/dashboard/summary", Recover(h.AuthMiddleware(h.HandleDashboardSummary)))

	// Users (admin only)
	mux.HandleFunc("/api/users", Recover(h.AdminMiddleware(h.HandleUsersList)))
	mux.HandleFunc("/api/users/create", Recover(h.AdminMiddleware(h.HandleUsersCreate)))
	mux.HandleFunc("/api/users/delete", Recover(h.AdminMiddleware(h.HandleUsersDelete)))

	// Order
	mux.HandleFunc("/api/order/generate", Recover(h.AuthMiddleware(h.HandleOrderGenerate)))
	mux.HandleFunc("/api/order/list", Recover(h.AuthMiddleware(h.HandleOrderList)))
	mux.HandleFunc("/api/order/tick", Recover(h.AuthMiddleware(h.HandleOrderTick)))

	// SPA
	mux.HandleFunc("/", Recover(h.HandleSPA))
	return mux
}
