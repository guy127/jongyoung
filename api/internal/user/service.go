package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

type Repository interface {
	FindByUID(ctx context.Context, uid string) (User, error)
	FindByID(ctx context.Context, id uuid.UUID) (User, error)
	Upsert(ctx context.Context, uid, email, displayName string) (User, error)
	OwnedRestaurants(ctx context.Context, userID uuid.UUID) ([]OwnedRestaurant, error)
}

type service struct {
	repository Repository
}

func NewService(repo Repository) *service {
	return &service{repository: repo}
}

// EnsureExists คืนผู้ใช้ของ sub นี้ สร้างใหม่ถ้ายังไม่มี และอัปเดตถ้า email/ชื่อใน Keycloak เปลี่ยน
// อ่านก่อนเขียน เพื่อไม่ต้องเขียน DB ทุก request ในกรณีปกติ (ข้อมูลไม่เปลี่ยน)
func (s *service) EnsureExists(ctx context.Context, uid, email, displayName string) (User, error) {
	existing, err := s.repository.FindByUID(ctx, uid)
	if err == nil && existing.Email == email && existing.DisplayName == displayName {
		return existing, nil
	}
	if err != nil && !errors.Is(err, ErrUserNotFound) {
		return User{}, fmt.Errorf("find user: %w", err)
	}
	u, err := s.repository.Upsert(ctx, uid, email, displayName)
	if err != nil {
		return User{}, fmt.Errorf("upsert user: %w", err)
	}
	return u, nil
}

// Me คือโปรไฟล์ของผู้ใช้ปัจจุบัน + ร้านที่เป็นเจ้าของ
type Me struct {
	User        User
	Restaurants []OwnedRestaurant
}

func (s *service) Me(ctx context.Context, id uuid.UUID) (Me, error) {
	u, err := s.repository.FindByID(ctx, id)
	if err != nil {
		return Me{}, err
	}
	restaurants, err := s.repository.OwnedRestaurants(ctx, id)
	if err != nil {
		return Me{}, fmt.Errorf("owned restaurants: %w", err)
	}
	return Me{User: u, Restaurants: restaurants}, nil
}
