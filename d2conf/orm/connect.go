package orm

import (
	"context"
	"crypto/md5"
	"fmt"
	"gorm.io/driver/mysql"
	_ "gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	_ "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"math/rand"
	"strings"
	"sync"
	"time"
)

// 連接緩存
var connCache sync.Map

// Connect 数据库连接
type Connect struct {
	db   *gorm.DB
	roDb []*gorm.DB
}

// RW 返回主(读写)连接
func (c *Connect) RW(opts ...*Options) *gorm.DB {
	if c.db != nil {
		return opt(opts...).parsed(c.db)
	}
	return nil
}

// RO 返回只读库连接, 若未设置只读连接，则返回主连接
func (c *Connect) RO(opts ...*Options) *gorm.DB {
	if len(c.roDb) == 0 {
		return opt(opts...).parsed(c.db)
	}
	return opt(opts...).parsed(c.roDb[rand.Intn(len(c.roDb))])
}

// Close 关闭数据库连接
func (c *Connect) Close() {
	if c.db != nil {
		sqlDB, err := c.db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	}

	for _, cc := range c.roDb {
		if db, err := cc.DB(); err != nil {
			_ = db.Close()
		}
	}
}

// Ping 主连接 ping
func (c *Connect) Ping(ctx context.Context) error {
	db, _ := c.RW().DB()
	if db == nil {
		return fmt.Errorf("rw disconnection")
	}
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping rw connection error,%v", err.Error())
	}

	var errMsg []string
	for _, cc := range c.roDb {
		if db, _ := cc.DB(); db == nil {
			continue
		}
		if err := db.PingContext(ctx); err != nil {
			errMsg = append(errMsg, err.Error())
		}
	}
	if len(errMsg) > 0 {
		return fmt.Errorf("ping ro's connection error,%v", strings.Join(errMsg, ";"))
	}

	return nil
}

// newConnect 建立数据库连接
func newConnect(dialect string, c *Config) *Connect {
	cacheKey := func(endpoint string) string {
		return fmt.Sprintf("%s-%x", dialect, md5.Sum([]byte(endpoint)))
	}

	conn := func(e string) *gorm.DB {
		v, ok := connCache.Load(cacheKey(e))
		if ok && v != nil && v.(*gorm.DB) != nil {
			if sqlDB, err := v.(*gorm.DB).DB(); err != nil && sqlDB.Ping() == nil {
				return v.(*gorm.DB)
			}
		}
		var db *gorm.DB
		var err error
		if dialect == "postgres" {
			db, err = gorm.Open(postgres.Open(e))
			if err != nil {
				panic(err)
			}
		} else if dialect == "mysql" {
			db, err = gorm.Open(mysql.Open(e))
			if err != nil {
				panic(err)
			}
		}
		sqlDB, err := db.DB()
		if err != nil {
			panic(err)
		}
		sqlDB.SetMaxIdleConns(c.Idle)
		sqlDB.SetMaxOpenConns(c.Active)
		sqlDB.SetConnMaxLifetime(c.IdleTimeout * time.Second)
		connCache.Store(cacheKey(e), db)
		return db
	}

	rc := &Connect{
		db: conn(c.Endpoint),
	}

	for _, e := range c.RoEndpoint {
		rc.roDb = append(rc.roDb, conn(e))
	}
	return rc
}

// NewPostgreSQL create new postgresql connect
// Endpoint format:
// host=pg-host port=5432 user=db-user password=db-password dbname=database-name sslmode=require application_name=application-name
func NewPostgreSQL(c *Config) *Connect {
	if !strings.Contains(strings.ToLower(c.Endpoint), "application_name=") {
		c.Endpoint += " application_name=golang-e2util "
	}
	for idx := range c.RoEndpoint {
		rc := c.RoEndpoint[idx]
		if !strings.Contains(strings.ToLower(rc), "application_name=") {
			c.RoEndpoint[idx] += " application_name=golang-e2util "
		}
	}
	return newConnect("postgres", c)
}

// NewMySQL create new postgresql connect
// Endpoint format:
// db-user:db-password@tcp(db-host:db-port)/database-name
func NewMySQL(c *Config) *Connect {
	return newConnect("mysql", c)
}
