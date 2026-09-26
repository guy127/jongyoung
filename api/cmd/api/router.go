package main

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"jongyoung/internal/booking"
	"jongyoung/internal/config"
	"jongyoung/internal/middleware"
	"jongyoung/internal/notification"
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
// ลำดับ middleware: Recovery → RequestLog (request id + log) → CORS → RateLimit (เฉพาะการเขียน) → (JWT เฉพาะ route ที่ต้อง login)
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
	notificationHandler := notification.NewHandler(notification.NewService(notification.NewRepository(db), time.Now))

	var auth gin.HandlerFunc
	if cfg.App.DevAuth {
		auth = middleware.DevAuth(userService)
	} else {
		verifier := middleware.NewOIDCVerifier(ctx, cfg.Keycloak.Issuer(), cfg.Keycloak.Audience)
		auth = middleware.JWT(verifier, userService)
	}

	r := gin.New()
	// api อยู่หลัง Caddy: เชื่อ X-Forwarded-For เฉพาะเมื่อมาจากเครือข่ายภายใน (docker) เท่านั้น
	// ไม่งั้น ClientIP จะเป็น IP ของ Caddy สำหรับทุกคน หรือใครก็ปลอม header มาเลี่ยง rate limit ได้
	if err := r.SetTrustedProxies([]string{"127.0.0.1", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"}); err != nil {
		panic(err) // ค่าคงที่ในโค้ด — ผิดได้แค่ตอนเขียนผิดเท่านั้น
	}
	r.Use(gin.Recovery(), middleware.RequestLog())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{cfg.App.CORSOrigin},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{middleware.RequestIDHeader, "Retry-After"}, // ให้ JavaScript ฝั่ง browser อ่านได้
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	// กันยิงจอง/รีวิวถี่ ๆ แบบเบา ๆ: 30 คำขอที่เขียนข้อมูลต่อนาทีต่อ IP (คนกดจริงไม่มีทางถึง)
	r.Use(middleware.RateLimit(30, time.Minute, time.Now))

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
	rest.GET("/:id/closures", restaurantHandler.ListClosures)
	rest.POST("/:id/closures", auth, restaurantHandler.CreateClosure)
	rest.DELETE("/:id/closures/:closureId", auth, restaurantHandler.DeleteClosure)
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

	v1.GET("/me/notifications", auth, notificationHandler.ListMine)
	v1.PUT("/me/notifications/read-all", auth, notificationHandler.MarkAllRead)
	v1.PUT("/me/notifications/:id/read", auth, notificationHandler.MarkRead)

	return r
}
