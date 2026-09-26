package restaurant

import (
	"context"
	"fmt"
	"time"

	"jongyoung/internal/booking"

	"github.com/google/uuid"
)

type Repository interface {
	Transaction(ctx context.Context, fn func(tx Repository) error) error
	Create(ctx context.Context, rest *Restaurant) error
	FindByID(ctx context.Context, id uuid.UUID) (Restaurant, error)
	LockByID(ctx context.Context, id uuid.UUID) (Restaurant, error)
	Update(ctx context.Context, rest *Restaurant) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
	FutureBookings(ctx context.Context, restaurantID uuid.UUID, now time.Time) ([]booking.Booking, error)
	CancelFutureBookings(ctx context.Context, restaurantID uuid.UUID, now time.Time) (int64, error)
	BookingsBetween(ctx context.Context, restaurantIDs []uuid.UUID, from, to time.Time) ([]booking.Booking, error)
	CountImages(ctx context.Context, restaurantID uuid.UUID) (int64, error)
	AddImage(ctx context.Context, img *Image) error
	DeleteImage(ctx context.Context, restaurantID, imageID uuid.UUID) error
	List(ctx context.Context, q ListQuery) ([]Restaurant, int64, error)
}

// nextAvailableDays = ค้นวันถัดไปที่ว่างได้ไกลสุดกี่วัน
const nextAvailableDays = 14

type service struct {
	repository Repository
	now        func() time.Time // ฉีดเวลาได้ในเทสต์
}

func NewService(repo Repository, now func() time.Time) *service {
	return &service{repository: repo, now: now}
}

func (s *service) Create(ctx context.Context, ownerID uuid.UUID, in Input, imageURLs []string) (Restaurant, error) {
	if len(imageURLs) == 0 {
		return Restaurant{}, ErrImageRequired
	}
	rest := Restaurant{OwnerID: ownerID}
	apply(&rest, in)
	for i, url := range imageURLs {
		rest.Images = append(rest.Images, Image{URL: url, SortOrder: i})
	}
	if err := s.repository.Create(ctx, &rest); err != nil {
		return Restaurant{}, fmt.Errorf("create restaurant: %w", err)
	}
	return rest, nil
}

func (s *service) Get(ctx context.Context, id uuid.UUID) (Restaurant, error) {
	return s.repository.FindByID(ctx, id)
}

// Update แก้ข้อมูลร้าน — ถ้าลดที่นั่งหรือเปลี่ยนเวลาเปิด ต้องไม่ทำให้ booking ที่มีอยู่ผิดกติกา
// ล็อกแถวร้านก่อนตรวจ ไม่งั้นมีคนจองแทรกระหว่างตรวจกับบันทึก
func (s *service) Update(ctx context.Context, userID, id uuid.UUID, in Input) (Restaurant, error) {
	err := s.repository.Transaction(ctx, func(tx Repository) error {
		rest, err := tx.LockByID(ctx, id)
		if err != nil {
			return err
		}
		if rest.OwnerID != userID {
			return ErrNotOwner
		}
		if in.Seats < rest.Seats || in.Hours() != rest.Hours() {
			future, err := tx.FutureBookings(ctx, id, s.now())
			if err != nil {
				return err
			}
			if err := booking.CheckSeatReduction(future, in.Seats); err != nil {
				return err
			}
			if err := booking.CheckHoursChange(future, in.Hours()); err != nil {
				return err
			}
		}
		apply(&rest, in)
		return tx.Update(ctx, &rest)
	})
	if err != nil {
		return Restaurant{}, err
	}
	return s.repository.FindByID(ctx, id)
}

// Delete = soft delete + ยกเลิก booking ที่ยังไม่เริ่มในทรานแซกชันเดียวกัน
// เลือกแบบนี้แทนการห้ามลบ: เจ้าของปิดร้านได้จริง และลูกค้าเห็นว่าการจองถูกยกเลิก (ไม่ใช่มาแล้วเจอร้านปิด)
func (s *service) Delete(ctx context.Context, userID, id uuid.UUID) error {
	return s.repository.Transaction(ctx, func(tx Repository) error {
		rest, err := tx.LockByID(ctx, id)
		if err != nil {
			return err
		}
		if rest.OwnerID != userID {
			return ErrNotOwner
		}
		if _, err := tx.CancelFutureBookings(ctx, id, s.now()); err != nil {
			return err
		}
		return tx.SoftDelete(ctx, id)
	})
}

func (s *service) AddImage(ctx context.Context, userID, id uuid.UUID, url string) (Image, error) {
	rest, err := s.repository.FindByID(ctx, id)
	if err != nil {
		return Image{}, err
	}
	if rest.OwnerID != userID {
		return Image{}, ErrNotOwner
	}
	img := Image{RestaurantID: id, URL: url}
	if err := s.repository.AddImage(ctx, &img); err != nil {
		return Image{}, err
	}
	return img, nil
}

// DeleteImage ห้ามลบรูปสุดท้าย (ร้านต้องมีรูปอย่างน้อย 1 รูป) — ล็อกร้านกันลบสองรูปสุดท้ายพร้อมกัน
func (s *service) DeleteImage(ctx context.Context, userID, id, imageID uuid.UUID) error {
	return s.repository.Transaction(ctx, func(tx Repository) error {
		rest, err := tx.LockByID(ctx, id)
		if err != nil {
			return err
		}
		if rest.OwnerID != userID {
			return ErrNotOwner
		}
		count, err := tx.CountImages(ctx, id)
		if err != nil {
			return err
		}
		if count <= 1 {
			return ErrImageRequired
		}
		return tx.DeleteImage(ctx, id, imageID)
	})
}

