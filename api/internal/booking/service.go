package booking

import (
	"context"
	"time"

	"jongyoung/internal/notification"

	"github.com/google/uuid"
)

type Repository interface {
	Transaction(ctx context.Context, fn func(tx Repository) error) error
	LockRestaurant(ctx context.Context, id uuid.UUID) (RestaurantInfo, error)
	FindRestaurant(ctx context.Context, id uuid.UUID, includeDeleted bool) (RestaurantInfo, error)
	FindOverlappingOwn(ctx context.Context, restaurantID, userID uuid.UUID, start, end time.Time, excludeID uuid.UUID) (*Booking, error)
	Overlapping(ctx context.Context, restaurantID uuid.UUID, start, end time.Time, excludeID uuid.UUID) ([]Booking, error)
	Create(ctx context.Context, b *Booking) error
	FindByID(ctx context.Context, id uuid.UUID) (Booking, error)
	LockByID(ctx context.Context, id uuid.UUID) (Booking, error)
	UpdateTime(ctx context.Context, b *Booking) error
	Cancel(ctx context.Context, id uuid.UUID, now time.Time) error
	FindView(ctx context.Context, id uuid.UUID) (View, error)
	ListByUser(ctx context.Context, userID uuid.UUID, status string, now time.Time) ([]View, error)
	ListForRestaurant(ctx context.Context, restaurantID uuid.UUID, from, to time.Time) ([]View, error)
	ClosuresBetween(ctx context.Context, restaurantID uuid.UUID, from, to time.Time) ([]Closure, error)
	Notify(ctx context.Context, d notification.Draft) error
}

// Slot ที่ผู้ใช้เลือก: วันทำการ + เวลาบนนาฬิกา (server แปลงเป็นเวลาจริงเอง รองรับร้านข้ามคืน)
type Choice struct {
	Date        time.Time // เที่ยงคืนของวันทำการ (เวลาไทย)
	StartMinute int
	EndMinute   int
	PartySize   int
}

// toRequest แปลงตัวเลือกเป็นเวลาจริงผ่าน Hours.Span (เวลาที่น้อยกว่าเวลาเปิด = หลังเที่ยงคืนของรอบนั้น)
func (c Choice) toRequest(h Hours) Request {
	start, end := h.Span(c.Date, c.StartMinute, c.EndMinute)
	return Request{StartAt: start, EndAt: end, PartySize: c.PartySize}
}

type service struct {
	repository Repository
	now        func() time.Time
}

func NewService(repo Repository, now func() time.Time) *service {
	return &service{repository: repo, now: now}
}

// Create จองโต๊ะ — ทุกการตรวจอยู่ในทรานแซกชันเดียวที่ล็อกแถวร้านแล้ว:
// 1) ล็อกร้าน  2) ตรวจกฎที่ไม่ใช้ DB  3) กฎ 8 จองซ้อนตัวเอง (ก่อนกฎ 7)  4) กฎ 7 ที่นั่ง  5) เขียน
func (s *service) Create(ctx context.Context, userID, restaurantID uuid.UUID, c Choice) (Booking, error) {
	var created Booking
	err := s.repository.Transaction(ctx, func(tx Repository) error {
		r, err := tx.LockRestaurant(ctx, restaurantID)
		if err != nil {
			return err
		}
		req := c.toRequest(r.Hours())
		if err := s.checkSlot(ctx, tx, r, userID, req, uuid.Nil); err != nil {
			return err
		}
		created = Booking{RestaurantID: r.ID, UserID: userID, PartySize: req.PartySize,
			StartAt: req.StartAt, EndAt: req.EndAt, Status: StatusActive}
		if err := tx.Create(ctx, &created); err != nil {
			return err
		}
		return notifyOwner(ctx, tx, r, userID, notification.KindBookingCreated, created)
	})
	return created, err
}

