package tenant

import "gorm.io/gorm"

const Default = "default"

func Scope(db *gorm.DB) *gorm.DB {
	return db.Where("tenant_id = ?", Default)
}
