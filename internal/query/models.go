package query

import (
	"reflect"
	"strconv"
	"strings"

	"etlgo/internal/dto"
	"etlgo/internal/model"
	"etlgo/internal/protocol"
	"etlgo/internal/tenant"

	"gorm.io/gorm"
)

type fieldKind int

const (
	fieldLike fieldKind = iota
	fieldEq
	fieldUint
	fieldBool
)

type definition struct {
	kind         string
	permission   string
	filters      map[string]fieldKind
	sorts        map[string]string
	defaultOrder string
	fetch        func(db *gorm.DB, filters map[string]any, order string, offset, limit int) (any, int64, error)
}

var registry = map[string]definition{}

func init() {
	register("pageForSource", "listSource", "oneSource", PermSourceView, sourceDef())
	register("pageForPipeline", "listPipeline", "onePipeline", PermPipelineView, pipelineDef())
	register("pageForRun", "listRun", "oneRun", PermRunView, runDef())
	register("pageForDataset", "listDataset", "oneDataset", PermDatasetView, datasetDef())
}

func register(pageName, listName, oneName, permission string, base definition) {
	page := base
	page.kind = ModeSearch
	page.permission = permission
	list := base
	list.kind = ModeList
	list.permission = permission
	one := base
	one.kind = ModeOne
	one.permission = permission
	registry[pageName] = page
	registry[listName] = list
	registry[oneName] = one
}

func sourceDef() definition {
	return definition{
		filters: map[string]fieldKind{"name": fieldLike, "type": fieldEq, "id": fieldUint},
		sorts: map[string]string{
			"id": "id", "name": "name", "createdAt": "created_at", "updatedAt": "updated_at",
		},
		defaultOrder: "id desc",
		fetch: func(db *gorm.DB, filters map[string]any, order string, offset, limit int) (any, int64, error) {
			items := []model.Source{}
			total, err := countAndFind(func() *gorm.DB {
				q := tenant.Scope(db.Session(&gorm.Session{}).Model(&model.Source{}))
				if value, ok := filters["name"].(string); ok {
					q = q.Where("name LIKE ? ESCAPE '\\'", "%"+escapeLike(value)+"%")
				}
				if value, ok := filters["type"].(string); ok {
					q = q.Where("type = ?", value)
				}
				if value, ok := filters["id"].(uint); ok {
					q = q.Where("id = ?", value)
				}
				return q
			}, order, offset, limit, &items)
			return dto.Sources(items), total, err
		},
	}
}

func pipelineDef() definition {
	return definition{
		filters: map[string]fieldKind{"name": fieldLike, "enabled": fieldBool, "id": fieldUint},
		sorts: map[string]string{
			"id": "id", "name": "name", "createdAt": "created_at", "updatedAt": "updated_at",
		},
		defaultOrder: "id desc",
		fetch: func(db *gorm.DB, filters map[string]any, order string, offset, limit int) (any, int64, error) {
			items := []model.Pipeline{}
			total, err := countAndFind(func() *gorm.DB {
				q := tenant.Scope(db.Session(&gorm.Session{}).Model(&model.Pipeline{}))
				if value, ok := filters["name"].(string); ok {
					q = q.Where("name LIKE ? ESCAPE '\\'", "%"+escapeLike(value)+"%")
				}
				if value, ok := filters["enabled"].(bool); ok {
					q = q.Where("enabled = ?", value)
				}
				if value, ok := filters["id"].(uint); ok {
					q = q.Where("id = ?", value)
				}
				return q
			}, order, offset, limit, &items)
			return dto.Pipelines(items), total, err
		},
	}
}

