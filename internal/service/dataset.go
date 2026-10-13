package service

import (
	"encoding/json"
	"os"
	"strings"

	"etlgo/internal/config"
	"etlgo/internal/engine"
	"etlgo/internal/model"

	"gorm.io/gorm"
)

type DatasetService struct {
	db      *gorm.DB
	dataDir string
}

func NewDatasetService(db *gorm.DB, cfg *config.Config) *DatasetService {
	return &DatasetService{db: db, dataDir: cfg.DataDir}
}

func (s *DatasetService) List() ([]model.Dataset, error) {
	items := []model.Dataset{}
	err := s.db.Order("updated_at desc").Find(&items).Error
	return items, err
}

func (s *DatasetService) Get(name string) (*model.Dataset, error) {
	var item model.Dataset
	if err := s.db.Where("name = ?", name).First(&item).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, miss("数据集不存在")
		}
		return nil, err
	}
	return &item, nil
}

func (s *DatasetService) Preview(name string, limit int) (*engine.Preview, error) {
	item, err := s.Get(name)
	if err != nil {
		return nil, err
	}
	table, total, err := s.read(item, limit)
	if err != nil {
		return nil, bad(err.Error())
	}
	view := previewOf(table, limit, 0, nil)
	view.Total = total
	view.Truncated = total > len(table.Rows)
	return view, nil
}

func (s *DatasetService) Export(name, format string) (string, []byte, error) {
	item, err := s.Get(name)
	if err != nil {
		return "", nil, err
	}
	format = strings.ToLower(format)
	if format != "csv" && format != "json" {
		return "", nil, bad("仅支持导出 csv 或 json")
	}
	table, _, err := s.read(item, 100000)
	if err != nil {
		return "", nil, bad(err.Error())
	}
	filename := item.Name + "." + format
	body, err := encodeTable(format, table)
	if err != nil {
		return "", nil, err
	}
	return filename, body, nil
}

func (s *DatasetService) Delete(name string) error {
	item, err := s.Get(name)
	if err != nil {
		return err
	}
	switch item.Kind {
	case model.TargetSQLite:
		if err := engine.DropSQLite(s.db, item.Name); err != nil {
			return bad(err.Error())
		}
	case model.TargetCSV, model.TargetJSON:
		if item.Path != "" {
			full, err := engine.ResolveUnder(s.dataDir, item.Path)
			if err == nil {
				_ = os.Remove(full)
			}
		}
	}
	return s.db.Delete(item).Error
}

func (s *DatasetService) read(item *model.Dataset, limit int) (*engine.Table, int, error) {
	switch item.Kind {
	case model.TargetSQLite:
		return engine.PreviewSQLite(s.db, item.Name, limit)
	case model.TargetCSV:
		full, err := engine.ResolveUnder(s.dataDir, item.Path)
		if err != nil {
			return nil, 0, err
		}
		table, err := engine.ExtractCSV(full, ",", true)
		if err != nil {
			return nil, 0, err
		}
		total := len(table.Rows)
		if limit > 0 && len(table.Rows) > limit {
			table.Rows = table.Rows[:limit]
		}
		return table, total, nil
	case model.TargetJSON:
		full, err := engine.ResolveUnder(s.dataDir, item.Path)
		if err != nil {
			return nil, 0, err
		}
		table, err := engine.ExtractJSON(full)
		if err != nil {
			return nil, 0, err
		}
		total := len(table.Rows)
		if limit > 0 && len(table.Rows) > limit {
			table.Rows = table.Rows[:limit]
		}
		return table, total, nil
	default:
		return nil, 0, bad("未知数据集类型")
	}
}

func (s *DatasetService) Stats() (map[string]any, error) {
	count := func(dest any, conds ...any) int64 {
		var n int64
		q := s.db.Model(dest)
		if len(conds) > 0 {
			q = q.Where(conds[0], conds[1:]...)
		}
		_ = q.Count(&n).Error
		return n
	}
	var rows int64
	_ = s.db.Raw("SELECT coalesce(sum(rows_loaded), 0) FROM runs WHERE status = ?", model.RunSuccess).Scan(&rows).Error
	runs, err := NewRunLister(s.db).latest(8)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"sources":     count(&model.Source{}),
		"pipelines":   count(&model.Pipeline{}),
		"runs":        count(&model.Run{}),
		"successRuns": count(&model.Run{}, "status = ?", model.RunSuccess),
		"failedRuns":  count(&model.Run{}, "status = ?", model.RunFailed),
		"runningRuns": count(&model.Run{}, "status = ?", model.RunRunning),
		"datasets":    count(&model.Dataset{}),
		"rowsLoaded":  rows,
		"recentRuns":  runs,
	}, nil
}

type runLister struct{ db *gorm.DB }

func NewRunLister(db *gorm.DB) *runLister { return &runLister{db: db} }

func (r *runLister) latest(limit int) ([]model.Run, error) {
	items := []model.Run{}
	err := r.db.Order("id desc").Limit(limit).Find(&items).Error
	for i := range items {
		items[i].Log = ""
	}
	return items, err
}

func encodeTable(format string, table *engine.Table) ([]byte, error) {
	if format == "json" {
		rows := make([]map[string]any, len(table.Rows))
		for i, row := range table.Rows {
			obj := make(map[string]any, len(table.Columns))
			for _, col := range table.Columns {
				obj[col] = row[col]
			}
			rows[i] = obj
		}
		return json.MarshalIndent(rows, "", "  ")
	}
	var b strings.Builder
	b.WriteString(strings.Join(table.Columns, ","))
	b.WriteByte('\n')
	for _, row := range table.Rows {
		cells := make([]string, len(table.Columns))
		for i, col := range table.Columns {
			cells[i] = csvEscape(engineCell(row[col]))
		}
		b.WriteString(strings.Join(cells, ","))
		b.WriteByte('\n')
	}
	return []byte(b.String()), nil
}

func engineCell(v any) string {
	if v == nil {
		return ""
	}
	switch n := v.(type) {
	case string:
		return n
	default:
		raw, _ := json.Marshal(n)
		text := string(raw)
		if len(text) >= 2 && text[0] == '"' {
			var s string
			if json.Unmarshal(raw, &s) == nil {
				return s
			}
		}
		return strings.Trim(text, `"`)
	}
}

func csvEscape(s string) string {
	if strings.ContainsAny(s, ",\"\n\r") {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}
