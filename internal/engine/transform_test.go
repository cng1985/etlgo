package engine

import "testing"

func sampleTable() *Table {
	return &Table{
		Columns: []string{"id", "name", "amount", "qty", "status"},
		Rows: []Record{
			{"id": "1", "name": " 张伟 ", "amount": "10.5", "qty": "2", "status": "paid"},
			{"id": "2", "name": "李娜", "amount": "8", "qty": "1", "status": "pending"},
			{"id": "3", "name": "王芳", "amount": "3", "qty": "4", "status": "paid"},
		},
	}
}

func TestApplyPipeline(t *testing.T) {
	steps := []Step{
		{Type: "filter", Column: "status", Op: "eq", Value: "paid"},
		{Type: "cast", Column: "amount", To: "float"},
		{Type: "cast", Column: "qty", To: "int"},
		{Type: "derive", Column: "total", Op: "mul", Left: "amount", Right: "qty"},
		{Type: "derive", Column: "name", Op: "trim", Left: "name"},
		{Type: "rename", Mapping: map[string]string{"name": "customer"}},
		{Type: "select", Columns: []string{"id", "customer", "total"}},
	}
	out, filtered, logs, err := Apply(sampleTable(), steps)
	if err != nil {
		t.Fatal(err)
	}
	if filtered != 1 {
		t.Fatalf("filtered = %d", filtered)
	}
	if len(out.Rows) != 2 || len(logs) != len(steps) {
		t.Fatalf("rows=%d logs=%d", len(out.Rows), len(logs))
	}
	if out.Columns[1] != "customer" {
		t.Fatalf("columns = %#v", out.Columns)
	}
	if out.Rows[0]["customer"] != "张伟" {
		t.Fatalf("trim/rename = %#v", out.Rows[0]["customer"])
	}
	if out.Rows[0]["total"] != 21.0 {
		t.Fatalf("total = %#v", out.Rows[0]["total"])
	}
	if out.Rows[1]["total"] != 12.0 {
		t.Fatalf("total2 = %#v", out.Rows[1]["total"])
	}
}

func TestFilterNumericAndContains(t *testing.T) {
	table := sampleTable()
	filtered, removed, _, err := applyFilter(table, Step{Column: "amount", Op: "gte", Value: "8"})
	if err != nil {
		t.Fatal(err)
	}
	if removed != 1 || len(filtered.Rows) != 2 {
		t.Fatalf("gte rows=%d removed=%d", len(filtered.Rows), removed)
	}
	contains, _, _, err := applyFilter(table, Step{Column: "name", Op: "contains", Value: "娜"})
	if err != nil {
		t.Fatal(err)
	}
	if len(contains.Rows) != 1 {
		t.Fatalf("contains rows=%d", len(contains.Rows))
	}
}

func TestDeriveConcatAndDivZero(t *testing.T) {
	table := &Table{
		Columns: []string{"a", "b", "left_name", "right_name"},
		Rows: []Record{
			{"a": "10", "b": "2", "left_name": "Go", "right_name": "ETL"},
			{"a": "1", "b": "0", "left_name": "A", "right_name": "B"},
		},
	}
	out, _, _, err := Apply(table, []Step{
		{Type: "derive", Column: "title", Op: "concat", Columns: []string{"left_name", "right_name"}, Sep: "-"},
		{Type: "cast", Column: "a", To: "float"},
		{Type: "cast", Column: "b", To: "float"},
		{Type: "derive", Column: "ratio", Op: "div", Left: "a", Right: "b"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Rows[0]["title"] != "Go-ETL" {
		t.Fatal(out.Rows[0]["title"])
	}
	if out.Rows[0]["ratio"] != 5.0 {
		t.Fatalf("ratio = %#v", out.Rows[0]["ratio"])
	}
	if out.Rows[1]["ratio"] != nil {
		t.Fatalf("div zero = %#v", out.Rows[1]["ratio"])
	}
}

func TestRejectUnknownStep(t *testing.T) {
	_, _, _, err := Apply(sampleTable(), []Step{{Type: "explode"}})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveUnderRejectsEscape(t *testing.T) {
	if _, err := ResolveUnder("/tmp/etl-data", "../etc/passwd"); err == nil {
		t.Fatal("expected escape error")
	}
	if _, err := ResolveUnder("/tmp/etl-data", "/etc/passwd"); err == nil {
		t.Fatal("expected absolute path error")
	}
}
