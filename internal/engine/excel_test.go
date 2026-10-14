package engine

import (
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestExtractExcel(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "orders.xlsx")
	file := excelize.NewFile()
	defer file.Close()
	sheet := "订单"
	file.SetSheetName("Sheet1", sheet)
	rows := [][]any{
		{"id", "客户", "金额"},
		{"1", "张伟", "10.5"},
		{"2", "李娜", ""},
	}
	for r, row := range rows {
		for c, value := range row {
			cell, err := excelize.CoordinatesToCellName(c+1, r+1)
			if err != nil {
				t.Fatal(err)
			}
			if err := file.SetCellValue(sheet, cell, value); err != nil {
				t.Fatal(err)
			}
		}
	}
	if _, err := file.NewSheet("空表"); err != nil {
		t.Fatal(err)
	}
	if err := file.SaveAs(path); err != nil {
		t.Fatal(err)
	}

	table, err := ExtractExcel(path, sheet, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(table.Columns) != 3 || len(table.Rows) != 2 {
		t.Fatalf("columns=%v rows=%d", table.Columns, len(table.Rows))
	}
	if table.Rows[0]["客户"] != "张伟" || table.Rows[1]["金额"] != "" {
		t.Fatalf("%#v", table.Rows)
	}

	first, err := ExtractExcel(path, "", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Rows) != 2 {
		t.Fatalf("first sheet rows=%d", len(first.Rows))
	}
	if _, err := ExtractExcel(path, "不存在", true); err == nil {
		t.Fatal("expected missing sheet")
	}
	if err := ValidateSheet("bad/name"); err == nil {
		t.Fatal("expected invalid sheet")
	}
}
