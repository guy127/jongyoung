package notification

import (
	"context"

	"gorm.io/gorm"
)

// Insert เขียนแจ้งเตือนหนึ่งรายการ โดย snapshot ข้อมูลการจองจาก DB ในคำสั่งเดียว (INSERT … SELECT)
// รับ *gorm.DB ของผู้เรียก — booking/restaurant ส่ง tx ของตัวเองมา แจ้งเตือนจึงอยู่ในทรานแซกชันเดียวกับงานนั้นเสมอ
// (จองไม่สำเร็จ = ไม่มีแจ้งเตือนหลงเหลือ; ยกเลิกสำเร็จ = ลูกค้าได้รับแจ้งแน่นอน)
func Insert(ctx context.Context, db *gorm.DB, d Draft) error {
	return db.WithContext(ctx).Exec(`
		INSERT INTO notifications (user_id, kind, booking_id, restaurant_id, restaurant_name, customer_name,
			business_date, start_at, end_at, party_size, reason)
		SELECT ?, ?, b.id, r.id, r.name, u.display_name, ?, b.start_at, b.end_at, b.party_size, ?
		FROM bookings b
		JOIN restaurants r ON r.id = b.restaurant_id
		JOIN users u ON u.id = b.user_id
		WHERE b.id = ?`, d.Recipient, d.Kind, d.BusinessDate, d.Reason, d.BookingID).Error
}
