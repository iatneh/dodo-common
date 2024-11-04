package d2model

import "time"

// Model 基础数据模型，所有表都有的属性
type Model struct {
	Id        int64      `json:"-" gorm:"primary_key;column:id"`
	CreatedAt *time.Time `json:"created_at,omitempty" gorm:"column:created_at"`
	UpdatedAt *time.Time `json:"updated_at,omitempty" gorm:"column:updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" gorm:"column:deleted_at"`
	Version   int64      `json:"-" gorm:"column:version"`
}