// Update แก้จำนวนคน/วัน/เวลา — ใช้กติกาเดียวกับจองใหม่ทุกข้อ + excludeID = booking นี้
// ลำดับล็อก: ร้านก่อน แล้วค่อย booking (เหมือนตอนลบร้าน) เพื่อไม่ให้เกิด deadlock
func (s *service) Update(ctx context.Context, userID, bookingID uuid.UUID, c Choice) (Booking, error) {
	current, err := s.repository.FindByID(ctx, bookingID)
	if err != nil {
		return Booking{}, err
	}
	var updated Booking
	err = s.repository.Transaction(ctx, func(tx Repository) error {
		r, err := tx.LockRestaurant(ctx, current.RestaurantID)
		if err != nil {
			return err
		}
		b, err := tx.LockByID(ctx, bookingID)
		if err != nil {
			return err
		}
		if err := s.checkChangeable(b, userID, r.CancelBeforeMinutes); err != nil {
			return err
		}
		req := c.toRequest(r.Hours())
		if err := s.checkSlot(ctx, tx, r, userID, req, b.ID); err != nil {
			return err
		}
		b.PartySize, b.StartAt, b.EndAt = req.PartySize, req.StartAt, req.EndAt
		updated = b
		if err := tx.UpdateTime(ctx, &b); err != nil {
			return err
		}
		return notifyOwner(ctx, tx, r, userID, notification.KindBookingUpdated, b)
	})
	return updated, err
}

// Cancel ยกเลิก (เปลี่ยนสถานะ ไม่ลบจริง) — ได้เมื่อ now <= start - cancel_before_minutes
func (s *service) Cancel(ctx context.Context, userID, bookingID uuid.UUID) error {
	return s.repository.Transaction(ctx, func(tx Repository) error {
		b, err := tx.LockByID(ctx, bookingID)
		if err != nil {
			return err
		}
		r, err := tx.FindRestaurant(ctx, b.RestaurantID, true)
		if err != nil {
			return err
		}
		if err := s.checkChangeable(b, userID, r.CancelBeforeMinutes); err != nil {
			return err
		}
		if err := tx.Cancel(ctx, b.ID, s.now()); err != nil {
			return err
		}
		return notifyOwner(ctx, tx, r, userID, notification.KindBookingCancelled, b)
	})
}

// Get: เห็นได้เฉพาะเจ้าของการจอง หรือเจ้าของร้าน
func (s *service) Get(ctx context.Context, userID, bookingID uuid.UUID) (View, error) {
	v, err := s.repository.FindView(ctx, bookingID)
	if err != nil {
		return View{}, err
	}
	if v.UserID != userID && v.RestaurantOwnerID != userID {
		return View{}, ErrForbidden
	}
	return v, nil
}

func (s *service) ListMine(ctx context.Context, userID uuid.UUID, status string) ([]View, error) {
	return s.repository.ListByUser(ctx, userID, status, s.now())
}

// Board คือบอร์ดรายวันทำการของเจ้าของร้าน: การจองทุกสถานะ + ที่ว่างต่อช่วง (seat bar)
type Board struct {
	Restaurant RestaurantInfo
	Closed     bool // วันปิดประจำสัปดาห์ — ไม่มีช่วงเวลา แต่ยังโชว์ booking เดิม (ถ้ามีก่อนตั้งวันปิด) ได้
	OpensAt    time.Time
	ClosesAt   time.Time
	Bookings   []View
	Slots      []Slot
}

