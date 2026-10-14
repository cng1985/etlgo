package engine

import (
	"context"
	"strings"
	"testing"
)

func TestRemoteDatabaseRejectsClosedPort(t *testing.T) {
	cfg := DBConfig{
		Host:     "127.0.0.1",
		Port:     1,
		User:     "reader",
		Password: "secret",
		Database: "shop",
		Query:    "select 1",
	}
	cases := []struct {
		name string
		fn   func() error
	}{
		{"mysql", func() error { _, err := ExtractMySQL(context.Background(), cfg); return err }},
		{"pg", func() error { _, err := ExtractPostgres(context.Background(), cfg); return err }},
		{"sqlserver", func() error { _, err := ExtractSQLServer(context.Background(), cfg); return err }},
	}
	for _, item := range cases {
		err := item.fn()
		if err == nil || !strings.Contains(err.Error(), "连接数据库失败") {
			t.Fatalf("%s err=%v", item.name, err)
		}
		if strings.Contains(err.Error(), "secret") {
			t.Fatalf("%s leaked password: %v", item.name, err)
		}
	}
}
