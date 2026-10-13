package engine

import "encoding/json"

type Record map[string]any

type Table struct {
	Columns []string `json:"columns"`
	Rows    []Record `json:"rows"`
}

type Step struct {
	Type    string            `json:"type"`
	Column  string            `json:"column,omitempty"`
	Columns []string          `json:"columns,omitempty"`
	Op      string            `json:"op,omitempty"`
	Value   string            `json:"value,omitempty"`
	To      string            `json:"to,omitempty"`
	Mapping map[string]string `json:"mapping,omitempty"`
	Left    string            `json:"left,omitempty"`
	Right   string            `json:"right,omitempty"`
	Sep     string            `json:"sep,omitempty"`
	Old     string            `json:"old,omitempty"`
	New     string            `json:"new,omitempty"`
}

type Preview struct {
	Columns   []string `json:"columns"`
	Rows      []Record `json:"rows"`
	Total     int      `json:"total"`
	Filtered  int      `json:"filtered,omitempty"`
	Logs      []string `json:"logs,omitempty"`
	Truncated bool     `json:"truncated"`
}

func ParseSteps(raw string) ([]Step, error) {
	if raw == "" || raw == "null" {
		return nil, nil
	}
	var steps []Step
	if err := json.Unmarshal([]byte(raw), &steps); err != nil {
		return nil, err
	}
	return steps, nil
}

func MustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}
