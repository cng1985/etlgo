package engine

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"unicode"

	"gorm.io/gorm"
)

var identRe = regexp.MustCompile(`^[\p{L}_][\p{L}\p{N}_]{0,63}$`)

type LoadResult struct {
	Name    string
	Kind    string
	Path    string
	Columns []string
	Rows    int
}

func LoadSQLite(db *gorm.DB, tableName, mode string, table *Table) (*LoadResult, error) {
	name := strings.TrimSpace(tableName)
	if !identRe.MatchString(name) {
		return nil, fmt.Errorf("表名不合法: %s", tableName)
	}
	if reservedTable(name) {
		return nil, fmt.Errorf("不能写入系统表 %s", name)
	}
	if mode == "" {
		mode = "replace"
	}
	if mode != "replace" && mode != "append" {
		return nil, fmt.Errorf("写入模式仅支持 replace 或 append")
	}
	for _, col := range table.Columns {
		if !identRe.MatchString(col) {
			return nil, fmt.Errorf("列名不合法: %s", col)
		}
	}
	quoted, err := quoteIdent(name)
	if err != nil {
		return nil, err
	}
	tx := db.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()
	if mode == "replace" {
		if err := tx.Exec("DROP TABLE IF EXISTS " + quoted).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
	} else if err := ensureAppendable(tx, name, quoted, table.Columns); err != nil {
		tx.Rollback()
		return nil, err
	}
	if err := tx.Exec(createTableSQL(quoted, table)).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("创建表失败: %w", err)
	}
	if len(table.Rows) > 0 {
		if err := insertRows(tx, quoted, table); err != nil {
			tx.Rollback()
			return nil, err
		}
	}
	if err := tx.Commit().Error; err != nil {
		return nil, err
	}
	return &LoadResult{Name: name, Kind: "sqlite", Columns: table.Columns, Rows: len(table.Rows)}, nil
}

func LoadFile(path, kind string, table *Table) (*LoadResult, error) {
	if err := EnsureParent(path); err != nil {
		return nil, err
	}
	switch kind {
	case "csv":
		if err := writeCSV(path, table); err != nil {
			return nil, err
		}
	case "json":
		if err := writeJSON(path, table); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("不支持的文件类型 %s", kind)
	}
	return &LoadResult{Name: fileDatasetName(path), Kind: kind, Path: path, Columns: table.Columns, Rows: len(table.Rows)}, nil
}

func PreviewSQLite(db *gorm.DB, tableName string, limit int) (*Table, int, error) {
	if !identRe.MatchString(tableName) {
		return nil, 0, fmt.Errorf("表名不合法")
	}
	quoted, err := quoteIdent(tableName)
	if err != nil {
		return nil, 0, err
	}
	var total int64
	if err := db.Raw("SELECT COUNT(*) FROM " + quoted).Scan(&total).Error; err != nil {
		return nil, 0, err
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := db.Raw(fmt.Sprintf("SELECT * FROM %s LIMIT %d", quoted, limit)).Rows()
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return nil, 0, err
	}
	table := &Table{Columns: columns}
	for rows.Next() {
		holders := make([]any, len(columns))
		ptrs := make([]any, len(columns))
		for i := range holders {
			ptrs[i] = &holders[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, 0, err
		}
		row := make(Record, len(columns))
		for i, col := range columns {
			row[col] = normalizeSQL(holders[i])
		}
		table.Rows = append(table.Rows, row)
	}
	return table, int(total), rows.Err()
}

func reservedTable(name string) bool {
	switch strings.ToLower(name) {
	case "sources", "pipelines", "runs", "datasets", "seed_products", "sqlite_sequence":
		return true
	default:
		return false
	}
}

func DropSQLite(db *gorm.DB, tableName string) error {
	if !identRe.MatchString(tableName) {
		return fmt.Errorf("表名不合法")
	}
	if reservedTable(tableName) {
		return fmt.Errorf("不能删除系统表")
	}
	quoted, err := quoteIdent(tableName)
	if err != nil {
		return err
	}
	return db.Exec("DROP TABLE IF EXISTS " + quoted).Error
}

func ensureAppendable(tx *gorm.DB, name, quoted string, columns []string) error {
	var count int64
	if err := tx.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?", name).Scan(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return nil
	}
	rows, err := tx.Raw("PRAGMA table_info(" + quoted + ")").Rows()
	if err != nil {
		return err
	}
	defer rows.Close()
	existing := map[string]struct{}{}
	for rows.Next() {
		var cid int
		var colName, colType string
		var notNull, pk int
		var dflt any
		if err := rows.Scan(&cid, &colName, &colType, &notNull, &dflt, &pk); err != nil {
			return err
		}
		existing[colName] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(existing) != len(columns) {
		return fmt.Errorf("追加失败，目标表字段与当前结果不一致")
	}
	for _, col := range columns {
		if _, ok := existing[col]; !ok {
			return fmt.Errorf("追加失败，目标表缺少字段 %s", col)
		}
	}
	return nil
}

func quoteIdent(name string) (string, error) {
	if !identRe.MatchString(name) {
		return "", fmt.Errorf("标识符不合法: %s", name)
	}
	return `"` + name + `"`, nil
}

func createTableSQL(table string, data *Table) string {
	parts := make([]string, len(data.Columns))
	for i, col := range data.Columns {
		q, _ := quoteIdent(col)
		parts[i] = q + " " + sqlType(data.Rows, col)
	}
	if len(parts) == 0 {
		parts = []string{`"_empty" TEXT`}
	}
	return fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)", table, strings.Join(parts, ", "))
}

