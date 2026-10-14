package engine

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DBConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	Database string `json:"database"`
	Query    string `json:"query"`
	SSLMode  string `json:"sslMode"`
}

func PrepareDB(kind string, cfg DBConfig) (DBConfig, error) {
	cfg.Host = strings.TrimSpace(cfg.Host)
	cfg.User = strings.TrimSpace(cfg.User)
	cfg.Database = strings.TrimSpace(cfg.Database)
	cfg.Query = strings.TrimSpace(cfg.Query)
	cfg.SSLMode = strings.TrimSpace(cfg.SSLMode)
	if strings.HasPrefix(cfg.Host, "[") && strings.HasSuffix(cfg.Host, "]") {
		cfg.Host = strings.TrimSuffix(strings.TrimPrefix(cfg.Host, "["), "]")
	}
	if !validHost(cfg.Host) {
		return cfg, fmt.Errorf("主机无效")
	}
	if cfg.Port == 0 {
		switch kind {
		case "mysql":
			cfg.Port = 3306
		case "pg":
			cfg.Port = 5432
		case "sqlserver":
			cfg.Port = 1433
		}
	}
	if cfg.Port < 1 || cfg.Port > 65535 {
		return cfg, fmt.Errorf("端口无效")
	}
	if cfg.User == "" || hasControl(cfg.User) || len([]rune(cfg.User)) > 128 {
		return cfg, fmt.Errorf("用户名无效")
	}
	if hasControl(cfg.Password) || len([]rune(cfg.Password)) > 256 {
		return cfg, fmt.Errorf("密码无效")
	}
	if cfg.Database == "" || hasControl(cfg.Database) || strings.ContainsAny(cfg.Database, "/\\?#") || len([]rune(cfg.Database)) > 128 {
		return cfg, fmt.Errorf("数据库名无效")
	}
	if err := assertReadOnlySQL(cfg.Query); err != nil {
		return cfg, err
	}
	switch kind {
	case "mysql":
		cfg.SSLMode = ""
	case "pg":
		if cfg.SSLMode == "" {
			cfg.SSLMode = "disable"
		}
		switch cfg.SSLMode {
		case "disable", "require", "verify-ca", "verify-full":
		default:
			return cfg, fmt.Errorf("不支持的 SSL 模式")
		}
	case "sqlserver":
		if cfg.SSLMode == "" {
			cfg.SSLMode = "disable"
		}
		switch cfg.SSLMode {
		case "disable", "false", "true":
		default:
			return cfg, fmt.Errorf("不支持的加密模式")
		}
	default:
		return cfg, fmt.Errorf("不支持的数据源类型")
	}
	return cfg, nil
}

func ExtractMySQL(ctx context.Context, cfg DBConfig) (*Table, error) {
	cfg, err := PrepareDB("mysql", cfg)
	if err != nil {
		return nil, err
	}
	return extractRemote(ctx, mysql.Open(mysqlDSN(cfg)), cfg.Query)
}

func ExtractPostgres(ctx context.Context, cfg DBConfig) (*Table, error) {
	cfg, err := PrepareDB("pg", cfg)
	if err != nil {
		return nil, err
	}
	return extractRemote(ctx, postgres.Open(postgresDSN(cfg)), cfg.Query)
}

func ExtractSQLServer(ctx context.Context, cfg DBConfig) (*Table, error) {
	cfg, err := PrepareDB("sqlserver", cfg)
	if err != nil {
		return nil, err
	}
	return extractRemote(ctx, sqlserver.Open(sqlServerDSN(cfg)), cfg.Query)
}

func mysqlDSN(cfg DBConfig) string {
	conf := mysqldriver.Config{
		User:                 cfg.User,
		Passwd:               cfg.Password,
		Net:                  "tcp",
		Addr:                 net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)),
		DBName:               cfg.Database,
		ParseTime:            true,
		Loc:                  time.UTC,
		Timeout:              8 * time.Second,
		ReadTimeout:          60 * time.Second,
		WriteTimeout:         15 * time.Second,
		AllowNativePasswords: true,
		Params: map[string]string{
			"charset": "utf8mb4",
		},
	}
	return conf.FormatDSN()
}

func postgresDSN(cfg DBConfig) string {
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.User, cfg.Password),
		Host:   net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)),
		Path:   "/" + url.PathEscape(cfg.Database),
	}
	q := url.Values{}
	q.Set("sslmode", cfg.SSLMode)
	q.Set("connect_timeout", "8")
	u.RawQuery = q.Encode()
	return u.String()
}

func sqlServerDSN(cfg DBConfig) string {
	u := url.URL{
		Scheme: "sqlserver",
		User:   url.UserPassword(cfg.User, cfg.Password),
		Host:   net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)),
	}
	q := url.Values{}
	q.Set("database", cfg.Database)
	q.Set("encrypt", cfg.SSLMode)
	q.Set("dial timeout", "8")
	u.RawQuery = q.Encode()
	return u.String()
}

func extractRemote(ctx context.Context, dialector gorm.Dialector, query string) (*Table, error) {
	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %s", sanitizeDBError(err))
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %s", sanitizeDBError(err))
	}
	defer sqlDB.Close()
	sqlDB.SetMaxOpenConns(2)
	sqlDB.SetConnMaxLifetime(time.Minute)
	pingCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(pingCtx); err != nil {
		return nil, fmt.Errorf("连接数据库失败: %s", sanitizeDBError(err))
	}
	return readSQL(ctx, db, query)
}

func validHost(host string) bool {
	if host == "" || len(host) > 253 || strings.ContainsAny(host, " \t\r\n/@\\?#") {
		return false
	}
	if ip := net.ParseIP(host); ip != nil {
		return true
	}
	if strings.Contains(host, ":") {
		return false
	}
	labels := strings.Split(host, ".")
	for _, label := range labels {
		if label == "" || len(label) > 63 {
			return false
		}
		for i, r := range label {
			ok := unicodeLetterOrDigit(r) || r == '-' || r == '_'
			if !ok {
				return false
			}
			if (i == 0 || i == len(label)-1) && (r == '-' || r == '_') {
				return false
			}
		}
	}
	return true
}

func unicodeLetterOrDigit(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}

func hasControl(s string) bool {
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			return true
		}
	}
	return false
}

var (
	passwordPattern = regexp.MustCompile(`(?i)(password|pwd)\s*=\s*[^&\s']+`)
	userInfoPattern = regexp.MustCompile(`://[^/\s]+@`)
)

func sanitizeDBError(err error) string {
	msg := passwordPattern.ReplaceAllString(err.Error(), "$1=***")
	msg = userInfoPattern.ReplaceAllString(msg, "://***@")
	if len([]rune(msg)) > 300 {
		msg = string([]rune(msg)[:300])
	}
	return msg
}
