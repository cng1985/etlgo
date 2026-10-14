package model

import "time"

const (
	SourceCSV       = "csv"
	SourceExcel     = "excel"
	SourceJSON      = "json"
	SourceSQLite    = "sqlite"
	SourceHTTP      = "http"
	SourceMySQL     = "mysql"
	SourcePostgres  = "pg"
	SourceSQLServer = "sqlserver"

	TargetSQLite = "sqlite"
	TargetCSV    = "csv"
	TargetJSON   = "json"

	RunRunning = "running"
	RunSuccess = "success"
	RunFailed  = "failed"
)

type Source struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	TenantID  string    `json:"-" gorm:"size:64;not null;default:'default';index"`
	Name      string    `json:"name" gorm:"size:120;not null"`
	Type      string    `json:"type" gorm:"size:32;not null;index"`
	Config    string    `json:"config" gorm:"type:text;not null"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Pipeline struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	TenantID        string    `json:"-" gorm:"size:64;not null;default:'default';index"`
	Name            string    `json:"name" gorm:"size:120;not null"`
	Description     string    `json:"description" gorm:"size:500"`
	Enabled         bool      `json:"enabled" gorm:"not null;default:true"`
	SourceID        uint      `json:"sourceId" gorm:"index;not null"`
	Steps           string    `json:"steps" gorm:"type:text;not null"`
	TargetType      string    `json:"targetType" gorm:"size:32;not null"`
	TargetConfig    string    `json:"targetConfig" gorm:"type:text;not null"`
	ScheduleMinutes int       `json:"scheduleMinutes" gorm:"not null;default:0"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type Run struct {
	ID            uint       `json:"id" gorm:"primaryKey"`
	TenantID      string     `json:"-" gorm:"size:64;not null;default:'default';index"`
	PipelineID    uint       `json:"pipelineId" gorm:"index;not null"`
	PipelineName  string     `json:"pipelineName" gorm:"size:120;not null"`
	Trigger       string     `json:"trigger" gorm:"size:32;not null"`
	Status        string     `json:"status" gorm:"size:32;not null;index"`
	RowsExtracted int        `json:"rowsExtracted"`
	RowsFiltered  int        `json:"rowsFiltered"`
	RowsLoaded    int        `json:"rowsLoaded"`
	Error         string     `json:"error" gorm:"type:text"`
	Log           string     `json:"log,omitempty" gorm:"type:text"`
	StartedAt     time.Time  `json:"startedAt"`
	FinishedAt    *time.Time `json:"finishedAt"`
}

type Dataset struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	TenantID   string    `json:"-" gorm:"size:64;not null;default:'default';index"`
	Name       string    `json:"name" gorm:"uniqueIndex;size:128;not null"`
	Kind       string    `json:"kind" gorm:"size:32;not null"`
	Path       string    `json:"path" gorm:"size:500"`
	PipelineID uint      `json:"pipelineId" gorm:"index"`
	Columns    string    `json:"columns" gorm:"type:text"`
	RowCount   int       `json:"rowCount"`
	UpdatedAt  time.Time `json:"updatedAt"`
}
