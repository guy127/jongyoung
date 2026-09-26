package review

import "errors"

var (
	ErrRestaurantNotFound = errors.New("restaurant not found")
	ErrOwnRestaurant      = errors.New("owner cannot review own restaurant") // 403 OWN_RESTAURANT
	ErrReviewExists       = errors.New("review already exists")              // 409 REVIEW_EXISTS
	ErrReviewNotFound     = errors.New("review not found")                   // 404
)
