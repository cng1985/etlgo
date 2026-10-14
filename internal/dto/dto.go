package dto

import (
	"encoding/json"
	"time"

	"etlgo/internal/model"
)

type Clock time.Time

func (c Clock) MarshalJSON() ([]byte, error) {
	t := time.Time(c)
	if t.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(t.Format("2006-01-02 15:04:05"))
}

func clockPtr(t *time.Time) *Clock {
	if t == nil || t.IsZero() {
		return nil
	}
	value := Clock(*t)
	return &value
}

type SourceView struct {
	ID        uint            `json:"id"`
	Name      string          `json:"name"`
	Type      string          `json:"type"`
	Config    json.RawMessage `json:"config"`
	CreatedAt Clock           `json:"createdAt"`
	UpdatedAt Clock           `json:"updatedAt"`
}

type PipelineView struct {
	ID              uint            `json:"id"`
	Name            string          `json:"name"`
	Description     string          `json:"description"`
	Enabled         bool            `json:"enabled"`
	SourceID        uint            `json:"sourceId"`
	Steps           json.RawMessage `json:"steps"`
	TargetType      string          `json:"targetType"`
	TargetConfig    json.RawMessage `json:"targetConfig"`
	ScheduleMinutes int             `json:"scheduleMinutes"`
	CreatedAt       Clock           `json:"createdAt"`
	UpdatedAt       Clock           `json:"updatedAt"`
}

type RunView struct {
	ID            uint   `json:"id"`
	PipelineID    uint   `json:"pipelineId"`
	PipelineName  string `json:"pipelineName"`
	Trigger       string `json:"trigger"`
	Status        string `json:"status"`
	State         int    `json:"state"`
	StateName     string `json:"stateName"`
	RowsExtracted int    `json:"rowsExtracted"`
	RowsFiltered  int    `json:"rowsFiltered"`
	RowsLoaded    int    `json:"rowsLoaded"`
	Error         string `json:"error"`
	Log           string `json:"log,omitempty"`
	StartedAt     Clock  `json:"startedAt"`
	FinishedAt    *Clock `json:"finishedAt"`
}

type DatasetView struct {
	ID         uint            `json:"id"`
	Name       string          `json:"name"`
	Kind       string          `json:"kind"`
	Path       string          `json:"path"`
	PipelineID uint            `json:"pipelineId"`
	Columns    json.RawMessage `json:"columns"`
	RowCount   int             `json:"rowCount"`
	UpdatedAt  Clock           `json:"updatedAt"`
}

func Source(item model.Source) SourceView {
	return SourceView{
		ID:        item.ID,
		Name:      item.Name,
		Type:      item.Type,
		Config:    rawJSON(item.Config),
		CreatedAt: Clock(item.CreatedAt),
		UpdatedAt: Clock(item.UpdatedAt),
	}
}

func Sources(items []model.Source) []SourceView {
	out := make([]SourceView, len(items))
	for i, item := range items {
		out[i] = Source(item)
	}
	return out
}

func Pipeline(item model.Pipeline) PipelineView {
	return PipelineView{
		ID:              item.ID,
		Name:            item.Name,
		Description:     item.Description,
		Enabled:         item.Enabled,
		SourceID:        item.SourceID,
		Steps:           rawJSON(item.Steps),
		TargetType:      item.TargetType,
		TargetConfig:    rawJSON(item.TargetConfig),
		ScheduleMinutes: item.ScheduleMinutes,
		CreatedAt:       Clock(item.CreatedAt),
		UpdatedAt:       Clock(item.UpdatedAt),
	}
}

func Pipelines(items []model.Pipeline) []PipelineView {
	out := make([]PipelineView, len(items))
	for i, item := range items {
		out[i] = Pipeline(item)
	}
	return out
}

func Run(item model.Run, withLog bool) RunView {
	state, name := runState(item.Status)
	view := RunView{
		ID:            item.ID,
		PipelineID:    item.PipelineID,
		PipelineName:  item.PipelineName,
		Trigger:       item.Trigger,
		Status:        item.Status,
		State:         state,
		StateName:     name,
		RowsExtracted: item.RowsExtracted,
		RowsFiltered:  item.RowsFiltered,
		RowsLoaded:    item.RowsLoaded,
		Error:         item.Error,
		StartedAt:     Clock(item.StartedAt),
		FinishedAt:    clockPtr(item.FinishedAt),
	}
	if withLog {
		view.Log = item.Log
	}
	return view
}

func Runs(items []model.Run) []RunView {
	out := make([]RunView, len(items))
	for i, item := range items {
		out[i] = Run(item, false)
	}
	return out
}

func Dataset(item model.Dataset) DatasetView {
	return DatasetView{
		ID:         item.ID,
		Name:       item.Name,
		Kind:       item.Kind,
		Path:       item.Path,
		PipelineID: item.PipelineID,
		Columns:    rawJSON(item.Columns),
		RowCount:   item.RowCount,
		UpdatedAt:  Clock(item.UpdatedAt),
	}
}

func Datasets(items []model.Dataset) []DatasetView {
	out := make([]DatasetView, len(items))
	for i, item := range items {
		out[i] = Dataset(item)
	}
	return out
}

func rawJSON(text string) json.RawMessage {
	if text == "" {
		return json.RawMessage("null")
	}
	if json.Valid([]byte(text)) {
		return json.RawMessage(text)
	}
	encoded, _ := json.Marshal(text)
	return encoded
}

func runState(status string) (int, string) {
	switch status {
	case model.RunRunning:
		return 1, "运行中"
	case model.RunSuccess:
		return 2, "成功"
	case model.RunFailed:
		return 3, "失败"
	default:
		return 0, "未知"
	}
}
