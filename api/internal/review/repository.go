package review

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

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

// RestaurantOwner คืน owner_id ของร้านที่ยังไม่ถูกลบ (ใช้ตรวจว่ารีวิวร้านตัวเองไหม)
func (r *repository) RestaurantOwner(ctx context.Context, restaurantID uuid.UUID) (uuid.UUID, error) {
	var owner uuid.UUID
	// อ่านตาราง restaurants ตรง ๆ โดยตั้งใจ — ถูกเรียกในทรานแซกชันเดียวกับการเขียนรีวิว
	// ถ้าเรียกผ่าน package restaurant ต้องส่ง *gorm.DB ของทรานแซกชันข้ามแพ็กเกจ (ดู ARCHITECTURE.md)
	err := r.db.WithContext(ctx).Table("restaurants").Select("owner_id").
		Where("id = ? AND deleted_at IS NULL", restaurantID).Row().Scan(&owner)
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, ErrRestaurantNotFound
	}
	return owner, err
}

// FindOwn หารีวิวของผู้ใช้ในร้านนี้ พร้อมล็อกแถว (แก้/ลบพร้อมกันจะได้ไม่ปรับคะแนนซ้ำ)
func (r *repository) FindOwn(ctx context.Context, restaurantID, userID uuid.UUID) (Review, error) {
	var rv Review
	err := r.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("restaurant_id = ? AND user_id = ?", restaurantID, userID).First(&rv).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Review{}, ErrReviewNotFound
	}
	return rv, err
}

// Create — ถ้าชน unique (restaurant_id, user_id) จากการกดส่งซ้ำพร้อมกัน แปลงเป็น ErrReviewExists
func (r *repository) Create(ctx context.Context, rv *Review) error {
	err := r.db.WithContext(ctx).Create(rv).Error
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
		return ErrReviewExists
	}
	return err
}

func (r *repository) Update(ctx context.Context, rv *Review) error {
	return r.db.WithContext(ctx).Model(rv).Select("score", "body", "updated_at").Updates(rv).Error
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&Review{}, "id = ?", id).Error
}

// AdjustRating ปรับผลรวมคะแนนแบบ atomic ในคำสั่งเดียว
// ห้ามอ่านค่ามาบวกใน Go แล้วเขียนกลับ — สองคนรีวิวพร้อมกันค่าจะหาย (lost update)
func (r *repository) AdjustRating(ctx context.Context, restaurantID uuid.UUID, sumDelta, countDelta int) error {
	// อัปเดตตาราง restaurants ตรง ๆ โดยตั้งใจ — ต้องอยู่ในทรานแซกชันเดียวกับการเพิ่ม/แก้/ลบรีวิว (CLAUDE.md 5.6)
	// ถ้าเรียกผ่าน package restaurant ต้องส่ง *gorm.DB ของทรานแซกชันข้ามแพ็กเกจ
	return r.db.WithContext(ctx).Exec(`
		UPDATE restaurants
		SET rating_sum = rating_sum + ?, rating_count = rating_count + ?
		WHERE id = ?`, sumDelta, countDelta, restaurantID).Error
}

func (r *repository) List(ctx context.Context, restaurantID uuid.UUID, limit, offset int) ([]View, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&Review{}).Where("restaurant_id = ?", restaurantID).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	list := []View{}
	err := r.db.WithContext(ctx).Table("reviews rv").
		Select("rv.*, u.display_name AS author_name").
		Joins("JOIN users u ON u.id = rv.user_id").
		Where("rv.restaurant_id = ?", restaurantID).
		Order("rv.created_at DESC, rv.id").
		Limit(limit).Offset(offset).
		Scan(&list).Error
	return list, total, err
}
