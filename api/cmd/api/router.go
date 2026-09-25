package main

import (
	"database/sql"
	"net/http"
	"time"

	"jongyoung/internal/config"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// newRouter ประกอบ middleware และ route ทั้งหมด
// ลำดับ middleware: Recovery → Logger → CORS → (JWT ต่อ route)
func newRouter(cfg config.Config, sqlDB *sql.DB) *gin.Engine {
	if !cfg.App.IsDevelopment() {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{cfg.App.CORSOrigin},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.GET("/healthz", func(c *gin.Context) {
		if err := sqlDB.PingContext(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "db unavailable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	return r
}
