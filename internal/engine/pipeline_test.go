package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestCSVToSQLite(t *testing.T) {
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "orders.csv")
	content := "id,customer,amount,qty,status\n1,张伟,10.5,2,paid\n2,李娜,8,1,pending\n3,王芳,3,4,paid\n"
	if err := os.WriteFile(csvPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	table, err := ExtractCSV(csvPath, ",", true)
	if err != nil {
		t.Fatal(err)
	}
	out, filtered, _, err := Apply(table, []Step{
		{Type: "filter", Column: "status", Op: "eq", Value: "paid"},
		{Type: "cast", Column: "amount", To: "float"},
		{Type: "cast", Column: "qty", To: "int"},
		{Type: "derive", Column: "total", Op: "mul", Left: "amount", Right: "qty"},
		{Type: "drop", Columns: []string{"status"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if filtered != 1 || len(out.Rows) != 2 {
		t.Fatalf("unexpected rows filtered=%d len=%d", filtered, len(out.Rows))
	}

	db, err := gorm.Open(sqlite.Open(filepath.Join(dir, "out.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := LoadSQLite(db, "orders_paid", "replace", out); err != nil {
		t.Fatal(err)
	}
	preview, total, err := PreviewSQLite(db, "orders_paid", 10)
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(preview.Rows) != 2 {
		t.Fatalf("preview total=%d rows=%d", total, len(preview.Rows))
	}
	again, err := ExtractSQLite(db, "select id, customer, total from orders_paid where total >= 20")
	if err != nil {
		t.Fatal(err)
	}
	if len(again.Rows) != 1 {
		t.Fatalf("sql extract rows=%d", len(again.Rows))
	}

	jsonPath := filepath.Join(dir, "out.json")
	if _, err := LoadFile(jsonPath, "json", out); err != nil {
		t.Fatal(err)
	}
	loaded, err := ExtractJSON(jsonPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Rows) != 2 {
		t.Fatal(len(loaded.Rows))
	}
}