func runDef() definition {
	return definition{
		filters: map[string]fieldKind{"status": fieldEq, "pipelineId": fieldUint, "id": fieldUint},
		sorts: map[string]string{
			"id": "id", "startedAt": "started_at",
		},
		defaultOrder: "id desc",
		fetch: func(db *gorm.DB, filters map[string]any, order string, offset, limit int) (any, int64, error) {
			items := []model.Run{}
			total, err := countAndFind(func() *gorm.DB {
				q := tenant.Scope(db.Session(&gorm.Session{}).Model(&model.Run{}))
				if value, ok := filters["status"].(string); ok {
					q = q.Where("status = ?", value)
				}
				if value, ok := filters["pipelineId"].(uint); ok {
					q = q.Where("pipeline_id = ?", value)
				}
				if value, ok := filters["id"].(uint); ok {
					q = q.Where("id = ?", value)
				}
				return q
			}, order, offset, limit, &items)
			return dto.Runs(items), total, err
		},
	}
}

func datasetDef() definition {
	return definition{
		filters: map[string]fieldKind{"name": fieldLike, "kind": fieldEq},
		sorts: map[string]string{
			"id": "id", "name": "name", "updatedAt": "updated_at",
		},
		defaultOrder: "updated_at desc",
		fetch: func(db *gorm.DB, filters map[string]any, order string, offset, limit int) (any, int64, error) {
			items := []model.Dataset{}
			total, err := countAndFind(func() *gorm.DB {
				q := tenant.Scope(db.Session(&gorm.Session{}).Model(&model.Dataset{}))
				if value, ok := filters["name"].(string); ok {
					q = q.Where("name LIKE ? ESCAPE '\\'", "%"+escapeLike(value)+"%")
				}
				if value, ok := filters["kind"].(string); ok {
					q = q.Where("kind = ?", value)
				}
				return q
			}, order, offset, limit, &items)
			return dto.Datasets(items), total, err
		},
	}
}

func countAndFind(query func() *gorm.DB, order string, offset, limit int, dest any) (int64, error) {
	var total int64
	if err := query().Count(&total).Error; err != nil {
		return 0, err
	}
	err := query().Order(order).Offset(offset).Limit(limit).Find(dest).Error
	return total, err
}

func castFilter(kind fieldKind, value any) (any, error) {
	switch kind {
	case fieldLike, fieldEq:
		text, ok := value.(string)
		if !ok {
			return nil, protocol.Bad("查询条件必须是字符串")
		}
		return strings.TrimSpace(text), nil
	case fieldUint:
		switch n := value.(type) {
		case float64:
			if n < 1 || n != float64(uint(n)) {
				return nil, protocol.Bad("查询条件必须是正整数")
			}
			return uint(n), nil
		case uint:
			if n == 0 {
				return nil, protocol.Bad("查询条件必须是正整数")
			}
			return n, nil
		case int:
			if n < 1 {
				return nil, protocol.Bad("查询条件必须是正整数")
			}
			return uint(n), nil
		case string:
			parsed, err := strconv.ParseUint(strings.TrimSpace(n), 10, 64)
			if err != nil || parsed == 0 {
				return nil, protocol.Bad("查询条件必须是正整数")
			}
			return uint(parsed), nil
		default:
			return nil, protocol.Bad("查询条件必须是正整数")
		}
	case fieldBool:
		switch n := value.(type) {
		case bool:
			return n, nil
		case string:
			switch strings.ToLower(strings.TrimSpace(n)) {
			case "true", "1":
				return true, nil
			case "false", "0":
				return false, nil
			default:
				return nil, protocol.Bad("查询条件必须是布尔值")
			}
		default:
			return nil, protocol.Bad("查询条件必须是布尔值")
		}
	default:
		return nil, protocol.Bad("不支持的查询条件")
	}
}

func escapeLike(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `%`, `\%`)
	value = strings.ReplaceAll(value, `_`, `\_`)
	return value
}

func firstRow(rows any) (any, error) {
	value := reflect.ValueOf(rows)
	if value.Kind() != reflect.Slice || value.Len() == 0 {
		return nil, protocol.NotFound("数据不存在")
	}
	return value.Index(0).Interface(), nil
}
