package restaurant

import (
	"context"
	"errors"
	"time"

	"jongyoung/internal/booking"
	"jongyoung/internal/notification"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *repository {
	return &repository{db: db}
}

// Transaction เรียก fn ด้วย repository ที่ผูกกับทรานแซกชันเดียวกัน — fn คืน error = rollback
func (r *repository) Transaction(ctx context.Context, fn func(tx Repository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&repository{db: tx})
	})
}

func (r *repository) Create(ctx context.Context, rest *Restaurant) error {
	return r.db.WithContext(ctx).Create(rest).Error // GORM สร้าง Images ให้ในทรานแซกชันเดียวกัน
}

func (r *repository) FindByID(ctx context.Context, id uuid.UUID) (Restaurant, error) {
	var rest Restaurant
	err := r.db.WithContext(ctx).
		Preload("Images", func(db *gorm.DB) *gorm.DB { return db.Order("sort_order, id") }).
		First(&rest, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Restaurant{}, ErrNotFound
	}
	return rest, err
}

// LockByID = SELECT ... FOR UPDATE — ใครแก้/จองร้านเดียวกันพร้อมกันต้องรอจนทรานแซกชันนี้จบ
func (r *repository) LockByID(ctx context.Context, id uuid.UUID) (Restaurant, error) {
	var rest Restaurant
	err := r.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&rest, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Restaurant{}, ErrNotFound
	}
	return rest, err
}

func (r *repository) Update(ctx context.Context, rest *Restaurant) error {
	return r.db.WithContext(ctx).Model(rest).Select(
		"name", "description", "cuisine", "address", "map_url", "seats",
		"open_minute", "close_minute", "closed_weekdays", "break_start_minute", "break_end_minute",
		"cancel_before_minutes", "updated_at",
	).Updates(rest).Error
}

func (r *repository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&Restaurant{}, "id = ?", id).Error
}

// FutureBookings = booking ที่ยัง active และยังไม่จบ (ใช้ตรวจตอนลดที่นั่ง/ย่นเวลา)
func (r *repository) FutureBookings(ctx context.Context, restaurantID uuid.UUID, now time.Time) ([]booking.Booking, error) {
	var bookings []booking.Booking
	err := r.db.WithContext(ctx).
		Where("restaurant_id = ? AND status = ? AND end_at > ?", restaurantID, booking.StatusActive, now).
		Order("start_at").
		Find(&bookings).Error
	return bookings, err
}

// upcomingQuery = การจอง active ที่ยังไม่เริ่ม + ชื่อลูกค้า — คนที่นั่งอยู่ในร้านแล้ว (เริ่มไปแล้ว) ไม่ถูกแตะ
// ล็อกแถวการจองที่จะยกเลิก — ลูกค้ากดยกเลิกเองพร้อมกัน จะไม่ถูกเขียนทับเป็น "ร้านยกเลิก" และไม่ได้แจ้งเตือนผิด
func (r *repository) upcomingQuery(ctx context.Context, restaurantID uuid.UUID, now time.Time) *gorm.DB {
	return r.db.WithContext(ctx).Table("bookings b").
		Clauses(clause.Locking{Strength: "UPDATE", Table: clause.Table{Name: "b"}}).
		Select("b.*, u.display_name AS customer_name").
		Joins("JOIN users u ON u.id = b.user_id").
		Where("b.restaurant_id = ? AND b.status = ? AND b.start_at > ?", restaurantID, booking.StatusActive, now).
		Order("b.start_at, b.id")
}

// UpcomingBookings = ทุกการจองที่ยังไม่เริ่ม (ใช้ตอนลบร้าน)
func (r *repository) UpcomingBookings(ctx context.Context, restaurantID uuid.UUID, now time.Time) ([]AffectedBooking, error) {
	list := []AffectedBooking{}
	err := r.upcomingQuery(ctx, restaurantID, now).Scan(&list).Error
	return list, err
}

// AffectedBookings = การจองที่ยังไม่เริ่มและทับช่วงปิด [start,end)
func (r *repository) AffectedBookings(ctx context.Context, restaurantID uuid.UUID, start, end, now time.Time) ([]AffectedBooking, error) {
	list := []AffectedBooking{}
	err := r.upcomingQuery(ctx, restaurantID, now).Where("b.start_at < ? AND b.end_at > ?", end, start).Scan(&list).Error
	return list, err
}

// CancelByRestaurant ยกเลิกการจองโดยร้าน (ปิดชั่วคราว/ลบร้าน) พร้อมเหตุผลที่ลูกค้าจะเห็น
func (r *repository) CancelByRestaurant(ctx context.Context, ids []uuid.UUID, reason string, now time.Time) error {
	return r.db.WithContext(ctx).Model(&booking.Booking{}).Where("id IN ?", ids).
		Updates(map[string]any{"status": booking.StatusCancelled, "cancelled_at": now, "updated_at": now,
			"cancelled_by": booking.CancelledByRestaurant, "cancel_reason": reason}).Error
}

// Notify เขียนแจ้งเตือนด้วย db ของ repository นี้ — ใน Transaction คือ tx เดียวกับการยกเลิก
func (r *repository) Notify(ctx context.Context, d notification.Draft) error {
	return notification.Insert(ctx, r.db, d)
}

func (r *repository) CreateClosure(ctx context.Context, c *booking.Closure) error {
	return r.db.WithContext(ctx).Create(c).Error
}

