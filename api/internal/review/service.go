package review

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

type Repository interface {
	Transaction(ctx context.Context, fn func(tx Repository) error) error
	RestaurantOwner(ctx context.Context, restaurantID uuid.UUID) (uuid.UUID, error)
	FindOwn(ctx context.Context, restaurantID, userID uuid.UUID) (Review, error)
	Create(ctx context.Context, rv *Review) error
	Update(ctx context.Context, rv *Review) error
	Delete(ctx context.Context, id uuid.UUID) error
	AdjustRating(ctx context.Context, restaurantID uuid.UUID, sumDelta, countDelta int) error
	List(ctx context.Context, restaurantID uuid.UUID, limit, offset int) ([]View, int64, error)
}

type service struct {
	repository Repository
}

func NewService(repo Repository) *service {
	return &service{repository: repo}
}

// Create: ห้ามรีวิวร้านตัวเอง, 1 คน 1 รีวิวต่อร้าน — เพิ่มรีวิวกับปรับคะแนนรวมอยู่ในทรานแซกชันเดียวกัน
func (s *service) Create(ctx context.Context, userID, restaurantID uuid.UUID, score int, body string) (Review, error) {
	var created Review
	err := s.repository.Transaction(ctx, func(tx Repository) error {
		owner, err := tx.RestaurantOwner(ctx, restaurantID)
		if err != nil {
			return err
		}
		if owner == userID {
			return ErrOwnRestaurant
		}
		if _, err := tx.FindOwn(ctx, restaurantID, userID); err == nil {
			return ErrReviewExists
		} else if !errors.Is(err, ErrReviewNotFound) {
			return err
		}
		created = Review{RestaurantID: restaurantID, UserID: userID, Score: score, Body: body}
		if err := tx.Create(ctx, &created); err != nil {
			return err
		}
		return tx.AdjustRating(ctx, restaurantID, score, 1)
	})
	return created, err
}

// Update แก้รีวิวของตัวเอง — ปรับผลรวมด้วยผลต่าง (ใหม่ − เดิม) จำนวนรีวิวไม่เปลี่ยน
func (s *service) Update(ctx context.Context, userID, restaurantID uuid.UUID, score int, body string) (Review, error) {
	var updated Review
	err := s.repository.Transaction(ctx, func(tx Repository) error {
		rv, err := tx.FindOwn(ctx, restaurantID, userID)
		if err != nil {
			return err
		}
		oldScore := rv.Score
		rv.Score, rv.Body = score, body
		if err := tx.Update(ctx, &rv); err != nil {
			return err
		}
		updated = rv
		return tx.AdjustRating(ctx, restaurantID, score-oldScore, 0)
	})
	return updated, err
}

// Delete ลบรีวิวของตัวเอง — หักคะแนนเดิมและจำนวนรีวิวออก
func (s *service) Delete(ctx context.Context, userID, restaurantID uuid.UUID) error {
	return s.repository.Transaction(ctx, func(tx Repository) error {
		rv, err := tx.FindOwn(ctx, restaurantID, userID)
		if err != nil {
			return err
		}
		if err := tx.Delete(ctx, rv.ID); err != nil {
			return err
		}
		return tx.AdjustRating(ctx, restaurantID, -rv.Score, -1)
	})
}

// Mine = รีวิวของฉันในร้านนี้ (หน้าเว็บใช้ตัดสินว่าจะเรียก POST หรือ PUT)
func (s *service) Mine(ctx context.Context, userID, restaurantID uuid.UUID) (Review, error) {
	return s.repository.FindOwn(ctx, restaurantID, userID)
}

func (s *service) List(ctx context.Context, restaurantID uuid.UUID, limit, offset int) ([]View, int64, error) {
	return s.repository.List(ctx, restaurantID, limit, offset)
}