func (s *service) Board(ctx context.Context, userID, restaurantID uuid.UUID, date time.Time) (Board, error) {
	r, err := s.repository.FindRestaurant(ctx, restaurantID, false)
	if err != nil {
		return Board{}, err
	}
	if r.OwnerID != userID {
		return Board{}, ErrNotRestaurantOwner
	}
	opensAt, closesAt := r.Hours().Window(date)
	views, err := s.repository.ListForRestaurant(ctx, restaurantID, opensAt, closesAt)
	if err != nil {
		return Board{}, err
	}
	var active []Booking
	for _, v := range views {
		if v.Status == StatusActive {
			active = append(active, v.Booking)
		}
	}
	closures, err := s.repository.ClosuresBetween(ctx, restaurantID, opensAt, closesAt)
	if err != nil {
		return Board{}, err
	}
	// now = เวลาศูนย์ → ไม่ตัดช่วงที่ผ่านไปแล้วออก (เจ้าของต้องเห็นทั้งรอบ)
	slots := Slots(r.Hours(), r.Seats, active, closures, date, time.Time{})
	return Board{Restaurant: r, Closed: r.Hours().ClosedOn(date), OpensAt: opensAt, ClosesAt: closesAt, Bookings: views, Slots: slots}, nil
}

// checkSlot ตรวจกฎทั้งหมดของช่วงที่ขอ (ต้องเรียกหลัง LockRestaurant เท่านั้น)
func (s *service) checkSlot(ctx context.Context, tx Repository, r RestaurantInfo, userID uuid.UUID, req Request, excludeID uuid.UUID) error {
	if err := ValidateRequest(req, r.Hours(), r.Seats, s.now()); err != nil {
		return err
	}
	// ช่วงปิดชั่วคราว — อ่านในล็อกเดียวกัน: เจ้าของร้านกดปิดพร้อมกับมีคนจอง ก็ต่อคิวที่ล็อกแถวร้านเดียวกัน
	closures, err := tx.ClosuresBetween(ctx, r.ID, req.StartAt, req.EndAt)
	if err != nil {
		return err
	}
	if c := closureAt(closures, req.StartAt, req.EndAt); c != nil {
		return &ClosedError{Closure: *c}
	}
	// กฎ 8 ก่อนกฎ 7: กดซ้ำแล้วคำขอแรกสำเร็จ → คำขอที่สองต้องได้ DUPLICATE_BOOKING ไม่ใช่ NOT_ENOUGH_SEATS
	dup, err := tx.FindOverlappingOwn(ctx, r.ID, userID, req.StartAt, req.EndAt, excludeID)
	if err != nil {
		return err
	}
	if dup != nil {
		return &DuplicateBookingError{BookingID: dup.ID}
	}
	existing, err := tx.Overlapping(ctx, r.ID, req.StartAt, req.EndAt, excludeID)
	if err != nil {
		return err
	}
	return checkSeats(existing, r.Seats, req.PartySize, req.StartAt, req.EndAt)
}

// checkChangeable: แก้/ยกเลิกได้เฉพาะของตัวเอง, ยังไม่ถูกยกเลิก, ยังไม่เริ่ม, และยังไม่เลยเส้นตายของเวลาเดิม
func (s *service) checkChangeable(b Booking, userID uuid.UUID, cancelBefore int) error {
	now := s.now()
	switch {
	case b.UserID != userID:
		return ErrForbidden
	case b.Status == StatusCancelled:
		return ErrBookingCancelled
	case !b.StartAt.After(now):
		return ErrAlreadyStarted
	case !CanCancel(b.StartAt, cancelBefore, now):
		return &CancelWindowError{Until: CancelDeadline(b.StartAt, cancelBefore)}
	}
	return nil
}

// notifyOwner แจ้งเจ้าของร้านเมื่อลูกค้าจอง/แก้/ยกเลิก — เรียกใน tx เดียวกับงานนั้น
// ไม่แจ้งถ้าเจ้าของร้านเป็นคนทำเอง (จองร้านตัวเองได้ — ดู README)
func notifyOwner(ctx context.Context, tx Repository, r RestaurantInfo, actor uuid.UUID, kind string, b Booking) error {
	if r.OwnerID == actor {
		return nil
	}
	return tx.Notify(ctx, notification.Draft{Recipient: r.OwnerID, Kind: kind, BookingID: b.ID,
		BusinessDate: BusinessDateOf(b.StartAt, r.Hours())})
}
