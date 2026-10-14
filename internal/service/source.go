package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"etlgo/internal/config"
	"etlgo/internal/engine"
	"etlgo/internal/model"
	"etlgo/internal/store"

	"gorm.io/gorm"
)

type SourceService struct {
	db      *gorm.DB
	dataDir string
}

func NewSourceService(db *gorm.DB, cfg *config.Config) *SourceService {
	return &SourceService{db: db, dataDir: cfg.DataDir}
}

type SourceInput struct {
	Name   string          `json:"name"`
	Type   string          `json:"type"`
	Config json.RawMessage `json:"config"`
}

func (s *SourceService) List() ([]model.Source, error) {
	items := []model.Source{}
	err := s.db.Order("id desc").Find(&items).Error
	return items, err
}

func (s *SourceService) Get(id uint) (*model.Source, error) {
	var item model.Source
	if err := s.db.First(&item, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, miss("数据源不存在")
		}
		return nil, err
	}
	return &item, nil
}

func (s *SourceService) Create(in SourceInput) (*model.Source, error) {
	item, err := s.build(in)
	if err != nil {
		return nil, err
	}
	if err := s.db.Create(item).Error; err != nil {
		return nil, err
	}
	return item, nil
}

func (s *SourceService) Update(id uint, in SourceInput) (*model.Source, error) {
	item, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	next, err := s.build(in)
	if err != nil {
		return nil, err
	}
	item.Name = next.Name
	item.Type = next.Type
	item.Config = next.Config
	if err := s.db.Save(item).Error; err != nil {
		return nil, err
	}
	return item, nil
}

func (s *SourceService) Delete(id uint) error {
	if _, err := s.Get(id); err != nil {
		return err
	}
	var n int64
	if err := s.db.Model(&model.Pipeline{}).Where("source_id = ?", id).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return deny("仍有管道使用该数据源")
	}
	return s.db.Delete(&model.Source{}, id).Error
}

func (s *SourceService) Preview(ctx context.Context, id uint, limit int) (*engine.Preview, error) {
	item, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	table, err := s.Extract(ctx, item)
	if err != nil {
		return nil, bad(err.Error())
	}
	return previewOf(table, limit, 0, nil), nil
}

func (s *SourceService) SaveUpload(name, kind string, r io.Reader, original string) (*model.Source, error) {
	kind = strings.ToLower(kind)
	if kind != model.SourceCSV && kind != model.SourceJSON {
		return nil, bad("仅支持上传 CSV 或 JSON")
	}
	ext := ".csv"
	if kind == model.SourceJSON {
		ext = ".json"
	}
	rel := filepath.Join("uploads", newFileID()+ext)
	full, err := engine.ResolveUnder(s.dataDir, rel)
	if err != nil {
		return nil, bad(err.Error())
	}
	if err := engine.EnsureParent(full); err != nil {
		return nil, err
	}
	f, err := os.Create(full)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if _, err := io.Copy(f, io.LimitReader(r, 32<<20)); err != nil {
		return nil, err
	}
	if strings.TrimSpace(name) == "" {
		name = strings.TrimSuffix(original, filepath.Ext(original))
	}
	cfg := map[string]any{"path": filepath.ToSlash(rel), "hasHeader": true, "delimiter": ","}
	raw, _ := json.Marshal(cfg)
	return s.Create(SourceInput{Name: name, Type: kind, Config: raw})
}

