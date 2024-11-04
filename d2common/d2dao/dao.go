package d2dao

import (
	"fmt"
	"github.com/iatneh/dodo-common/d2conf/orm"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"reflect"
	"strings"
	"time"
)

// Dao 业务数据库
type Dao struct {
	conn *orm.Connect
}

// New 建立数据库连接
func New(config *orm.Config) *Dao {
	var dao *Dao
	ce := strings.ToLower(config.Endpoint)
	if strings.Contains(ce, "host=") && strings.Contains(ce, "dbname=") {
		dao = &Dao{conn: orm.NewPostgreSQL(config)}
	} else {
		dao = &Dao{conn: orm.NewMySQL(config)}
	}

	// 注册回调方法
	err := dao.conn.RW().Callback().Create().Replace("gorm:update_time_stamp", dao.createCallback)
	if err != nil {
		panic(fmt.Sprintf("callback create err: %v", err))
	}
	err = dao.conn.RW().Callback().Update().Replace("gorm:update_time_stamp", dao.updateCallback)
	if err != nil {
		panic(fmt.Sprintf("callback update err: %v", err))
	}
	return dao
}

// Delete 删除记录
func (d *Dao) Delete(v interface{}, where ...interface{}) error {
	if len(where) > 0 {
		return d.conn.RW().Delete(v, where...).Error
	}
	return d.conn.RW().Delete(v).Error
}

// Exists 檢查記錄指定條件的記錄是否存在
func (d *Dao) Exists(v interface{}, query string, where ...interface{}) (bool, error) {
	var count int64
	if err := d.conn.RW(&orm.Options{Unscoped: true}).Model(v).
		Where(query, where...).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// RO 返回数据库只读连接
func (d *Dao) RO(opts ...*orm.Options) *gorm.DB {
	return d.conn.RO(opts...)
}

// RW 返回数据库读写连接
func (d *Dao) RW(opts ...*orm.Options) *gorm.DB {
	return d.conn.RW(opts...)
}

// Close 关闭数据库连接
func (d *Dao) Close() {
	d.conn.Close()
}

// Save 保存数据
func (d *Dao) Save(v interface{}) error {
	return d.conn.RW().Model(v).Save(v).Error
}

// SaveBatch 批量保存数据
func (d *Dao) SaveBatch(v interface{}, batchSize int) error {
	return d.conn.RW().Model(v).CreateInBatches(v, batchSize).Error
}

// createCallback 创建新记录回调
func (d *Dao) createCallback(db *gorm.DB) {
	if db.Statement.Schema == nil {
		return
	}
	createdAt := db.Statement.Schema.LookUpField("CreatedAt")
	nowTime := time.Now().UTC()
	if createdAt != nil {
		_ = d.setFieldValue(db, createdAt, nowTime)
	}

	updatedAt := db.Statement.Schema.LookUpField("UpdatedAt")
	if updatedAt != nil {
		_ = d.setFieldValue(db, updatedAt, nowTime)
	}
}

// updateCallback 更新记录回调
func (d *Dao) updateCallback(db *gorm.DB) {
	if db.Statement.Schema == nil {
		return
	}
	updatedAt := db.Statement.Schema.LookUpField("UpdatedAt")
	if updatedAt != nil {
		_ = d.setFieldValue(db, updatedAt, time.Now().UTC())
	}
}

func (d *Dao) setFieldValue(db *gorm.DB, field *schema.Field, fieldValue interface{}) error {
	switch db.Statement.ReflectValue.Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < db.Statement.ReflectValue.Len(); i++ {
			// Get value from field
			if _, isZero := field.ValueOf(db.Statement.Context, db.Statement.ReflectValue.Index(i)); isZero {
				return field.Set(db.Statement.Context, db.Statement.ReflectValue, fieldValue)
			}
		}
	case reflect.Struct:
		// Get value from field
		if _, isZero := field.ValueOf(db.Statement.Context, db.Statement.ReflectValue); isZero {
			return field.Set(db.Statement.Context, db.Statement.ReflectValue, fieldValue)
		}
	default:
		return fmt.Errorf("unsupport type:%d", db.Statement.ReflectValue.Kind())
	}
	return nil
}
