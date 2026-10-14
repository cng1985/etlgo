package store

import (
	"fmt"

	"etlgo/internal/config"
	"etlgo/internal/model"
	"etlgo/internal/tenant"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func New(cfg *config.Config) (*gorm.DB, error) {
	return open(cfg.DBPath, true)
}

func Open(path string) (*gorm.DB, error) {
	return open(path, false)
}

func open(path string, migrate bool) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(dsn(path)), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(1)
	if migrate {
		if err := db.AutoMigrate(&model.Source{}, &model.Pipeline{}, &model.Run{}, &model.Dataset{}); err != nil {
			return nil, err
		}
		for _, table := range []string{"sources", "pipelines", "runs", "datasets"} {
			if err := db.Exec("UPDATE "+table+" SET tenant_id = ? WHERE tenant_id IS NULL OR tenant_id = ''", tenant.Default).Error; err != nil {
				return nil, err
			}
		}
	}
	return db, nil
}

func dsn(path string) string {
	return fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)", path)
}
