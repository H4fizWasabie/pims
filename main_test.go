package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/H4fizWasabie/pims/internal/config"
	"github.com/H4fizWasabie/pims/internal/db"
	"github.com/H4fizWasabie/pims/internal/handler"
)

// testServer wraps a test HTTP server with DB access
type testServer struct {
	*httptest.Server
	db  *sql.DB
	cfg *config.Config
}

type sqlDB = *sql.DB // alias for readability

func newTestServer(t *testing.T) *testServer {
	t.Helper()

	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		databaseURL = os.Getenv("DATABASE_URL")
	}
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL or DATABASE_URL not set")
	}

	database, err := db.Connect(databaseURL)
	if err != nil {
		t.Fatalf("db connect: %v", err)
	}

	if err := db.Migrate(database); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	cfg := &config.Config{
		Port:            "0",
		DatabaseURL:     databaseURL,
		SessionSecret:   "test-secret",
		OpenRouterAPIKey: os.Getenv("OPENROUTER_API_KEY"),
		OpenRouterModel: "google/gemma-4-31b-it:free",
		GeminiAPIKey:    os.Getenv("GEMINI_API_KEY"),
		IndentApprovers: []string{"admin@pims.local", "approver@test.com", "kisame350@gmail.com"},
		SpecApprovers:   []string{"admin@pims.local", "spec@test.com", "kisame350@gmail.com"},
		MasterAdmins:    []string{"admin@pims.local", "admin@test.com", "kisame350@gmail.com"},
	}

	h := &handler.Handler{DB: database, Cfg: cfg}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/auth/login", handler.Recover(h.HandleLogin))
	mux.HandleFunc("/api/auth/logout", handler.Recover(h.HandleLogout))
	mux.HandleFunc("/api/auth/me", handler.Recover(h.HandleMe))
	mux.HandleFunc("/api/master/chunk", handler.Recover(h.HandleMasterChunk))
	mux.HandleFunc("/api/master/search", handler.Recover(h.HandleMasterSearch))
	mux.HandleFunc("/api/master/replace", handler.Recover(h.AdminMiddleware(h.HandleMasterReplace)))
	mux.HandleFunc("/api/master/all", handler.Recover(h.HandleMasterAll))
	mux.HandleFunc("/api/inventory/chunk", handler.Recover(h.HandleInventoryChunk))
	mux.HandleFunc("/api/inventory/replace", handler.Recover(h.AdminMiddleware(h.HandleInventoryReplace)))
	mux.HandleFunc("/api/indent/master-data", handler.Recover(h.HandleIndentMasterData))
	mux.HandleFunc("/api/indent/submit", handler.Recover(h.AuthMiddleware(h.HandleIndentSubmit)))
	mux.HandleFunc("/api/indent/approve", handler.Recover(h.AuthMiddleware(h.HandleIndentApprove)))
	mux.HandleFunc("/api/indent/reject", handler.Recover(h.AuthMiddleware(h.HandleIndentReject)))
	mux.HandleFunc("/api/grn/master-data", handler.Recover(h.HandleGRNMasterData))
	mux.HandleFunc("/api/grn/submit", handler.Recover(h.AuthMiddleware(h.HandleGRNSubmit)))
	mux.HandleFunc("/api/stocktake/submit", handler.Recover(h.AuthMiddleware(h.HandleStockTakeSubmit)))
	mux.HandleFunc("/api/stocktake/today", handler.Recover(h.HandleStockTakeToday))
	mux.HandleFunc("/api/disposal/search", handler.Recover(h.HandleDisposalSearch))
	mux.HandleFunc("/api/disposal/submit", handler.Recover(h.AuthMiddleware(h.HandleDisposalSubmit)))
	mux.HandleFunc("/api/analysis/run", handler.Recover(h.HandleAnalysisRun))
	mux.HandleFunc("/api/analysis/today", handler.Recover(h.HandleAnalysisToday))
	mux.HandleFunc("/api/expiry/list", handler.Recover(h.HandleExpiryList))
	mux.HandleFunc("/api/expiry/update-remark", handler.Recover(h.AuthMiddleware(h.HandleExpiryUpdateRemark)))
	mux.HandleFunc("/api/spec/submit", handler.Recover(h.AuthMiddleware(h.HandleSpecSubmit)))
	mux.HandleFunc("/api/spec/approve", handler.Recover(h.AuthMiddleware(h.HandleSpecApprove)))
	mux.HandleFunc("/api/spec/reject", handler.Recover(h.AuthMiddleware(h.HandleSpecReject)))
	mux.HandleFunc("/api/dashboard/summary", handler.Recover(h.AuthMiddleware(h.HandleDashboardSummary)))
	mux.HandleFunc("/api/order/prf-number", handler.Recover(h.AuthMiddleware(h.HandleOrderPRFNumber)))
	mux.HandleFunc("/api/order/generate", handler.Recover(h.AuthMiddleware(h.HandleOrderGenerate)))

	ts := httptest.NewServer(mux)
	return &testServer{Server: ts, db: database, cfg: cfg}
}

