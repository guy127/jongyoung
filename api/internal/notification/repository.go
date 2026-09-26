package notification

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Insert เขียนแจ้งเตือนหนึ่งรายการ โดย snapshot ข้อมูลการจองจาก DB ในคำสั่งเดียว (INSERT … SELECT)
// รับ *gorm.DB ของผู้เรียก — booking/restaurant ส่ง tx ของตัวเองมา แจ้งเตือนจึงอยู่ในทรานแซกชันเดียวกับงานนั้นเสมอ
// (จองไม่สำเร็จ = ไม่มีแจ้งเตือนหลงเหลือ; ยกเลิกสำเร็จ = ลูกค้าได้รับแจ้งแน่นอน)
// ต้องคืน error เมื่อไม่พบการจอง (0 rows affected) — ป้องกันแจ้งเตือนหายเงียบ
func Insert(ctx context.Context, db *gorm.DB, d Draft) error {
	res := db.WithContext(ctx).Exec(`
		INSERT INTO notifications (user_id, kind, booking_id, restaurant_id, restaurant_name, customer_name,
			business_date, start_at, end_at, party_size, reason)
		SELECT ?, ?, b.id, r.id, r.name, u.display_name, ?, b.start_at, b.end_at, b.party_size, ?
		FROM bookings b
		JOIN restaurants r ON r.id = b.restaurant_id
		JOIN users u ON u.id = b.user_id
		WHERE b.id = ?`, d.Recipient, d.Kind, d.BusinessDate, d.Reason, d.BookingID)

	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("notification: ไม่พบการจอง %s", d.BookingID)
	}
	return nil
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *repository {
	return &repository{db: db}
}

// ListByUser = แจ้งเตือนล่าสุดของผู้รับ (ใหม่สุดก่อน, id เป็น tie-breaker)
func (r *repository) ListByUser(ctx context.Context, userID uuid.UUID, limit int) ([]Notification, error) {
	list := []Notification{}
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC, id").Limit(limit).Find(&list).Error
	return list, err
}

func (r *repository) CountUnread(ctx context.Context, userID uuid.UUID) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&Notification{}).Where("user_id = ? AND read_at IS NULL", userID).Count(&n).Error
	return n, err
}

// MarkRead ใส่ user_id ใน WHERE ด้วย — ของคนอื่นได้ 0 แถว (ไม่ต้องอ่านมาเช็คเจ้าของก่อน); อ่านแล้วไม่ทับเวลาเดิม
func (r *repository) MarkRead(ctx context.Context, userID, id uuid.UUID, now time.Time) (bool, error) {
	res := r.db.WithContext(ctx).Model(&Notification{}).Where("id = ? AND user_id = ?", id, userID).
		Update("read_at", gorm.Expr("COALESCE(read_at, ?)", now))
	return res.RowsAffected > 0, res.Error
}

func (r *repository) MarkAllRead(ctx context.Context, userID uuid.UUID, now time.Time) error {
	return r.db.WithContext(ctx).Model(&Notification{}).Where("user_id = ? AND read_at IS NULL", userID).
		Update("read_at", now).Error
}
