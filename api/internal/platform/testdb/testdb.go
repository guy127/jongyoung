// Package testdb เปิด PostgreSQL จริงใน container (testcontainers) สำหรับ repository/integration test
// และรัน migrations ทุกไฟล์ใน migrations/ — จึงห้ามใส่ seed data ใน migrations/
package testdb

import (
	"context"
	"database/sql"
	"testing"

	"jongyoung/migrations"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	gormpg "gorm.io/driver/postgres"
)

// New คืน *gorm.DB ที่ต่อกับฐานข้อมูลใหม่เอี่ยม (หนึ่ง container ต่อหนึ่งการเรียก)
// ข้ามเมื่อรัน go test -short เพราะต้องใช้ Docker
func New(t *testing.T) *gorm.DB {
	t.Helper()
	if testing.Short() {
		t.Skip("ข้าม integration test ใน -short mode (ต้องใช้ Docker)")
	}
	ctx := context.Background()

	ctr, err := postgres.Run(ctx, "postgres:17-alpine",
		postgres.WithDatabase("jongyoung_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		postgres.BasicWaitStrategies(),
	)
	testcontainers.CleanupContainer(t, ctr)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}

	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open sql: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatal(err)
	}
	goose.SetLogger(goose.NopLogger())
	if err := goose.Up(sqlDB, "."); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	db, err := gorm.Open(gormpg.New(gormpg.Config{Conn: sqlDB}), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("open gorm: %v", err)
	}
	return db
}
