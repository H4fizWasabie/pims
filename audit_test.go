package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"
)

// Regression tests for the hardening audit. Each one fails on the old code.

func (ts *testServer) adminCookie(t *testing.T) string {
	return ts.login(t, "admin@pims.local", "admin123")
}

func decodeMap(resp *http.Response) map[string]any {
	var m map[string]any
	json.NewDecoder(resp.Body).Decode(&m)
	return m
}

func uniq(prefix string) string { return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano()) }

func TestIndentApproveUsesStoredQty(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	defer ts.db.Close()
	cookie := ts.adminCookie(t)

	id := uniq("QTY")
	ts.db.Exec(`INSERT INTO master_items (stock_id, item_name) VALUES ($1, 'Qty item')`, id)
	ts.db.Exec(`INSERT INTO inventory (stock_id, item_name, current_stock) VALUES ($1, 'Qty item', 5)`, id)
	ts.post("/api/indent/submit", mustJSON(t, map[string]any{
		"requester": "Lab",
		"items":     []map[string]any{{"itemName": "Qty item", "stockId": id, "uom": "pc", "reqQty": 1000}},
	}), cookie)
	var row int
	ts.db.QueryRow(`SELECT id FROM indents WHERE stock_id = $1`, id).Scan(&row)

	// Client claims qty 1, but the stored request is 1000 against 5 in stock.
	resp := ts.post("/api/indent/approve", mustJSON(t, map[string]any{"indentRowIndex": row, "reqQty": 1}), cookie)
	assertStatus(t, resp, 400)
}

func TestIndentIDsUniqueAndValidated(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	defer ts.db.Close()
	cookie := ts.adminCookie(t)

	submit := func() string {
		resp := ts.post("/api/indent/submit", mustJSON(t, map[string]any{
			"requester": "Lab",
			"items":     []map[string]any{{"itemName": "X", "stockId": "S1", "uom": "pc", "reqQty": 1}},
		}), cookie)
		return fmt.Sprint(decodeMap(resp)["message"])
	}
	// Same minute, same department: the old HHMM id collided.
	if a, b := submit(), submit(); a == b {
		t.Fatalf("indent ids collided: %q", a)
	}

	resp := ts.post("/api/indent/submit", mustJSON(t, map[string]any{
		"requester": "Lab",
		"items":     []map[string]any{{"itemName": "X", "stockId": "S1", "uom": "pc", "reqQty": -3}},
	}), cookie)
	assertStatus(t, resp, 400)
}

func TestOrderTotalsAndNumbering(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	defer ts.db.Close()
	cookie := ts.adminCookie(t)

	order := map[string]any{"department": uniq("Dept"), "items": []map[string]any{
		{"itemName": "I", "uom": "pc", "cost": 2.5, "qty": 4, "total": 999999},
	}}
	resp := ts.post("/api/order/generate", mustJSON(t, order), cookie)
	assertStatus(t, resp, 200)
	var total float64
	ts.db.QueryRow(`SELECT total_cost FROM orders WHERE department = $1`, order["department"]).Scan(&total)
	if total != 10 {
		t.Errorf("total_cost = %v, want 10 (qty*cost, not the client's value)", total)
	}

	for name, bad := range map[string]map[string]any{
		"no department": {"department": " ", "items": order["items"]},
		"zero qty":      {"department": "Lab", "items": []map[string]any{{"itemName": "I", "qty": 0, "cost": 1}}},
		"negative cost": {"department": "Lab", "items": []map[string]any{{"itemName": "I", "qty": 1, "cost": -1}}},
	} {
		if r := ts.post("/api/order/generate", mustJSON(t, bad), cookie); r.StatusCode != 400 {
			t.Errorf("%s: status %d, want 400", name, r.StatusCode)
		}
	}

	// A rejected order must not burn a PRF number.
	prf := func() int {
		var n int
		ts.db.QueryRow(`SELECT COALESCE(MAX(counter), 0) FROM id_counters WHERE key = 'PRF_' || to_char(now(), 'YYYYMMDD')`).Scan(&n)
		return n
	}
	before := prf()
	ts.post("/api/order/generate", mustJSON(t, map[string]any{"department": "Lab", "items": []map[string]any{{"itemName": "I", "qty": 0}}}), cookie)
	if prf() != before {
		t.Error("rejected order consumed a PRF number")
	}
}

