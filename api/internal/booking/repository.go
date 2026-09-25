package booking

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// RestaurantInfo คือข้อมูลร้านที่การจองต้องใช้ — อ่านจากตาราง restaurants ตรง ๆ
// (ไม่ import package restaurant เพราะ restaurant import booking อยู่แล้ว จะเป็น import cycle)
type RestaurantInfo struct {
	ID                  uuid.UUID
	OwnerID             uuid.UUID
	Name                string
	Address             string
	Seats               int
	OpenMinute          int
	CloseMinute         int
	CancelBeforeMinutes int
}

func (r RestaurantInfo) Hours() Hours {
	return Hours{OpenMinute: r.OpenMinute, CloseMinute: r.CloseMinute}
}

// View = booking + ข้อมูลร้าน + ชื่อลูกค้า สำหรับแสดงผล
type View struct {
	Booking
	RestaurantName      string
	RestaurantAddress   string
	RestaurantOwnerID   uuid.UUID
	OpenMinute          int
	CloseMinute         int
	CancelBeforeMinutes int
	CustomerName        string
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *repository {
	return &repository{db: db}
}

func (r *repository) Transaction(ctx context.Context, fn func(tx Repository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&repository{db: tx})
	})
}

const restaurantColumns = "id, owner_id, name, address, seats, open_minute, close_minute, cancel_before_minutes"

// LockRestaurant = SELECT ... FOR UPDATE บนแถวร้าน — คำขอจองร้านเดียวกันพร้อมกันจะต่อคิวที่บรรทัดนี้
func (r *repository) LockRestaurant(ctx context.Context, id uuid.UUID) (RestaurantInfo, error) {
	var info RestaurantInfo
	res := r.db.WithContext(ctx).Table("restaurants").Select(restaurantColumns).
		Where("id = ? AND deleted_at IS NULL", id).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Limit(1).Scan(&info)
	if res.Error != nil {
		return RestaurantInfo{}, res.Error
	}
	if res.RowsAffected == 0 {
		return RestaurantInfo{}, ErrRestaurantNotFound
	}
	return info, nil
}

// FindRestaurant อ่านแบบไม่ล็อก; includeDeleted ใช้ตอนยกเลิกการจองของร้านที่ถูกลบไปแล้ว
func (r *repository) FindRestaurant(ctx context.Context, id uuid.UUID, includeDeleted bool) (RestaurantInfo, error) {
	var info RestaurantInfo
	q := r.db.WithContext(ctx).Table("restaurants").Select(restaurantColumns).Where("id = ?", id)
	if !includeDeleted {
		q = q.Where("deleted_at IS NULL")
	}
	res := q.Limit(1).Scan(&info)
	if res.Error != nil {
		return RestaurantInfo{}, res.Error
	}
	if res.RowsAffected == 0 {
		return RestaurantInfo{}, ErrRestaurantNotFound
	}
	return info, nil
}

// FindOverlappingOwn หา booking active ของผู้ใช้คนนี้ที่ร้านนี้ซึ่งทับ [start,end) — ไม่เจอคืน nil (กฎข้อ 8)
func (r *repository) FindOverlappingOwn(ctx context.Context, restaurantID, userID uuid.UUID, start, end time.Time, excludeID uuid.UUID) (*Booking, error) {
	var b Booking
	err := r.db.WithContext(ctx).
		Where("restaurant_id = ? AND user_id = ? AND status = ?", restaurantID, userID, StatusActive).
		Where("start_at < ? AND end_at > ?", end, start).
		Where("id <> ?", excludeID).
		First(&b).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}

// Overlapping = booking active ทุกคนที่ทับ [start,end) ไม่รวม excludeID (กฎข้อ 7 — ตอนแก้ไขห้ามนับที่นั่งเดิมของตัวเอง)
func (r *repository) Overlapping(ctx context.Context, restaurantID uuid.UUID, start, end time.Time, excludeID uuid.UUID) ([]Booking, error) {
	var list []Booking
	err := r.db.WithContext(ctx).
		Where("restaurant_id = ? AND status = ?", restaurantID, StatusActive).
		Where("start_at < ? AND end_at > ?", end, start).
		Where("id <> ?", excludeID).
		Find(&list).Error
	return list, err
}

