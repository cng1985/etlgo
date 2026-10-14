package engine

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode"

	"gorm.io/gorm"
)

const maxRows = 100000

func ExtractCSV(path, delimiter string, hasHeader bool) (*Table, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("打开 CSV 失败: %w", err)
	}
	defer f.Close()
	reader := csv.NewReader(f)
	reader.FieldsPerRecord = -1
	if delimiter != "" {
		runes := []rune(delimiter)
		if len(runes) != 1 {
			return nil, fmt.Errorf("分隔符必须是单个字符")
		}
		reader.Comma = runes[0]
	}
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("解析 CSV 失败: %w", err)
	}
	if len(records) == 0 {
		return &Table{}, nil
	}
	start := 0
	var columns []string
	if hasHeader {
		columns = sanitizeHeaders(records[0])
		start = 1
	} else {
		columns = make([]string, len(records[0]))
		for i := range columns {
			columns[i] = fmt.Sprintf("col_%d", i+1)
		}
	}
	if len(records)-start > maxRows {
		return nil, fmt.Errorf("行数超过上限 %d", maxRows)
	}
	rows := make([]Record, 0, len(records)-start)
	for _, rec := range records[start:] {
		row := make(Record, len(columns))
		for i, col := range columns {
			if i < len(rec) {
				row[col] = rec[i]
			} else {
				row[col] = ""
			}
		}
		rows = append(rows, row)
	}
	return &Table{Columns: columns, Rows: rows}, nil
}

func ExtractJSON(path string) (*Table, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("打开 JSON 失败: %w", err)
	}
	return tableFromJSON(b, "")
}

func ExtractHTTP(ctx context.Context, url, dataPath string) (*Table, error) {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return nil, fmt.Errorf("仅支持 http 或 https 地址")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求数据源失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("数据源返回状态 %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return nil, err
	}
	return tableFromJSON(body, dataPath)
}

func ExtractSQLite(db *gorm.DB, query string) (*Table, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return nil, fmt.Errorf("SQL 不能为空")
	}
	lower := strings.ToLower(q)
	if !(strings.HasPrefix(lower, "select") || strings.HasPrefix(lower, "with")) {
		return nil, fmt.Errorf("仅允许 SELECT 查询")
	}
	if strings.Contains(q, ";") {
		return nil, fmt.Errorf("不允许多条 SQL")
	}
	rows, err := db.Raw(q).Rows()
	if err != nil {
		return nil, fmt.Errorf("执行查询失败: %w", err)
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	table := &Table{Columns: columns}
	for rows.Next() {
		if len(table.Rows) >= maxRows {
			return nil, fmt.Errorf("行数超过上限 %d", maxRows)
		}
		holders := make([]any, len(columns))
		ptrs := make([]any, len(columns))
		for i := range holders {
			ptrs[i] = &holders[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		row := make(Record, len(columns))
		for i, col := range columns {
			row[col] = normalizeSQL(holders[i])
		}
		table.Rows = append(table.Rows, row)
	}
	return table, rows.Err()
}

func tableFromJSON(body []byte, dataPath string) (*Table, error) {
	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("解析 JSON 失败: %w", err)
	}
	if dataPath != "" {
		current := payload
		for _, part := range strings.Split(dataPath, ".") {
			obj, ok := current.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("路径 %s 不是对象", dataPath)
			}
			current, ok = obj[part]
			if !ok {
				return nil, fmt.Errorf("找不到路径 %s", dataPath)
			}
		}
		payload = current
	}
	arr, ok := payload.([]any)
	if !ok {
		return nil, fmt.Errorf("JSON 需要是对象数组")
	}
	if len(arr) > maxRows {
		return nil, fmt.Errorf("行数超过上限 %d", maxRows)
	}
	columns := make([]string, 0)
	seen := map[string]struct{}{}
	rows := make([]Record, 0, len(arr))
	for _, item := range arr {
		obj, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("JSON 数组元素必须是对象")
		}
		row := make(Record, len(obj))
		for k, v := range obj {
			row[k] = v
			if _, ok := seen[k]; !ok {
				seen[k] = struct{}{}
				columns = append(columns, k)
			}
		}
		rows = append(rows, row)
	}
	return &Table{Columns: columns, Rows: rows}, nil
}

func sanitizeHeaders(headers []string) []string {
	used := map[string]int{}
	out := make([]string, len(headers))
	for i, header := range headers {
		name := sanitizeIdent(header)
		if name == "" {
			name = fmt.Sprintf("col_%d", i+1)
		}
		if n, ok := used[name]; ok {
			n++
			used[name] = n
			name = fmt.Sprintf("%s_%d", name, n)
		} else {
			used[name] = 1
		}
		out[i] = name
	}
	return out
}

func sanitizeIdent(name string) string {
	name = strings.TrimSpace(name)
	var b strings.Builder
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			b.WriteRune(r)
		} else if r == ' ' || r == '-' {
			b.WriteRune('_')
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return ""
	}
	first := []rune(out)[0]
	if unicode.IsDigit(first) {
		out = "c_" + out
	}
	if len([]rune(out)) > 64 {
		out = string([]rune(out)[:64])
	}
	return out
}

func normalizeSQL(v any) any {
	switch n := v.(type) {
	case nil:
		return nil
	case []byte:
		return string(n)
	default:
		return n
	}
}
