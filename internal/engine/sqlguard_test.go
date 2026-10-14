package engine

import (
	"strings"
	"testing"
)

func TestAssertReadOnlySQL(t *testing.T) {
	ok := []string{
		"select id, name from orders",
		"SELECT * FROM t WHERE name = 'drop table'",
		"with c as (select 1 as id) select * from c",
		"-- note\nselect 1",
		"select updated_at from t",
	}
	for _, query := range ok {
		if err := assertReadOnlySQL(query); err != nil {
			t.Fatalf("%s: %v", query, err)
		}
	}
	bad := []string{
		"",
		"delete from t",
		"select 1; drop table t",
		"with c as (select 1) insert into t select * from c",
		"select * from t into outfile '/tmp/x'",
		"select load_file('/etc/passwd')",
		"select pg_read_file('x')",
		"/* unfinished",
		"select 'unterminated",
	}
	for _, query := range bad {
		if err := assertReadOnlySQL(query); err == nil {
			t.Fatalf("expected reject: %s", query)
		}
	}
}

func TestPrepareDB(t *testing.T) {
	cfg, err := PrepareDB("mysql", DBConfig{
		Host:     "db.internal",
		User:     "reader",
		Password: "p@ss word",
		Database: "sales",
		Query:    "select 1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != 3306 {
		t.Fatalf("port=%d", cfg.Port)
	}
	dsn := mysqlDSN(cfg)
	if !strings.Contains(dsn, "tcp(db.internal:3306)/sales") || strings.Contains(dsn, "allowAllFiles=true") {
		t.Fatal(dsn)
	}

	pg, err := PrepareDB("pg", DBConfig{
		Host: "[::1]", User: "reader", Password: "a/b c", Database: "app db", Query: "select 1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if pg.Port != 5432 || pg.SSLMode != "disable" || pg.Host != "::1" {
		t.Fatalf("%#v", pg)
	}
	pgDSN := postgresDSN(pg)
	if !strings.Contains(pgDSN, "postgres://") || !strings.Contains(pgDSN, "sslmode=disable") || strings.Contains(pgDSN, "a/b c") {
		t.Fatal(pgDSN)
	}

	ms, err := PrepareDB("sqlserver", DBConfig{
		Host: "10.0.0.8", Port: 1433, User: "reader", Database: "erp", Query: "select 1", SSLMode: "true",
	})
	if err != nil {
		t.Fatal(err)
	}
	msDSN := sqlServerDSN(ms)
	if !strings.Contains(msDSN, "encrypt=true") || !strings.Contains(msDSN, "database=erp") {
		t.Fatal(msDSN)
	}

	if _, err := PrepareDB("mysql", DBConfig{Host: "bad host", User: "a", Database: "b", Query: "select 1"}); err == nil {
		t.Fatal("expected bad host")
	}
	if _, err := PrepareDB("pg", DBConfig{Host: "localhost", User: "a", Database: "b", Query: "select 1", SSLMode: "prefer"}); err == nil {
		t.Fatal("expected bad ssl")
	}
	if _, err := PrepareDB("mysql", DBConfig{Host: "localhost", User: "a", Database: "b", Query: "delete from t"}); err == nil {
		t.Fatal("expected bad query")
	}
}
