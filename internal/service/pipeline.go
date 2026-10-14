package service

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"etlgo/internal/config"
	"etlgo/internal/engine"
	"etlgo/internal/model"
	"etlgo/internal/tenant"

	"go.uber.org/fx"
	"gorm.io/gorm"
)

type PipelineService struct {
	db      *gorm.DB
	dataDir string
	sources *SourceService
}

func NewPipelineService(db *gorm.DB, cfg *config.Config, sources *SourceService) *PipelineService {
	return &PipelineService{db: db, dataDir: cfg.DataDir, sources: sources}
}

type PipelineInput struct {
	Name            string          `json:"name"`
	Description     string          `json:"description"`
	Enabled         *bool           `json:"enabled"`
	SourceID        uint            `json:"sourceId"`
	Steps           json.RawMessage `json:"steps"`
	TargetType      string          `json:"targetType"`
	TargetConfig    json.RawMessage `json:"targetConfig"`
	ScheduleMinutes int             `json:"scheduleMinutes"`
}

type PreviewInput struct {
	SourceID uint            `json:"sourceId"`
	Steps    json.RawMessage `json:"steps"`
	Limit    int             `json:"limit"`
}

func (s *PipelineService) List() ([]model.Pipeline, error) {
	items := []model.Pipeline{}
	err := s.db.Order("id desc").Find(&items).Error
	return items, err
}

func (s *PipelineService) Get(id uint) (*model.Pipeline, error) {
	var item model.Pipeline
	if err := tenant.Scope(s.db).First(&item, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, miss("管道不存在")
		}
		return nil, err
	}
	return &item, nil
}

func (s *PipelineService) Create(in PipelineInput) (*model.Pipeline, error) {
	item, err := s.build(in)
	if err != nil {
		return nil, err
	}
	if err := s.db.Create(item).Error; err != nil {
		return nil, err
	}
	return item, nil
}

func (s *PipelineService) Update(id uint, in PipelineInput) (*model.Pipeline, error) {
	item, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	next, err := s.build(in)
	if err != nil {
		return nil, err
	}
	item.Name = next.Name
	item.Description = next.Description
	item.Enabled = next.Enabled
	item.SourceID = next.SourceID
	item.Steps = next.Steps
	item.TargetType = next.TargetType
	item.TargetConfig = next.TargetConfig
	item.ScheduleMinutes = next.ScheduleMinutes
	if err := s.db.Save(item).Error; err != nil {
		return nil, err
	}
	return item, nil
}

func (s *PipelineService) Delete(id uint) error {
	if _, err := s.Get(id); err != nil {
		return err
	}
	var running int64
	if err := s.db.Model(&model.Run{}).Where("pipeline_id = ? AND status = ?", id, model.RunRunning).Count(&running).Error; err != nil {
		return err
	}
	if running > 0 {
		return deny("管道正在运行")
	}
	return s.db.Delete(&model.Pipeline{}, id).Error
}

func (s *PipelineService) Preview(ctx context.Context, in PreviewInput) (*engine.Preview, error) {
	source, err := s.sources.Get(in.SourceID)
	if err != nil {
		return nil, err
	}
	steps, err := decodeSteps(in.Steps)
	if err != nil {
		return nil, err
	}
	table, err := s.sources.Extract(ctx, source)
	if err != nil {
		return nil, bad(err.Error())
	}
	transformed, filtered, logs, err := engine.Apply(table, steps)
	if err != nil {
		return nil, bad(err.Error())
	}
	logs = append([]string{fmt.Sprintf("抽取 %d 行", len(table.Rows))}, logs...)
	return previewOf(transformed, in.Limit, filtered, logs), nil
}

func (s *PipelineService) StartRun(id uint, trigger string) (*model.Run, error) {
	pipeline, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if !pipeline.Enabled && trigger == "schedule" {
		return nil, deny("管道已停用")
	}
	tenantID := pipeline.TenantID
	if tenantID == "" {
		tenantID = tenant.Default
	}
	run := &model.Run{
		TenantID:     tenantID,
		PipelineID:   pipeline.ID,
		PipelineName: pipeline.Name,
		Trigger:      trigger,
		Status:       model.RunRunning,
		StartedAt:    time.Now(),
	}
	err = s.db.Transaction(func(tx *gorm.DB) error {
		var n int64
		if err := tx.Model(&model.Run{}).Where("pipeline_id = ? AND status = ?", id, model.RunRunning).Count(&n).Error; err != nil {
			return err
		}
		if n > 0 {
			return deny("已有运行中的任务")
		}
		return tx.Create(run).Error
	})
	if err != nil {
		return nil, err
	}
	go s.execute(run.ID)
	run.Log = ""
	return run, nil
}

