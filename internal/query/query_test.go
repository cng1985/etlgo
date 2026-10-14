package query

import (
	"context"
	"path/filepath"
	"testing"

	"etlgo/internal/config"
	"etlgo/internal/model"
	"etlgo/internal/protocol"
	"etlgo/internal/store"
	"etlgo/internal/tenant"
)

func TestQueryPipelineRejectsUnsafeInput(t *testing.T) {
	dir := t.TempDir()
	db, err := store.New(&config.Config{DBPath: filepath.Join(dir, "q.db"), DataDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Source{TenantID: tenant.Default, Name: "订单", Type: model.SourceCSV, Config: `{"path":"samples/orders.csv"}`}).Error; err != nil {
		t.Fatal(err)
	}
	exec := New(db)
	_, err = exec.Execute(context.Background(), FullAccess(), ModeSearch, Request{
		Model:   "pageForSource",
		Filters: map[string]any{"name;drop": "x"},
	})
	if err == nil {
		t.Fatal("expected unknown filter error")
	}
	_, err = exec.Execute(context.Background(), FullAccess(), ModeSearch, Request{
		Model: "pageForSource",
		Sort:  &Sort{Field: "id;drop table sources", Method: "desc"},
	})
	status, ok := err.(*protocol.Error)
	if !ok || status.Code != protocol.CodeBadRequest {
		t.Fatalf("sort error = %#v", err)
	}
	_, err = exec.Execute(context.Background(), Principal{}, ModeSearch, Request{Model: "pageForSource"})
	status, ok = err.(*protocol.Error)
	if !ok || status.Code != protocol.CodeForbidden {
		t.Fatalf("permission error = %#v", err)
	}
	out, err := exec.Execute(context.Background(), FullAccess(), ModeSearch, Request{
		Model:   "pageForSource",
		Page:    1,
		Size:    10,
		Filters: map[string]any{"name": "订"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.ResponseType != "page" || out.Total != 1 || out.Page != 1 {
		t.Fatalf("outcome = %#v", out)
	}
}
