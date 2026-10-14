package engine

import (
	"fmt"
	"strings"
	"unicode"
)

func assertReadOnlySQL(query string) error {
	q := strings.TrimSpace(query)
	if q == "" {
		return fmt.Errorf("SQL 不能为空")
	}
	if strings.ContainsAny(q, ";\x00") || strings.Contains(q, "$$") {
		return fmt.Errorf("不允许多条 SQL")
	}
	words, err := sqlWords(q)
	if err != nil {
		return err
	}
	if len(words) == 0 || (words[0] != "select" && words[0] != "with") {
		return fmt.Errorf("仅允许 SELECT 查询")
	}
	forbidden := map[string]struct{}{
		"insert": {}, "update": {}, "delete": {}, "drop": {}, "alter": {},
		"create": {}, "truncate": {}, "grant": {}, "revoke": {}, "exec": {},
		"execute": {}, "merge": {}, "copy": {}, "call": {}, "attach": {},
		"detach": {}, "pragma": {}, "outfile": {}, "dumpfile": {},
		"openrowset": {}, "opendatasource": {}, "xp_cmdshell": {},
		"load_file": {}, "pg_read_file": {}, "pg_read_binary_file": {}, "pg_ls_dir": {},
	}
	for i, word := range words {
		if _, ok := forbidden[word]; ok {
			return fmt.Errorf("查询包含不允许的操作")
		}
		if word == "load" && i+1 < len(words) && words[i+1] == "data" {
			return fmt.Errorf("查询包含不允许的操作")
		}
		if word == "into" && i+1 < len(words) && (words[i+1] == "outfile" || words[i+1] == "dumpfile") {
			return fmt.Errorf("查询包含不允许的操作")
		}
	}
	return nil
}

func sqlWords(query string) ([]string, error) {
	var words []string
	var b strings.Builder
	flush := func() {
		if b.Len() == 0 {
			return
		}
		words = append(words, strings.ToLower(b.String()))
		b.Reset()
	}
	runes := []rune(query)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		switch r {
		case '\'':
			flush()
			i++
			for i < len(runes) {
				if runes[i] == '\'' {
					if i+1 < len(runes) && runes[i+1] == '\'' {
						i += 2
						continue
					}
					break
				}
				i++
			}
			if i >= len(runes) {
				return nil, fmt.Errorf("SQL 引号未闭合")
			}
		case '"', '`':
			quote := r
			flush()
			i++
			for i < len(runes) && runes[i] != quote {
				i++
			}
			if i >= len(runes) {
				return nil, fmt.Errorf("SQL 引号未闭合")
			}
		case '-':
			if i+1 < len(runes) && runes[i+1] == '-' {
				flush()
				for i < len(runes) && runes[i] != '\n' {
					i++
				}
				continue
			}
			flush()
		case '/':
			if i+1 < len(runes) && runes[i+1] == '*' {
				flush()
				i += 2
				closed := false
				for i+1 < len(runes) {
					if runes[i] == '*' && runes[i+1] == '/' {
						i++
						closed = true
						break
					}
					i++
				}
				if !closed {
					return nil, fmt.Errorf("SQL 注释未闭合")
				}
				continue
			}
			flush()
		default:
			if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
				b.WriteRune(r)
			} else {
				flush()
			}
		}
	}
	flush()
	return words, nil
}
