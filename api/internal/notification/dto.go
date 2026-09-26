package notification

import (
	"time"

	"github.com/google/uuid"
)

// bangkok = เวลาไทย (ไม่มี DST) — ไม่ import booking.Bangkok เพราะ booking import package นี้ (import cycle)
var bangkok = time.FixedZone("Asia/Bangkok", 7*60*60)

type NotificationResponse struct {
	ID             uuid.UUID  `json:"id"`
	Kind           string     `json:"kind" example:"booking_cancelled_by_restaurant"`
	BookingID      uuid.UUID  `json:"booking_id"`
	RestaurantID   uuid.UUID  `json:"restaurant_id"`
	RestaurantName string     `json:"restaurant_name"`
	CustomerName   string     `json:"customer_name"`
	BusinessDate   string     `json:"business_date" example:"2026-10-10"`
	StartAt        time.Time  `json:"start_at"`
	EndAt          time.Time  `json:"end_at"`
	PartySize      int        `json:"party_size"`
	Reason         string     `json:"reason"`
	ReadAt         *time.Time `json:"read_at"`
	CreatedAt      time.Time  `json:"created_at"`
}

type ListResponse struct {
	Items       []NotificationResponse `json:"items"`
	UnreadCount int64                  `json:"unread_count"`
}

func NewNotificationResponse(n Notification) NotificationResponse {
	return NotificationResponse{
		ID: n.ID, Kind: n.Kind, BookingID: n.BookingID, RestaurantID: n.RestaurantID,
		RestaurantName: n.RestaurantName, CustomerName: n.CustomerName,
		BusinessDate: n.BusinessDate.Format("2006-01-02"),
		StartAt:      n.StartAt.In(bangkok), EndAt: n.EndAt.In(bangkok),
		PartySize: n.PartySize, Reason: n.Reason, ReadAt: n.ReadAt, CreatedAt: n.CreatedAt.In(bangkok),
	}
}
