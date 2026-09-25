package user

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User คือผู้ใช้ในฐานข้อมูลเรา ผูกกับ Keycloak ด้วย KeycloakUID (= claims.sub)
// Keycloak เก็บ identity ส่วนตารางนี้มีไว้ให้ FK ของ restaurants/bookings/reviews ชี้มา
type User struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	KeycloakUID string    `gorm:"not null"`
	Email       string    `gorm:"not null"`
	DisplayName string    `gorm:"not null"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

// OwnedRestaurant คือร้านที่ผู้ใช้เป็นเจ้าของ (ใช้ใน GET /me)
type OwnedRestaurant struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}