func sqlType(rows []Record, col string) string {
	kind := ""
	for _, row := range rows {
		switch row[col].(type) {
		case nil:
			continue
		case int, int64, int32, bool:
			if kind == "TEXT" {
				return "TEXT"
			}
			if kind == "" {
				kind = "INTEGER"
			}
		case float64, float32:
			if kind == "TEXT" {
				return "TEXT"
			}
			kind = "REAL"
		default:
			if isEmpty(row[col]) {
				continue
			}
			return "TEXT"
		}
	}
	if kind == "" {
		return "TEXT"
	}
	return kind
}

func insertRows(tx *gorm.DB, table string, data *Table) error {
	if len(data.Columns) == 0 {
		return nil
	}
	cols := make([]string, len(data.Columns))
	holders := make([]string, len(data.Columns))
	for i, col := range data.Columns {
		q, err := quoteIdent(col)
		if err != nil {
			return err
		}
		cols[i] = q
		holders[i] = "?"
	}
	stmt := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, strings.Join(cols, ", "), strings.Join(holders, ", "))
	for _, row := range data.Rows {
		args := make([]any, len(data.Columns))
		for i, col := range data.Columns {
			args[i] = sqlArg(row[col])
		}
		if err := tx.Exec(stmt, args...).Error; err != nil {
			return fmt.Errorf("写入行失败: %w", err)
		}
	}
	return nil
}

func sqlArg(v any) any {
	switch n := v.(type) {
	case nil:
		return nil
	case bool:
		if n {
			return 1
		}
		return 0
	default:
		return n
	}
}

func writeCSV(path string, table *Table) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	if err := w.Write(table.Columns); err != nil {
		return err
	}
	for _, row := range table.Rows {
		rec := make([]string, len(table.Columns))
		for i, col := range table.Columns {
			rec[i] = formatCell(row[col])
		}
		if err := w.Write(rec); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

func writeJSON(path string, table *Table) error {
	rows := make([]map[string]any, len(table.Rows))
	for i, row := range table.Rows {
		obj := make(map[string]any, len(table.Columns))
		for _, col := range table.Columns {
			obj[col] = row[col]
		}
		rows[i] = obj
	}
	b, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func fileDatasetName(path string) string {
	base := path
	if i := strings.LastIndex(path, "/"); i >= 0 {
		base = path[i+1:]
	}
	ext := ""
	if i := strings.LastIndex(base, "."); i >= 0 {
		ext = base[i:]
		base = base[:i]
	}
	var b strings.Builder
	for _, r := range base {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	name := strings.Trim(b.String(), "_")
	if name == "" {
		name = "dataset"
	}
	if ext == ".csv" || ext == ".json" {
		name += strings.TrimPrefix(ext, ".")
	}
	return name
}
