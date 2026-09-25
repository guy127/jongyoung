package review

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type Review struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	RestaurantID uuid.UUID `gorm:"type:uuid;not null"`
	UserID       uuid.UUID `gorm:"type:uuid;not null"`
	Score        int
	Body         string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// View = รีวิว + ชื่อผู้เขียน
type View struct {
	Review
	AuthorName string
}

var (
	ErrRestaurantNotFound = errors.New("restaurant not found")
	ErrOwnRestaurant      = errors.New("owner cannot review own restaurant") // 403 OWN_RESTAURANT
	ErrReviewExists       = errors.New("review already exists")              // 409 REVIEW_EXISTS
	ErrReviewNotFound     = errors.New("review not found")                   // 404
)
