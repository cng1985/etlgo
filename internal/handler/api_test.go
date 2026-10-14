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
	"etlgo/internal/query"
	"etlgo/internal/seed"
	"etlgo/internal/service"
	"etlgo/internal/store"

	"github.com/gin-gonic/gin"
)

func TestSeededPipelineRun(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()
	cfg := &config.Config{DataDir: dir, DBPath: filepath.Join(dir, "etl.db"), Addr: ":0", GinMode: "test", Token: "etl-demo-token"}
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
	h := New(cfg, sources, pipelines, datasets, query.New(db))
	r := gin.New()
	h.Register(r)

	unauthorized := httptest.NewRequest(http.MethodPost, "/api/v1/pipeline/list", bytes.NewReader([]byte(`{}`)))
	unauthorized.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, unauthorized)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized = %d %s", rec.Code, rec.Body.String())
	}

	tenantBody := postRaw(t, r, "/api/v1/source/list", map[string]any{"tenantId": 1})
	if tenantBody.Code != http.StatusBadRequest {
		t.Fatalf("tenant = %d %s", tenantBody.Code, tenantBody.raw)
	}

	list := postRaw(t, r, "/api/v1/pipeline/list", map[string]any{"page": 1, "size": 20})
	if list.Code != 200 || list.ResponseType != "page" || list.Total != 3 {
		t.Fatalf("pipelines = %#v", list)
	}
	var pipes []map[string]any
	if err := json.Unmarshal(list.Data, &pipes); err != nil {
		t.Fatal(err)
	}
	for _, pipe := range pipes {
		id := int(pipe["id"].(float64))
		created := pipe["createdAt"].(string)
		if len(created) != len("2006-01-02 15:04:05") {
			t.Fatalf("createdAt = %s", created)
		}
		res := postRaw(t, r, "/api/v1/pipeline/execute", map[string]any{"id": id})
		if res.Code != http.StatusOK {
			t.Fatalf("run %s: %d %s", pipe["name"], res.Code, res.raw)
		}
	}
	deadline := time.Now().Add(8 * time.Second)
	var runs []map[string]any
	for time.Now().Before(deadline) {
		page := postRaw(t, r, "/api/v1/query/search", map[string]any{
			"model": "pageForRun",
			"page":  1,
			"size":  20,
		})
		if err := json.Unmarshal(page.Data, &runs); err != nil {
			t.Fatal(err)
		}
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
				detail := postRaw(t, r, "/api/v1/run/get", map[string]any{"id": run["id"]})
				t.Fatalf("run failed: %s", detail.raw)
			}
		}
		if done && page.ResponseType == "page" {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if len(runs) < 3 {
		t.Fatalf("runs = %#v", runs)
	}
	sets := postRaw(t, r, "/api/v1/dataset/list", map[string]any{"page": 1, "size": 20})
	if sets.Total != 3 {
		t.Fatalf("datasets = %s", sets.raw)
	}
	preview := postRaw(t, r, "/api/v1/dataset/preview", map[string]any{"name": "orders_paid", "limit": 20})
	var previewBody map[string]any
	if err := json.Unmarshal(preview.Data, &previewBody); err != nil {
		t.Fatal(err)
	}
	if int(previewBody["total"].(float64)) != 10 {
		t.Fatalf("orders_paid total=%v", previewBody["total"])
	}
	statsEnv := postRaw(t, r, "/api/v1/stats/get", map[string]any{})
	var stats map[string]any
	if err := json.Unmarshal(statsEnv.Data, &stats); err != nil {
		t.Fatal(err)
	}
	if int(stats["successRuns"].(float64)) != 3 {
		t.Fatalf("stats=%v", stats)
	}
}

type apiBody struct {
	Code         int
	Msg          string
	Data         json.RawMessage
	Total        float64
	ResponseType string
	raw          string
}

func postRaw(t *testing.T, r http.Handler, path string, payload any) apiBody {
	t.Helper()
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer etl-demo-token")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	var parsed apiBody
	parsed.raw = rec.Body.String()
	parsed.Code = rec.Code
	_ = json.Unmarshal(rec.Body.Bytes(), &parsed)
	if rec.Code != parsed.Code && parsed.Code == 0 {
		parsed.Code = rec.Code
	}
	return parsed
}
