// Package migrations ฝังไฟล์ SQL ของ goose ไว้ในไบนารี
// ทั้ง cmd/migrate และ integration test (testcontainers) ใช้ชุดเดียวกันนี้
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
