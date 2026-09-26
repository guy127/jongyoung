package user

import "github.com/google/uuid"

type MeResponse struct {
	ID          uuid.UUID         `json:"id"`
	Email       string            `json:"email"`
	DisplayName string            `json:"display_name"`
	Restaurants []OwnedRestaurant `json:"restaurants"`
}
