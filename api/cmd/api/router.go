package main

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"jongyoung/internal/booking"
	"jongyoung/internal/config"
	"jongyoung/internal/middleware"
	"jongyoung/internal/restaurant"
	"jongyoung/internal/review"
	"jongyoung/internal/user"

	_ "jongyoung/docs" // register swagger spec

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"
)

// newRouter ประกอบ dependency, middleware และ route ทั้งหมด
// ลำดับ middleware: Recovery → Logger → CORS → (JWT เฉพาะ route ที่ต้อง login)
func newRouter(ctx context.Context, cfg config.Config, db *gorm.DB, sqlDB *sql.DB) *gin.Engine {
	if !cfg.App.IsDevelopment() {
		gin.SetMode(gin.ReleaseMode)
	}

	// dependency injection
	userService := user.NewService(user.NewRepository(db))
	userHandler := user.NewHandler(userService)
	restaurantHandler := restaurant.NewHandler(restaurant.NewService(restaurant.NewRepository(db), time.Now))
	reviewHandler := review.NewHandler(review.NewService(review.NewRepository(db)))
	bookingHandler := booking.NewHandler(booking.NewService(booking.NewRepository(db), time.Now), time.Now)

	var auth gin.HandlerFunc
	if cfg.App.DevAuth {
		auth = middleware.DevAuth(userService)
	} else {
		verifier := middleware.NewOIDCVerifier(ctx, cfg.Keycloak.Issuer(), cfg.Keycloak.Audience)
		auth = middleware.JWT(verifier, userService)
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

	// เอกสาร API: http://api.jongyoung.localhost/swagger/index.html (สร้างใหม่ด้วย: swag init -g cmd/api/main.go -o docs --parseInternal)
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	v1 := r.Group("/api/v1")
	v1.GET("/me", auth, userHandler.Me)

	rest := v1.Group("/restaurants")
	rest.GET("", restaurantHandler.List)
	rest.GET("/:id", restaurantHandler.Get)
	rest.GET("/:id/availability", restaurantHandler.Availability)
	rest.GET("/:id/next-available", restaurantHandler.NextAvailable)
	rest.POST("", auth, restaurantHandler.Create)
	rest.PUT("/:id", auth, restaurantHandler.Update)
	rest.DELETE("/:id", auth, restaurantHandler.Delete)
	rest.POST("/:id/images", auth, restaurantHandler.AddImage)
	rest.DELETE("/:id/images/:imageId", auth, restaurantHandler.DeleteImage)
	rest.GET("/:id/bookings", auth, bookingHandler.Board)
	rest.GET("/:id/reviews", reviewHandler.List)
	rest.GET("/:id/reviews/mine", auth, reviewHandler.Mine)
	rest.POST("/:id/reviews", auth, reviewHandler.Create)
	rest.PUT("/:id/reviews", auth, reviewHandler.Update)
	rest.DELETE("/:id/reviews", auth, reviewHandler.Delete)

	v1.GET("/me/bookings", auth, bookingHandler.ListMine)
	bookings := v1.Group("/bookings", auth)
	bookings.POST("", bookingHandler.Create)
	bookings.GET("/:id", bookingHandler.Get)
	bookings.PUT("/:id", bookingHandler.Update)
	bookings.DELETE("/:id", bookingHandler.Cancel)

	return r
}