func TestOrderTickRejectsUnknownField(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	defer ts.db.Close()
	resp := ts.post("/api/order/tick", mustJSON(t, map[string]any{"id": 1, "field": "bogus"}), ts.adminCookie(t))
	assertStatus(t, resp, 400)
}

func TestGRNConcurrentDoubleSubmit(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	defer ts.db.Close()
	cookie := ts.adminCookie(t)

	grn := mustJSON(t, map[string]any{
		"supplier": "S", "submissionToken": uniq("race"),
		"items": []map[string]any{{"itemName": "I", "qtyPo": 1, "qtyDo": 1, "qtyInv": 1}},
	})
	var wg sync.WaitGroup
	codes := make([]int, 6)
	for i := range codes {
		wg.Add(1)
		go func() { defer wg.Done(); codes[i] = ts.post("/api/grn/submit", grn, cookie).StatusCode }()
	}
	wg.Wait()
	ok := 0
	for _, c := range codes {
		if c == 200 {
			ok++
		} else if c != 409 {
			t.Errorf("unexpected status %d", c)
		}
	}
	if ok != 1 {
		t.Errorf("%d submissions succeeded (%v), want exactly 1", ok, codes)
	}

	// Validation
	resp := ts.post("/api/grn/submit", mustJSON(t, map[string]any{"supplier": "", "submissionToken": uniq("v"), "items": []any{}}), cookie)
	assertStatus(t, resp, 400)
}

func TestExpiryPagination(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	defer ts.db.Close()
	cookie := ts.adminCookie(t)

	ts.db.Exec(`DELETE FROM expiry_tracking`)
	for i, days := range []int{-5, 10, 100, 2000} { // last one is beyond the one-year window
		ts.db.Exec(`INSERT INTO expiry_tracking (stock_id, item_name, batch_no, expiry_date) VALUES ($1, 'I', 'B', CURRENT_DATE + $2::int)`, fmt.Sprint("E", i), days)
	}
	page := func(p int) map[string]any {
		return decodeMap(ts.get(fmt.Sprintf("/api/expiry/list?page=%d&pageSize=2", p), cookie))
	}
	p1, p2 := page(1), page(2)
	if p1["totalItems"] != float64(3) || p1["totalPages"] != float64(2) || p1["hasNext"] != true {
		t.Errorf("page 1 meta wrong: %v", p1)
	}
	if n := len(p1["items"].([]any)); n != 2 {
		t.Errorf("page 1 has %d items, want 2 (the far-future batch must not eat a slot)", n)
	}
	if n := len(p2["items"].([]any)); n != 1 || p2["hasNext"] != false {
		t.Errorf("page 2 wrong: %v", p2)
	}
	if first := p1["items"].([]any)[0].(map[string]any); first["label"] != "EXPIRED" {
		t.Errorf("first item = %v, want the expired one", first)
	}
}

func TestMasterReplaceValidates(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	defer ts.db.Close()
	cookie := ts.adminCookie(t)

	good := [][]string{{"KEEP-1", "Keep me", "pc", "G", "1.00", "S", "Available"}}
	assertStatus(t, ts.post("/api/master/replace", mustJSON(t, good), cookie), 200)

	for name, bad := range map[string][][]string{
		"empty":        {},
		"short row":    {{"A", "B", "C"}},
		"bad cost":     {{"A", "B", "pc", "G", "abc", "S", "Available"}},
		"duplicate id": {{"A", "B", "pc", "G", "1", "S", "Available"}, {"A", "C", "pc", "G", "1", "S", "Available"}},
		"missing id":   {{"", "B", "pc", "G", "1", "S", "Available"}},
	} {
		if r := ts.post("/api/master/replace", mustJSON(t, bad), cookie); r.StatusCode != 400 {
			t.Errorf("%s: status %d, want 400", name, r.StatusCode)
		}
	}
	var n int
	ts.db.QueryRow(`SELECT COUNT(*) FROM master_items WHERE stock_id = 'KEEP-1'`).Scan(&n)
	if n != 1 {
		t.Error("a rejected upload wiped the item master")
	}
}