func (s *PipelineService) ListRuns(pipelineID uint, status string, limit int) ([]model.Run, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	q := s.db.Order("id desc").Limit(limit)
	if pipelineID > 0 {
		q = q.Where("pipeline_id = ?", pipelineID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	items := []model.Run{}
	if err := q.Find(&items).Error; err != nil {
		return nil, err
	}
	for i := range items {
		items[i].Log = ""
	}
	return items, nil
}

func (s *PipelineService) GetRun(id uint) (*model.Run, error) {
	var item model.Run
	if err := tenant.Scope(s.db).First(&item, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, miss("运行记录不存在")
		}
		return nil, err
	}
	return &item, nil
}

func (s *PipelineService) execute(runID uint) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	var run model.Run
	if err := s.db.First(&run, runID).Error; err != nil {
		return
	}
	logs := make([]string, 0, 8)
	fail := func(err error) {
		now := time.Now()
		run.Status = model.RunFailed
		run.Error = err.Error()
		run.FinishedAt = &now
		run.Log = joinLogs(append(logs, "失败: "+err.Error()))
		_ = s.db.Save(&run).Error
	}
	defer func() {
		if rec := recover(); rec != nil {
			fail(fmt.Errorf("内部错误: %v", rec))
		}
	}()

	pipeline, err := s.Get(run.PipelineID)
	if err != nil {
		fail(err)
		return
	}
	source, err := s.sources.Get(pipeline.SourceID)
	if err != nil {
		fail(err)
		return
	}
	table, err := s.sources.Extract(ctx, source)
	if err != nil {
		fail(err)
		return
	}
	run.RowsExtracted = len(table.Rows)
	logs = append(logs, fmt.Sprintf("抽取 %s，得到 %d 行、%d 列", source.Name, len(table.Rows), len(table.Columns)))

	steps, err := engine.ParseSteps(pipeline.Steps)
	if err != nil {
		fail(err)
		return
	}
	transformed, filtered, stepLogs, err := engine.Apply(table, steps)
	if err != nil {
		fail(err)
		return
	}
	logs = append(logs, stepLogs...)
	run.RowsFiltered = filtered

	loaded, err := s.load(pipeline, transformed)
	if err != nil {
		fail(err)
		return
	}
	run.RowsLoaded = loaded.Rows
	logs = append(logs, fmt.Sprintf("写入 %s %s，%d 行", loaded.Kind, loaded.Name, loaded.Rows))
	if err := s.upsertDataset(pipeline, loaded); err != nil {
		fail(err)
		return
	}
	now := time.Now()
	run.Status = model.RunSuccess
	run.FinishedAt = &now
	run.Log = joinLogs(logs)
	_ = s.db.Save(&run).Error
}

func (s *PipelineService) load(pipeline *model.Pipeline, table *engine.Table) (*engine.LoadResult, error) {
	switch pipeline.TargetType {
	case model.TargetSQLite:
		var cfg struct {
			Table string `json:"table"`
			Mode  string `json:"mode"`
		}
		if err := json.Unmarshal([]byte(pipeline.TargetConfig), &cfg); err != nil {
			return nil, fmt.Errorf("目标配置无效")
		}
		return engine.LoadSQLite(s.db, cfg.Table, cfg.Mode, table)
	case model.TargetCSV, model.TargetJSON:
		var cfg struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal([]byte(pipeline.TargetConfig), &cfg); err != nil {
			return nil, fmt.Errorf("目标配置无效")
		}
		full, err := engine.ResolveUnder(s.dataDir, cfg.Path)
		if err != nil {
			return nil, err
		}
		result, err := engine.LoadFile(full, pipeline.TargetType, table)
		if err != nil {
			return nil, err
		}
		result.Name = datasetName(cfg.Path)
		result.Path = filepath.ToSlash(cfg.Path)
		return result, nil
	default:
		return nil, fmt.Errorf("不支持的目标类型 %s", pipeline.TargetType)
	}
}

func (s *PipelineService) upsertDataset(pipeline *model.Pipeline, loaded *engine.LoadResult) error {
	columns, _ := json.Marshal(loaded.Columns)
	rowCount := loaded.Rows
	if loaded.Kind == model.TargetSQLite {
		var total int64
		if err := s.db.Raw(`SELECT COUNT(*) FROM "` + loaded.Name + `"`).Scan(&total).Error; err == nil {
			rowCount = int(total)
		}
	}
	var item model.Dataset
	err := tenant.Scope(s.db).Where("name = ?", loaded.Name).First(&item).Error
	if err == gorm.ErrRecordNotFound {
		item = model.Dataset{
			TenantID:   pipeline.TenantID,
			Name:       loaded.Name,
			Kind:       loaded.Kind,
			Path:       loaded.Path,
			PipelineID: pipeline.ID,
			Columns:    string(columns),
			RowCount:   rowCount,
		}
		return s.db.Create(&item).Error
	}
	if err != nil {
		return err
	}
	item.Kind = loaded.Kind
	item.Path = loaded.Path
	item.PipelineID = pipeline.ID
	if item.TenantID == "" {
		item.TenantID = pipeline.TenantID
	}
	item.Columns = string(columns)
	item.RowCount = rowCount
	return s.db.Save(&item).Error
}