// ClosuresBetween = ช่วงปิดของหลายร้านที่ทับ [from,to) — query เดียวต่อหน้า กัน N+1 (แบบเดียวกับ BookingsBetween)
func (r *repository) ClosuresBetween(ctx context.Context, restaurantIDs []uuid.UUID, from, to time.Time) ([]booking.Closure, error) {
	var list []booking.Closure
	if len(restaurantIDs) == 0 {
		return list, nil
	}
	err := r.db.WithContext(ctx).
		Where("restaurant_id IN ? AND start_at < ? AND end_at > ?", restaurantIDs, to, from).
		Order("start_at").
		Find(&list).Error
	return list, err
}

// UpcomingClosures = ช่วงปิดที่ยังไม่จบ เรียงตามเวลาเริ่ม
func (r *repository) UpcomingClosures(ctx context.Context, restaurantID uuid.UUID, now time.Time) ([]booking.Closure, error) {
	list := []booking.Closure{}
	err := r.db.WithContext(ctx).Where("restaurant_id = ? AND end_at > ?", restaurantID, now).Order("start_at").Find(&list).Error
	return list, err
}

// DeleteClosure ใส่ restaurant_id ใน WHERE ด้วย — ลบช่วงปิดของร้านอื่นผ่าน URL ร้านตัวเองไม่ได้
func (r *repository) DeleteClosure(ctx context.Context, restaurantID, closureID uuid.UUID) (bool, error) {
	res := r.db.WithContext(ctx).Where("id = ? AND restaurant_id = ?", closureID, restaurantID).Delete(&booking.Closure{})
	return res.RowsAffected > 0, res.Error
}

// BookingsBetween = booking active ของหลายร้านที่ทับช่วง [from,to) — query เดียวต่อหน้า กัน N+1
func (r *repository) BookingsBetween(ctx context.Context, restaurantIDs []uuid.UUID, from, to time.Time) ([]booking.Booking, error) {
	var bookings []booking.Booking
	if len(restaurantIDs) == 0 {
		return bookings, nil
	}
	err := r.db.WithContext(ctx).
		Where("restaurant_id IN ? AND status = ? AND start_at < ? AND end_at > ?", restaurantIDs, booking.StatusActive, to, from).
		Find(&bookings).Error
	return bookings, err
}

func (r *repository) CountImages(ctx context.Context, restaurantID uuid.UUID) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&Image{}).Where("restaurant_id = ?", restaurantID).Count(&n).Error
	return n, err
}

func (r *repository) AddImage(ctx context.Context, img *Image) error {
	// รูปใหม่ต่อท้าย: sort_order = ค่ามากสุด + 1
	return r.db.WithContext(ctx).Raw(`
		INSERT INTO restaurant_images (restaurant_id, url, sort_order)
		VALUES (?, ?, (SELECT COALESCE(MAX(sort_order), -1) + 1 FROM restaurant_images WHERE restaurant_id = ?))
		RETURNING *`, img.RestaurantID, img.URL, img.RestaurantID).Scan(img).Error
}

func (r *repository) DeleteImage(ctx context.Context, restaurantID, imageID uuid.UUID) error {
	res := r.db.WithContext(ctx).Where("id = ? AND restaurant_id = ?", imageID, restaurantID).Delete(&Image{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrImageNotFound
	}
	return nil
}

// List คืนร้านตามตัวกรอง + จำนวนทั้งหมด (สำหรับ pagination)
// sort=rating ใช้ Bayesian average: score = (C·m + rating_sum) / (C + rating_count), C = 5
// m = ค่าเฉลี่ยของทุกรีวิว — score ไม่ใช่คอลัมน์ จึงคำนวณใน query; ร้านไม่มีรีวิวอยู่ท้ายสุดเสมอ
func (r *repository) List(ctx context.Context, q ListQuery) ([]Restaurant, int64, error) {
	base := r.db.WithContext(ctx).Model(&Restaurant{})
	if q.Q != "" {
		like := "%" + q.Q + "%"
		base = base.Where("(restaurants.name ILIKE ? OR restaurants.description ILIKE ? OR restaurants.cuisine ILIKE ?)", like, like, like)
	}
	if q.Cuisine != "" {
		base = base.Where("restaurants.cuisine = ?", q.Cuisine)
	}

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query := base.Session(&gorm.Session{})
	switch q.Sort {
	case "rating":
		query = query.
			Select("restaurants.*, (5 * g.m + restaurants.rating_sum) / (5 + restaurants.rating_count) AS score").
			Joins(`CROSS JOIN (
				SELECT COALESCE(SUM(rating_sum)::float / NULLIF(SUM(rating_count), 0), 0) AS m
				FROM restaurants WHERE deleted_at IS NULL) g`).
			Order("restaurants.rating_count = 0").
			Order("score DESC").
			Order("restaurants.rating_count DESC").
			Order("restaurants.id")
	case "reviews":
		query = query.Order("restaurants.rating_count DESC").Order("restaurants.id")
	default:
		query = query.Order("restaurants.created_at DESC").Order("restaurants.id")
	}

	var list []Restaurant
	err := query.
		Preload("Images", func(db *gorm.DB) *gorm.DB { return db.Order("sort_order, id") }).
		Limit(q.Limit).Offset(q.Offset).
		Find(&list).Error
	return list, total, err
}
