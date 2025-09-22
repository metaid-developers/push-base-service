package major

import (
	"database/sql"
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"push-base-service/conf"
)

var (
	db    *gorm.DB
	sqlDB *sql.DB
)

func InitSqlConfig() {
	dsn := conf.RdsDsn
	fmt.Println(dsn)
	gdb, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(fmt.Errorf("DB init error %s", err.Error()))
	}
	sqlDB, err = gdb.DB()
	if err != nil {
		panic(fmt.Errorf("sqlDB error %s", err.Error()))
	}
	sqlDB.SetMaxOpenConns(conf.RdsMaxOpenConns)
	sqlDB.SetMaxIdleConns(conf.RdsMaxIgleConns)
	db = gdb
}

func GetSqlDB() *gorm.DB {
	if db != nil {
		return db
	}
	return nil
}