func TestDisposalUsesMasterCost(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	defer ts.db.Close()
	cookie := ts.adminCookie(t)

	id := uniq("DISP")
	ts.db.Exec(`INSERT INTO master_items (stock_id, item_name, cost) VALUES ($1, 'D', 7)`, id)
	resp := ts.post("/api/disposal/submit", mustJSON(t, map[string]any{
		"stockId": id, "itemName": "D", "qty": 3, "reason": "Expired", "cost": 0.01,
	}), cookie)
	assertStatus(t, resp, 200)
	var loss float64
	ts.db.QueryRow(`SELECT total_loss FROM disposal_logs WHERE stock_id = $1`, id).Scan(&loss)
	if loss != 21 {
		t.Errorf("total_loss = %v, want 21 (qty * master cost)", loss)
	}
}

func TestSpecApproveCollisionFails(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	defer ts.db.Close()
	cookie := ts.adminCookie(t)

	var rowID int
	ts.db.QueryRow(`INSERT INTO new_item_requests (req_id, requester, item_name) VALUES ($1, 'x', 'Clash') RETURNING id`, uniq("R")).Scan(&rowID)
	ts.db.Exec(`INSERT INTO master_items (stock_id, item_name) VALUES ($1, 'squatter') ON CONFLICT DO NOTHING`, fmt.Sprint("NEW-", 1000+rowID))

	resp := ts.post("/api/spec/approve", mustJSON(t, map[string]any{"rowIndex": rowID}), cookie)
	assertStatus(t, resp, 400)
	var status string
	ts.db.QueryRow(`SELECT status FROM new_item_requests WHERE id = $1`, rowID).Scan(&status)
	if status != "Pending Review" {
		t.Errorf("status = %q; a failed approval must leave the request pending", status)
	}
}

func TestStockTakeRejectsNegativeQty(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	defer ts.db.Close()
	resp := ts.post("/api/stocktake/submit", mustJSON(t, map[string]any{"stockId": "S", "location": "Lab Level 1", "qty": -1}), ts.adminCookie(t))
	assertStatus(t, resp, 400)
}

func TestWritesRequireJSONContentType(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	defer ts.db.Close()
	cookie := ts.adminCookie(t)

	req, _ := http.NewRequest("POST", ts.URL+"/api/order/generate", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "text/plain") // what a cross-site <form> can send
	req.Header.Set("Cookie", cookie)
	resp, _ := http.DefaultClient.Do(req)
	assertStatus(t, resp, 415)
}

func TestOversizedBodyRejected(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	defer ts.db.Close()
	big := `{"department":"` + strings.Repeat("a", 2<<20) + `"}`
	resp := ts.post("/api/order/generate", big, ts.adminCookie(t))
	assertStatus(t, resp, 400)
}

func TestUnknownAPIPathIsJSON404(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	defer ts.db.Close()
	resp := ts.get("/api/does-not-exist", "")
	assertStatus(t, resp, 404)
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("content-type %q, want JSON", ct)
	}
}

func TestServerErrorsDoNotLeakSQL(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	defer ts.db.Close()
	cookie := ts.adminCookie(t)

	ts.db.Exec(`ALTER TABLE orders RENAME TO orders_gone`)
	defer ts.db.Exec(`ALTER TABLE orders_gone RENAME TO orders`)
	resp := ts.get("/api/order/list", cookie)
	assertStatus(t, resp, 500)
	if body := readBody(resp); strings.Contains(body, "orders") || strings.Contains(body, "pq") {
		t.Errorf("response leaks internals: %s", body)
	}
}

