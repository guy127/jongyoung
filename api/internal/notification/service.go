package notification

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Repository interface {
	ListByUser(ctx context.Context, userID uuid.UUID, limit int) ([]Notification, error)
	CountUnread(ctx context.Context, userID uuid.UUID) (int64, error)
	MarkRead(ctx context.Context, userID, id uuid.UUID, now time.Time) (bool, error)
	MarkAllRead(ctx context.Context, userID uuid.UUID, now time.Time) error
}

// listLimit = แสดง 20 รายการล่าสุดในแผงกระดิ่ง
const listLimit = 20

type service struct {
	repository Repository
	now        func() time.Time
}

func NewService(repo Repository, now func() time.Time) *service {
	return &service{repository: repo, now: now}
}

// List คืนรายการล่าสุด + จำนวนที่ยังไม่อ่านทั้งหมด (อาจมากกว่าที่แสดง)
func (s *service) List(ctx context.Context, userID uuid.UUID) ([]Notification, int64, error) {
	list, err := s.repository.ListByUser(ctx, userID, listLimit)
	if err != nil {
		return nil, 0, err
	}
	unread, err := s.repository.CountUnread(ctx, userID)
	return list, unread, err
}

func (s *service) MarkRead(ctx context.Context, userID, id uuid.UUID) error {
	ok, err := s.repository.MarkRead(ctx, userID, id, s.now())
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	return nil
}

func (s *service) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	return s.repository.MarkAllRead(ctx, userID, s.now())
}