func (r *repository) Create(ctx context.Context, b *Booking) error {
	return r.db.WithContext(ctx).Create(b).Error
}

func (r *repository) FindByID(ctx context.Context, id uuid.UUID) (Booking, error) {
	var b Booking
	err := r.db.WithContext(ctx).First(&b, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Booking{}, ErrBookingNotFound
	}
	return b, err
}

func (r *repository) LockByID(ctx context.Context, id uuid.UUID) (Booking, error) {
	var b Booking
	err := r.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&b, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Booking{}, ErrBookingNotFound
	}
	return b, err
}

func (r *repository) UpdateTime(ctx context.Context, b *Booking) error {
	return r.db.WithContext(ctx).Model(b).Select("party_size", "start_at", "end_at", "updated_at").Updates(b).Error
}

// Cancel เปลี่ยนสถานะเป็น cancelled — ไม่ลบแถวจริง เพื่อเก็บประวัติ
func (r *repository) Cancel(ctx context.Context, id uuid.UUID, now time.Time) error {
	return r.db.WithContext(ctx).Model(&Booking{}).Where("id = ?", id).
		Updates(map[string]any{"status": StatusCancelled, "cancelled_at": now, "updated_at": now}).Error
}

// viewQuery = booking + ร้าน + ชื่อลูกค้า (JOIN ครั้งเดียว ไม่ N+1)
func (r *repository) viewQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Table("bookings b").
		Select(`b.*, r.name AS restaurant_name, r.address AS restaurant_address, r.owner_id AS restaurant_owner_id,
			r.open_minute, r.close_minute, r.cancel_before_minutes, u.display_name AS customer_name`).
		Joins("JOIN restaurants r ON r.id = b.restaurant_id").
		Joins("JOIN users u ON u.id = b.user_id")
}

func (r *repository) FindView(ctx context.Context, id uuid.UUID) (View, error) {
	var v View
	res := r.viewQuery(ctx).Where("b.id = ?", id).Limit(1).Scan(&v)
	if res.Error != nil {
		return View{}, res.Error
	}
	if res.RowsAffected == 0 {
		return View{}, ErrBookingNotFound
	}
	return v, nil
}

// ListByUser: upcoming = ยังไม่จบ (เร็วสุดก่อน), past = จบแล้ว, cancelled = ยกเลิกแล้ว (ล่าสุดก่อน)
func (r *repository) ListByUser(ctx context.Context, userID uuid.UUID, status string, now time.Time) ([]View, error) {
	q := r.viewQuery(ctx).Where("b.user_id = ?", userID)
	switch status {
	case "past":
		q = q.Where("b.status = ? AND b.end_at <= ?", StatusActive, now).Order("b.start_at DESC")
	case "cancelled":
		q = q.Where("b.status = ?", StatusCancelled).Order("b.start_at DESC")
	default:
		q = q.Where("b.status = ? AND b.end_at > ?", StatusActive, now).Order("b.start_at")
	}
	list := []View{}
	err := q.Limit(100).Scan(&list).Error
	return list, err
}

// ListForRestaurant = การจองทุกสถานะที่เริ่มในรอบ [from,to) ของร้าน (บอร์ดรายวันของเจ้าของ)
func (r *repository) ListForRestaurant(ctx context.Context, restaurantID uuid.UUID, from, to time.Time) ([]View, error) {
	list := []View{}
	err := r.viewQuery(ctx).
		Where("b.restaurant_id = ? AND b.start_at >= ? AND b.start_at < ?", restaurantID, from, to).
		Order("b.start_at, b.created_at").
		Scan(&list).Error
	return list, err
}