func TestUserManagement(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	defer ts.db.Close()
	cookie := ts.adminCookie(t)

	email := uniq("Mixed.Case") + "@test.com"
	create := func(e, pw, role string) int {
		return ts.post("/api/users/create", mustJSON(t, map[string]string{"email": e, "password": pw, "role": role}), cookie).StatusCode
	}
	if c := create(email, "longenough1", "user"); c != 200 {
		t.Fatalf("create: %d", c)
	}
	if c := create(strings.ToUpper(email), "longenough1", "user"); c != 409 {
		t.Errorf("case-variant duplicate: %d, want 409", c)
	}
	if c := create(uniq("a")+"@test.com", "short", "user"); c != 400 {
		t.Errorf("short password: %d, want 400", c)
	}
	if c := create(uniq("b")+"@test.com", "longenough1", "superuser"); c != 400 {
		t.Errorf("bad role: %d, want 400", c)
	}
	// Login is case-insensitive too.
	ts.login(t, strings.ToUpper(email), "longenough1")

	var adminID int
	ts.db.QueryRow(`SELECT id FROM users WHERE email = 'admin@pims.local'`).Scan(&adminID)
	assertStatus(t, ts.post(fmt.Sprintf("/api/users/delete?id=%d", adminID), "", cookie), 400)
}

func TestChangePasswordSignsOutOtherSessions(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	defer ts.db.Close()
	admin := ts.adminCookie(t)

	email := uniq("pw") + "@test.com"
	ts.post("/api/users/create", mustJSON(t, map[string]string{"email": email, "password": "oldpassword1"}), admin)
	keep, other := ts.login(t, email, "oldpassword1"), ts.login(t, email, "oldpassword1")

	resp := ts.post("/api/auth/change-password", mustJSON(t, map[string]string{"oldPassword": "oldpassword1", "newPassword": "newpassword2"}), keep)
	assertStatus(t, resp, 200)
	assertStatus(t, ts.get("/api/auth/me", keep), 200)
	assertStatus(t, ts.get("/api/auth/me", other), 401)

	bad := ts.post("/api/auth/change-password", mustJSON(t, map[string]string{"oldPassword": "nope", "newPassword": "newpassword3"}), keep)
	assertStatus(t, bad, 400)
}

func TestDemoLoginIsRateLimited(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	defer ts.db.Close()
	limited := false
	for i := 0; i < 30; i++ {
		// A private client IP keeps this from starving other tests' demo logins.
		req, _ := http.NewRequest("POST", ts.URL+"/api/auth/demo", nil)
		req.Header.Set("X-Forwarded-For", "203.0.113.77")
		resp, _ := http.DefaultClient.Do(req)
		if resp.StatusCode == 429 {
			limited = true
			break
		}
	}
	if !limited {
		t.Error("demo login never rate-limited")
	}
}

func TestMigrationSeedsNoDefaultPassword(t *testing.T) {
	sql, err := os.ReadFile("internal/db/migration.sql")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(sql), "admin123") {
		t.Error("migration.sql still seeds a default password")
	}
}

// Guards the XSS fix: server/user text must go through esc() before innerHTML.
func TestFrontendEscapesUserText(t *testing.T) {
	html, err := os.ReadFile("static/index.html")
	if err != nil {
		t.Fatal(err)
	}
	raw := regexp.MustCompile(`\$\{(item|i)\.(itemName|stockId|uom|batch|requester|reqId|justification|name|expiry|label)\}|'\s*\+\s*(u|r)\.(email|itemName|department|prfNo|location|uom)\s*\+\s*'`)
	for n, line := range strings.Split(string(html), "\n") {
		if strings.Contains(line, ".value =") {
			continue // form fields take text, not HTML
		}
		if raw.MatchString(line) {
			t.Errorf("index.html:%d interpolates unescaped text: %s", n+1, strings.TrimSpace(line))
		}
	}
	if strings.Contains(string(html), "_addCustom('${") {
		t.Error("custom-item onclick still splices user text into inline JS")
	}
}
