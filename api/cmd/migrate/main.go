// cmd/migrate รัน goose migration ที่ฝังอยู่ในไบนารี
// ใช้เป็น service "migrate" ใน compose (รันก่อน api) และรันเองได้:
//
//	go run ./cmd/migrate up
//	go run ./cmd/migrate down   # ถอย 1 ขั้น
package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"

	"jongyoung/internal/config"
	"jongyoung/migrations"

	_ "github.com/jackc/pgx/v5/stdlib" // driver "pgx" สำหรับ database/sql
	"github.com/pressly/goose/v3"
)

func main() {
	if err := run(); err != nil {
		slog.Error("migrate ล้มเหลว", "error", err)
		os.Exit(1)
	}
}

func run() error {
	command := "up"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	db, err := sql.Open("pgx", cfg.Database.DSN)
	if err != nil {
		return err
	}
	defer db.Close()

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	switch command {
	case "up":
		return goose.Up(db, ".")
	case "down":
		return goose.Down(db, ".")
	case "status":
		return goose.Status(db, ".")
	default:
		return fmt.Errorf("ไม่รู้จักคำสั่ง %q (ใช้ up, down, status)", command)
	}
}