func (s *PipelineService) build(in PipelineInput) (*model.Pipeline, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, bad("名称不能为空")
	}
	if utf8.RuneCountInString(in.Description) > 500 {
		return nil, bad("描述过长")
	}
	if _, err := s.sources.Get(in.SourceID); err != nil {
		return nil, err
	}
	steps, err := decodeSteps(in.Steps)
	if err != nil {
		return nil, err
	}
	rawSteps, _ := json.Marshal(steps)
	if in.ScheduleMinutes < 0 || in.ScheduleMinutes > 10080 {
		return nil, bad("调度间隔不合法")
	}
	enabled := true
	if in.Enabled != nil {
		enabled = *in.Enabled
	}
	target, err := s.normalizeTarget(in.TargetType, in.TargetConfig)
	if err != nil {
		return nil, err
	}
	return &model.Pipeline{
		TenantID:        tenant.Default,
		Name:            name,
		Description:     strings.TrimSpace(in.Description),
		Enabled:         enabled,
		SourceID:        in.SourceID,
		Steps:           string(rawSteps),
		TargetType:      in.TargetType,
		TargetConfig:    target,
		ScheduleMinutes: in.ScheduleMinutes,
	}, nil
}

func (s *PipelineService) normalizeTarget(kind string, raw json.RawMessage) (string, error) {
	if len(raw) == 0 {
		return "", bad("目标配置不能为空")
	}
	switch kind {
	case model.TargetSQLite:
		var cfg struct {
			Table string `json:"table"`
			Mode  string `json:"mode"`
		}
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return "", bad("目标配置无效")
		}
		cfg.Table = strings.TrimSpace(cfg.Table)
		if cfg.Table == "" {
			return "", bad("目标表名不能为空")
		}
		if cfg.Mode == "" {
			cfg.Mode = "replace"
		}
		if cfg.Mode != "replace" && cfg.Mode != "append" {
			return "", bad("写入模式仅支持 replace 或 append")
		}
		b, _ := json.Marshal(cfg)
		return string(b), nil
	case model.TargetCSV, model.TargetJSON:
		var cfg struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return "", bad("目标配置无效")
		}
		cfg.Path = filepath.ToSlash(strings.TrimSpace(cfg.Path))
		if _, err := engine.ResolveUnder(s.dataDir, cfg.Path); err != nil {
			return "", bad(err.Error())
		}
		ext := strings.ToLower(filepath.Ext(cfg.Path))
		if kind == model.TargetCSV && ext != ".csv" {
			return "", bad("CSV 目标需要 .csv 路径")
		}
		if kind == model.TargetJSON && ext != ".json" {
			return "", bad("JSON 目标需要 .json 路径")
		}
		b, _ := json.Marshal(cfg)
		return string(b), nil
	default:
		return "", bad("不支持的目标类型")
	}
}

func (s *PipelineService) Tick() {
	var pipelines []model.Pipeline
	if err := tenant.Scope(s.db).Where("enabled = ? AND schedule_minutes > 0", true).Find(&pipelines).Error; err != nil {
		return
	}
	now := time.Now()
	for _, pipeline := range pipelines {
		var last model.Run
		err := s.db.Where("pipeline_id = ?", pipeline.ID).Order("id desc").First(&last).Error
		if err == nil {
			if last.Status == model.RunRunning {
				continue
			}
			if last.StartedAt.Add(time.Duration(pipeline.ScheduleMinutes) * time.Minute).After(now) {
				continue
			}
		}
		_, _ = s.StartRun(pipeline.ID, "schedule")
	}
}

func StartScheduler(lc fx.Lifecycle, pipelines *PipelineService) {
	ctx, cancel := context.WithCancel(context.Background())
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go func() {
				ticker := time.NewTicker(20 * time.Second)
				defer ticker.Stop()
				for {
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
						pipelines.Tick()
					}
				}
			}()
			return nil
		},
		OnStop: func(context.Context) error {
			cancel()
			return nil
		},
	})
}

func decodeSteps(raw json.RawMessage) ([]engine.Step, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return []engine.Step{}, nil
	}
	var steps []engine.Step
	if err := json.Unmarshal(raw, &steps); err != nil {
		return nil, bad("转换步骤必须是数组")
	}
	for i, step := range steps {
		switch step.Type {
		case "select", "drop", "rename", "filter", "cast", "derive":
		default:
			return nil, bad(fmt.Sprintf("第 %d 步类型不支持", i+1))
		}
	}
	return steps, nil
}

func joinLogs(lines []string) string {
	text := strings.Join(lines, "\n")
	if len(text) > 32000 {
		return text[:32000]
	}
	return text
}

func datasetName(rel string) string {
	base := filepath.Base(rel)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	name = strings.Map(func(r rune) rune {
		if r == '_' || r == '-' || (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			if r == '-' {
				return '_'
			}
			return r
		}
		return '_'
	}, name)
	name = strings.Trim(name, "_")
	if name == "" {
		name = "dataset"
	}
	return name + "_" + strings.TrimPrefix(ext, ".")
}
