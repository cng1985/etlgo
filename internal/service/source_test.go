package service

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"etlgo/internal/config"
	"etlgo/internal/model"
	"etlgo/internal/store"

	"github.com/xuri/excelize/v2"
)

func TestSourceCreateAndExcelExtract(t *testing.T) {
	dir := t.TempDir()
	db, err := store.New(&config.Config{DBPath: filepath.Join(dir, "etl.db"), DataDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	svc := NewSourceService(db, &config.Config{DataDir: dir})

	if _, err := svc.Create(SourceInput{Name: "订单库", Type: model.SourceMySQL, Config: []byte(`{"host":"127.0.0.1","user":"reader","password":"secret","database":"shop","query":"delete from orders"}`)}); err == nil {
		t.Fatal("expected reject write query")
	}
	mysql, err := svc.Create(SourceInput{
		Name:   "订单库",
		Type:   model.SourceMySQL,
		Config: []byte(`{"host":"db.internal","user":"reader","password":"secret","database":"shop","query":"select id from orders"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if mysql.Type != model.SourceMySQL {
		t.Fatal(mysql.Type)
	}
	if _, err := svc.Create(SourceInput{Name: "pg", Type: model.SourcePostgres, Config: []byte(`{"host":"localhost","user":"u","database":"d","query":"select 1","sslMode":"prefer"}`)}); err == nil {
		t.Fatal("expected bad ssl")
	}
	if _, err := svc.Create(SourceInput{Name: "mssql", Type: model.SourceSQLServer, Config: []byte(`{"host":"localhost","user":"u","database":"d","query":"select 1"}`)}); err != nil {
		t.Fatal(err)
	}

	file := excelize.NewFile()
	file.SetCellValue("Sheet1", "A1", "sku")
	file.SetCellValue("Sheet1", "B1", "name")
	file.SetCellValue("Sheet1", "A2", "A-1")
	file.SetCellValue("Sheet1", "B2", "键盘")
	var buf bytes.Buffer
	if err := file.Write(&buf); err != nil {
		t.Fatal(err)
	}
	file.Close()
	uploaded, err := svc.SaveUpload("库存表", model.SourceExcel, bytes.NewReader(buf.Bytes()), "stock.xlsx")
	if err != nil {
		t.Fatal(err)
	}
	table, err := svc.Extract(context.Background(), uploaded)
	if err != nil {
		t.Fatal(err)
	}
	if len(table.Rows) != 1 || table.Rows[0]["name"] != "键盘" {
		t.Fatalf("%#v", table.Rows)
	}
	if _, err := svc.SaveUpload("", model.SourceExcel, strings.NewReader("nope"), "old.xls"); err == nil {
		t.Fatal("expected xls reject")
	}

	csv, err := svc.SaveUpload("名单", model.SourceCSV, strings.NewReader("id,name\n1,张伟\n"), "people.csv")
	if err != nil {
		t.Fatal(err)
	}
	csvTable, err := svc.Extract(context.Background(), csv)
	if err != nil {
		t.Fatal(err)
	}
	if len(csvTable.Rows) != 1 || csvTable.Rows[0]["name"] != "张伟" {
		t.Fatalf("%#v", csvTable.Rows)
	}
}
