package db

import (
	"database/sql"
	"path/filepath"
	"testing"
)

func TestConnectProcuraReadsStockReadOnly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "procura.sqlite")
	seed, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := seed.Exec(`CREATE TABLE items (stock_id TEXT PRIMARY KEY, current_stock REAL); INSERT INTO items VALUES ('A', 4)`); err != nil {
		t.Fatal(err)
	}
	seed.Close()

	procura, err := ConnectProcura(path)
	if err != nil {
		t.Fatal(err)
	}
	defer procura.Close()

	stocks, err := currentStocks(nil, procura, []string{"A"})
	if err != nil {
		t.Fatal(err)
	}
	if stocks["A"] != 4 {
		t.Fatalf("stock A: want 4, got %v", stocks["A"])
	}
	if _, err := procura.Exec("INSERT INTO items VALUES ('B', 2)"); err == nil {
		t.Fatal("expected Procura connection to be read-only")
	}
}
