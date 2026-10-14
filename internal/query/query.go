package query

import (
	"context"
	"strings"

	"etlgo/internal/protocol"

	"gorm.io/gorm"
)

const (
	ModeSearch = "search"
	ModeList   = "list"
	ModeOne    = "one"

	PermSourceView   = "etl:source:view"
	PermSourceEdit   = "etl:source:edit"
	PermPipelineView = "etl:pipeline:view"
	PermPipelineEdit = "etl:pipeline:edit"
	PermExecute      = "etl:pipeline:execute"
	PermRunView      = "etl:run:view"
	PermDatasetView  = "etl:dataset:view"
	PermDatasetEdit  = "etl:dataset:edit"
	PermStatsView    = "etl:stats:view"
	PermFileUpload   = "etl:file:upload"
)

type Request struct {
	Model   string         `json:"model"`
	Page    int            `json:"page"`
	Size    int            `json:"size"`
	Filters map[string]any `json:"filters"`
	Sort    *Sort          `json:"sort"`
}

type Sort struct {
	Field  string `json:"field"`
	Method string `json:"method"`
}

type Principal struct {
	perms map[string]struct{}
}

func (p Principal) Allow(permission string) bool {
	_, ok := p.perms[permission]
	return ok
}

func FullAccess() Principal {
	names := []string{
		PermSourceView, PermSourceEdit, PermPipelineView, PermPipelineEdit, PermExecute,
		PermRunView, PermDatasetView, PermDatasetEdit, PermStatsView, PermFileUpload,
	}
	perms := make(map[string]struct{}, len(names))
	for _, name := range names {
		perms[name] = struct{}{}
	}
	return Principal{perms: perms}
}

type Outcome struct {
	ResponseType string
	Data         any
	Page         int
	Size         int
	Total        int64
}

type Executor struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Executor {
	return &Executor{db: db}
}

func (e *Executor) Execute(ctx context.Context, principal Principal, mode string, req Request) (Outcome, error) {
	def, ok := registry[req.Model]
	if !ok {
		return Outcome{}, protocol.Bad("查询模型不存在")
	}
	if def.kind != mode {
		return Outcome{}, protocol.Bad("查询模型与接口不匹配")
	}
	if !principal.Allow(def.permission) {
		return Outcome{}, protocol.Forbidden("无权限")
	}
	filters, err := sanitizeFilters(def, req.Filters)
	if err != nil {
		return Outcome{}, err
	}
	order, err := orderBy(def, req.Sort)
	if err != nil {
		return Outcome{}, err
	}
	db := e.db.WithContext(ctx)
	switch mode {
	case ModeSearch:
		page, size, err := protocol.NormalizePage(req.Page, req.Size)
		if err != nil {
			return Outcome{}, err
		}
		rows, total, err := def.fetch(db, filters, order, (page-1)*size, size)
		if err != nil {
			return Outcome{}, err
		}
		return Outcome{ResponseType: "page", Data: rows, Page: page, Size: size, Total: total}, nil
	case ModeList:
		rows, _, err := def.fetch(db, filters, order, 0, 200)
		if err != nil {
			return Outcome{}, err
		}
		return Outcome{Data: rows}, nil
	case ModeOne:
		rows, total, err := def.fetch(db, filters, order, 0, 2)
		if err != nil {
			return Outcome{}, err
		}
		if total == 0 {
			return Outcome{}, protocol.NotFound("数据不存在")
		}
		if total > 1 {
			return Outcome{}, protocol.Bad("查询条件不唯一")
		}
		item, err := firstRow(rows)
		if err != nil {
			return Outcome{}, err
		}
		return Outcome{Data: item}, nil
	default:
		return Outcome{}, protocol.Bad("不支持的查询方式")
	}
}

func sanitizeFilters(def definition, filters map[string]any) (map[string]any, error) {
	if filters == nil {
		return map[string]any{}, nil
	}
	out := make(map[string]any, len(filters))
	for key, value := range filters {
		if key == "tenantId" {
			return nil, protocol.Bad("不能指定租户")
		}
		kind, ok := def.filters[key]
		if !ok {
			return nil, protocol.Bad("不支持的查询条件: " + key)
		}
		if value == nil {
			continue
		}
		if text, ok := value.(string); ok && strings.TrimSpace(text) == "" {
			continue
		}
		parsed, err := castFilter(kind, value)
		if err != nil {
			return nil, err
		}
		out[key] = parsed
	}
	return out, nil
}

func orderBy(def definition, sort *Sort) (string, error) {
	if sort == nil || strings.TrimSpace(sort.Field) == "" {
		return def.defaultOrder, nil
	}
	column, ok := def.sorts[sort.Field]
	if !ok {
		return "", protocol.Bad("不支持的排序字段")
	}
	method := strings.ToLower(strings.TrimSpace(sort.Method))
	if method != "asc" && method != "desc" {
		return "", protocol.Bad("排序方式仅支持 asc 或 desc")
	}
	return column + " " + method, nil
}