// List คืนร้าน + ถ้ามีวัน/เวลา/จำนวนคน จะคำนวณปุ่มเวลาให้ทุกร้านในหน้า ด้วย booking query เดียว
func (s *service) List(ctx context.Context, q ListQuery) ([]ListItem, int64, error) {
	list, total, err := s.repository.List(ctx, q)
	if err != nil {
		return nil, 0, err
	}
	items := make([]ListItem, len(list))
	for i, r := range list {
		items[i] = ListItem{Restaurant: r}
	}
	if q.Date == nil || len(list) == 0 {
		return items, total, nil
	}

	// ทุกร้านในหน้ามีรอบของวันนั้นอยู่ในช่วง [เที่ยงคืนวันนั้น, เที่ยงคืนอีก 2 วัน) (ร้านข้ามคืนจบไม่เกินวันถัดไป)
	from := *q.Date
	to := from.AddDate(0, 0, 2)
	ids := make([]uuid.UUID, len(list))
	for i, r := range list {
		ids[i] = r.ID
	}
	bookings, err := s.repository.BookingsBetween(ctx, ids, from, to)
	if err != nil {
		return nil, 0, err
	}
	byRestaurant := map[uuid.UUID][]booking.Booking{}
	for _, b := range bookings {
		byRestaurant[b.RestaurantID] = append(byRestaurant[b.RestaurantID], b)
	}
	now := s.now()
	for i, r := range list {
		items[i].Slots = booking.SlotsAround(r.Hours(), r.Seats, byRestaurant[r.ID], *q.Date, q.Minute, now)
	}
	return items, total, nil
}

// Availability คืนรอบของวันทำการ date และที่ว่างทุกช่วง 30 นาที
func (s *service) Availability(ctx context.Context, id uuid.UUID, date time.Time) (Restaurant, []booking.Slot, error) {
	rest, err := s.repository.FindByID(ctx, id)
	if err != nil {
		return Restaurant{}, nil, err
	}
	opensAt, closesAt := rest.Hours().Window(date)
	bookings, err := s.repository.BookingsBetween(ctx, []uuid.UUID{id}, opensAt, closesAt)
	if err != nil {
		return Restaurant{}, nil, err
	}
	return rest, booking.Slots(rest.Hours(), rest.Seats, bookings, date, s.now()), nil
}

// NextAvailable หาวันทำการแรก (ตั้งแต่ date ถึง date+14) ที่ยังมีช่วงว่างพอสำหรับ party คน
//   - ร้านเปิดตอนเวลาที่ค้นแต่เต็ม → หาช่วงรอบเวลาเดิมของวันถัด ๆ ไป (การ์ดแสดงวันที่ค้นอยู่แล้ว จึงเริ่มวันถัดไป)
//   - ร้านไม่เปิดตอนเวลาที่ค้นเลย (ค้น 10:30 ที่ร้าน 17:00–23:00) → ช่วงว่างแรก ๆ ของรอบนั้นแทน เริ่มจากวันที่ค้น
//
// ดึง booking ทั้งช่วงด้วย query เดียว แล้วไล่ทีละวันใน Go; ไม่เจอ → nil
func (s *service) NextAvailable(ctx context.Context, id uuid.UUID, date time.Time, minute, party int) (*NextAvailable, error) {
	rest, err := s.repository.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	h := rest.Hours()
	from, _ := h.Window(date)
	_, to := h.Window(date.AddDate(0, 0, nextAvailableDays))
	bookings, err := s.repository.BookingsBetween(ctx, []uuid.UUID{id}, from, to)
	if err != nil {
		return nil, err
	}
	now := s.now()
	for day := 0; day <= nextAvailableDays; day++ {
		d := date.AddDate(0, 0, day)
		around := booking.SlotsAround(h, rest.Seats, bookings, d, minute, now)
		var ok []booking.CardSlot
		if day > 0 {
			ok = freeSlots(around, party)
		}
		if len(ok) == 0 && !h.OpenAtMinute(minute) {
			ok = firstFree(booking.Slots(h, rest.Seats, bookings, d, now), party, len(around))
		}
		if len(ok) > 0 {
			return &NextAvailable{Date: d, Slots: ok}, nil
		}
	}
	return nil, nil
}

// freeSlots เลือกเฉพาะช่วงที่เปิดและว่างพอ
func freeSlots(slots []booking.CardSlot, party int) []booking.CardSlot {
	var ok []booking.CardSlot
	for _, s := range slots {
		if !s.Closed && s.Available >= party {
			ok = append(ok, s)
		}
	}
	return ok
}

// firstFree คืนช่วงที่ว่างพอ n ช่วงแรกของรอบ
func firstFree(slots []booking.Slot, party, n int) []booking.CardSlot {
	var ok []booking.CardSlot
	for _, s := range slots {
		if len(ok) == n {
			break
		}
		if s.Available >= party {
			ok = append(ok, booking.CardSlot{StartAt: s.StartAt, EndAt: s.EndAt, Available: s.Available})
		}
	}
	return ok
}

func apply(rest *Restaurant, in Input) {
	rest.Name = in.Name
	rest.Description = in.Description
	rest.Cuisine = in.Cuisine
	rest.Address = in.Address
	rest.Seats = in.Seats
	rest.OpenMinute = in.OpenMinute
	rest.CloseMinute = in.CloseMinute
	rest.ClosedWeekdays = in.ClosedWeekdays
	rest.CancelBeforeMinutes = in.CancelBeforeMinutes
}