func (s *SourceService) Extract(ctx context.Context, item *model.Source) (*engine.Table, error) {
	switch item.Type {
	case model.SourceCSV:
		var cfg struct {
			Path      string `json:"path"`
			Delimiter string `json:"delimiter"`
			HasHeader *bool  `json:"hasHeader"`
		}
		if err := json.Unmarshal([]byte(item.Config), &cfg); err != nil {
			return nil, fmt.Errorf("CSV 配置无效")
		}
		full, err := engine.ResolveUnder(s.dataDir, cfg.Path)
		if err != nil {
			return nil, err
		}
		hasHeader := true
		if cfg.HasHeader != nil {
			hasHeader = *cfg.HasHeader
		}
		return engine.ExtractCSV(full, cfg.Delimiter, hasHeader)
	case model.SourceJSON:
		var cfg struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal([]byte(item.Config), &cfg); err != nil {
			return nil, fmt.Errorf("JSON 配置无效")
		}
		full, err := engine.ResolveUnder(s.dataDir, cfg.Path)
		if err != nil {
			return nil, err
		}
		return engine.ExtractJSON(full)
	case model.SourceSQLite:
		var cfg struct {
			Path  string `json:"path"`
			Query string `json:"query"`
		}
		if err := json.Unmarshal([]byte(item.Config), &cfg); err != nil {
			return nil, fmt.Errorf("SQLite 配置无效")
		}
		db := s.db
		if strings.TrimSpace(cfg.Path) != "" {
			full, err := engine.ResolveUnder(s.dataDir, cfg.Path)
			if err != nil {
				return nil, err
			}
			opened, err := store.Open(full)
			if err != nil {
				return nil, fmt.Errorf("打开 SQLite 失败: %w", err)
			}
			sqlDB, _ := opened.DB()
			defer sqlDB.Close()
			db = opened
		}
		return engine.ExtractSQLite(db, cfg.Query)
	case model.SourceHTTP:
		var cfg struct {
			URL      string `json:"url"`
			DataPath string `json:"dataPath"`
		}
		if err := json.Unmarshal([]byte(item.Config), &cfg); err != nil {
			return nil, fmt.Errorf("HTTP 配置无效")
		}
		return engine.ExtractHTTP(ctx, cfg.URL, cfg.DataPath)
	default:
		return nil, fmt.Errorf("不支持的数据源类型 %s", item.Type)
	}
}

func (s *SourceService) build(in SourceInput) (*model.Source, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, bad("名称不能为空")
	}
	if len([]rune(name)) > 120 {
		return nil, bad("名称过长")
	}
	switch in.Type {
	case model.SourceCSV, model.SourceJSON, model.SourceSQLite, model.SourceHTTP:
	default:
		return nil, bad("不支持的数据源类型")
	}
	if len(in.Config) == 0 {
		return nil, bad("配置不能为空")
	}
	if !json.Valid(in.Config) {
		return nil, bad("配置必须是 JSON 对象")
	}
	if err := s.validateConfig(in.Type, in.Config); err != nil {
		return nil, err
	}
	return &model.Source{Name: name, Type: in.Type, Config: string(in.Config)}, nil
}

func (s *SourceService) validateConfig(kind string, raw json.RawMessage) error {
	switch kind {
	case model.SourceCSV:
		var cfg struct {
			Path      string `json:"path"`
			Delimiter string `json:"delimiter"`
		}
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return bad("CSV 配置无效")
		}
		if _, err := engine.ResolveUnder(s.dataDir, cfg.Path); err != nil {
			return bad(err.Error())
		}
		if cfg.Delimiter != "" && len([]rune(cfg.Delimiter)) != 1 {
			return bad("分隔符必须是单个字符")
		}
	case model.SourceJSON:
		var cfg struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return bad("JSON 配置无效")
		}
		if _, err := engine.ResolveUnder(s.dataDir, cfg.Path); err != nil {
			return bad(err.Error())
		}
	case model.SourceSQLite:
		var cfg struct {
			Path  string `json:"path"`
			Query string `json:"query"`
		}
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return bad("SQLite 配置无效")
		}
		if strings.TrimSpace(cfg.Query) == "" {
			return bad("SQL 不能为空")
		}
		if strings.TrimSpace(cfg.Path) != "" {
			if _, err := engine.ResolveUnder(s.dataDir, cfg.Path); err != nil {
				return bad(err.Error())
			}
		}
	case model.SourceHTTP:
		var cfg struct {
			URL string `json:"url"`
		}
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return bad("HTTP 配置无效")
		}
		if !strings.HasPrefix(cfg.URL, "http://") && !strings.HasPrefix(cfg.URL, "https://") {
			return bad("仅支持 http 或 https 地址")
		}
	}
	return nil
}

func newFileID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", os.Getpid())
	}
	return hex.EncodeToString(buf)
}

func previewOf(table *engine.Table, limit, filtered int, logs []string) *engine.Preview {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows := table.Rows
	truncated := false
	if len(rows) > limit {
		rows = rows[:limit]
		truncated = true
	}
	copied := make([]engine.Record, len(rows))
	for i, row := range rows {
		next := make(engine.Record, len(row))
		for k, v := range row {
			next[k] = v
		}
		copied[i] = next
	}
	return &engine.Preview{
		Columns:   table.Columns,
		Rows:      copied,
		Total:     len(table.Rows),
		Filtered:  filtered,
		Logs:      logs,
		Truncated: truncated,
	}
}
