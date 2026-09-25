// Package database เปิดการเชื่อมต่อ PostgreSQL ผ่าน GORM
package database

import (
	"context"
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Open เปิด connection แล้ว ping ให้แน่ใจว่าต่อได้จริงก่อนเริ่มรับ request
func Open(ctx context.Context, dsn string, debug bool) (*gorm.DB, error) {
	level := logger.Warn
	if debug {
		level = logger.Info
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:  logger.Default.LogMode(level),
		NowFunc: func() time.Time { return time.Now().UTC() }, // เก็บเวลาเป็น UTC ทั้งหมด
	})
	if err != nil {
		return nil, fmt.Errorf("เปิด database: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetMaxIdleConns(5)
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return db, nil
}
