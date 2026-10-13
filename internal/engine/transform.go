package engine

import (
	"fmt"
	"strings"
)

func Apply(table *Table, steps []Step) (*Table, int, []string, error) {
	current := &Table{Columns: append([]string{}, table.Columns...), Rows: cloneRows(table.Rows)}
	filtered := 0
	logs := make([]string, 0, len(steps))
	for i, step := range steps {
		var err error
		var removed int
		var note string
		switch step.Type {
		case "select":
			current, note, err = applySelect(current, step)
		case "drop":
			current, note, err = applyDrop(current, step)
		case "rename":
			current, note, err = applyRename(current, step)
		case "filter":
			current, removed, note, err = applyFilter(current, step)
			filtered += removed
		case "cast":
			current, note, err = applyCast(current, step)
		case "derive":
			current, note, err = applyDerive(current, step)
		default:
			err = fmt.Errorf("不支持的步骤类型 %s", step.Type)
		}
		if err != nil {
			return nil, filtered, logs, fmt.Errorf("第 %d 步（%s）失败: %w", i+1, step.Type, err)
		}
		logs = append(logs, note)
	}
	return current, filtered, logs, nil
}

func cloneRows(rows []Record) []Record {
	out := make([]Record, len(rows))
	for i, row := range rows {
		next := make(Record, len(row))
		for k, v := range row {
			next[k] = v
		}
		out[i] = next
	}
	return out
}

func applySelect(table *Table, step Step) (*Table, string, error) {
	if len(step.Columns) == 0 {
		return nil, "", fmt.Errorf("select 需要指定列")
	}
	known := columnSet(table.Columns)
	for _, col := range step.Columns {
		if _, ok := known[col]; !ok {
			return nil, "", fmt.Errorf("列 %s 不存在", col)
		}
	}
	rows := make([]Record, len(table.Rows))
	for i, row := range table.Rows {
		next := make(Record, len(step.Columns))
		for _, col := range step.Columns {
			next[col] = row[col]
		}
		rows[i] = next
	}
	return &Table{Columns: append([]string{}, step.Columns...), Rows: rows},
		fmt.Sprintf("保留列 %s", strings.Join(step.Columns, ", ")), nil
}

func applyDrop(table *Table, step Step) (*Table, string, error) {
	if len(step.Columns) == 0 {
		return nil, "", fmt.Errorf("drop 需要指定列")
	}
	drop := columnSet(step.Columns)
	cols := make([]string, 0, len(table.Columns))
	for _, col := range table.Columns {
		if _, ok := drop[col]; !ok {
			cols = append(cols, col)
		}
	}
	if len(cols) == len(table.Columns) {
		return nil, "", fmt.Errorf("没有匹配到要删除的列")
	}
	rows := make([]Record, len(table.Rows))
	for i, row := range table.Rows {
		next := make(Record, len(cols))
		for _, col := range cols {
			next[col] = row[col]
		}
		rows[i] = next
	}
	return &Table{Columns: cols, Rows: rows},
		fmt.Sprintf("删除列 %s", strings.Join(step.Columns, ", ")), nil
}

func applyRename(table *Table, step Step) (*Table, string, error) {
	if len(step.Mapping) == 0 {
		return nil, "", fmt.Errorf("rename 需要字段映射")
	}
	known := columnSet(table.Columns)
	for from := range step.Mapping {
		if _, ok := known[from]; !ok {
			return nil, "", fmt.Errorf("列 %s 不存在", from)
		}
	}
	used := map[string]struct{}{}
	cols := make([]string, len(table.Columns))
	for i, col := range table.Columns {
		name := col
		if to, ok := step.Mapping[col]; ok {
			name = strings.TrimSpace(to)
			if name == "" {
				return nil, "", fmt.Errorf("列 %s 的新名称不能为空", col)
			}
		}
		if _, ok := used[name]; ok {
			return nil, "", fmt.Errorf("重命名后列名重复: %s", name)
		}
		used[name] = struct{}{}
		cols[i] = name
	}
	rows := make([]Record, len(table.Rows))
	for i, row := range table.Rows {
		next := make(Record, len(cols))
		for idx, col := range table.Columns {
			next[cols[idx]] = row[col]
		}
		rows[i] = next
	}
	return &Table{Columns: cols, Rows: rows}, fmt.Sprintf("重命名 %d 个字段", len(step.Mapping)), nil
}