func (ts *testServer) login(t *testing.T, email, password string) string {
	t.Helper()
	body, _ := JSON(map[string]string{"email": email, "password": password})
	resp := ts.post("/api/auth/login", body, "")
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("login failed: %s", readBody(resp))
	}
	cookies := resp.Header.Values("Set-Cookie")
	for _, c := range cookies {
		if strings.HasPrefix(c, "pims_session=") {
			return strings.Split(c, ";")[0]
		}
	}
	t.Fatal("no session cookie")
	return ""
}

func (ts *testServer) get(path, cookie string) *http.Response {
	req, _ := http.NewRequest("GET", ts.URL+path, nil)
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	resp, _ := http.DefaultClient.Do(req)
	return resp
}

func (ts *testServer) post(path, body, cookie string) *http.Response {
	req, _ := http.NewRequest("POST", ts.URL+path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	resp, _ := http.DefaultClient.Do(req)
	return resp
}

func JSON(v any) (string, error) {
	b, err := json.Marshal(v)
	return string(b), err
}

func readBody(resp *http.Response) string {
	b, _ := io.ReadAll(resp.Body)
	return string(b)
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	s, err := JSON(v)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func assertStatus(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	if resp.StatusCode != want {
		t.Errorf("status: want %d, got %d (%s)", want, resp.StatusCode, readBody(resp))
	}
}

func assertSuccess(t *testing.T, resp *http.Response) {
	t.Helper()
	var m map[string]any
	json.NewDecoder(resp.Body).Decode(&m)
	if s, ok := m["success"]; !ok || s != true {
		t.Errorf("expected success: %v", m)
	}
}

func assertJSONKey(t *testing.T, resp *http.Response, key string, want any) {
	t.Helper()
	body := readBody(resp)
	var m map[string]any
	json.Unmarshal([]byte(body), &m)
	got, ok := m[key]
	if !ok {
		t.Errorf("key %q not found in %s", key, body)
		return
	}
	if got != want {
		t.Errorf("key %q: want %v, got %v", key, want, got)
	}
}

// --- TESTS ---

func TestAuthLogin(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	defer ts.db.Close()

	// Admin login (default user from migration)
	cookie := ts.login(t, "admin@pims.local", "admin123")

	// Verify session
	resp := ts.get("/api/auth/me", cookie)
	assertStatus(t, resp, 200)
	assertJSONKey(t, resp, "email", "admin@pims.local")
	// re-request: assertJSONKey consumes the body
	resp = ts.get("/api/auth/me", cookie)
	assertJSONKey(t, resp, "role", "admin")

	// Bad login
	resp = ts.post("/api/auth/login", mustJSON(t, map[string]string{
		"email": "admin@pims.local", "password": "wrong",
	}), "")
	assertStatus(t, resp, 401)

	// Logout
	resp = ts.post("/api/auth/logout", "", cookie)
	assertStatus(t, resp, 200)
}

func TestMasterItems(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	defer ts.db.Close()

	cookie := ts.login(t, "admin@pims.local", "admin123")

	// Replace master data (admin only)
	data := [][]string{
		{"STK001", "Paracetamol 500mg", "TAB", "Pharmacy", "0.50", "MedSupply Co", "Available"},
		{"STK002", "Syringe 5ml", "PCS", "Lab", "0.25", "LabEquip Ltd", "Available"},
	}
	resp := ts.post("/api/master/replace", mustJSON(t, data), cookie)
	assertSuccess(t, resp)

	// Get all
	resp = ts.get("/api/master/all", cookie)
	assertStatus(t, resp, 200)
	var master []map[string]any
	json.NewDecoder(resp.Body).Decode(&master)
	if len(master) == 0 {
		t.Errorf("master/all: expected items, got %d", len(master))
	}

	// Search
	resp = ts.get("/api/master/search?q=para", cookie)
	assertStatus(t, resp, 200)

	// Pagination
	resp = ts.get("/api/master/chunk?page=0&pageSize=10", cookie)
	assertStatus(t, resp, 200)
}

func TestInventoryReplace(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	defer ts.db.Close()

	cookie := ts.login(t, "admin@pims.local", "admin123")

	// First add master items (FK constraint)
	data := [][]string{
		{"STK001", "Item A", "TAB", "G1", "1.00", "Sup1", "Available"},
	}
	ts.post("/api/master/replace", mustJSON(t, data), cookie)

	// Replace inventory
	invData := [][]string{
		{"STK001", "Item A", "50"},
	}
	resp := ts.post("/api/inventory/replace", mustJSON(t, invData), cookie)
	assertSuccess(t, resp)

	// Get chunk
	resp = ts.get("/api/inventory/chunk?page=0&pageSize=10", cookie)
	assertStatus(t, resp, 200)
}

func TestIndentSubmitAndApprove(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	defer ts.db.Close()

	cookie := ts.login(t, "admin@pims.local", "admin123")

	// Setup: add master items and inventory
	master := [][]string{{"STK001", "Item X", "BOX", "G1", "5.00", "Sup1", "Available"}}
	ts.post("/api/master/replace", mustJSON(t, master), cookie)
	ts.post("/api/inventory/replace", mustJSON(t, [][]string{{"STK001", "Item X", "100"}}), cookie)

	// Submit indent
	indent := map[string]any{
		"requester": "Lab",
		"items": []map[string]any{
			{"itemName": "Item X", "stockId": "STK001", "uom": "BOX", "reqQty": 10},
		},
	}
	resp := ts.post("/api/indent/submit", mustJSON(t, indent), cookie)
	assertSuccess(t, resp)

	// Get the indent row ID from DB
	var indentID int
	ts.db.QueryRow("SELECT id FROM indents WHERE status = 'Pending' ORDER BY id DESC LIMIT 1").Scan(&indentID)
	if indentID == 0 {
		t.Fatal("no pending indent found")
	}

	// Approve (needs indent_approver role)
	// admin@pims.local is also in IndentApprovers list
	approve := map[string]any{
		"indentRowIndex": indentID,
		"stockId":        "STK001",
		"reqQty":         10,
	}
	resp = ts.post("/api/indent/approve", mustJSON(t, approve), cookie)
	assertSuccess(t, resp)

	// Verify stock deducted
	var stock float64
	ts.db.QueryRow("SELECT current_stock FROM inventory WHERE stock_id = 'STK001'").Scan(&stock)
	if stock != 90 {
		t.Errorf("stock: want 90, got %v", stock)
	}
}

func TestGRNSubmit(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	defer ts.db.Close()

	cookie := ts.login(t, "admin@pims.local", "admin123")

	// Setup master items
	master := [][]string{{"STK001", "Item X", "BOX", "G1", "5.00", "Sup1", "Available"}}
	ts.post("/api/master/replace", mustJSON(t, master), cookie)

	// Submit GRN
	grn := map[string]any{
		"supplier":        "Test Supplier",
		"doDate":          "2025-01-01",
		"invNo":           "INV-001",
		"poNo":            "PO-001",
		"submissionToken": "test-token-001",
		"items": []map[string]any{
			{"itemName": "Item X", "qtyPo": 10, "qtyDo": 10, "qtyInv": 10, "uom": "BOX", "batch": "B001", "status": "Match", "remarks": ""},
		},
	}
	resp := ts.post("/api/grn/submit", mustJSON(t, grn), cookie)
	assertSuccess(t, resp)

	// Double entry check
	resp = ts.post("/api/grn/submit", mustJSON(t, grn), cookie)
	assertStatus(t, resp, 409) // conflict
}

func TestStockTake(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	defer ts.db.Close()

	cookie := ts.login(t, "admin@pims.local", "admin123")

	entry := map[string]any{
		"location": "Pharmacy",
		"stockId":  "STK001",
		"itemName": "Test Item",
		"uom":      "BOX",
		"group":    "G1",
		"cost":     5.00,
		"qty":      10,
		"batch":    "B001",
		"expiry":   "01/12/2026",
	}
	resp := ts.post("/api/stocktake/submit", mustJSON(t, entry), cookie)
	assertSuccess(t, resp)

	// Get today
	resp = ts.get("/api/stocktake/today", cookie)
	assertStatus(t, resp, 200)
}

func TestDisposal(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	defer ts.db.Close()

	cookie := ts.login(t, "admin@pims.local", "admin123")

	// Setup
	master := [][]string{{"STK001", "Item X", "BOX", "G1", "5.00", "Sup1", "Available"}}
	ts.post("/api/master/replace", mustJSON(t, master), cookie)
	ts.post("/api/inventory/replace", mustJSON(t, [][]string{{"STK001", "Item X", "50"}}), cookie)

	// Also add expiry tracking so disposal search works
	ts.db.Exec(`INSERT INTO expiry_tracking (stock_id, item_name, batch_no, expiry_date, uom)
		VALUES ('STK001', 'Item X', 'B001', '2026-12-01', 'BOX')
		ON CONFLICT (stock_id, batch_no) DO NOTHING`)

	// Search
	resp := ts.get("/api/disposal/search?q=B001", cookie)
	assertStatus(t, resp, 200)

	// Submit disposal
	disposal := map[string]any{
		"stockId":  "STK001",
		"itemName": "Item X",
		"qty":      5,
		"reason":   "Expired",
		"remarks":  "test",
		"batch":    "B001",
		"cost":     5.00,
	}
	resp = ts.post("/api/disposal/submit", mustJSON(t, disposal), cookie)
	assertSuccess(t, resp)

	// Verify stock deducted
	var stock float64
	ts.db.QueryRow("SELECT current_stock FROM inventory WHERE stock_id = 'STK001'").Scan(&stock)
	if stock != 45 {
		t.Errorf("stock: want 45, got %v", stock)
	}
}

func TestExpiryList(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	defer ts.db.Close()

	cookie := ts.login(t, "admin@pims.local", "admin123")

	// Add an expiry entry
	ts.db.Exec(`INSERT INTO expiry_tracking (stock_id, item_name, batch_no, expiry_date, uom)
		VALUES ('STK001', 'Test Item', 'B001', CURRENT_DATE + 30, 'BOX')
		ON CONFLICT (stock_id, batch_no) DO NOTHING`)

	resp := ts.get("/api/expiry/list?page=1&pageSize=20", cookie)
	assertStatus(t, resp, 200)

	// Update remark
	remark := map[string]any{"rowIndex": 1, "remark": "Test remark"}
	resp = ts.post("/api/expiry/update-remark", mustJSON(t, remark), cookie)
	assertSuccess(t, resp)
}

func TestSpecRequest(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	defer ts.db.Close()

	cookie := ts.login(t, "admin@pims.local", "admin123")

	// Submit spec
	spec := map[string]any{
		"itemName":      "New Test Item",
		"category":      "Pharmacy",
		"uom":           "BOX",
		"estCost":       10.00,
		"justification": "Needed for testing",
	}
	resp := ts.post("/api/spec/submit", mustJSON(t, spec), cookie)
	assertSuccess(t, resp)

	// Get the row ID
	var specID int
	ts.db.QueryRow("SELECT id FROM new_item_requests ORDER BY id DESC LIMIT 1").Scan(&specID)

	// Approve (admin is also spec approver in test config)
	approve := map[string]any{"rowIndex": specID}
	resp = ts.post("/api/spec/approve", mustJSON(t, approve), cookie)
	assertSuccess(t, resp)
}

func TestDashboard(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	defer ts.db.Close()

	cookie := ts.login(t, "admin@pims.local", "admin123")

	resp := ts.get("/api/dashboard/summary", cookie)
	assertStatus(t, resp, 200)
}

func TestOrderPRF(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	defer ts.db.Close()

	cookie := ts.login(t, "admin@pims.local", "admin123")

	resp := ts.get("/api/order/prf-number", cookie)
	assertStatus(t, resp, 200)
	var prf struct {
		PrfNo string `json:"prfNo"`
	}
	json.NewDecoder(resp.Body).Decode(&prf)
	if prf.PrfNo == "" {
		t.Errorf("prfNo: expected non-empty PRF number")
	}

	// Test order generate (PDF deferred, returns PRF number)
	orderData := map[string]any{
		"department": "Lab",
		"items": []map[string]any{
			{"stockId": "STK001", "itemName": "Item", "uom": "BOX", "cost": 5.0, "qty": 10, "supplier": "Sup1", "reason": "test"},
		},
	}
	resp = ts.post("/api/order/generate", mustJSON(t, orderData), cookie)
	assertSuccess(t, resp)
}

func TestAnalysis(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	defer ts.db.Close()

	cookie := ts.login(t, "admin@pims.local", "admin123")

	// Add a stock take entry so analysis has data
	ts.db.Exec(`INSERT INTO stock_takes (take_date, location, stock_id, item_name, physical_qty)
		VALUES (CURRENT_DATE, 'Test', 'STK001', 'Item', 10)`)

	resp := ts.post("/api/analysis/run", "", cookie)
	assertStatus(t, resp, 200)

	resp = ts.get("/api/analysis/today", cookie)
	assertStatus(t, resp, 200)
}

func TestSystemLogs(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	defer ts.db.Close()

	db.LogEvent(ts.db, "TEST", "Integration test running", "test@test.com")
	db.LogError(ts.db, "TestContext", fmt.Errorf("test error"), "test@test.com")

	var count int
	ts.db.QueryRow("SELECT COUNT(*) FROM system_logs WHERE log_type = 'TEST'").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 test log, got %d", count)
	}
}

func TestUnauthorizedAccess(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	defer ts.db.Close()

	// No auth
	resp := ts.get("/api/dashboard/summary", "")
	assertStatus(t, resp, 401)

	resp = ts.post("/api/indent/submit", `{}`, "")
	assertStatus(t, resp, 401)

	// Non-admin trying admin endpoint
	// First create a regular user (hash below is bcrypt of "admin123")
	ts.db.Exec(`INSERT INTO users (email, password_hash, role) VALUES ('user@test.com', '$2a$10$N6NM1w1zUprcd8vwvp/BceLfeyGv6JqcdyhS5gSntCiyM0oEWGR2S', 'user') ON CONFLICT DO NOTHING`)
	userCookie := ts.login(t, "user@test.com", "admin123")

	// Master replace requires admin
	resp = ts.post("/api/master/replace", `[]`, userCookie)
	assertStatus(t, resp, 403)

	// Cleanup
	ts.db.Exec("DELETE FROM sessions WHERE user_id = (SELECT id FROM users WHERE email = 'user@test.com')")
	ts.db.Exec("DELETE FROM users WHERE email = 'user@test.com'")
}
