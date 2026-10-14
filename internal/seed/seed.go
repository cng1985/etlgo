package seed

import (
	"embed"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"

	"etlgo/internal/config"
	"etlgo/internal/engine"
	"etlgo/internal/model"

	"gorm.io/gorm"
)

//go:embed samples/*
var samples embed.FS

func Apply(db *gorm.DB, cfg *config.Config) error {
	if err := writeSamples(cfg.DataDir); err != nil {
		return err
	}
	if err := seedProducts(db); err != nil {
		return err
	}
	var n int64
	if err := db.Model(&model.Source{}).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	orders := model.Source{
		Name:   "示例订单 CSV",
		Type:   model.SourceCSV,
		Config: `{"path":"samples/orders.csv","delimiter":",","hasHeader":true}`,
	}
	customers := model.Source{
		Name:   "示例客户 JSON",
		Type:   model.SourceJSON,
		Config: `{"path":"samples/customers.json"}`,
	}
	products := model.Source{
		Name:   "示例产品表",
		Type:   model.SourceSQLite,
		Config: `{"query":"select sku, name, category, price, stock from seed_products"}`,
	}
	if err := db.Create(&orders).Error; err != nil {
		return err
	}
	if err := db.Create(&customers).Error; err != nil {
		return err
	}
	if err := db.Create(&products).Error; err != nil {
		return err
	}
	paidSteps := []engine.Step{
		{Type: "filter", Column: "status", Op: "eq", Value: "paid"},
		{Type: "cast", Column: "amount", To: "float"},
		{Type: "cast", Column: "quantity", To: "int"},
		{Type: "derive", Column: "total", Op: "mul", Left: "amount", Right: "quantity"},
		{Type: "rename", Mapping: map[string]string{"customer": "customer_name", "created_at": "order_date"}},
		{Type: "select", Columns: []string{"id", "customer_name", "product", "amount", "quantity", "total", "order_date"}},
	}
	vipSteps := []engine.Step{
		{Type: "filter", Column: "vip", Op: "eq", Value: "true"},
		{Type: "derive", Column: "label", Op: "concat", Columns: []string{"name", "city"}, Sep: " · "},
		{Type: "select", Columns: []string{"id", "label", "email"}},
	}
	stockSteps := []engine.Step{
		{Type: "filter", Column: "stock", Op: "lt", Value: "40"},
		{Type: "cast", Column: "price", To: "float"},
		{Type: "cast", Column: "stock", To: "int"},
	}
	pipelines := []model.Pipeline{
		{
			Name:            "已支付订单汇总",
			Description:     "从订单 CSV 中保留已支付记录，计算金额合计并写入 SQLite。",
			Enabled:         true,
			SourceID:        orders.ID,
			Steps:           must(paidSteps),
			TargetType:      model.TargetSQLite,
			TargetConfig:    `{"table":"orders_paid","mode":"replace"}`,
			ScheduleMinutes: 0,
		},
		{
			Name:            "会员客户导入",
			Description:     "从客户 JSON 中筛出会员，生成展示名后写入 SQLite。",
			Enabled:         true,
			SourceID:        customers.ID,
			Steps:           must(vipSteps),
			TargetType:      model.TargetSQLite,
			TargetConfig:    `{"table":"vip_customers","mode":"replace"}`,
			ScheduleMinutes: 0,
		},
		{
			Name:            "低库存产品导出",
			Description:     "查询产品表，导出库存低于 40 的记录为 CSV。",
			Enabled:         true,
			SourceID:        products.ID,
			Steps:           must(stockSteps),
			TargetType:      model.TargetCSV,
			TargetConfig:    `{"path":"outputs/low_stock.csv"}`,
			ScheduleMinutes: 0,
		},
	}
	return db.Create(&pipelines).Error
}

func writeSamples(dataDir string) error {
	return fs.WalkDir(samples, "samples", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		body, err := samples.ReadFile(path)
		if err != nil {
			return err
		}
		target := filepath.Join(dataDir, filepath.FromSlash(path))
		if _, err := os.Stat(target); err == nil {
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, body, 0o644)
	})
}

func seedProducts(db *gorm.DB) error {
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS seed_products (
		sku TEXT,
		name TEXT,
		category TEXT,
		price REAL,
		stock INTEGER
	)`).Error; err != nil {
		return err
	}
	var n int64
	if err := db.Raw("SELECT COUNT(*) FROM seed_products").Scan(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	rows := []struct {
		SKU, Name, Category string
		Price               float64
		Stock               int
	}{
		{"KB-01", "机械键盘", "外设", 299, 80},
		{"MN-02", "显示器", "显示", 1899, 12},
		{"MS-03", "鼠标", "外设", 89, 140},
		{"HS-04", "耳机", "音频", 459, 26},
		{"DK-05", "扩展坞", "配件", 199, 8},
		{"CAM-06", "摄像头", "视频", 329, 35},
		{"MIC-07", "麦克风", "音频", 699, 6},
		{"HUB-08", "集线器", "配件", 99, 54},
	}
	for _, row := range rows {
		if err := db.Exec(
			"INSERT INTO seed_products (sku, name, category, price, stock) VALUES (?, ?, ?, ?, ?)",
			row.SKU, row.Name, row.Category, row.Price, row.Stock,
		).Error; err != nil {
			return err
		}
	}
	return nil
}

func must(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