func applyFilter(table *Table, step Step) (*Table, int, string, error) {
	if step.Column == "" {
		return nil, 0, "", fmt.Errorf("filter 需要指定列")
	}
	if !hasColumn(table.Columns, step.Column) {
		return nil, 0, "", fmt.Errorf("列 %s 不存在", step.Column)
	}
	kept := make([]Record, 0, len(table.Rows))
	for _, row := range table.Rows {
		ok, err := match(row[step.Column], step.Op, step.Value)
		if err != nil {
			return nil, 0, "", err
		}
		if ok {
			kept = append(kept, row)
		}
	}
	removed := len(table.Rows) - len(kept)
	return &Table{Columns: table.Columns, Rows: kept}, removed,
		fmt.Sprintf("过滤 %s %s %s，保留 %d 行，去掉 %d 行", step.Column, step.Op, step.Value, len(kept), removed), nil
}

func match(val any, op, expect string) (bool, error) {
	switch op {
	case "eq":
		return equals(val, expect), nil
	case "ne":
		return !equals(val, expect), nil
	case "contains":
		return strings.Contains(strings.ToLower(asString(val)), strings.ToLower(expect)), nil
	case "not_contains":
		return !strings.Contains(strings.ToLower(asString(val)), strings.ToLower(expect)), nil
	case "empty":
		return isEmpty(val), nil
	case "not_empty":
		return !isEmpty(val), nil
	case "gt", "gte", "lt", "lte":
		return compare(val, expect, op)
	default:
		return false, fmt.Errorf("不支持的过滤条件 %s", op)
	}
}

func applyCast(table *Table, step Step) (*Table, string, error) {
	if step.Column == "" || step.To == "" {
		return nil, "", fmt.Errorf("cast 需要列和目标类型")
	}
	if !hasColumn(table.Columns, step.Column) {
		return nil, "", fmt.Errorf("列 %s 不存在", step.Column)
	}
	for _, row := range table.Rows {
		casted, err := castValue(row[step.Column], step.To)
		if err != nil {
			return nil, "", err
		}
		row[step.Column] = casted
	}
	return table, fmt.Sprintf("将 %s 转为 %s", step.Column, step.To), nil
}

func applyDerive(table *Table, step Step) (*Table, string, error) {
	name := strings.TrimSpace(step.Column)
	if name == "" {
		return nil, "", fmt.Errorf("derive 需要新列名")
	}
	cols := append([]string{}, table.Columns...)
	if !hasColumn(cols, name) {
		cols = append(cols, name)
	}
	for _, row := range table.Rows {
		val, err := deriveValue(row, step)
		if err != nil {
			return nil, "", err
		}
		row[name] = val
	}
	return &Table{Columns: cols, Rows: table.Rows},
		fmt.Sprintf("生成列 %s（%s）", name, step.Op), nil
}

func deriveValue(row Record, step Step) (any, error) {
	switch step.Op {
	case "const":
		return step.Value, nil
	case "concat":
		if len(step.Columns) == 0 {
			return nil, fmt.Errorf("concat 需要列")
		}
		sep := step.Sep
		parts := make([]string, 0, len(step.Columns))
		for _, col := range step.Columns {
			parts = append(parts, asString(row[col]))
		}
		return strings.Join(parts, sep), nil
	case "lower":
		return strings.ToLower(asString(cell(row, step.Left))), nil
	case "upper":
		return strings.ToUpper(asString(cell(row, step.Left))), nil
	case "trim":
		return strings.TrimSpace(asString(cell(row, step.Left))), nil
	case "replace":
		return strings.ReplaceAll(asString(cell(row, step.Left)), step.Old, step.New), nil
	case "add", "sub", "mul", "div":
		left, ok := asNumber(cell(row, step.Left))
		if !ok {
			return nil, fmt.Errorf("列 %s 不是数值", step.Left)
		}
		right, ok := operand(row, step.Right)
		if !ok {
			return nil, fmt.Errorf("右操作数 %s 不是数值", step.Right)
		}
		switch step.Op {
		case "add":
			return left + right, nil
		case "sub":
			return left - right, nil
		case "mul":
			return left * right, nil
		case "div":
			if right == 0 {
				return nil, nil
			}
			return left / right, nil
		}
	case "coalesce":
		for _, col := range step.Columns {
			if !isEmpty(row[col]) {
				return row[col], nil
			}
		}
		return step.Value, nil
	default:
		return nil, fmt.Errorf("不支持的派生运算 %s", step.Op)
	}
	return nil, fmt.Errorf("不支持的派生运算 %s", step.Op)
}

func cell(row Record, col string) any {
	if col == "" {
		return nil
	}
	return row[col]
}

func operand(row Record, token string) (float64, bool) {
	if token == "" {
		return 0, false
	}
	if v, ok := row[token]; ok {
		return asNumber(v)
	}
	return asNumber(token)
}

func columnSet(cols []string) map[string]struct{} {
	set := make(map[string]struct{}, len(cols))
	for _, col := range cols {
		set[col] = struct{}{}
	}
	return set
}

func hasColumn(cols []string, name string) bool {
	_, ok := columnSet(cols)[name]
	return ok
}
