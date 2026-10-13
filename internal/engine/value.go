package engine

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

func asString(v any) string {
	switch n := v.(type) {
	case nil:
		return ""
	case string:
		return n
	case float64:
		if math.IsNaN(n) || math.IsInf(n, 0) {
			return ""
		}
		if n == math.Trunc(n) {
			return strconv.FormatInt(int64(n), 10)
		}
		return strconv.FormatFloat(n, 'f', -1, 64)
	case float32:
		return asString(float64(n))
	case int:
		return strconv.Itoa(n)
	case int64:
		return strconv.FormatInt(n, 10)
	case int32:
		return strconv.FormatInt(int64(n), 10)
	case bool:
		if n {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprint(v)
	}
}

func asNumber(v any) (float64, bool) {
	switch n := v.(type) {
	case nil:
		return 0, false
	case float64:
		if math.IsNaN(n) || math.IsInf(n, 0) {
			return 0, false
		}
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case int32:
		return float64(n), true
	case string:
		s := strings.TrimSpace(n)
		if s == "" {
			return 0, false
		}
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, false
		}
		return f, true
	default:
		return asNumber(asString(v))
	}
}

func isEmpty(v any) bool {
	return strings.TrimSpace(asString(v)) == ""
}

func equals(left any, expect string) bool {
	ln, lok := asNumber(left)
	rn, rok := asNumber(expect)
	if lok && rok {
		return ln == rn
	}
	return asString(left) == expect
}

func compare(left any, expect, op string) (bool, error) {
	ln, lok := asNumber(left)
	rn, rok := asNumber(expect)
	if !lok || !rok {
		return false, fmt.Errorf("无法按数值比较 %q 与 %q", asString(left), expect)
	}
	switch op {
	case "gt":
		return ln > rn, nil
	case "gte":
		return ln >= rn, nil
	case "lt":
		return ln < rn, nil
	case "lte":
		return ln <= rn, nil
	default:
		return false, fmt.Errorf("不支持的比较运算 %s", op)
	}
}

func castValue(v any, to string) (any, error) {
	switch to {
	case "string":
		return asString(v), nil
	case "int":
		n, ok := asNumber(v)
		if !ok {
			if isEmpty(v) {
				return nil, nil
			}
			return nil, fmt.Errorf("无法将 %q 转为整数", asString(v))
		}
		return int64(math.Trunc(n)), nil
	case "float":
		n, ok := asNumber(v)
		if !ok {
			if isEmpty(v) {
				return nil, nil
			}
			return nil, fmt.Errorf("无法将 %q 转为小数", asString(v))
		}
		return n, nil
	case "bool":
		s := strings.ToLower(strings.TrimSpace(asString(v)))
		switch s {
		case "1", "true", "yes", "y", "是":
			return true, nil
		case "0", "false", "no", "n", "否", "":
			return false, nil
		default:
			return nil, fmt.Errorf("无法将 %q 转为布尔值", asString(v))
		}
	default:
		return nil, fmt.Errorf("不支持的类型 %s", to)
	}
}

func formatCell(v any) string {
	return asString(v)
}
