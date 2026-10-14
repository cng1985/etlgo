package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"etlgo/internal/config"
	"etlgo/internal/seed"
	"etlgo/internal/service"
	"etlgo/internal/store"

	"github.com/gin-gonic/gin"
)

func TestSeededPipelineRun(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()
	cfg := &config.Config{DataDir: dir, DBPath: filepath.Join(dir, "etl.db"), Addr: ":0", GinMode: "test"}
	db, err := store.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := seed.Apply(db, cfg); err != nil {
		t.Fatal(err)
	}
	sources := service.NewSourceService(db, cfg)
	pipelines := service.NewPipelineService(db, cfg, sources)
	datasets := service.NewDatasetService(db, cfg)
	h := New(sources, pipelines, datasets)
	r := gin.New()
	h.Register(r)

	var pipes []map[string]any
	getJSON(t, r, "/api/pipelines", &pipes)
	if len(pipes) != 3 {
		t.Fatalf("pipelines = %d", len(pipes))
	}
	for _, pipe := range pipes {
		id := int(pipe["id"].(float64))
		req := httptest.NewRequest(http.MethodPost, "/api/pipelines/"+itoa(id)+"/run", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusAccepted {
			t.Fatalf("run %s: %d %s", pipe["name"], rec.Code, rec.Body.String())
		}
	}
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		var runs []map[string]any
		getJSON(t, r, "/api/runs?limit=10", &runs)
		if len(runs) < 3 {
			time.Sleep(50 * time.Millisecond)
			continue
		}
		done := true
		for _, run := range runs {
			if run["status"] == "running" {
				done = false
			}
			if run["status"] == "failed" {
				detail := map[string]any{}
				getJSON(t, r, "/api/runs/"+itoa(int(run["id"].(float64))), &detail)
				t.Fatalf("run failed: %v log=%v", detail["error"], detail["log"])
			}
		}
		if done {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	var sets []map[string]any
	getJSON(t, r, "/api/datasets", &sets)
	if len(sets) != 3 {
		t.Fatalf("datasets = %#v", sets)
	}
	var preview map[string]any
	getJSON(t, r, "/api/datasets/orders_paid/preview", &preview)
	if int(preview["total"].(float64)) != 10 {
		t.Fatalf("orders_paid total=%v", preview["total"])
	}
	var stats map[string]any
	getJSON(t, r, "/api/stats", &stats)
	if int(stats["successRuns"].(float64)) != 3 {
		t.Fatalf("stats=%v", stats)
	}
}

func getJSON(t *testing.T, r http.Handler, path string, dest any) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("%s -> %d %s", path, rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), dest); err != nil {
		t.Fatal(err)
	}
}

func itoa(n int) string {
	return string(bytes.TrimLeft([]byte{
		byte('0' + n/10),
		byte('0' + n%10),
	}, "0"))
}
