# ปิดร้านชั่วคราว + Notification Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** เจ้าของร้านปิดร้านเป็นช่วงเวลาได้ (ยืนยันก่อนยกเลิกการจองที่ทับ) และทั้งลูกค้า/เจ้าของร้านได้รับแจ้งเตือนในเว็บ

**Architecture:** ช่วงปิดเก็บเป็นช่วงเวลาจริงใน `restaurant_closures`; กติกา "ทับช่วงปิดไหม" มีฟังก์ชันเดียว `closureAt` ใน `booking/businessday.go` ที่ `checkSlot`/`Slots`/`SlotsAround` ใช้ร่วมกัน การปิดร้านและยกเลิกการจองอยู่ใน `restaurant.Service.CreateClosure` ในทรานแซกชันที่ล็อกแถวร้าน notification เขียนด้วยฟังก์ชันเดียว `notification.Insert(ctx, tx, Draft)` (INSERT … SELECT snapshot จาก DB) ในทรานแซกชันของงานนั้นเสมอ domain `notification` มี API อ่าน/กดอ่าน หน้าเว็บ poll ทุก 60 วินาที

**Tech Stack:** Go + Gin + GORM + goose + mockery + testcontainers (api), Next.js + TanStack Query + Vitest + Playwright (web/e2e)

**Spec:** `docs/design-notes/specs/2026-09-26-closures-notifications-design.md`

## Global Constraints

- ข้อความ commit ภาษาไทย ลงท้าย `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`
- `user_id` มาจาก token เท่านั้น (CLAUDE.md ข้อ 5 กฎข้อศูนย์)
- ทุกการตรวจ/เขียนที่เกี่ยวกับการจองอยู่ในทรานแซกชันที่ล็อกแถวร้าน (`FOR UPDATE`) — รวมการปิดร้าน
- แปลงวันทำการ ↔ เวลาจริงผ่าน `businessday.go` เท่านั้น; เช็คช่วงปิดผ่าน `closureAt` เท่านั้น
- notification สร้างในทรานแซกชันเดียวกับงานที่ทำให้เกิด และ **ไม่แจ้งคนที่เป็นคนทำเอง**
- API ส่ง timestamp เต็มพร้อม offset; error รูปแบบเดียว `{error:{code,message,details}}`
- Error code ใหม่: `RESTAURANT_CLOSED` 400 (`reason, start_at, end_at`), `INVALID_CLOSURE` 400, `CLOSURE_AFFECTS_BOOKINGS` 409 (`bookings`)
- เหตุผลการปิด: บังคับ, ≤ 200 ตัวอักษร; ช่วงปิดยาว ≤ 90 วัน; เริ่มไม่ก่อน "ตอนนี้ปัดลง :00/:30"
- ไอคอน lucide ไม่ใช้อีโมจิ; ตัวอักษรไทย ≥ 13px; ฝั่ง owner ไม่มี gradient/glass, ปุ่มแดงสงวนให้การกระทำที่ทำลายข้อมูล
- ห้ามรัน `./dev.sh reset` / `./dev.sh e2e` บน DB ที่ผู้ใช้มีข้อมูลของตัวเองโดยไม่ถามก่อน (reset = TRUNCATE)

## Review Focus

1. **ยืนยันด้วยรายการเก่า แต่มีการจองใหม่แทรกระหว่างดูรายการ** → ต้องได้ 409 ใหม่และไม่ยกเลิกอะไรเลย → Task 5 service test "ยืนยันรายการเก่า…"
2. **"ปิดร้านตอนนี้" ขณะมีลูกค้านั่งอยู่** → การจองที่เริ่มแล้วต้องไม่ถูกยกเลิก → Task 5 integration `AffectedBookings` ("ไม่รวมที่เริ่มแล้ว")
3. **เจ้าของร้านจองร้านตัวเอง** → ไม่มี notification ถึงตัวเองทั้งตอนจองและตอนปิดร้านทับการจองนั้น → Task 3 "เจ้าของร้านจองร้านตัวเอง…", Task 5 "การจองของเจ้าของร้านเอง…"
4. **ปิดบางช่วงของร้านข้ามคืน (เช่น 00:30–01:30 ของรอบวันที่ 10)** → ต้องเป็นเช้าวันที่ 11 ไม่ใช่วันที่ 10 → Task 1 `TestSpanAndDays`
5. **อ่าน/กดอ่าน notification ของคนอื่น** → 404 และ `read-all` ไม่แตะของคนอื่น; route `/:id/read` กับ `/read-all` อยู่ร่วมกันได้ → Task 4 handler + integration tests

---

## File Structure

| ไฟล์ | หน้าที่ |
|---|---|
| `api/internal/booking/businessday.go` | `Hours.Span`, `Hours.Days`, `closureAt` |
| `api/internal/booking/model.go` | `Closure` (ตาราง restaurant_closures), `Booking.CancelledBy/CancelReason` |
| `api/internal/booking/availability.go`, `changes.go` | `Slots`/`SlotsAround` รับ `closures` |
| `api/internal/booking/service.go`, `repository.go`, `error.go`, `dto.go`, `handler.go` | ตรวจช่วงปิดตอนจอง, แจ้งเจ้าของร้าน, `cancelled_by`, `RESTAURANT_CLOSED` |
| `api/migrations/00008…00010` | ตารางช่วงปิด, คอลัมน์ยกเลิก, ตาราง notifications |
| `api/internal/notification/*` | domain ใหม่: model, `Insert`, repository/service/handler/dto/error |
| `api/internal/restaurant/*` | CRUD ช่วงปิด, ยกเลิกโดยร้าน + แจ้งลูกค้า, ส่งช่วงปิดให้ slot |
| `api/cmd/api/router.go` | route ใหม่ |
| `api/cmd/seed/main.go` | ช่วงปิด + การจองที่ร้านยกเลิก + notification |
| `web/lib/types.ts`, `format.ts`, `notifications.ts`, `errors.ts` | type + ฟังก์ชันแสดงผล |
| `web/services/notifications.ts`, `closures.ts` | hook |
| `web/containers/NotificationBell.tsx` + navbars | กระดิ่ง |
| `web/components/restaurant/ClosureBanner.tsx`, `BookingPanel.tsx`, `BookingError.tsx`, หน้าการจอง | ฝั่งลูกค้า |
| `web/app/owner/closures/page.tsx`, `app/owner/bookings/page.tsx`, `OwnerNavbar.tsx` | ฝั่งเจ้าของร้าน |
| `e2e/tests/owner.spec.ts`, `helpers.ts` | E2E |

---

### Task 1: แกนกลาง — ช่วงปิดใน `businessday.go` และ `Slots`

**Files:**
- Modify: `api/internal/booking/businessday.go`, `api/internal/booking/model.go`, `api/internal/booking/availability.go`, `api/internal/booking/changes.go`, `api/internal/booking/service.go` (`toRequest`, `Board`), `api/internal/booking/dto.go` (rename), `api/internal/restaurant/service.go` (ส่ง `nil` ชั่วคราว)
- Test: `api/internal/booking/businessday_test.go`, `availability_test.go`, `changes_test.go`, `service_test.go`

**Interfaces:**
- Produces:
  - `type booking.Closure struct { ID, RestaurantID uuid.UUID; StartAt, EndAt time.Time; Reason string; CreatedAt time.Time }` + `TableName() = "restaurant_closures"`
  - `func (h Hours) Span(date time.Time, startMinute, endMinute int) (start, end time.Time)`
  - `func (h Hours) Days(from, to time.Time) (start, end time.Time)`
  - `func closureAt(closures []Closure, start, end time.Time) *Closure` (unexported)
  - `func Slots(h Hours, seats int, bookings []Booking, closures []Closure, date, now time.Time) []Slot`
  - `func SlotsAround(h Hours, seats int, bookings []Booking, closures []Closure, date time.Time, minuteOfDay int, now time.Time) []CardSlot`
  - `func BusinessDateOf(start time.Time, h Hours) string` (เปลี่ยนชื่อจาก `businessDate` ใน dto.go ให้ package restaurant ใช้ได้)

- [ ] **Step 1: ปรับ call site เดิมของ `Slots`/`SlotsAround` ในเทสต์ให้รับ `closures` (ใส่ `nil`)**

Run:
```bash
cd api && perl -pi -e 's/\b(Slots(?:Around)?\(\w+, \d+, (?:nil|\[\]Booking\{b\}), )/$1nil, /g' internal/booking/availability_test.go internal/booking/changes_test.go
grep -n "Slots(\|SlotsAround(" internal/booking/availability_test.go internal/booking/changes_test.go | grep -v "func Test"
```
Expected: ทุกบรรทัดที่เรียกมีอาร์กิวเมนต์ตัวที่สี่เป็น `nil` แล้ว เช่น `Slots(overnight, 10, nil, nil, bkk(…` และ `SlotsAround(overnight, 10, []Booking{b}, nil, date, …` (14 บรรทัด)

และเปลี่ยนชื่อ `businessDate(` → `BusinessDateOf(`:
```bash
cd api && perl -pi -e 's/\bbusinessDate\(/BusinessDateOf(/g' internal/booking/dto.go internal/booking/service_test.go
```

- [ ] **Step 2: เขียนเทสต์ใหม่**

ต่อท้าย `businessday_test.go`:
```go
func TestSpanAndDays(t *testing.T) {
	start, end := overnight.Span(bkk(2026, 10, 10, 0, 0), 23*60, 60)
	assert.True(t, start.Equal(bkk(2026, 10, 10, 23, 0)), "start = %s", start)
	assert.True(t, end.Equal(bkk(2026, 10, 11, 1, 0)), "01:00 ของรอบวันที่ 10 = เช้าวันที่ 11 (end = %s)", end)

	start, end = overnight.Span(bkk(2026, 10, 10, 0, 0), 30, 90)
	assert.True(t, start.Equal(bkk(2026, 10, 11, 0, 30)), "00:30 ของรอบวันที่ 10 = เช้าวันที่ 11 (start = %s)", start)
	assert.True(t, end.Equal(bkk(2026, 10, 11, 1, 30)))

	start, end = overnight.Days(bkk(2026, 10, 10, 0, 0), bkk(2026, 10, 12, 0, 0))
	assert.True(t, start.Equal(bkk(2026, 10, 10, 18, 0)))
	assert.True(t, end.Equal(bkk(2026, 10, 13, 2, 0)), "ถึงเวลาปิดของรอบวันที่ 12 = ตีสองวันที่ 13 (end = %s)", end)
}

func TestClosureAt(t *testing.T) {
	a := Closure{Reason: "ไฟดับ", StartAt: bkk(2026, 10, 10, 18, 0), EndAt: bkk(2026, 10, 10, 20, 0)}
	b := Closure{Reason: "ปรับปรุง", StartAt: bkk(2026, 10, 10, 21, 0), EndAt: bkk(2026, 10, 10, 22, 0)}
	list := []Closure{a, b}

	assert.Nil(t, closureAt(nil, bkk(2026, 10, 10, 18, 0), bkk(2026, 10, 10, 19, 0)))
	assert.Nil(t, closureAt(list, bkk(2026, 10, 10, 17, 0), bkk(2026, 10, 10, 18, 0)), "จบตอนเริ่มปิดพอดี = ไม่ทับ")
	assert.Nil(t, closureAt(list, bkk(2026, 10, 10, 20, 0), bkk(2026, 10, 10, 21, 0)), "อยู่ระหว่างสองช่วงพอดี")
	got := closureAt(list, bkk(2026, 10, 10, 19, 30), bkk(2026, 10, 10, 21, 30))
	require.NotNil(t, got)
	assert.Equal(t, "ไฟดับ", got.Reason, "ทับสองช่วง → คืนตัวแรก")
}
```

ต่อท้าย `availability_test.go`:
```go
func TestSlotsSkipClosure(t *testing.T) {
	longAgo := bkk(2026, 10, 1, 0, 0)
	closed := []Closure{{Reason: "ไฟดับ", StartAt: bkk(2026, 10, 10, 18, 0), EndAt: bkk(2026, 10, 10, 20, 0)}}

	s := Slots(normal, 10, nil, closed, bkk(2026, 10, 10, 0, 0), longAgo)
	require.Len(t, s, 18, "22 ช่วง − 4 ช่วงที่ปิด")
	assert.True(t, s[13].StartAt.Equal(bkk(2026, 10, 10, 17, 30)))
	assert.True(t, s[14].StartAt.Equal(bkk(2026, 10, 10, 20, 0)), "ถัดจาก 17:30 คือ 20:00")

	cards := SlotsAround(normal, 10, nil, closed, bkk(2026, 10, 10, 0, 0), 18*60, longAgo)
	assert.Equal(t, []bool{false, false, true, true, true},
		[]bool{cards[0].Closed, cards[1].Closed, cards[2].Closed, cards[3].Closed, cards[4].Closed}, "17:00 17:30 เปิด / 18:00–19:00 ปิด")
}
```

- [ ] **Step 3: รันให้เห็นว่าไม่ผ่าน**

Run: `cd api && go test ./internal/booking/ 2>&1 | head -5`
Expected: compile error เช่น `undefined: Closure` / `too many arguments in call to Slots` / `undefined: BusinessDateOf`

- [ ] **Step 4: เขียนโค้ด**

`model.go` ต่อท้ายไฟล์:
```go
// Closure คือช่วงที่ร้านปิดชั่วคราว (ตาราง restaurant_closures)
// อยู่ใน package booking เพราะการจองและตารางเวลาว่างต้องใช้ — package restaurant เป็นคนสร้าง/ลบ
type Closure struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	RestaurantID uuid.UUID `gorm:"type:uuid;not null"`
	StartAt      time.Time
	EndAt        time.Time
	Reason       string
	CreatedAt    time.Time
}

func (Closure) TableName() string { return "restaurant_closures" }
```

`businessday.go` ต่อท้ายไฟล์:
```go
// Span แปลงช่วงเวลาบนนาฬิกาในวันทำการ date เป็นช่วงจริง (ใช้ทั้งการจองและการปิดร้านบางช่วง)
// ผ่าน At จึงรองรับร้านข้ามคืน; เวลาจบที่ไม่หลังเวลาเริ่ม (เช่นร้าน 24 ชม. 23:30–00:00) แปลว่าจบวันถัดไป
func (h Hours) Span(date time.Time, startMinute, endMinute int) (start, end time.Time) {
	start = h.At(date, startMinute)
	end = h.At(date, endMinute)
	if !end.After(start) {
		end = end.AddDate(0, 0, 1)
	}
	return start, end
}

// Days คืนช่วงตั้งแต่เวลาเปิดของวันทำการ from ถึงเวลาปิดของวันทำการ to (ปิดร้านทั้งวัน/หลายวัน)
func (h Hours) Days(from, to time.Time) (start, end time.Time) {
	start, _ = h.Window(from)
	_, end = h.Window(to)
	return start, end
}

// closureAt คืนช่วงปิดตัวแรกที่ทับ [start,end) — nil ถ้าไม่ทับ (จบตอนเริ่มปิดพอดี = ไม่ทับ)
// ใช้ทั้งตอนจอง/แก้การจองและตอนคำนวณตารางเวลาว่าง — มีฟังก์ชันเดียว ห้ามเขียนซ้ำ
func closureAt(closures []Closure, start, end time.Time) *Closure {
	for i := range closures {
		if start.Before(closures[i].EndAt) && end.After(closures[i].StartAt) {
			return &closures[i]
		}
	}
	return nil
}
```

`service.go` — `toRequest` ใช้ `Span`:
```go
// toRequest แปลงตัวเลือกเป็นเวลาจริงผ่าน Hours.Span (เวลาที่น้อยกว่าเวลาเปิด = หลังเที่ยงคืนของรอบนั้น)
func (c Choice) toRequest(h Hours) Request {
	start, end := h.Span(c.Date, c.StartMinute, c.EndMinute)
	return Request{StartAt: start, EndAt: end, PartySize: c.PartySize}
}
```
และใน `Board`: `slots := Slots(r.Hours(), r.Seats, active, nil, date, time.Time{})` (Task 3 จะใส่ช่วงปิดจริง)

`availability.go` — signature + เงื่อนไขข้าม:
```go
// Slots คืนทุกช่วง 30 นาทีของรอบวันทำการ date
// bookings = booking 'active' ที่ทับกับรอบนั้น; ช่วงที่เริ่มก่อน now+LeadTime จองไม่ได้แล้ว, ช่วงพัก และช่วงที่ร้านปิดชั่วคราว จึงไม่อยู่ในลิสต์
// วันปิดประจำสัปดาห์ไม่มีรอบ → คืนลิสต์ว่าง
func Slots(h Hours, seats int, bookings []Booking, closures []Closure, date, now time.Time) []Slot {
```
และบรรทัดเงื่อนไขใน loop:
```go
		if t.Before(earliest) || h.overlapsBreak(date, t, end) || closureAt(closures, t, end) != nil {
```

`changes.go` — `SlotsAround`:
```go
func SlotsAround(h Hours, seats int, bookings []Booking, closures []Closure, date time.Time, minuteOfDay int, now time.Time) []CardSlot {
```
และ
```go
		if !h.Fits(start, end) || start.Before(earliest) || closureAt(closures, start, end) != nil {
```
และแก้คอมเมนต์ของ `CardSlot.Closed` เป็น `// ร้านปิดช่วงนั้น (นอกเวลา/ช่วงพัก/ปิดชั่วคราว) หรือจองไม่ทันแล้ว (เลย lead time)`

`dto.go` — คอมเมนต์ฟังก์ชันที่เปลี่ยนชื่อ:
```go
// BusinessDateOf: วันทำการของการจองที่เริ่ม start — ถ้าเวลาเริ่มอยู่ก่อนเวลาเปิดของวันปฏิทินนั้น แปลว่าเป็นส่วนหลังเที่ยงคืนของรอบเมื่อวาน
// export ไว้ให้ package restaurant ใช้ทำ notification ด้วยกติกาเดียวกัน
func BusinessDateOf(start time.Time, h Hours) string {
```

`restaurant/service.go` — ใส่ `nil` ชั่วคราวที่ 4 จุด (Task 5 ใส่ของจริง):
```go
		items[i].Slots = booking.SlotsAround(r.Hours(), r.Seats, byRestaurant[r.ID], nil, *q.Date, q.Minute, now)
```
```go
	return rest, booking.Slots(rest.Hours(), rest.Seats, bookings, nil, date, s.now()), nil
```
```go
		around := booking.SlotsAround(h, rest.Seats, bookings, nil, d, minute, now)
```
```go
			ok = firstFree(booking.Slots(h, rest.Seats, bookings, nil, d, now), party, len(around))
```

- [ ] **Step 5: รันเทสต์**

Run: `cd api && go build ./... && go test ./internal/booking/ ./internal/restaurant/ 2>&1 | tail -3`
Expected: `ok` ทั้งสอง package

- [ ] **Step 6: Commit**

```bash
git add api/internal
git commit -m "เพิ่มช่วงปิดชั่วคราวในแกนกลาง: closureAt, Hours.Span/Days และตารางเวลาว่างข้ามช่วงปิด

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 2: Migration + `notification.Insert`

**Files:**
- Create: `api/migrations/00008_create_restaurant_closures.sql`, `00009_add_booking_cancel_info.sql`, `00010_create_notifications.sql`
- Create: `api/internal/notification/model.go`, `api/internal/notification/repository.go`
- Test: `api/internal/notification/repository_integration_test.go`

**Interfaces:**
- Produces (package `notification`):
  - const `KindBookingCreated`, `KindBookingUpdated`, `KindBookingCancelled`, `KindCancelledByRestaurant`
  - `type Notification struct {...}` (ตาราง notifications)
  - `type Draft struct { Recipient uuid.UUID; Kind string; BookingID uuid.UUID; BusinessDate string; Reason string }`
  - `func Insert(ctx context.Context, db *gorm.DB, d Draft) error`

- [ ] **Step 1: Migrations**

`00008_create_restaurant_closures.sql`:
```sql
-- +goose Up
-- ช่วงที่ร้านปิดชั่วคราว (ไฟดับ, ปิดปรับปรุง, หยุดยาว) เป็นช่วงเวลาจริง [start_at, end_at)
-- ปิดทั้งวัน/หลายวันก็เก็บรูปแบบเดียวกัน (= ช่วงที่ครอบทั้งรอบของวันนั้น)
CREATE TABLE restaurant_closures (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    restaurant_id uuid NOT NULL REFERENCES restaurants(id) ON DELETE CASCADE,
    start_at      timestamptz NOT NULL,
    end_at        timestamptz NOT NULL,
    reason        text NOT NULL CHECK (reason <> ''),
    created_at    timestamptz NOT NULL DEFAULT now(),
    CHECK (end_at > start_at)
);
CREATE INDEX restaurant_closures_restaurant_end_idx ON restaurant_closures (restaurant_id, end_at);

-- +goose Down
DROP TABLE restaurant_closures;
```

`00009_add_booking_cancel_info.sql`:
```sql
-- +goose Up
-- ใครเป็นคนยกเลิก: ลูกค้ายกเลิกเอง หรือร้านยกเลิก (ปิดชั่วคราว/ลบร้าน) พร้อมเหตุผลที่ลูกค้าจะเห็น
-- การจองที่ยกเลิกไปก่อนมีคอลัมน์นี้คงเป็น null (ไม่ backfill เพราะไม่รู้จริงว่าใครยกเลิก)
ALTER TABLE bookings
    ADD COLUMN cancelled_by  text CHECK (cancelled_by IN ('customer', 'restaurant')),
    ADD COLUMN cancel_reason text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE bookings DROP COLUMN cancelled_by, DROP COLUMN cancel_reason;
```

`00010_create_notifications.sql`:
```sql
-- +goose Up
-- แจ้งเตือนในเว็บ — เก็บ snapshot ของการจองตอนเกิดเหตุ (การจองอาจถูกแก้ภายหลัง แต่แจ้งเตือนต้องบอกสิ่งที่เกิดตอนนั้น)
-- ไม่เก็บข้อความ: หน้าเว็บประกอบข้อความไทยจาก kind เอง
CREATE TABLE notifications (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         uuid NOT NULL REFERENCES users(id),
    kind            text NOT NULL CHECK (kind IN ('booking_created', 'booking_updated', 'booking_cancelled', 'booking_cancelled_by_restaurant')),
    booking_id      uuid NOT NULL REFERENCES bookings(id),
    restaurant_id   uuid NOT NULL REFERENCES restaurants(id),
    restaurant_name text NOT NULL,
    customer_name   text NOT NULL,
    business_date   date NOT NULL,
    start_at        timestamptz NOT NULL,
    end_at          timestamptz NOT NULL,
    party_size      int NOT NULL,
    reason          text NOT NULL DEFAULT '',
    read_at         timestamptz,
    created_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX notifications_user_created_idx ON notifications (user_id, created_at DESC);

-- +goose Down
DROP TABLE notifications;
```

- [ ] **Step 2: เขียนเทสต์ integration** — `api/internal/notification/repository_integration_test.go`

```go
package notification

import (
	"context"
	"testing"
	"time"

	"jongyoung/internal/platform/testdb"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// เวลาไทย (ไม่มี DST) — ไม่ import booking เพราะ booking import notification (import cycle)
var bangkokTest = time.FixedZone("Asia/Bangkok", 7*60*60)

func at(h int) time.Time { return time.Date(2026, 10, 10, h, 0, 0, 0, bangkokTest) }

func insertUser(t *testing.T, db *gorm.DB, name string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	require.NoError(t, db.Raw(`INSERT INTO users (keycloak_uid, email, display_name) VALUES (?, 'u@x.y', ?) RETURNING id`, uuid.NewString(), name).Row().Scan(&id))
	return id
}

func insertBooking(t *testing.T, db *gorm.DB, owner, customer uuid.UUID, party int) uuid.UUID {
	t.Helper()
	var rid, bid uuid.UUID
	require.NoError(t, db.Raw(`INSERT INTO restaurants (owner_id, name, address, seats, open_minute, close_minute)
		VALUES (?, 'ครัวบ้านสวน', 'กรุงเทพฯ', 10, 660, 1320) RETURNING id`, owner).Row().Scan(&rid))
	require.NoError(t, db.Raw(`INSERT INTO bookings (restaurant_id, user_id, party_size, start_at, end_at)
		VALUES (?, ?, ?, ?, ?) RETURNING id`, rid, customer, party, at(18), at(19)).Row().Scan(&bid))
	return bid
}

func TestInsertSnapshot(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	owner, customer := insertUser(t, db, "เจ้าของ"), insertUser(t, db, "มะลิ")
	bid := insertBooking(t, db, owner, customer, 4)

	require.NoError(t, Insert(ctx, db, Draft{Recipient: owner, Kind: KindBookingCreated, BookingID: bid, BusinessDate: "2026-10-10"}))
	require.NoError(t, db.Exec(`UPDATE bookings SET party_size = 2 WHERE id = ?`, bid).Error)

	var n Notification
	require.NoError(t, db.First(&n, "user_id = ?", owner).Error)
	assert.Equal(t, KindBookingCreated, n.Kind)
	assert.Equal(t, "ครัวบ้านสวน", n.RestaurantName)
	assert.Equal(t, "มะลิ", n.CustomerName)
	assert.Equal(t, 4, n.PartySize, "snapshot ตอนเกิดเหตุ — แก้การจองทีหลังไม่กระทบ")
	assert.Equal(t, "2026-10-10", n.BusinessDate.Format("2006-01-02"))
	assert.True(t, n.StartAt.Equal(at(18)))
	assert.Nil(t, n.ReadAt)
}
```

- [ ] **Step 3: รันให้เห็นว่าไม่ผ่าน**

Run: `cd api && go test ./internal/notification/ 2>&1 | head -3`
Expected: `undefined: Insert` / `undefined: Draft`

- [ ] **Step 4: เขียนโค้ด**

`api/internal/notification/model.go`:
```go
package notification

import (
	"time"

	"github.com/google/uuid"
)

// kind ของแจ้งเตือน — หน้าเว็บแปลงเป็นข้อความไทยเอง
const (
	KindBookingCreated        = "booking_created"                 // ลูกค้าจองใหม่ → เจ้าของร้าน
	KindBookingUpdated        = "booking_updated"                 // ลูกค้าแก้การจอง → เจ้าของร้าน
	KindBookingCancelled      = "booking_cancelled"               // ลูกค้ายกเลิก → เจ้าของร้าน
	KindCancelledByRestaurant = "booking_cancelled_by_restaurant" // ร้านปิดชั่วคราว/ลบร้าน → ลูกค้า
)

// Notification คือแจ้งเตือนหนึ่งรายการ (ตาราง notifications) — ข้อมูลการจองเป็น snapshot ตอนเกิดเหตุ
type Notification struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID         uuid.UUID `gorm:"type:uuid;not null"` // ผู้รับ
	Kind           string
	BookingID      uuid.UUID `gorm:"type:uuid"`
	RestaurantID   uuid.UUID `gorm:"type:uuid"`
	RestaurantName string
	CustomerName   string
	BusinessDate   time.Time // วันทำการ (ร้านข้ามคืน ≠ วันปฏิทิน) ใช้ทำลิงก์ไปบอร์ดเจ้าของร้าน
	StartAt        time.Time
	EndAt          time.Time
	PartySize      int
	Reason         string
	ReadAt         *time.Time
	CreatedAt      time.Time
}

// Draft = ส่วนที่ผู้สร้างแจ้งเตือนกำหนดเอง ส่วนที่เหลือ (ชื่อร้าน ชื่อลูกค้า เวลา จำนวนคน) Insert ดึงจาก DB เอง
type Draft struct {
	Recipient    uuid.UUID
	Kind         string
	BookingID    uuid.UUID
	BusinessDate string // YYYY-MM-DD
	Reason       string
}
```

`api/internal/notification/repository.go`:
```go
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
```

- [ ] **Step 5: รันเทสต์ทั้ง api** (migration ใหม่ต้องไม่ทำให้เทสต์เดิมพัง)

Run: `cd api && go test ./... 2>&1 | grep -v "no test files"`
Expected: ทุก package `ok`

- [ ] **Step 6: Commit**

```bash
git add api/migrations api/internal/notification
git commit -m "เพิ่มตารางช่วงปิดร้าน, ผู้ยกเลิกการจอง และ notifications + ฟังก์ชันเขียนแจ้งเตือนแบบ snapshot

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 3: การจองรู้จักช่วงปิด + แจ้งเจ้าของร้าน

**Files:**
- Modify: `api/internal/booking/model.go`, `repository.go`, `service.go`, `error.go`, `dto.go`, `handler.go`, `mocks_test.go` (สร้างใหม่ด้วย mockery)
- Test: `api/internal/booking/service_test.go`, `concurrency_integration_test.go`

**Interfaces:**
- Consumes: `Closure`, `closureAt`, `Slots(…, closures, …)` (Task 1); `notification.Draft`, `notification.Insert`, `notification.Kind*` (Task 2)
- Produces:
  - `booking.Repository` เพิ่ม `ClosuresBetween(ctx context.Context, restaurantID uuid.UUID, from, to time.Time) ([]Closure, error)` และ `Notify(ctx context.Context, d notification.Draft) error`
  - `Booking.CancelledBy *string`, `Booking.CancelReason string`; const `CancelledByCustomer = "customer"`, `CancelledByRestaurant = "restaurant"`
  - `type ClosedError struct{ Closure Closure }` → 400 `RESTAURANT_CLOSED`
  - JSON `cancelled_by`, `cancel_reason` ใน `BookingResponse`

- [ ] **Step 1: model + interface + mock**

`model.go` — ต่อจาก const status:
```go
// ใครเป็นคนยกเลิก (คอลัมน์ cancelled_by) — ลูกค้าต้องรู้ว่าร้านยกเลิกให้ ไม่ใช่ตัวเองกดพลาด
const (
	CancelledByCustomer   = "customer"
	CancelledByRestaurant = "restaurant"
)
```
ใน `Booking` ต่อจาก `CancelledAt  *time.Time`:
```go
	CancelledBy  *string // nil = ยังไม่ยกเลิก (หรือข้อมูลเก่าก่อนมีคอลัมน์นี้)
	CancelReason string  // เหตุผลจากร้าน
```

`service.go` — ใน `Repository` interface เพิ่มสองบรรทัดท้าย:
```go
	ClosuresBetween(ctx context.Context, restaurantID uuid.UUID, from, to time.Time) ([]Closure, error)
	Notify(ctx context.Context, d notification.Draft) error
```
และ import `"jongyoung/internal/notification"`

`repository.go` — เพิ่ม (import `"jongyoung/internal/notification"`):
```go
// ClosuresBetween = ช่วงปิดชั่วคราวของร้านที่ทับ [from,to)
func (r *repository) ClosuresBetween(ctx context.Context, restaurantID uuid.UUID, from, to time.Time) ([]Closure, error) {
	var list []Closure
	err := r.db.WithContext(ctx).
		Where("restaurant_id = ? AND start_at < ? AND end_at > ?", restaurantID, to, from).
		Order("start_at").
		Find(&list).Error
	return list, err
}

// Notify เขียนแจ้งเตือนด้วย db ของ repository นี้ — ใน Transaction คือ tx เดียวกับการเขียน booking
func (r *repository) Notify(ctx context.Context, d notification.Draft) error {
	return notification.Insert(ctx, r.db, d)
}
```
และ `Cancel` บันทึกว่าลูกค้าเป็นคนยกเลิก:
```go
// Cancel เปลี่ยนสถานะเป็น cancelled โดยลูกค้า — ไม่ลบแถวจริง เพื่อเก็บประวัติ
func (r *repository) Cancel(ctx context.Context, id uuid.UUID, now time.Time) error {
	return r.db.WithContext(ctx).Model(&Booking{}).Where("id = ?", id).
		Updates(map[string]any{"status": StatusCancelled, "cancelled_at": now, "updated_at": now, "cancelled_by": CancelledByCustomer}).Error
}
```

Run: `cd api && mockery 2>&1 | tail -2 && go build ./...`
Expected: mock ใหม่ใน `internal/booking/mocks_test.go` มี `ClosuresBetween`/`Notify`; build ผ่าน

- [ ] **Step 2: ปรับเทสต์เดิมที่ mock จะไม่ยอมเพราะ service เรียก repository เพิ่ม + เขียนเทสต์ใหม่**

ใน `service_test.go` เพิ่ม import `"jongyoung/internal/notification"` แล้วแก้:

(a) ทุกบรรทัด `\t\trepo.EXPECT().FindOverlappingOwn(ctx, rid, user, start, end, uuid.Nil)` ใน `TestServiceCreate` (3 จุด) ใส่บรรทัดนี้ไว้ข้างบน:
```go
		repo.EXPECT().ClosuresBetween(ctx, rid, start, end).Return(nil, nil)
```
Run: `cd api && perl -0pi -e 's/(\t\trepo\.EXPECT\(\)\.FindOverlappingOwn\(ctx, rid, user, start, end, uuid\.Nil\))/\t\trepo.EXPECT().ClosuresBetween(ctx, rid, start, end).Return(nil, nil)\n$1/g' internal/booking/service_test.go && grep -c "ClosuresBetween(ctx, rid, start, end)" internal/booking/service_test.go`
Expected: `3`

(b) ใน subtest "ข้ามเที่ยงคืน" ต่อจากบรรทัด `})).Return(nil)` ของ `repo.EXPECT().Create(...)`:
```go
		repo.EXPECT().Notify(ctx, notification.Draft{Recipient: uuid.Nil, Kind: notification.KindBookingCreated,
			BookingID: uuid.Nil, BusinessDate: "2026-10-10"}).Return(nil)
```
(`rest.OwnerID` เป็นค่าว่าง ≠ `user` จึงต้องแจ้ง; mock `Create` ไม่ใส่ ID ให้ → `uuid.Nil`)

(c) ใน subtest "3) A แก้เป็น 8 คน…" ก่อนบรรทัด `repo.EXPECT().FindOverlappingOwn(ctx, rid, user, mine.StartAt, mine.EndAt, bid)`:
```go
		repo.EXPECT().ClosuresBetween(ctx, rid, mine.StartAt, mine.EndAt).Return(nil, nil)
```

(d) ใน subtest "Cancel: ตรงเส้นตายพอดี…" ต่อจาก `repo.EXPECT().Cancel(ctx, bid, deadline).Return(nil)`:
```go
		repo.EXPECT().Notify(ctx, notification.Draft{Recipient: uuid.Nil, Kind: notification.KindBookingCancelled,
			BookingID: bid, BusinessDate: "2026-10-10"}).Return(nil)
```

(e) ใน subtest "20) บอร์ดวันเสาร์…" ต่อจาก `repo.EXPECT().FindRestaurant(ctx, rid, false).Return(rest, nil)`:
```go
		repo.EXPECT().ClosuresBetween(ctx, rid, bkk(2026, 10, 10, 18, 0), bkk(2026, 10, 11, 2, 0)).Return(nil, nil)
```

เพิ่มเทสต์ใหม่ท้าย `service_test.go`:
```go
func TestServiceClosuresAndNotify(t *testing.T) {
	ctx := context.Background()
	user, owner, rid := uuid.New(), uuid.New(), uuid.New()
	rest := RestaurantInfo{ID: rid, OwnerID: owner, Seats: 10, OpenMinute: 11 * 60, CloseMinute: 22 * 60, CancelBeforeMinutes: 30}
	choice := Choice{Date: bkk(2026, 10, 10, 0, 0), StartMinute: 19 * 60, EndMinute: 20 * 60, PartySize: 2}
	start, end := bkk(2026, 10, 10, 19, 0), bkk(2026, 10, 10, 20, 0)

	t.Run("ทับช่วงปิด → ClosedError พร้อมเหตุผล และไม่ตรวจกฎ 8/7 ต่อ", func(t *testing.T) {
		repo := NewMockRepository(t)
		expectTx(repo)
		repo.EXPECT().LockRestaurant(ctx, rid).Return(rest, nil)
		repo.EXPECT().ClosuresBetween(ctx, rid, start, end).Return([]Closure{
			{Reason: "ไฟดับ", StartAt: bkk(2026, 10, 10, 18, 0), EndAt: bkk(2026, 10, 10, 22, 0)},
		}, nil)
		_, err := newSvc(repo).Create(ctx, user, rid, choice)
		var closed *ClosedError
		require.True(t, errors.As(err, &closed))
		assert.Equal(t, "ไฟดับ", closed.Closure.Reason)
	})

	t.Run("จองสำเร็จ → แจ้งเจ้าของร้าน", func(t *testing.T) {
		repo := NewMockRepository(t)
		expectTx(repo)
		repo.EXPECT().LockRestaurant(ctx, rid).Return(rest, nil)
		repo.EXPECT().ClosuresBetween(ctx, rid, start, end).Return(nil, nil)
		repo.EXPECT().FindOverlappingOwn(ctx, rid, user, start, end, uuid.Nil).Return(nil, nil)
		repo.EXPECT().Overlapping(ctx, rid, start, end, uuid.Nil).Return(nil, nil)
		repo.EXPECT().Create(ctx, mock.Anything).Return(nil)
		repo.EXPECT().Notify(ctx, notification.Draft{Recipient: owner, Kind: notification.KindBookingCreated,
			BookingID: uuid.Nil, BusinessDate: "2026-10-10"}).Return(nil)
		_, err := newSvc(repo).Create(ctx, user, rid, choice)
		assert.NoError(t, err)
	})

	t.Run("เจ้าของร้านจองร้านตัวเอง → ไม่แจ้งตัวเอง (ไม่มี EXPECT Notify — ถ้าเรียก mock จะ fail)", func(t *testing.T) {
		repo := NewMockRepository(t)
		expectTx(repo)
		repo.EXPECT().LockRestaurant(ctx, rid).Return(rest, nil)
		repo.EXPECT().ClosuresBetween(ctx, rid, start, end).Return(nil, nil)
		repo.EXPECT().FindOverlappingOwn(ctx, rid, owner, start, end, uuid.Nil).Return(nil, nil)
		repo.EXPECT().Overlapping(ctx, rid, start, end, uuid.Nil).Return(nil, nil)
		repo.EXPECT().Create(ctx, mock.Anything).Return(nil)
		_, err := newSvc(repo).Create(ctx, owner, rid, choice)
		assert.NoError(t, err)
	})
}
```

ต่อท้าย `concurrency_integration_test.go`:
```go
// ช่วงปิดชั่วคราวต้องถูกตรวจในล็อกเดียวกับการจอง และจองสำเร็จต้องมีแจ้งเตือนถึงเจ้าของร้านในทรานแซกชันเดียวกัน
func TestBookingClosureAndNotify(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	now := func() time.Time { return bkk(2026, 10, 1, 12, 0) }
	svc := NewService(NewRepository(db), now)
	owner := insertUser(t, db)
	rid := insertRestaurant(t, db, owner, 10) // 11:00–22:00
	require.NoError(t, db.Exec(`INSERT INTO restaurant_closures (restaurant_id, start_at, end_at, reason) VALUES (?, ?, ?, 'ไฟดับ')`,
		rid, bkk(2026, 10, 10, 18, 0), bkk(2026, 10, 10, 20, 0)).Error)
	date := bkk(2026, 10, 10, 0, 0)

	_, err := svc.Create(ctx, insertUser(t, db), rid, Choice{Date: date, StartMinute: 19 * 60, EndMinute: 20 * 60, PartySize: 2})
	var closed *ClosedError
	require.True(t, errors.As(err, &closed), "err = %v", err)
	assert.Equal(t, "ไฟดับ", closed.Closure.Reason)

	b, err := svc.Create(ctx, insertUser(t, db), rid, Choice{Date: date, StartMinute: 20 * 60, EndMinute: 21 * 60, PartySize: 2})
	require.NoError(t, err)
	var n int64
	require.NoError(t, db.Table("notifications").Where("user_id = ? AND booking_id = ? AND kind = 'booking_created'", owner, b.ID).Count(&n).Error)
	assert.EqualValues(t, 1, n)
}
```

- [ ] **Step 3: รันให้เห็นว่าไม่ผ่าน**

Run: `cd api && go test ./internal/booking/ 2>&1 | grep -E "^(--- FAIL|FAIL|ok)|undefined" | head`
Expected: compile error `undefined: ClosedError` (หรือ FAIL จาก mock ที่ยังไม่ถูกเรียก)

- [ ] **Step 4: เขียนโค้ด**

`error.go` ต่อท้าย:
```go
// ClosedError → 400 RESTAURANT_CLOSED: ช่วงที่ขอทับช่วงที่ร้านปิดชั่วคราว (หน้าเว็บบอกเหตุผลและช่วงปิดได้)
type ClosedError struct {
	Closure Closure
}

func (e *ClosedError) Error() string { return "RESTAURANT_CLOSED: " + e.Closure.Reason }
```

`service.go` — ใน `checkSlot` ต่อจากบล็อก `ValidateRequest`:
```go
	// ช่วงปิดชั่วคราว — อ่านในล็อกเดียวกัน: เจ้าของร้านกดปิดพร้อมกับมีคนจอง ก็ต่อคิวที่ล็อกแถวร้านเดียวกัน
	closures, err := tx.ClosuresBetween(ctx, r.ID, req.StartAt, req.EndAt)
	if err != nil {
		return err
	}
	if c := closureAt(closures, req.StartAt, req.EndAt); c != nil {
		return &ClosedError{Closure: *c}
	}
```
และเปลี่ยนบรรทัดถัดไป `dup, err := tx.FindOverlappingOwn(...)` ให้ยังคอมไพล์ได้ (`err` ประกาศแล้ว → ใช้ `dup, err := …` ได้เพราะ `dup` เป็นตัวแปรใหม่)

เพิ่ม helper ท้ายไฟล์ `service.go`:
```go
// notifyOwner แจ้งเจ้าของร้านเมื่อลูกค้าจอง/แก้/ยกเลิก — เรียกใน tx เดียวกับงานนั้น
// ไม่แจ้งถ้าเจ้าของร้านเป็นคนทำเอง (จองร้านตัวเองได้ — ดู README)
func notifyOwner(ctx context.Context, tx Repository, r RestaurantInfo, actor uuid.UUID, kind string, b Booking) error {
	if r.OwnerID == actor {
		return nil
	}
	return tx.Notify(ctx, notification.Draft{Recipient: r.OwnerID, Kind: kind, BookingID: b.ID,
		BusinessDate: BusinessDateOf(b.StartAt, r.Hours())})
}
```
ใน `Create` แทน `return tx.Create(ctx, &created)` ด้วย:
```go
		if err := tx.Create(ctx, &created); err != nil {
			return err
		}
		return notifyOwner(ctx, tx, r, userID, notification.KindBookingCreated, created)
```
ใน `Update` แทน `return tx.UpdateTime(ctx, &b)` ด้วย:
```go
		if err := tx.UpdateTime(ctx, &b); err != nil {
			return err
		}
		return notifyOwner(ctx, tx, r, userID, notification.KindBookingUpdated, b)
```
ใน `Cancel` แทน `return tx.Cancel(ctx, b.ID, s.now())` ด้วย:
```go
		if err := tx.Cancel(ctx, b.ID, s.now()); err != nil {
			return err
		}
		return notifyOwner(ctx, tx, r, userID, notification.KindBookingCancelled, b)
```
ใน `Board` ต่อจาก `views, err := …` block:
```go
	closures, err := s.repository.ClosuresBetween(ctx, restaurantID, opensAt, closesAt)
	if err != nil {
		return Board{}, err
	}
```
และ `slots := Slots(r.Hours(), r.Seats, active, closures, date, time.Time{})`

`dto.go` — ใน `BookingResponse` ต่อจาก `CancelledAt`:
```go
	CancelledBy  *string           `json:"cancelled_by"`  // "customer" | "restaurant" | null
	CancelReason string            `json:"cancel_reason"` // เหตุผลเมื่อร้านยกเลิก
```
และใน `NewBookingResponse` struct literal ต่อจาก `Status: v.Status,`:
```go
		CancelledBy:  v.CancelledBy,
		CancelReason: v.CancelReason,
```

`handler.go` — ใน `fail` เพิ่ม `var closed *ClosedError` และ case (ก่อน `case errors.Is(err, ErrRestaurantNotFound):`):
```go
	case errors.As(err, &closed):
		httputil.Abort(c, http.StatusBadRequest, "RESTAURANT_CLOSED", "ร้านปิดในช่วงที่เลือก",
			gin.H{"reason": closed.Closure.Reason, "start_at": closed.Closure.StartAt.In(Bangkok), "end_at": closed.Closure.EndAt.In(Bangkok)})
```

- [ ] **Step 5: รันเทสต์**

Run: `cd api && go test ./internal/booking/ 2>&1 | tail -3`
Expected: `ok  	jongyoung/internal/booking`

- [ ] **Step 6: Commit**

```bash
git add api/internal/booking
git commit -m "การจองตรวจช่วงปิดชั่วคราว (RESTAURANT_CLOSED) + แจ้งเจ้าของร้านเมื่อจอง/แก้/ยกเลิก + บันทึกผู้ยกเลิก

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 4: API แจ้งเตือน (`internal/notification`)

**Files:**
- Modify: `api/internal/notification/repository.go`
- Create: `api/internal/notification/service.go`, `handler.go`, `dto.go`, `error.go`
- Modify: `api/.mockery.yml`, `api/cmd/api/router.go`
- Test: `api/internal/notification/service_test.go`, `handler_test.go`, `repository_integration_test.go`

**Interfaces:**
- Consumes: `Notification`, `Insert` (Task 2)
- Produces: `GET /me/notifications` → `{ items: NotificationResponse[], unread_count }`; `PUT /me/notifications/:id/read` → 204/404; `PUT /me/notifications/read-all` → 204
  - JSON item: `id, kind, booking_id, restaurant_id, restaurant_name, customer_name, business_date, start_at, end_at, party_size, reason, read_at, created_at`

- [ ] **Step 1: เขียนเทสต์ integration ของ repository** — ต่อท้าย `repository_integration_test.go`

```go
func TestRepositoryReadState(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	repo := NewRepository(db)
	owner, other := insertUser(t, db, "เจ้าของ"), insertUser(t, db, "คนอื่น")
	bid := insertBooking(t, db, owner, insertUser(t, db, "มะลิ"), 2)
	for range 3 {
		require.NoError(t, Insert(ctx, db, Draft{Recipient: owner, Kind: KindBookingCreated, BookingID: bid, BusinessDate: "2026-10-10"}))
	}
	require.NoError(t, Insert(ctx, db, Draft{Recipient: other, Kind: KindBookingCreated, BookingID: bid, BusinessDate: "2026-10-10"}))

	list, err := repo.ListByUser(ctx, owner, 20)
	require.NoError(t, err)
	require.Len(t, list, 3, "เห็นเฉพาะของตัวเอง")

	ok, err := repo.MarkRead(ctx, other, list[0].ID, at(12))
	require.NoError(t, err)
	assert.False(t, ok, "กดอ่านของคนอื่น → ไม่พบ")

	ok, err = repo.MarkRead(ctx, owner, list[0].ID, at(12))
	require.NoError(t, err)
	assert.True(t, ok)
	n, err := repo.CountUnread(ctx, owner)
	require.NoError(t, err)
	assert.EqualValues(t, 2, n)

	require.NoError(t, repo.MarkAllRead(ctx, owner, at(13)))
	n, _ = repo.CountUnread(ctx, owner)
	assert.Zero(t, n)
	n, _ = repo.CountUnread(ctx, other)
	assert.EqualValues(t, 1, n, "read-all ไม่แตะของคนอื่น")
}
```

- [ ] **Step 2: เขียน repository/service/dto/error/handler**

ต่อท้าย `repository.go` (เพิ่ม import `"time"`, `"github.com/google/uuid"`):
```go
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
```

`error.go`:
```go
package notification

import "errors"

// ErrNotFound: ไม่มี หรือเป็นของคนอื่น (ตอบ 404 เหมือนกัน ไม่บอกว่ามีอยู่จริง)
var ErrNotFound = errors.New("notification not found")
```

`service.go`:
```go
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
```

`dto.go`:
```go
package notification

import (
	"time"

	"github.com/google/uuid"
)

// bangkok = เวลาไทย (ไม่มี DST) — ไม่ import booking.Bangkok เพราะ booking import package นี้ (import cycle)
var bangkok = time.FixedZone("Asia/Bangkok", 7*60*60)

type NotificationResponse struct {
	ID             uuid.UUID  `json:"id"`
	Kind           string     `json:"kind" example:"booking_cancelled_by_restaurant"`
	BookingID      uuid.UUID  `json:"booking_id"`
	RestaurantID   uuid.UUID  `json:"restaurant_id"`
	RestaurantName string     `json:"restaurant_name"`
	CustomerName   string     `json:"customer_name"`
	BusinessDate   string     `json:"business_date" example:"2026-10-10"`
	StartAt        time.Time  `json:"start_at"`
	EndAt          time.Time  `json:"end_at"`
	PartySize      int        `json:"party_size"`
	Reason         string     `json:"reason"`
	ReadAt         *time.Time `json:"read_at"`
	CreatedAt      time.Time  `json:"created_at"`
}

type ListResponse struct {
	Items       []NotificationResponse `json:"items"`
	UnreadCount int64                  `json:"unread_count"`
}

func NewNotificationResponse(n Notification) NotificationResponse {
	return NotificationResponse{
		ID: n.ID, Kind: n.Kind, BookingID: n.BookingID, RestaurantID: n.RestaurantID,
		RestaurantName: n.RestaurantName, CustomerName: n.CustomerName,
		BusinessDate: n.BusinessDate.Format("2006-01-02"),
		StartAt:      n.StartAt.In(bangkok), EndAt: n.EndAt.In(bangkok),
		PartySize: n.PartySize, Reason: n.Reason, ReadAt: n.ReadAt, CreatedAt: n.CreatedAt.In(bangkok),
	}
}
```

`handler.go`:
```go
package notification

import (
	"context"
	"errors"
	"net/http"

	"jongyoung/internal/httputil"
	"jongyoung/internal/reqctx"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Service interface {
	List(ctx context.Context, userID uuid.UUID) ([]Notification, int64, error)
	MarkRead(ctx context.Context, userID, id uuid.UUID) error
	MarkAllRead(ctx context.Context, userID uuid.UUID) error
}

type handler struct {
	service Service
}

func NewHandler(service Service) *handler {
	return &handler{service: service}
}

// ListMine godoc
//
//	@Summary	แจ้งเตือนล่าสุดของฉัน + จำนวนที่ยังไม่อ่าน
//	@ID			listMyNotifications
//	@Tags		me
//	@Security	BearerAuth
//	@Produce	json
//	@Success	200	{object}	ListResponse
//	@Failure	401	{object}	httputil.ErrorResponse
//	@Router		/me/notifications [get]
func (h *handler) ListMine(c *gin.Context) {
	userID, ok := reqctx.UserID(c.Request.Context())
	if !ok {
		httputil.Unauthorized(c, "ต้องเข้าสู่ระบบ")
		return
	}
	list, unread, err := h.service.List(c.Request.Context(), userID)
	if err != nil {
		httputil.Internal(c, err)
		return
	}
	resp := ListResponse{Items: make([]NotificationResponse, len(list)), UnreadCount: unread}
	for i, n := range list {
		resp.Items[i] = NewNotificationResponse(n)
	}
	c.JSON(http.StatusOK, resp)
}

// MarkRead godoc
//
//	@Summary	กดอ่านแจ้งเตือนหนึ่งรายการ (ของคนอื่น → 404)
//	@ID			markNotificationRead
//	@Tags		me
//	@Security	BearerAuth
//	@Param		id	path	string	true	"notification id"
//	@Success	204
//	@Failure	404	{object}	httputil.ErrorResponse
//	@Router		/me/notifications/{id}/read [put]
func (h *handler) MarkRead(c *gin.Context) {
	id, ok := httputil.PathID(c, "id")
	if !ok {
		return
	}
	userID, ok := reqctx.UserID(c.Request.Context())
	if !ok {
		httputil.Unauthorized(c, "ต้องเข้าสู่ระบบ")
		return
	}
	err := h.service.MarkRead(c.Request.Context(), userID, id)
	if errors.Is(err, ErrNotFound) {
		httputil.NotFound(c, "ไม่พบการแจ้งเตือน")
		return
	}
	if err != nil {
		httputil.Internal(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// MarkAllRead godoc
//
//	@Summary	กดอ่านแจ้งเตือนทั้งหมดของฉัน
//	@ID			markAllNotificationsRead
//	@Tags		me
//	@Security	BearerAuth
//	@Success	204
//	@Router		/me/notifications/read-all [put]
func (h *handler) MarkAllRead(c *gin.Context) {
	userID, ok := reqctx.UserID(c.Request.Context())
	if !ok {
		httputil.Unauthorized(c, "ต้องเข้าสู่ระบบ")
		return
	}
	if err := h.service.MarkAllRead(c.Request.Context(), userID); err != nil {
		httputil.Internal(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
```

`.mockery.yml` — ใน `packages:` เพิ่ม:
```yaml
  jongyoung/internal/notification:
    config:
      all: true
```
Run: `cd api && mockery 2>&1 | tail -1 && go build ./...`

- [ ] **Step 3: เขียนเทสต์ service + handler**

`service_test.go`:
```go
package notification

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestServiceMarkRead(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 10, 12, 0, 0, 0, bangkok)
	user, id := uuid.New(), uuid.New()

	repo := NewMockRepository(t)
	repo.EXPECT().MarkRead(ctx, user, id, now).Return(false, nil)
	err := NewService(repo, func() time.Time { return now }).MarkRead(ctx, user, id)
	assert.ErrorIs(t, err, ErrNotFound, "ไม่มีแถวถูกแก้ = ไม่มีหรือเป็นของคนอื่น")
}
```

`handler_test.go`:
```go
package notification

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"jongyoung/internal/reqctx"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// router จริงที่มีทั้ง /:id/read และ /read-all — ยืนยันว่าสอง route นี้อยู่ร่วมกันได้ใน Gin
func newTestRouter(svc Service, userID uuid.UUID) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	auth := func(c *gin.Context) {
		if userID != uuid.Nil {
			c.Request = c.Request.WithContext(reqctx.WithUserID(c.Request.Context(), userID)) // แทน middleware.JWT
		}
	}
	h := NewHandler(svc)
	r.GET("/me/notifications", auth, h.ListMine)
	r.PUT("/me/notifications/read-all", auth, h.MarkAllRead)
	r.PUT("/me/notifications/:id/read", auth, h.MarkRead)
	return r
}

func TestHandler(t *testing.T) {
	t.Run("ไม่มีผู้ใช้ใน context → 401", func(t *testing.T) {
		w := httptest.NewRecorder()
		newTestRouter(NewMockService(t), uuid.Nil).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/me/notifications", nil))
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("กดอ่านของคนอื่น → 404", func(t *testing.T) {
		svc := NewMockService(t)
		svc.EXPECT().MarkRead(mock.Anything, mock.Anything, mock.Anything).Return(ErrNotFound)
		w := httptest.NewRecorder()
		newTestRouter(svc, uuid.New()).ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/me/notifications/"+uuid.NewString()+"/read", nil))
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("read-all ไม่ถูกตีความเป็น :id → 204", func(t *testing.T) {
		svc := NewMockService(t)
		svc.EXPECT().MarkAllRead(mock.Anything, mock.Anything).Return(nil)
		w := httptest.NewRecorder()
		newTestRouter(svc, uuid.New()).ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/me/notifications/read-all", nil))
		assert.Equal(t, http.StatusNoContent, w.Code)
	})
}
```

- [ ] **Step 4: รันเทสต์** (เขียนเทสต์หลังโค้ดใน task นี้เพราะไฟล์ใหม่ทั้งชุด — ให้ลองลบบรรทัด `if !ok { return ErrNotFound }` ใน service ชั่วคราว รันแล้วเห็น `TestServiceMarkRead` FAIL จากนั้นใส่คืน)

Run: `cd api && go test ./internal/notification/ -v 2>&1 | grep -E "^(--- |ok|FAIL)"`
Expected: ทุก test `--- PASS`, `ok`

- [ ] **Step 5: Route** — `api/cmd/api/router.go`

import `"jongyoung/internal/notification"`; ในส่วน DI:
```go
	notificationHandler := notification.NewHandler(notification.NewService(notification.NewRepository(db), time.Now))
```
ต่อจาก `v1.GET("/me/bookings", auth, bookingHandler.ListMine)`:
```go
	v1.GET("/me/notifications", auth, notificationHandler.ListMine)
	v1.PUT("/me/notifications/read-all", auth, notificationHandler.MarkAllRead)
	v1.PUT("/me/notifications/:id/read", auth, notificationHandler.MarkRead)
```

Run: `cd api && go build ./... && go vet ./... && go test ./... 2>&1 | grep -v "no test files"`
Expected: build ผ่าน, ทุก package `ok`

- [ ] **Step 6: Commit**

```bash
git add api/.mockery.yml api/internal/notification api/cmd/api/router.go
git commit -m "เพิ่ม API แจ้งเตือน: รายการของฉัน, กดอ่าน (ของคนอื่นได้ 404), อ่านทั้งหมด

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 5: ปิดร้านชั่วคราว (restaurant) + ยกเลิกโดยร้าน + แจ้งลูกค้า

**Files:**
- Modify: `api/internal/restaurant/model.go`, `dto.go`, `error.go`, `repository.go`, `service.go`, `handler.go`, `mocks_test.go` (mockery), `api/cmd/api/router.go`
- Test: `api/internal/restaurant/dto_test.go`, `service_test.go`, `repository_integration_test.go`, สร้าง `handler_test.go`

**Interfaces:**
- Consumes: `booking.Closure`, `Hours.Span/Days`, `booking.BusinessDateOf`, `booking.CancelledByRestaurant`, `booking.MaxAdvance` (Task 1, 3); `notification.Draft/Insert/KindCancelledByRestaurant` (Task 2)
- Produces:
  - `POST /restaurants/:id/closures` body `{date,start_time,end_time | from_date,to_date, reason, confirm_booking_ids}` → 201 `ClosureResponse{id,start_at,end_at,reason}` / 409 `CLOSURE_AFFECTS_BOOKINGS details.bookings[{id,code,customer_name,start_at,end_at,party_size}]` / 400 `INVALID_CLOSURE` / 403
  - `GET /restaurants/:id/closures` → `ClosureResponse[]` (ยังไม่จบ)
  - `DELETE /restaurants/:id/closures/:closureId` → 204 / 404
  - Repository ใหม่: `UpcomingBookings`, `AffectedBookings`, `CancelByRestaurant`, `Notify`, `CreateClosure`, `ClosuresBetween`, `UpcomingClosures`, `DeleteClosure` (ลบ `CancelFutureBookings`)

- [ ] **Step 1: model / error / dto**

`model.go` ต่อท้าย:
```go
// AffectedBooking = การจองที่ยังไม่เริ่มซึ่งจะถูกยกเลิก (แสดงให้เจ้าของร้านเห็นก่อนยืนยันปิดร้าน)
type AffectedBooking struct {
	booking.Booking
	CustomerName string
}

// ClosureInput = คำขอปิดร้านที่ parse แล้ว: Partial = บางช่วงของวันทำการ Date, ไม่งั้น = ทั้งวัน FromDate–ToDate
type ClosureInput struct {
	Partial                bool
	Date, FromDate, ToDate time.Time
	StartMinute, EndMinute int
	Reason                 string
	ConfirmBookingIDs      []uuid.UUID // รายการที่เจ้าของร้านเห็นแล้วยืนยันให้ยกเลิก
}
```

`error.go`:
```go
package restaurant

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound        = errors.New("restaurant not found")
	ErrNotOwner        = errors.New("not the owner of this restaurant")
	ErrImageRequired   = errors.New("restaurant must have at least one image")
	ErrImageNotFound   = errors.New("image not found")
	ErrInvalidClosure  = errors.New("ช่วงปิดไม่ถูกต้อง: ระบุ date + start_time + end_time หรือ from_date + to_date, ห้ามย้อนหลัง, ยาวไม่เกิน 90 วัน และต้องมีเหตุผล (ไม่เกิน 200 ตัวอักษร)") // → 400 INVALID_CLOSURE
	ErrClosureNotFound = errors.New("closure not found")
)

// AffectsBookingsError → 409 CLOSURE_AFFECTS_BOOKINGS: มีการจองที่จะถูกยกเลิก ต้องยืนยันด้วยรายการนี้ก่อน
type AffectsBookingsError struct {
	Bookings []AffectedBooking
}

func (e *AffectsBookingsError) Error() string {
	return fmt.Sprintf("CLOSURE_AFFECTS_BOOKINGS: %d bookings", len(e.Bookings))
}
```

`dto.go` ต่อท้าย (เพิ่ม import `"unicode/utf8"`):
```go
// ClosureRequest: ปิดบางช่วงของวันทำการ (date + start_time + end_time) หรือทั้งวัน/หลายวัน (from_date + to_date) อย่างใดอย่างหนึ่ง
// ส่งครั้งแรกไม่ต้องมี confirm_booking_ids — ถ้ามีการจองทับ API ตอบ 409 พร้อมรายการ แล้วส่งซ้ำพร้อม id ที่เห็น
type ClosureRequest struct {
	Date              string      `json:"date" example:"2026-10-15"`
	StartTime         string      `json:"start_time" example:"18:00"`
	EndTime           string      `json:"end_time" example:"20:00"`
	FromDate          string      `json:"from_date" example:"2026-10-20"`
	ToDate            string      `json:"to_date" example:"2026-10-22"`
	Reason            string      `json:"reason" example:"ไฟดับทั้งซอย"`
	ConfirmBookingIDs []uuid.UUID `json:"confirm_booking_ids"`
}

// ToInput ตรวจรูปแบบ (ช่วงเวลาจริงคำนวณใน service เพราะต้องรู้เวลาเปิดของร้าน)
func (req ClosureRequest) ToInput() (ClosureInput, error) {
	reason := strings.TrimSpace(req.Reason)
	if reason == "" || utf8.RuneCountInString(reason) > 200 {
		return ClosureInput{}, ErrInvalidClosure
	}
	in := ClosureInput{Reason: reason, ConfirmBookingIDs: req.ConfirmBookingIDs}
	partial := req.Date != "" || req.StartTime != "" || req.EndTime != ""
	days := req.FromDate != "" || req.ToDate != ""
	var err error
	switch {
	case partial && !days:
		in.Partial = true
		if in.Date, err = booking.ParseDate(req.Date); err != nil {
			return ClosureInput{}, ErrInvalidClosure
		}
		if in.StartMinute, err = ParseClock(req.StartTime); err != nil {
			return ClosureInput{}, ErrInvalidClosure
		}
		if in.EndMinute, err = ParseClock(req.EndTime); err != nil || in.EndMinute == in.StartMinute {
			return ClosureInput{}, ErrInvalidClosure
		}
	case days && !partial:
		if in.FromDate, err = booking.ParseDate(req.FromDate); err != nil {
			return ClosureInput{}, ErrInvalidClosure
		}
		if in.ToDate, err = booking.ParseDate(req.ToDate); err != nil || in.ToDate.Before(in.FromDate) {
			return ClosureInput{}, ErrInvalidClosure
		}
	default: // ผสมสองแบบ หรือไม่ส่งช่วงเวลาเลย
		return ClosureInput{}, ErrInvalidClosure
	}
	return in, nil
}

type ClosureResponse struct {
	ID      uuid.UUID `json:"id"`
	StartAt time.Time `json:"start_at"`
	EndAt   time.Time `json:"end_at"`
	Reason  string    `json:"reason"`
}

func newClosure(c booking.Closure) ClosureResponse {
	return ClosureResponse{ID: c.ID, StartAt: c.StartAt.In(booking.Bangkok), EndAt: c.EndAt.In(booking.Bangkok), Reason: c.Reason}
}

type AffectedBookingResponse struct {
	ID           uuid.UUID `json:"id"`
	Code         string    `json:"code" example:"JY-7F3K2A"`
	CustomerName string    `json:"customer_name"`
	StartAt      time.Time `json:"start_at"`
	EndAt        time.Time `json:"end_at"`
	PartySize    int       `json:"party_size"`
}

func newAffected(list []AffectedBooking) []AffectedBookingResponse {
	out := make([]AffectedBookingResponse, len(list))
	for i, b := range list {
		out[i] = AffectedBookingResponse{ID: b.ID, Code: booking.Code(b.ID), CustomerName: b.CustomerName,
			StartAt: b.StartAt.In(booking.Bangkok), EndAt: b.EndAt.In(booking.Bangkok), PartySize: b.PartySize}
	}
	return out
}
```

- [ ] **Step 2: เทสต์ dto** — ต่อท้าย `dto_test.go`
```go
func TestClosureRequestToInput(t *testing.T) {
	in, err := ClosureRequest{Date: "2026-10-10", StartTime: "18:00", EndTime: "20:00", Reason: " ไฟดับ "}.ToInput()
	require.NoError(t, err)
	assert.True(t, in.Partial)
	assert.Equal(t, 18*60, in.StartMinute)
	assert.Equal(t, "ไฟดับ", in.Reason)

	in, err = ClosureRequest{FromDate: "2026-10-20", ToDate: "2026-10-22", Reason: "หยุดยาว"}.ToInput()
	require.NoError(t, err)
	assert.False(t, in.Partial)

	for name, bad := range map[string]ClosureRequest{
		"ไม่มีเหตุผล":         {Date: "2026-10-10", StartTime: "18:00", EndTime: "20:00"},
		"เหตุผลยาวเกิน":       {Date: "2026-10-10", StartTime: "18:00", EndTime: "20:00", Reason: strings.Repeat("ก", 201)},
		"ผสมสองแบบ":           {Date: "2026-10-10", StartTime: "18:00", EndTime: "20:00", FromDate: "2026-10-10", ToDate: "2026-10-10", Reason: "x"},
		"ไม่ระบุช่วง":         {Reason: "x"},
		"ไม่ลง :30":           {Date: "2026-10-10", StartTime: "18:15", EndTime: "20:00", Reason: "x"},
		"เริ่มเท่ากับจบ":      {Date: "2026-10-10", StartTime: "18:00", EndTime: "18:00", Reason: "x"},
		"ถึงวันก่อนจากวัน":    {FromDate: "2026-10-22", ToDate: "2026-10-20", Reason: "x"},
		"ขาดเวลาจบ":           {Date: "2026-10-10", StartTime: "18:00", Reason: "x"},
	} {
		_, err := bad.ToInput()
		assert.ErrorIs(t, err, ErrInvalidClosure, name)
	}
}
```

- [ ] **Step 3: Repository**

ใน `service.go` `Repository` interface: ลบ `CancelFutureBookings(...)` แล้วเพิ่ม:
```go
	UpcomingBookings(ctx context.Context, restaurantID uuid.UUID, now time.Time) ([]AffectedBooking, error)
	AffectedBookings(ctx context.Context, restaurantID uuid.UUID, start, end, now time.Time) ([]AffectedBooking, error)
	CancelByRestaurant(ctx context.Context, ids []uuid.UUID, reason string, now time.Time) error
	Notify(ctx context.Context, d notification.Draft) error
	CreateClosure(ctx context.Context, c *booking.Closure) error
	ClosuresBetween(ctx context.Context, restaurantIDs []uuid.UUID, from, to time.Time) ([]booking.Closure, error)
	UpcomingClosures(ctx context.Context, restaurantID uuid.UUID, now time.Time) ([]booking.Closure, error)
	DeleteClosure(ctx context.Context, restaurantID, closureID uuid.UUID) (bool, error)
```

`repository.go` — แทนฟังก์ชัน `CancelFutureBookings` ทั้งฟังก์ชันด้วย (import `"jongyoung/internal/notification"`):
```go
// upcomingQuery = การจอง active ที่ยังไม่เริ่ม + ชื่อลูกค้า — คนที่นั่งอยู่ในร้านแล้ว (เริ่มไปแล้ว) ไม่ถูกแตะ
func (r *repository) upcomingQuery(ctx context.Context, restaurantID uuid.UUID, now time.Time) *gorm.DB {
	return r.db.WithContext(ctx).Table("bookings b").
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
```

Run: `cd api && mockery 2>&1 | tail -1`

- [ ] **Step 4: เทสต์ service** — `service_test.go`

เพิ่ม import `"jongyoung/internal/notification"`

(a) `TestNextAvailable`: ต่อจากทุกบรรทัด `repo.EXPECT().BookingsBetween(ctx, []uuid.UUID{id}, mock.Anything, mock.Anything)…` ใส่
```go
		repo.EXPECT().ClosuresBetween(ctx, []uuid.UUID{id}, mock.Anything, mock.Anything).Return(nil, nil)
```
Run: `cd api && perl -pi -e 's/^(\t\trepo\.EXPECT\(\)\.BookingsBetween\(ctx, \[\]uuid\.UUID\{id\}, mock\.Anything, mock\.Anything\).*\n)/$1\t\trepo.EXPECT().ClosuresBetween(ctx, []uuid.UUID{id}, mock.Anything, mock.Anything).Return(nil, nil)\n/' internal/restaurant/service_test.go && grep -c "ClosuresBetween(ctx, \[\]uuid.UUID{id}" internal/restaurant/service_test.go`
Expected: `3`

(b) `TestDelete` subtest แรก: แทน
```go
		repo.EXPECT().CancelFutureBookings(ctx, id, fixedNow).Return(3, nil)
```
ด้วย
```go
		customer, bid := uuid.New(), uuid.New()
		repo.EXPECT().UpcomingBookings(ctx, id, fixedNow).Return([]AffectedBooking{
			{Booking: booking.Booking{ID: bid, UserID: customer, StartAt: time.Date(2026, 10, 10, 19, 0, 0, 0, booking.Bangkok)}},
		}, nil)
		repo.EXPECT().CancelByRestaurant(ctx, []uuid.UUID{bid}, "ร้านปิดให้บริการ", fixedNow).Return(nil)
		repo.EXPECT().Notify(ctx, notification.Draft{Recipient: customer, Kind: notification.KindCancelledByRestaurant,
			BookingID: bid, BusinessDate: "2026-10-10", Reason: "ร้านปิดให้บริการ"}).Return(nil)
```
(ร้านใน subtest นี้ `OpenMinute/CloseMinute` เป็น 0 = 24 ชม. → วันทำการของ 19:00 วันที่ 10 คือ 2026-10-10)

(c) เทสต์ใหม่ท้ายไฟล์:
```go
func TestCreateClosure(t *testing.T) {
	ctx := context.Background()
	owner, id, customer := uuid.New(), uuid.New(), uuid.New()
	rest := Restaurant{ID: id, OwnerID: owner, Seats: 10, OpenMinute: 11 * 60, CloseMinute: 22 * 60}
	bkk := func(h, m int) time.Time { return time.Date(2026, 10, 10, h, m, 0, 0, booking.Bangkok) }
	// fixedNow = 10 ต.ค. 09:00 → ปิด 10 ต.ค. 18:00–20:00
	in := ClosureInput{Partial: true, Date: bkk(0, 0), StartMinute: 18 * 60, EndMinute: 20 * 60, Reason: "ไฟดับ"}
	start, end := bkk(18, 0), bkk(20, 0)
	hit := AffectedBooking{Booking: booking.Booking{ID: uuid.New(), UserID: customer, StartAt: bkk(19, 0), EndAt: bkk(20, 0)}, CustomerName: "มะลิ"}
	isClosure := mock.MatchedBy(func(c *booking.Closure) bool {
		return c.RestaurantID == id && c.StartAt.Equal(start) && c.EndAt.Equal(end) && c.Reason == "ไฟดับ"
	})

	t.Run("ไม่มีการจองทับ → บันทึกเลย", func(t *testing.T) {
		repo := NewMockRepository(t)
		expectTx(repo)
		repo.EXPECT().LockByID(ctx, id).Return(rest, nil)
		repo.EXPECT().AffectedBookings(ctx, id, start, end, fixedNow).Return(nil, nil)
		repo.EXPECT().CreateClosure(ctx, isClosure).Return(nil)
		_, err := newServiceWith(repo).CreateClosure(ctx, owner, id, in)
		assert.NoError(t, err)
	})

	t.Run("มีการจองทับแต่ยังไม่ยืนยัน → AffectsBookingsError พร้อมรายการ ไม่บันทึกอะไร", func(t *testing.T) {
		repo := NewMockRepository(t)
		expectTx(repo)
		repo.EXPECT().LockByID(ctx, id).Return(rest, nil)
		repo.EXPECT().AffectedBookings(ctx, id, start, end, fixedNow).Return([]AffectedBooking{hit}, nil)
		_, err := newServiceWith(repo).CreateClosure(ctx, owner, id, in)
		var affects *AffectsBookingsError
		require.True(t, errors.As(err, &affects))
		assert.Equal(t, []AffectedBooking{hit}, affects.Bookings)
	})

	t.Run("ยืนยันตรงรายการ → บันทึก + ยกเลิก + แจ้งลูกค้า", func(t *testing.T) {
		repo := NewMockRepository(t)
		expectTx(repo)
		repo.EXPECT().LockByID(ctx, id).Return(rest, nil)
		repo.EXPECT().AffectedBookings(ctx, id, start, end, fixedNow).Return([]AffectedBooking{hit}, nil)
		repo.EXPECT().CreateClosure(ctx, isClosure).Return(nil)
		repo.EXPECT().CancelByRestaurant(ctx, []uuid.UUID{hit.ID}, "ไฟดับ", fixedNow).Return(nil)
		repo.EXPECT().Notify(ctx, notification.Draft{Recipient: customer, Kind: notification.KindCancelledByRestaurant,
			BookingID: hit.ID, BusinessDate: "2026-10-10", Reason: "ไฟดับ"}).Return(nil)
		confirmed := in
		confirmed.ConfirmBookingIDs = []uuid.UUID{hit.ID}
		_, err := newServiceWith(repo).CreateClosure(ctx, owner, id, confirmed)
		assert.NoError(t, err)
	})

	t.Run("ยืนยันรายการเก่า แต่มีการจองใหม่แทรก → 409 ใหม่ ไม่ยกเลิกอะไร", func(t *testing.T) {
		newcomer := AffectedBooking{Booking: booking.Booking{ID: uuid.New(), UserID: uuid.New(), StartAt: bkk(18, 30), EndAt: bkk(19, 30)}}
		repo := NewMockRepository(t)
		expectTx(repo)
		repo.EXPECT().LockByID(ctx, id).Return(rest, nil)
		repo.EXPECT().AffectedBookings(ctx, id, start, end, fixedNow).Return([]AffectedBooking{newcomer, hit}, nil)
		confirmed := in
		confirmed.ConfirmBookingIDs = []uuid.UUID{hit.ID}
		_, err := newServiceWith(repo).CreateClosure(ctx, owner, id, confirmed)
		var affects *AffectsBookingsError
		require.True(t, errors.As(err, &affects))
		assert.Len(t, affects.Bookings, 2)
	})

	t.Run("การจองของเจ้าของร้านเอง → ยกเลิก แต่ไม่แจ้งตัวเอง", func(t *testing.T) {
		mine := AffectedBooking{Booking: booking.Booking{ID: uuid.New(), UserID: owner, StartAt: bkk(19, 0), EndAt: bkk(20, 0)}}
		repo := NewMockRepository(t)
		expectTx(repo)
		repo.EXPECT().LockByID(ctx, id).Return(rest, nil)
		repo.EXPECT().AffectedBookings(ctx, id, start, end, fixedNow).Return([]AffectedBooking{mine}, nil)
		repo.EXPECT().CreateClosure(ctx, isClosure).Return(nil)
		repo.EXPECT().CancelByRestaurant(ctx, []uuid.UUID{mine.ID}, "ไฟดับ", fixedNow).Return(nil)
		confirmed := in
		confirmed.ConfirmBookingIDs = []uuid.UUID{mine.ID}
		_, err := newServiceWith(repo).CreateClosure(ctx, owner, id, confirmed)
		assert.NoError(t, err)
	})

	t.Run("เริ่มก่อนตอนนี้ (08:00 ขณะที่ตอนนี้ 09:00) → ErrInvalidClosure", func(t *testing.T) {
		repo := NewMockRepository(t)
		expectTx(repo)
		repo.EXPECT().LockByID(ctx, id).Return(rest, nil)
		past := in
		past.StartMinute = 8 * 60
		_, err := newServiceWith(repo).CreateClosure(ctx, owner, id, past)
		assert.ErrorIs(t, err, ErrInvalidClosure)
	})

	t.Run("ไม่ใช่เจ้าของ → ErrNotOwner", func(t *testing.T) {
		repo := NewMockRepository(t)
		expectTx(repo)
		repo.EXPECT().LockByID(ctx, id).Return(rest, nil)
		_, err := newServiceWith(repo).CreateClosure(ctx, uuid.New(), id, in)
		assert.ErrorIs(t, err, ErrNotOwner)
	})
}
```

- [ ] **Step 5: เทสต์ integration** — `repository_integration_test.go`

แทน subtest `"CancelFutureBookings ยกเลิกเฉพาะที่ยังไม่เริ่ม"` ทั้งบล็อกด้วย:
```go
	t.Run("UpcomingBookings เฉพาะที่ยังไม่เริ่ม", func(t *testing.T) {
		list, err := repo.UpcomingBookings(ctx, r.ID, now)
		require.NoError(t, err)
		require.Len(t, list, 1)
		assert.Equal(t, future, list[0].ID)
		assert.NotEqual(t, past, list[0].ID, "การจองที่ผ่านไปแล้วต้องไม่ถูกยกเลิก (เก็บประวัติ)")
	})
```
ต่อท้ายไฟล์ (import เพิ่ม `"errors"`):
```go
func TestClosures(t *testing.T) {
	db := testdb.New(t)
	repo := NewRepository(db)
	ctx := context.Background()
	owner, customer := createOwner(t, db), createOwner(t, db)
	r := createRestaurant(t, repo, owner, "ร้านปิดชั่วคราว", 0, 0)
	at := func(h int) time.Time { return time.Date(2026, 10, 10, h, 0, 0, 0, booking.Bangkok) }
	insert := func(start, end time.Time) uuid.UUID {
		var id uuid.UUID
		require.NoError(t, db.Raw(`INSERT INTO bookings (restaurant_id, user_id, party_size, start_at, end_at) VALUES (?, ?, 2, ?, ?) RETURNING id`,
			r.ID, customer, start, end).Row().Scan(&id))
		return id
	}
	insert(at(14), at(16)) // เริ่มไปแล้วตอน 15:00 — ลูกค้านั่งอยู่ในร้าน
	later := insert(at(18), at(19))
	insert(at(20), at(21)) // ไม่ทับช่วงปิด 14:00–20:00
	now := at(15)

	t.Run("AffectedBookings ไม่รวมที่เริ่มแล้ว และไม่รวมที่ไม่ทับ", func(t *testing.T) {
		list, err := repo.AffectedBookings(ctx, r.ID, at(14), at(20), now)
		require.NoError(t, err)
		require.Len(t, list, 1)
		assert.Equal(t, later, list[0].ID)
		assert.Equal(t, "owner", list[0].CustomerName)
	})

	t.Run("CRUD ช่วงปิด + ลบของร้านอื่นไม่ได้", func(t *testing.T) {
		c := booking.Closure{RestaurantID: r.ID, StartAt: at(18), EndAt: at(20), Reason: "ไฟดับ"}
		require.NoError(t, repo.CreateClosure(ctx, &c))
		list, err := repo.ClosuresBetween(ctx, []uuid.UUID{r.ID}, at(19), at(21))
		require.NoError(t, err)
		require.Len(t, list, 1)
		upcoming, err := repo.UpcomingClosures(ctx, r.ID, now)
		require.NoError(t, err)
		require.Len(t, upcoming, 1)
		ok, err := repo.DeleteClosure(ctx, uuid.New(), c.ID)
		require.NoError(t, err)
		assert.False(t, ok, "id ร้านไม่ตรง → ลบไม่ได้")
		ok, err = repo.DeleteClosure(ctx, r.ID, c.ID)
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("service: ปิดร้าน → 409 → ยืนยัน → ช่วงปิด + ยกเลิก + แจ้งลูกค้า ในทรานแซกชันเดียว", func(t *testing.T) {
		svc := NewService(repo, func() time.Time { return now })
		in := ClosureInput{Partial: true, Date: at(0), StartMinute: 18 * 60, EndMinute: 20 * 60, Reason: "ไฟดับ"}
		_, err := svc.CreateClosure(ctx, owner, r.ID, in)
		var affects *AffectsBookingsError
		require.True(t, errors.As(err, &affects), "err = %v", err)
		require.Len(t, affects.Bookings, 1)

		in.ConfirmBookingIDs = []uuid.UUID{affects.Bookings[0].ID}
		_, err = svc.CreateClosure(ctx, owner, r.ID, in)
		require.NoError(t, err)

		var got booking.Booking
		require.NoError(t, db.First(&got, "id = ?", later).Error)
		assert.Equal(t, booking.StatusCancelled, got.Status)
		require.NotNil(t, got.CancelledBy)
		assert.Equal(t, booking.CancelledByRestaurant, *got.CancelledBy)
		assert.Equal(t, "ไฟดับ", got.CancelReason)
		var n int64
		require.NoError(t, db.Table("notifications").Where("user_id = ? AND booking_id = ? AND kind = 'booking_cancelled_by_restaurant'", customer, later).Count(&n).Error)
		assert.EqualValues(t, 1, n)
	})
}
```

- [ ] **Step 6: รันให้เห็นว่าไม่ผ่าน**

Run: `cd api && go test ./internal/restaurant/ 2>&1 | head -5`
Expected: `newServiceWith(repo).CreateClosure undefined` (หรือ compile error ที่ `CancelFutureBookings` ใน service.go)

- [ ] **Step 7: Service** — `service.go` (import `"jongyoung/internal/notification"`)

ค่าคงที่ต่อจาก `nextAvailableDays`:
```go
// maxClosure = ปิดชั่วคราวได้ครั้งละไม่เกิน 90 วัน (เท่าระยะจองล่วงหน้า — ไกลกว่านี้ไม่มีการจองให้กระทบ)
const maxClosure = booking.MaxAdvance

// deletedReason = เหตุผลที่ลูกค้าเห็นเมื่อร้านถูกลบ
const deletedReason = "ร้านปิดให้บริการ"
```

แทน `Delete` ทั้งฟังก์ชัน:
```go
// Delete = soft delete + ยกเลิก booking ที่ยังไม่เริ่มและแจ้งลูกค้า ในทรานแซกชันเดียวกัน
// เลือกแบบนี้แทนการห้ามลบ: เจ้าของปิดร้านได้จริง และลูกค้ารู้ว่าร้านยกเลิกให้ (ไม่ใช่มาแล้วเจอร้านปิด)
func (s *service) Delete(ctx context.Context, userID, id uuid.UUID) error {
	return s.repository.Transaction(ctx, func(tx Repository) error {
		rest, err := tx.LockByID(ctx, id)
		if err != nil {
			return err
		}
		if rest.OwnerID != userID {
			return ErrNotOwner
		}
		upcoming, err := tx.UpcomingBookings(ctx, id, s.now())
		if err != nil {
			return err
		}
		if err := cancelByRestaurant(ctx, tx, rest, upcoming, deletedReason, s.now()); err != nil {
			return err
		}
		return tx.SoftDelete(ctx, id)
	})
}
```

เพิ่มท้ายไฟล์:
```go
// CreateClosure ปิดร้านชั่วคราว — ตรวจและเขียนในทรานแซกชันเดียวที่ล็อกแถวร้าน (คำขอจองที่มาพร้อมกันต่อคิวที่ล็อกเดียวกัน)
// ถ้ามีการจองที่ยังไม่เริ่มทับช่วงปิด ต้องยืนยันด้วยรายการ id ที่ตรงกันพอดี ไม่งั้นคืน AffectsBookingsError พร้อมรายการล่าสุด
// → ไม่มีทางยกเลิกการจองที่เจ้าของร้านยังไม่เคยเห็น (เช่นมีคนจองแทรกระหว่างที่ดูรายการอยู่)
func (s *service) CreateClosure(ctx context.Context, userID, id uuid.UUID, in ClosureInput) (booking.Closure, error) {
	var created booking.Closure
	err := s.repository.Transaction(ctx, func(tx Repository) error {
		rest, err := tx.LockByID(ctx, id)
		if err != nil {
			return err
		}
		if rest.OwnerID != userID {
			return ErrNotOwner
		}
		now := s.now()
		start, end := closureRange(rest.Hours(), in)
		// ปัดตอนนี้ลงเป็น :00/:30 — "ปิดตอนนี้" ตอน 15:10 เริ่มได้ที่ 15:00 (เวลาไทยต่างจาก UTC เป็นชั่วโมงเต็ม จึงปัดตรงกัน)
		if !end.After(start) || end.Sub(start) > maxClosure || start.Before(now.Truncate(30*time.Minute)) {
			return ErrInvalidClosure
		}
		affected, err := tx.AffectedBookings(ctx, id, start, end, now)
		if err != nil {
			return err
		}
		if len(affected) > 0 && !sameIDs(affected, in.ConfirmBookingIDs) {
			return &AffectsBookingsError{Bookings: affected}
		}
		created = booking.Closure{RestaurantID: id, StartAt: start, EndAt: end, Reason: in.Reason}
		if err := tx.CreateClosure(ctx, &created); err != nil {
			return err
		}
		return cancelByRestaurant(ctx, tx, rest, affected, in.Reason, now)
	})
	return created, err
}

// closureRange แปลงคำขอเป็นช่วงเวลาจริงผ่าน businessday.go (รองรับร้านข้ามคืน)
func closureRange(h booking.Hours, in ClosureInput) (time.Time, time.Time) {
	if in.Partial {
		return h.Span(in.Date, in.StartMinute, in.EndMinute)
	}
	return h.Days(in.FromDate, in.ToDate)
}

// sameIDs: การจองที่จะถูกยกเลิกตรงกับที่เจ้าของร้านยืนยันมาพอดีไหม (ไม่สนลำดับ)
func sameIDs(affected []AffectedBooking, confirmed []uuid.UUID) bool {
	if len(affected) != len(confirmed) {
		return false
	}
	want := make(map[uuid.UUID]bool, len(confirmed))
	for _, id := range confirmed {
		want[id] = true
	}
	for _, b := range affected {
		if !want[b.ID] {
			return false
		}
	}
	return true
}

// cancelByRestaurant ยกเลิกการจองด้วยเหตุผลจากร้าน แล้วแจ้งลูกค้าแต่ละราย (เรียกใน tx เดียวกัน)
// ไม่แจ้งเจ้าของร้านที่จองร้านตัวเองไว้ — เขาเป็นคนปิดร้านเอง
func cancelByRestaurant(ctx context.Context, tx Repository, rest Restaurant, list []AffectedBooking, reason string, now time.Time) error {
	if len(list) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, len(list))
	for i, b := range list {
		ids[i] = b.ID
	}
	if err := tx.CancelByRestaurant(ctx, ids, reason, now); err != nil {
		return err
	}
	for _, b := range list {
		if b.UserID == rest.OwnerID {
			continue
		}
		d := notification.Draft{Recipient: b.UserID, Kind: notification.KindCancelledByRestaurant, BookingID: b.ID,
			BusinessDate: booking.BusinessDateOf(b.StartAt, rest.Hours()), Reason: reason}
		if err := tx.Notify(ctx, d); err != nil {
			return err
		}
	}
	return nil
}

// ListClosures = ช่วงปิดที่ยังไม่จบ (สาธารณะ — หน้าร้านแสดงให้ลูกค้าเห็น)
func (s *service) ListClosures(ctx context.Context, id uuid.UUID) ([]booking.Closure, error) {
	return s.repository.UpcomingClosures(ctx, id, s.now())
}

// DeleteClosure = เปิดร้านกลับ — การจองที่ยกเลิกไปแล้วไม่ฟื้น (ลูกค้าได้รับแจ้งแล้วและอาจไปจองที่อื่น)
func (s *service) DeleteClosure(ctx context.Context, userID, id, closureID uuid.UUID) error {
	rest, err := s.repository.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if rest.OwnerID != userID {
		return ErrNotOwner
	}
	ok, err := s.repository.DeleteClosure(ctx, id, closureID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrClosureNotFound
	}
	return nil
}
```

ส่งช่วงปิดให้ slot (แทน `nil` จาก Task 1):
- `List` ต่อจากบล็อก `byRestaurant` ของ booking:
```go
	closures, err := s.repository.ClosuresBetween(ctx, ids, from, to)
	if err != nil {
		return nil, 0, err
	}
	closuresOf := map[uuid.UUID][]booking.Closure{}
	for _, c := range closures {
		closuresOf[c.RestaurantID] = append(closuresOf[c.RestaurantID], c)
	}
```
  และ `items[i].Slots = booking.SlotsAround(r.Hours(), r.Seats, byRestaurant[r.ID], closuresOf[r.ID], *q.Date, q.Minute, now)`
- `Availability` ต่อจากบล็อก `bookings, err :=`:
```go
	closures, err := s.repository.ClosuresBetween(ctx, []uuid.UUID{id}, opensAt, closesAt)
	if err != nil {
		return Restaurant{}, nil, err
	}
	return rest, booking.Slots(rest.Hours(), rest.Seats, bookings, closures, date, s.now()), nil
```
- `NextAvailable` ต่อจากบล็อก `bookings, err :=`:
```go
	closures, err := s.repository.ClosuresBetween(ctx, []uuid.UUID{id}, from, to)
	if err != nil {
		return nil, err
	}
```
  และแทน `nil` สองจุดในลูปด้วย `closures`

- [ ] **Step 8: Handler + route**

`handler.go` — `Service` interface เพิ่ม:
```go
	CreateClosure(ctx context.Context, userID, id uuid.UUID, in ClosureInput) (booking.Closure, error)
	ListClosures(ctx context.Context, id uuid.UUID) ([]booking.Closure, error)
	DeleteClosure(ctx context.Context, userID, id, closureID uuid.UUID) error
```
handler ใหม่ (วางก่อน `fail`):
```go
// CreateClosure godoc
//
//	@Summary	ปิดร้านชั่วคราว (บางช่วงของวัน หรือทั้งวัน/หลายวัน) — ทับการจองต้องยืนยันด้วย confirm_booking_ids
//	@ID			createClosure
//	@Tags		restaurants
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string			true	"restaurant id"
//	@Param		request	body		ClosureRequest	true	"ช่วงปิด"
//	@Success	201		{object}	ClosureResponse
//	@Failure	400		{object}	httputil.ErrorResponse	"INVALID_CLOSURE"
//	@Failure	403		{object}	httputil.ErrorResponse
//	@Failure	409		{object}	httputil.ErrorResponse	"CLOSURE_AFFECTS_BOOKINGS"
//	@Router		/restaurants/{id}/closures [post]
func (h *handler) CreateClosure(c *gin.Context) {
	id, ok := httputil.PathID(c, "id")
	if !ok {
		return
	}
	var req ClosureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.fail(c, ErrInvalidClosure)
		return
	}
	in, err := req.ToInput()
	if err != nil {
		h.fail(c, err)
		return
	}
	userID, _ := reqctx.UserID(c.Request.Context())
	closure, err := h.service.CreateClosure(c.Request.Context(), userID, id, in)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, newClosure(closure))
}

// ListClosures godoc
//
//	@Summary	ช่วงปิดชั่วคราวที่ยังไม่จบ
//	@ID			listClosures
//	@Tags		restaurants
//	@Produce	json
//	@Param		id	path	string	true	"restaurant id"
//	@Success	200	{array}	ClosureResponse
//	@Router		/restaurants/{id}/closures [get]
func (h *handler) ListClosures(c *gin.Context) {
	id, ok := httputil.PathID(c, "id")
	if !ok {
		return
	}
	list, err := h.service.ListClosures(c.Request.Context(), id)
	if err != nil {
		h.fail(c, err)
		return
	}
	out := make([]ClosureResponse, len(list))
	for i, cl := range list {
		out[i] = newClosure(cl)
	}
	c.JSON(http.StatusOK, out)
}

// DeleteClosure godoc
//
//	@Summary	เปิดร้านกลับ (ลบช่วงปิด) — การจองที่ยกเลิกไปแล้วไม่ฟื้น
//	@ID			deleteClosure
//	@Tags		restaurants
//	@Security	BearerAuth
//	@Param		id			path	string	true	"restaurant id"
//	@Param		closureId	path	string	true	"closure id"
//	@Success	204
//	@Router		/restaurants/{id}/closures/{closureId} [delete]
func (h *handler) DeleteClosure(c *gin.Context) {
	id, ok := httputil.PathID(c, "id")
	if !ok {
		return
	}
	closureID, ok := httputil.PathID(c, "closureId")
	if !ok {
		return
	}
	userID, _ := reqctx.UserID(c.Request.Context())
	if err := h.service.DeleteClosure(c.Request.Context(), userID, id, closureID); err != nil {
		h.fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
```
ใน `fail` เพิ่ม `var affects *AffectsBookingsError` และ case ต่อจาก `case errors.Is(err, ErrNotOwner):` block:
```go
	case errors.Is(err, ErrInvalidClosure):
		httputil.Abort(c, http.StatusBadRequest, "INVALID_CLOSURE", ErrInvalidClosure.Error(), nil)
	case errors.Is(err, ErrClosureNotFound):
		httputil.NotFound(c, "ไม่พบช่วงปิด")
	case errors.As(err, &affects):
		httputil.Abort(c, http.StatusConflict, "CLOSURE_AFFECTS_BOOKINGS", "มีการจองที่จะถูกยกเลิก ต้องยืนยันก่อน",
			gin.H{"bookings": newAffected(affects.Bookings)})
```

`router.go` ต่อจาก `rest.DELETE("/:id/images/:imageId", …)`:
```go
	rest.GET("/:id/closures", restaurantHandler.ListClosures)
	rest.POST("/:id/closures", auth, restaurantHandler.CreateClosure)
	rest.DELETE("/:id/closures/:closureId", auth, restaurantHandler.DeleteClosure)
```

Run: `cd api && mockery 2>&1 | tail -1`

`handler_test.go` (ไฟล์ใหม่):
```go
package restaurant

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"jongyoung/internal/booking"
	"jongyoung/internal/reqctx"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func postClosure(t *testing.T, svc Service, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/restaurants/:id/closures", func(c *gin.Context) {
		c.Request = c.Request.WithContext(reqctx.WithUserID(c.Request.Context(), uuid.New())) // แทน middleware.JWT
	}, NewHandler(svc).CreateClosure)
	req := httptest.NewRequest(http.MethodPost, "/restaurants/"+uuid.NewString()+"/closures", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestHandlerCreateClosure(t *testing.T) {
	body := `{"date":"2026-10-10","start_time":"18:00","end_time":"20:00","reason":"ไฟดับ"}`

	t.Run("ไม่มีเหตุผล → 400 INVALID_CLOSURE (ไม่เรียก service)", func(t *testing.T) {
		w := postClosure(t, NewMockService(t), `{"date":"2026-10-10","start_time":"18:00","end_time":"20:00"}`)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "INVALID_CLOSURE")
	})

	t.Run("ไม่ใช่เจ้าของ → 403", func(t *testing.T) {
		svc := NewMockService(t)
		svc.EXPECT().CreateClosure(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(booking.Closure{}, ErrNotOwner)
		assert.Equal(t, http.StatusForbidden, postClosure(t, svc, body).Code)
	})

	t.Run("มีการจองทับ → 409 CLOSURE_AFFECTS_BOOKINGS พร้อมรายการ", func(t *testing.T) {
		svc := NewMockService(t)
		svc.EXPECT().CreateClosure(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(booking.Closure{}, &AffectsBookingsError{Bookings: []AffectedBooking{{CustomerName: "มะลิ"}}})
		w := postClosure(t, svc, body)
		assert.Equal(t, http.StatusConflict, w.Code)
		assert.Contains(t, w.Body.String(), "CLOSURE_AFFECTS_BOOKINGS")
		assert.Contains(t, w.Body.String(), "มะลิ")
	})
}
```

- [ ] **Step 9: รันเทสต์ทั้ง api**

Run: `cd api && go vet ./... && go test ./... 2>&1 | grep -v "no test files"`
Expected: ทุก package `ok`

- [ ] **Step 10: Commit**

```bash
git add api/internal/restaurant api/cmd/api/router.go
git commit -m "ปิดร้านชั่วคราว: ยืนยันด้วยรายการจองที่เห็น, ยกเลิกโดยร้านพร้อมแจ้งลูกค้า, ตารางเวลาว่างข้ามช่วงปิด

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 6: Seed + Swagger

**Files:**
- Modify: `api/cmd/seed/main.go`, `api/docs/*` (สร้างใหม่)

**Interfaces:**
- Consumes: `booking.Hours.Days`, `booking.CancelledByRestaurant/Customer`, `notification.Insert/Draft/Kind*`

- [ ] **Step 1: `book` คืน id** — แทนฟังก์ชัน `book`:
```go
func (s *seeder) book(restaurant, user uuid.UUID, party int, start, end time.Time, status string) uuid.UUID {
	var cancelledAt *time.Time
	var cancelledBy *string
	if status == booking.StatusCancelled {
		t := s.today.AddDate(0, 0, -1)
		by := booking.CancelledByCustomer
		cancelledAt, cancelledBy = &t, &by
	}
	return s.scanID(`INSERT INTO bookings (restaurant_id, user_id, party_size, start_at, end_at, status, cancelled_at, cancelled_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?) RETURNING id`,
		restaurant, user, party, start, end, status, cancelledAt, cancelledBy)
}

// notify เขียนแจ้งเตือนด้วยฟังก์ชันเดียวกับระบบจริง (snapshot จาก DB)
func (s *seeder) notify(recipient uuid.UUID, kind string, bookingID uuid.UUID, businessDate time.Time, reason string) {
	if s.err == nil {
		s.err = notification.Insert(context.Background(), s.tx, notification.Draft{Recipient: recipient, Kind: kind,
			BookingID: bookingID, BusinessDate: businessDate.Format("2006-01-02"), Reason: reason})
	}
}
```
(import `"context"` ถ้ายังไม่มี และ `"jongyoung/internal/notification"`)

- [ ] **Step 2: ข้อมูลตัวอย่าง** — ใน `seed()` แทนสองบรรทัดแรกของ "การจองของ customer1" ด้วยการเก็บ id และต่อท้ายส่วนการจองทั้งหมด (ก่อนบล็อก "คะแนนรวม"):

```go
	seaMidnight := s.book(seafood, customer1, 3, s.at(tomorrow, 23, 30), s.at(tomorrow.AddDate(0, 0, 1), 0, 30), booking.StatusActive)
	sushiDay3 := s.book(sushi, customer1, 2, s.at(tomorrow.AddDate(0, 0, 2), 19, 0), s.at(tomorrow.AddDate(0, 0, 2), 20, 30), booking.StatusActive)
```
และต่อท้าย:
```go
	// ร้านยกเลิกการจองของ customer1 (ร้านตามสั่ง 4 วันข้างหน้า) + แจ้งเตือนที่ยังไม่อ่าน → login แล้วเห็นตัวเลขบนกระดิ่ง
	day4 := s.today.AddDate(0, 0, 4)
	byShop := s.book(tamsang, customer1, 2, s.at(day4, 12, 0), s.at(day4, 13, 0), booking.StatusActive)
	s.exec(`UPDATE bookings SET status = ?, cancelled_at = ?, cancelled_by = ?, cancel_reason = ? WHERE id = ?`,
		booking.StatusCancelled, s.today, booking.CancelledByRestaurant, "ไฟดับทั้งซอย", byShop)
	s.notify(customer1, notification.KindCancelledByRestaurant, byShop, day4, "ไฟดับทั้งซอย")

	// เจ้าของร้านได้รับแจ้งการจองใหม่ (ยังไม่อ่าน)
	s.notify(owner2, notification.KindBookingCreated, sushiDay3, tomorrow.AddDate(0, 0, 2), "")
	s.notify(owner2, notification.KindBookingCreated, seaMidnight, tomorrow, "")

	// บุฟเฟ่ต์ปิดปรับปรุงทั้งวัน 5 วันข้างหน้า (ร้านนี้ไม่มีการจอง active จึงไม่มีการจองถูกยกเลิก)
	day5 := s.today.AddDate(0, 0, 5)
	closeStart, closeEnd := booking.Hours{OpenMinute: 17 * 60, CloseMinute: 23 * 60}.Days(day5, day5)
	s.exec(`INSERT INTO restaurant_closures (restaurant_id, start_at, end_at, reason) VALUES (?, ?, ?, ?)`,
		buffet, closeStart, closeEnd, "ปิดปรับปรุงร้าน")
```

Run: `cd api && go build ./... && go vet ./...`
Expected: ไม่มี output

- [ ] **Step 3: รัน seed กับ DB ชั่วคราว** (ไม่แตะ DB ของผู้ใช้)

Run: `cd api && go test ./... 2>&1 | grep -v "no test files" | tail -8`
Expected: ทุก package `ok` (seed ไม่มีเทสต์ — ตรวจตอน E2E ใน Task 11 ซึ่งรัน `--reset` หลังได้รับอนุญาต)

- [ ] **Step 4: Swagger**

Run: `./dev.sh docs && grep -c "closures\|notifications" api/docs/openapi.yaml`
Expected: ≥ 4

- [ ] **Step 5: Commit**

```bash
git add api/cmd/seed/main.go api/docs
git commit -m "seed: การจองที่ร้านยกเลิก, แจ้งเตือนที่ยังไม่อ่าน, ช่วงปิดปรับปรุงร้านบุฟเฟ่ต์ + swagger ใหม่

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 7: เว็บ — type, hook และฟังก์ชันแสดงผล

**Files:**
- Modify: `web/lib/types.ts`, `web/lib/format.ts`, `web/lib/errors.ts`
- Create: `web/lib/notifications.ts`, `web/lib/notifications.test.ts`, `web/services/notifications.ts`, `web/services/closures.ts`
- Test: `web/lib/format.test.ts`

**Interfaces:**
- Produces:
  - types: `Closure`, `AffectedBooking`, `NotificationKind`, `Notification`, `NotificationList`; `Booking.cancelled_by`, `Booking.cancel_reason`
  - `notificationText(n: Notification): string`, `notificationHref(n: Notification): string`, `bellLabel(unread: number): string`
  - `bookingStatusLabel(b, past: boolean): string`, `gapLabel(from: string, to: string, closures: Closure[]): string`, `closureRangeLabel(c: {start_at,end_at}): string`
  - hooks: `useNotifications()`, `useMarkRead()`, `useMarkAllRead()`, `useClosures(id)`, `useCreateClosure(id)`, `useDeleteClosure(id)`; type `ClosureInput`

- [ ] **Step 1: types** — `web/lib/types.ts`

ใน `Booking` ต่อจาก `cancelled_at?: string;`:
```ts
  cancelled_by: "customer" | "restaurant" | null; // null = ข้อมูลเก่าก่อนมีคอลัมน์นี้
  cancel_reason: string;
```
ต่อท้ายไฟล์:
```ts
export type Closure = { id: string; start_at: string; end_at: string; reason: string };

export type AffectedBooking = { id: string; code: string; customer_name: string; start_at: string; end_at: string; party_size: number };

export type NotificationKind = "booking_created" | "booking_updated" | "booking_cancelled" | "booking_cancelled_by_restaurant";

/** แจ้งเตือน — ข้อมูลการจองเป็น snapshot ตอนเกิดเหตุ, ข้อความประกอบที่ lib/notifications.ts */
export type Notification = {
  id: string;
  kind: NotificationKind;
  booking_id: string;
  restaurant_id: string;
  restaurant_name: string;
  customer_name: string;
  business_date: string;
  start_at: string;
  end_at: string;
  party_size: number;
  reason: string;
  read_at: string | null;
  created_at: string;
};

export type NotificationList = { items: Notification[]; unread_count: number };
```

- [ ] **Step 2: เขียนเทสต์**

`web/lib/notifications.test.ts`:
```ts
import { describe, expect, it } from "vitest";

import { bellLabel, notificationHref, notificationText } from "./notifications";
import type { Notification } from "./types";

const base: Notification = {
  id: "n1", kind: "booking_created", booking_id: "b1", restaurant_id: "r1", restaurant_name: "ท่าเรือซีฟู้ดบาร์",
  customer_name: "มะลิ วงศ์ดี", business_date: "2026-10-10", start_at: "2026-10-10T23:30:00+07:00", end_at: "2026-10-11T00:30:00+07:00",
  party_size: 3, reason: "", read_at: null, created_at: "2026-10-01T10:00:00+07:00",
};

describe("notificationText — ข้อความตาม kind", () => {
  it("ร้านยกเลิก → ชื่อร้าน + เวลา (มีป้ายข้ามวัน) + เหตุผล", () => {
    const text = notificationText({ ...base, kind: "booking_cancelled_by_restaurant", reason: "ไฟดับ" });
    expect(text).toContain("ท่าเรือซีฟู้ดบาร์ ยกเลิกการจองของคุณ");
    expect(text).toContain("23:30–00:30 (เช้าวันที่ 11)");
    expect(text).toContain("· ไฟดับ");
  });
  it("จองใหม่ → ชื่อลูกค้า + จำนวนคน", () => {
    expect(notificationText(base)).toContain("มะลิ วงศ์ดี จอง 3 คน");
  });
  it("แก้การจอง", () => {
    expect(notificationText({ ...base, kind: "booking_updated" })).toContain("มะลิ วงศ์ดี แก้การจองเป็น 3 คน");
  });
  it("ลูกค้ายกเลิก", () => {
    expect(notificationText({ ...base, kind: "booking_cancelled" })).toContain("มะลิ วงศ์ดี ยกเลิกการจอง");
  });
});

describe("notificationHref", () => {
  it("ลูกค้า → หน้าการจอง", () => {
    expect(notificationHref({ ...base, kind: "booking_cancelled_by_restaurant" })).toBe("/bookings/b1");
  });
  it("เจ้าของร้าน → บอร์ดของวันทำการ (ไม่ใช่วันปฏิทินของเวลาเริ่ม)", () => {
    expect(notificationHref(base)).toBe("/owner/bookings?restaurant=r1&date=2026-10-10");
  });
});

describe("bellLabel", () => {
  it("บอกจำนวนที่ยังไม่อ่านให้ screen reader", () => {
    expect(bellLabel(3)).toBe("การแจ้งเตือน 3 รายการที่ยังไม่อ่าน");
    expect(bellLabel(0)).toBe("การแจ้งเตือน");
  });
});
```

ต่อท้าย `web/lib/format.test.ts` (import เพิ่ม `bookingStatusLabel, closureRangeLabel, gapLabel`):
```ts
describe("bookingStatusLabel — ร้านยกเลิกต้องบอกว่าใครยกเลิกและเพราะอะไร", () => {
  const b = { status: "cancelled", cancelled_by: null, cancel_reason: "" } as const;
  it("ร้านยกเลิก → เหตุผล", () => {
    expect(bookingStatusLabel({ ...b, cancelled_by: "restaurant", cancel_reason: "ไฟดับ" }, false)).toBe("✕ ร้านยกเลิก · ไฟดับ");
  });
  it("ลูกค้ายกเลิกเอง / ข้อมูลเก่า", () => {
    expect(bookingStatusLabel({ ...b, cancelled_by: "customer" }, false)).toBe("✕ ยกเลิกแล้ว");
    expect(bookingStatusLabel(b, false)).toBe("✕ ยกเลิกแล้ว");
  });
  it("ยังไม่ยกเลิก", () => {
    expect(bookingStatusLabel({ ...b, status: "active" }, false)).toBe("✓ ยืนยันแล้ว");
    expect(bookingStatusLabel({ ...b, status: "active" }, true)).toBe("ไปแล้ว");
  });
});

describe("gapLabel — ช่องว่างระหว่าง slot เป็นช่วงพักหรือร้านปิด", () => {
  const closures = [{ id: "c1", start_at: "2026-10-10T18:00:00+07:00", end_at: "2026-10-10T20:00:00+07:00", reason: "ไฟดับ" }];
  it("ทับช่วงปิด → ร้านปิด + เหตุผล", () => {
    expect(gapLabel("2026-10-10T18:00:00+07:00", "2026-10-10T20:00:00+07:00", closures)).toBe("ร้านปิด 18:00–20:00 · ไฟดับ");
  });
  it("ไม่ทับ → พักร้าน", () => {
    expect(gapLabel("2026-10-10T14:00:00+07:00", "2026-10-10T17:00:00+07:00", closures)).toBe("พักร้าน 14:00–17:00");
  });
});

describe("closureRangeLabel", () => {
  it("วันเดียวกัน → วันที่ครั้งเดียว", () => {
    expect(closureRangeLabel({ start_at: "2026-10-10T18:00:00+07:00", end_at: "2026-10-10T20:00:00+07:00" })).toMatch(/10 ต\.ค\. 18:00–20:00$/);
  });
  it("คนละวัน → ใส่วันที่ทั้งสองฝั่ง", () => {
    const label = closureRangeLabel({ start_at: "2026-10-20T17:00:00+07:00", end_at: "2026-10-22T23:00:00+07:00" });
    expect(label).toContain("20 ต.ค. 17:00");
    expect(label).toContain("22 ต.ค. 23:00");
  });
});
```

- [ ] **Step 3: รันให้เห็นว่าไม่ผ่าน**

Run: `cd web && npx vitest run lib 2>&1 | grep -E "FAIL|Tests " | head -5`
Expected: FAIL (`Failed to resolve import "./notifications"`, `bookingStatusLabel is not a function`)

- [ ] **Step 4: เขียนโค้ด**

`web/lib/format.ts` ต่อจาก `contiguousFrom`:
```ts
/** "ส. 10 ต.ค. 18:00–20:00" หรือ "จ. 20 ต.ค. 17:00 – พ. 22 ต.ค. 23:00" (ช่วงปิดร้านหลายวัน) */
export function closureRangeLabel(c: { start_at: string; end_at: string }): string {
  if (dateKey(c.start_at) === dateKey(c.end_at)) return `${fmtShortDate(c.start_at)} ${fmtTime(c.start_at)}–${fmtTime(c.end_at)}`;
  return `${fmtShortDate(c.start_at)} ${fmtTime(c.start_at)} – ${fmtShortDate(c.end_at)} ${fmtTime(c.end_at)}`;
}

/** ป้ายของช่องว่างระหว่าง slot: ทับช่วงที่ร้านปิดชั่วคราว → "ร้านปิด … · เหตุผล" ไม่งั้นเป็นช่วงพักประจำ */
export function gapLabel(from: string, to: string, closures: { start_at: string; end_at: string; reason: string }[]): string {
  const a = new Date(from).getTime();
  const b = new Date(to).getTime();
  const c = closures.find((c) => new Date(c.start_at).getTime() < b && new Date(c.end_at).getTime() > a);
  return c ? `ร้านปิด ${fmtTime(from)}–${fmtTime(to)} · ${c.reason}` : `พักร้าน ${fmtTime(from)}–${fmtTime(to)}`;
}

/** ป้ายสถานะการจอง — ร้านยกเลิกต้องบอกว่าร้านเป็นคนยกเลิกและเพราะอะไร ไม่ใช่ "ยกเลิกแล้ว" เฉย ๆ */
export function bookingStatusLabel(b: { status: string; cancelled_by: string | null; cancel_reason: string }, past: boolean): string {
  if (b.status === "cancelled") {
    return b.cancelled_by === "restaurant" ? `✕ ร้านยกเลิก${b.cancel_reason ? ` · ${b.cancel_reason}` : ""}` : "✕ ยกเลิกแล้ว";
  }
  return past ? "ไปแล้ว" : "✓ ยืนยันแล้ว";
}
```

`web/lib/notifications.ts`:
```ts
// ข้อความและลิงก์ของแจ้งเตือน — API ส่งแค่ kind + snapshot ของการจอง หน้าเว็บประกอบข้อความไทยที่นี่ที่เดียว
import { fmtRange, fmtShortDate, keyToDate } from "./format";
import type { Notification } from "./types";

export function notificationText(n: Notification): string {
  const when = `${fmtShortDate(keyToDate(n.business_date))} ${fmtRange(n.start_at, n.end_at, n.business_date)}`;
  switch (n.kind) {
    case "booking_cancelled_by_restaurant":
      return `${n.restaurant_name} ยกเลิกการจองของคุณ ${when}${n.reason ? ` · ${n.reason}` : ""}`;
    case "booking_created":
      return `${n.customer_name} จอง ${n.party_size} คน ${when}`;
    case "booking_updated":
      return `${n.customer_name} แก้การจองเป็น ${n.party_size} คน ${when}`;
    case "booking_cancelled":
      return `${n.customer_name} ยกเลิกการจอง ${when}`;
  }
}

/** กดแล้วไปไหน: ลูกค้าไปหน้าการจอง, เจ้าของร้านไปบอร์ดของวันทำการนั้น (ร้านข้ามคืน วันทำการ ≠ วันปฏิทิน) */
export function notificationHref(n: Notification): string {
  if (n.kind === "booking_cancelled_by_restaurant") return `/bookings/${n.booking_id}`;
  return `/owner/bookings?restaurant=${n.restaurant_id}&date=${n.business_date}`;
}

export const bellLabel = (unread: number) => (unread > 0 ? `การแจ้งเตือน ${unread} รายการที่ยังไม่อ่าน` : "การแจ้งเตือน");
```

`web/lib/errors.ts` — ใน `messages` เพิ่ม:
```ts
  RESTAURANT_CLOSED: "ร้านปิดในช่วงที่เลือก",
  INVALID_CLOSURE: "ช่วงปิดไม่ถูกต้อง — ห้ามย้อนหลัง ยาวไม่เกิน 90 วัน และต้องมีเหตุผล",
  CLOSURE_AFFECTS_BOOKINGS: "มีการจองที่จะถูกยกเลิก",
```
และใน `switch` ของ `errorMessage` เพิ่ม:
```ts
    case "RESTAURANT_CLOSED":
      return `${base} (${closureRangeLabel({ start_at: String(d.start_at), end_at: String(d.end_at) })} · ${d.reason})`;
```
(import `closureRangeLabel` จาก `./format`)

`web/services/notifications.ts`:
```ts
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useSession } from "next-auth/react";

import { api } from "@/lib/api";
import type { NotificationList } from "@/lib/types";

/** แจ้งเตือนของฉัน — poll ทุก 60 วินาที + ตอนกลับมาที่แท็บ (ง่ายกว่า WebSocket และช้ากว่ากันไม่กี่วินาที) */
export function useNotifications() {
  const { status } = useSession();
  return useQuery({
    queryKey: ["notifications"],
    queryFn: async () => (await api.get<NotificationList>("/me/notifications")).data,
    enabled: status === "authenticated",
    refetchInterval: 60_000,
    refetchOnWindowFocus: true,
  });
}

export function useMarkRead() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      await api.put(`/me/notifications/${id}/read`);
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ["notifications"] }),
  });
}

export function useMarkAllRead() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      await api.put("/me/notifications/read-all");
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ["notifications"] }),
  });
}
```

`web/services/closures.ts`:
```ts
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api } from "@/lib/api";
import type { Closure } from "@/lib/types";

/** บางช่วงของวันทำการ (date + start_time + end_time) หรือทั้งวัน/หลายวัน (from_date + to_date) */
export type ClosureInput = {
  date?: string;
  start_time?: string;
  end_time?: string;
  from_date?: string;
  to_date?: string;
  reason: string;
  confirm_booking_ids?: string[]; // ส่งเมื่อยืนยันรายการจองที่จะถูกยกเลิก (ได้จาก 409 CLOSURE_AFFECTS_BOOKINGS)
};

/** ช่วงปิดที่ยังไม่จบ (สาธารณะ) */
export const useClosures = (restaurantId: string | undefined) =>
  useQuery({
    queryKey: ["closures", restaurantId],
    queryFn: async () => (await api.get<Closure[]>(`/restaurants/${restaurantId}/closures`)).data,
    enabled: !!restaurantId,
  });

// ปิด/เปิดร้านกระทบเวลาว่าง บอร์ด และการจอง → โหลดใหม่หมด
const refreshAfterChange = (qc: ReturnType<typeof useQueryClient>) =>
  Promise.all(["closures", "availability", "board", "restaurants"].map((key) => qc.invalidateQueries({ queryKey: [key] })));

export function useCreateClosure(restaurantId: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: ClosureInput) => (await api.post<Closure>(`/restaurants/${restaurantId}/closures`, input)).data,
    onSuccess: () => refreshAfterChange(qc),
  });
}

export function useDeleteClosure(restaurantId: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (closureId: string) => {
      await api.delete(`/restaurants/${restaurantId}/closures/${closureId}`);
    },
    onSuccess: () => refreshAfterChange(qc),
  });
}
```

- [ ] **Step 5: รันเทสต์ + type**

Run: `cd web && npx tsc --noEmit && npm test 2>&1 | grep -E "Test Files|Tests "`
Expected: ไม่มี error; ผ่านทั้งหมด

- [ ] **Step 6: Commit**

```bash
git add web/lib web/services
git commit -m "เว็บ: type/hook ของแจ้งเตือนและช่วงปิด + ข้อความแจ้งเตือน, ป้ายร้านยกเลิก, ป้ายช่วงปิด

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 8: เว็บ — กระดิ่งแจ้งเตือน

**Files:**
- Create: `web/containers/NotificationBell.tsx`
- Modify: `web/containers/Navbar.tsx`, `web/containers/OwnerNavbar.tsx`

**Interfaces:**
- Consumes: `useNotifications`, `useMarkRead`, `useMarkAllRead`, `notificationText`, `notificationHref`, `bellLabel` (Task 7)

- [ ] **Step 1: component** — `web/containers/NotificationBell.tsx`
```tsx
"use client";

import { Bell } from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";

import { fmtShortDate, fmtTime } from "@/lib/format";
import { bellLabel, notificationHref, notificationText } from "@/lib/notifications";
import type { Notification } from "@/lib/types";
import { useMarkAllRead, useMarkRead, useNotifications } from "@/services/notifications";

/**
 * กระดิ่งแจ้งเตือน (ทั้งโหมดลูกค้าและเจ้าของร้าน) — tone="dark" ใช้บนแถบทึบของ owner
 * ตัวเลขยังไม่อ่านมีใน aria-label ด้วย (ไม่สื่อด้วยสี/ตัวเลขลอย ๆ อย่างเดียว)
 */
export default function NotificationBell({ tone = "light" }: { tone?: "light" | "dark" }) {
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const list = useNotifications();
  const markRead = useMarkRead();
  const markAll = useMarkAllRead();
  if (!list.data) return null; // ยังไม่ login หรือกำลังโหลดครั้งแรก

  const unread = list.data.unread_count;
  const go = (n: Notification) => {
    setOpen(false);
    if (!n.read_at) markRead.mutate(n.id);
    router.push(notificationHref(n));
  };

  return (
    <div className="relative" onKeyDown={(e) => e.key === "Escape" && setOpen(false)}>
      <button type="button" aria-label={bellLabel(unread)} aria-expanded={open} aria-haspopup="true" onClick={() => setOpen(!open)}
        className={`relative grid size-11 place-items-center rounded-full ${tone === "dark" ? "text-white/85 transition-colors hover:bg-white/10" : "nav-link text-muted"}`}>
        <Bell size={19} aria-hidden />
        {unread > 0 && (
          <span aria-hidden className="absolute right-0.5 top-0.5 grid h-5 min-w-5 place-items-center rounded-full bg-full px-1 text-[12px] font-semibold text-white tabular">
            {unread > 99 ? "99+" : unread}
          </span>
        )}
      </button>
      {open && (
        <div className="absolute right-0 top-12 z-30 w-[min(360px,calc(100vw-2rem))] overflow-hidden rounded-2xl border border-border bg-surface text-sm text-text shadow-lg">
          <div className="flex items-center justify-between border-b border-border px-4 py-2.5">
            <span className="font-semibold">การแจ้งเตือน</span>
            {unread > 0 && <button type="button" className="link" onClick={() => markAll.mutate()}>อ่านทั้งหมด</button>}
          </div>
          {list.data.items.length === 0 ? (
            <p className="p-4 text-muted">ยังไม่มีการแจ้งเตือน</p>
          ) : (
            <ul className="max-h-96 overflow-y-auto">
              {list.data.items.map((n) => (
                <li key={n.id} className="border-b border-border last:border-0">
                  <button type="button" onClick={() => go(n)} className={`flex w-full gap-2.5 px-4 py-3 text-left hover:bg-chip ${n.read_at ? "text-muted" : ""}`}>
                    <span aria-hidden className={`mt-1.5 size-2 shrink-0 rounded-full ${n.read_at ? "" : "bg-sel"}`} />
                    <span className="flex flex-col gap-0.5">
                      <span>{!n.read_at && <span className="sr-only">ยังไม่อ่าน: </span>}{notificationText(n)}</span>
                      <span className="text-[13px] text-soft tabular">{fmtShortDate(n.created_at)} {fmtTime(n.created_at)}</span>
                    </span>
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </div>
  );
}
```

- [ ] **Step 2: ใส่ใน navbar**

`Navbar.tsx` — import `NotificationBell from "@/containers/NotificationBell"`; ก่อนบรรทัด `<ThemeToggle className="nav-link border-border" />`:
```tsx
        {status === "authenticated" && <NotificationBell />}
```

`OwnerNavbar.tsx` — import เดียวกัน; ก่อน `<ThemeToggle className="border-white/30 …" />`:
```tsx
        <NotificationBell tone="dark" />
```

- [ ] **Step 3: lint + type + test**

Run: `cd web && npm run lint && npx tsc --noEmit && npm test 2>&1 | grep -E "Tests "`
Expected: ไม่มี error; ผ่านทั้งหมด

- [ ] **Step 4: ดูกับระบบจริง** — `./dev.sh up` แล้ว login เป็น customer1 (ถ้า DB เป็น seed เดิม ใช้ SQL ใส่แจ้งเตือนทดสอบให้ customer1 ชั่วคราวแล้วลบออก) ตรวจ: ตัวเลขบนกระดิ่ง, เปิดแผง, กดรายการแล้วไปหน้าการจอง, ตัวเลขลด; หน้ามือถือ 375px navbar ไม่ล้น
  (ไม่มี browser tool → ใช้ E2E ใน Task 11 แทน และบันทึก Ruling)

- [ ] **Step 5: Commit**

```bash
git add web/containers
git commit -m "เว็บ: กระดิ่งแจ้งเตือนใน navbar ทั้งโหมดลูกค้าและเจ้าของร้าน

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 9: เว็บ — ฝั่งลูกค้ารู้จักช่วงปิดและการยกเลิกโดยร้าน

**Files:**
- Create: `web/components/restaurant/ClosureBanner.tsx`
- Modify: `web/app/(customer)/restaurants/[id]/page.tsx`, `web/components/booking/BookingPanel.tsx`, `web/components/booking/BookingError.tsx`, `web/app/(customer)/me/bookings/page.tsx`, `web/app/(customer)/bookings/[id]/page.tsx`, `web/app/owner/bookings/page.tsx`
- Test: `web/components/booking/BookingError.test.tsx`

**Interfaces:**
- Consumes: `useClosures`, `gapLabel`, `closureRangeLabel`, `bookingStatusLabel`, `errorMessage` (Task 7)

- [ ] **Step 1: เทสต์ BookingError** — ต่อท้าย `BookingError.test.tsx` (ดูรูปแบบ render/screen ที่ไฟล์ใช้อยู่)
```tsx
it("RESTAURANT_CLOSED → กล่องสีกลางบอกช่วงปิดและเหตุผล ไม่บอกว่าเต็ม", () => {
  render(<BookingError error={{ code: "RESTAURANT_CLOSED", message: "", details: { reason: "ไฟดับ", start_at: "2026-10-10T18:00:00+07:00", end_at: "2026-10-10T20:00:00+07:00" } }} />);
  const box = screen.getByRole("status");
  expect(box).toHaveTextContent("ร้านปิดในช่วงที่เลือก");
  expect(box).toHaveTextContent("ไฟดับ");
  expect(box).not.toHaveTextContent("เต็ม");
});
```
Run: `cd web && npx vitest run components/booking/BookingError.test.tsx 2>&1 | grep -E "✓|×|FAIL" | head`
Expected: เทสต์ใหม่ FAIL (ได้ role `alert` สีแดงแทน `status`)

- [ ] **Step 2: BookingError** — ก่อน `return` สุดท้าย:
```tsx
  if (error.code === "RESTAURANT_CLOSED") {
    return (
      <Alert tone="info" title={errorMessage(error)}>
        เวลาว่างด้านบนอัปเดตแล้ว ลองเลือกช่วงอื่นหรือวันอื่น
      </Alert>
    );
  }
```
และแก้คอมเมนต์หัวฟังก์ชันเพิ่มบรรทัด `* - RESTAURANT_CLOSED: สีกลาง — ร้านปิดชั่วคราว ไม่ใช่ความผิดพลาดและไม่ใช่ "เต็ม"`

- [ ] **Step 3: ClosureBanner** — `web/components/restaurant/ClosureBanner.tsx`
```tsx
"use client";

import { Alert } from "@/components/bases/ui";
import { closureRangeLabel } from "@/lib/format";
import { useClosures } from "@/services/closures";

/** แถบแจ้งบนหน้าร้านเมื่อมีช่วงปิดชั่วคราวที่ยังไม่จบ — ให้เห็นก่อนเลือกเวลา */
export default function ClosureBanner({ restaurantId }: { restaurantId: string }) {
  const closures = useClosures(restaurantId);
  if (!closures.data?.length) return null;
  return (
    <Alert tone="warn" title="ร้านปิดชั่วคราว">
      <ul className="flex flex-col gap-0.5">
        {closures.data.map((c) => <li key={c.id} className="tabular">{closureRangeLabel(c)} · {c.reason}</li>)}
      </ul>
    </Alert>
  );
}
```
`app/(customer)/restaurants/[id]/page.tsx` — import `ClosureBanner from "@/components/restaurant/ClosureBanner"`; ต่อจากบรรทัด `</header>`:
```tsx
      <ClosureBanner restaurantId={restaurant.id} />
```

- [ ] **Step 4: BookingPanel** — import `gapLabel` จาก `@/lib/format` และ `useClosures` จาก `@/services/closures`; ต่อจาก `const availability = useAvailability(r.id, date);`:
```tsx
  const closures = useClosures(r.id);
```
แทนเนื้อใน `<p className="col-span-4 …">` ของเส้นแบ่ง:
```tsx
                      {gapLabel(slots[i - 1].end_at, s.start_at, closures.data ?? [])}
```
แก้ Alert `hitsBreak` เป็น `title="ชนช่วงพักหรือช่วงที่ร้านปิด"` และใน `submit` แก้บรรทัด refetch:
```tsx
      if (e?.code === "NOT_ENOUGH_SEATS" || e?.code === "RESTAURANT_CLOSED") void availability.refetch();
```

- [ ] **Step 5: ป้ายสถานะการจอง** — import `bookingStatusLabel` จาก `@/lib/format` ในแต่ละไฟล์

`me/bookings/page.tsx`:
```tsx
              {bookingStatusLabel(b, past)}
```
(แทน `{cancelled ? "✕ ยกเลิกแล้ว" : past ? "ไปแล้ว" : "✓ ยืนยันแล้ว"}`)

`bookings/[id]/page.tsx` — แทน h1 และบรรทัดใต้:
```tsx
        <h1 id="ok-h" className="font-display text-[28px] text-balance sm:text-[32px]">
          {cancelled ? (b.cancelled_by === "restaurant" ? "ร้านยกเลิกการจองนี้" : "การจองนี้ถูกยกเลิกแล้ว") : "จองสำเร็จแล้ว"}
        </h1>
        {cancelled && b.cancelled_by === "restaurant" && <p className="text-muted">เหตุผลจากร้าน: {b.cancel_reason}</p>}
```

`owner/bookings/page.tsx` — cell สถานะในตาราง:
```tsx
                        <td className={`px-3.5 ${b.status === "cancelled" ? "text-full" : "text-ok"}`}>{b.status === "cancelled" ? bookingStatusLabel(b, false) : "✓ ยืนยัน"}</td>
```

- [ ] **Step 6: เส้นแบ่งบนบอร์ด owner** — ใน `owner/bookings/page.tsx` import `gapLabel` + `useClosures`; ต่อจาก `const board = useBoard(restaurantId, date);`:
```tsx
  const closures = useClosures(restaurantId);
```
แถวช่องว่าง (ที่เขียนว่า "พักร้าน") แทนด้วย:
```tsx
                        <li className="grid h-6 grid-cols-[92px_1fr] items-center gap-2.5 text-muted tabular">
                          <span>{fmtTime(all[i - 1].end_at)}–{fmtTime(s.start_at)}</span><span>{gapLabel(all[i - 1].end_at, s.start_at, closures.data ?? []).split(" ")[0]}</span>
                        </li>
```
(คำแรกของป้ายคือ "พักร้าน" หรือ "ร้านปิด")

- [ ] **Step 7: lint + type + test**

Run: `cd web && npm run lint && npx tsc --noEmit && npm test 2>&1 | grep -E "Tests "`
Expected: ผ่านทั้งหมด

- [ ] **Step 8: Commit**

```bash
git add web
git commit -m "เว็บ: ป้ายร้านปิดชั่วคราวบนหน้าร้าน/แผงจอง/บอร์ด, RESTAURANT_CLOSED สีกลาง, ป้าย \"ร้านยกเลิก\" พร้อมเหตุผล

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 10: เว็บ — หน้า "ปิดร้านชั่วคราว" ของเจ้าของร้าน

**Files:**
- Create: `web/app/owner/closures/page.tsx`
- Modify: `web/containers/OwnerNavbar.tsx`, `web/app/owner/bookings/page.tsx`

**Interfaces:**
- Consumes: `useClosures`, `useCreateClosure`, `useDeleteClosure`, `ClosureInput`, `closureRangeLabel`, `errorMessage`, `apiError`, `useMe`
- Produces: route `/owner/closures?restaurant=&date=&start=&end=` (บอร์ดส่ง query มาพร้อมค่าเริ่มต้น)

- [ ] **Step 1: หน้า** — `web/app/owner/closures/page.tsx`
```tsx
"use client";

import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useState } from "react";

import { apiError } from "@/lib/api";
import { errorMessage } from "@/lib/errors";
import { closureRangeLabel, fmtShortDate, fmtTime, minutesToClock, todayKey } from "@/lib/format";
import type { AffectedBooking, ApiError } from "@/lib/types";
import { type ClosureInput, useClosures, useCreateClosure, useDeleteClosure } from "@/services/closures";
import { useMe } from "@/services/me";

const clocks = Array.from({ length: 48 }, (_, i) => minutesToClock(i * 30));
const input = "h-9 rounded-md border border-border-strong bg-surface px-2.5 text-text";

export default function ClosuresPage() {
  return (
    <Suspense>
      <Closures />
    </Suspense>
  );
}

/**
 * ปิดร้านชั่วคราว (CLAUDE.md 5.1): รายการช่วงปิด + ฟอร์มสองแบบ
 * บันทึกแล้วได้ 409 CLOSURE_AFFECTS_BOOKINGS → แสดงการจองที่จะถูกยกเลิก แล้วยืนยันด้วยรายการ id ที่เห็นเท่านั้น
 */
function Closures() {
  const router = useRouter();
  const params = useSearchParams();
  const me = useMe();
  const restaurantId = params.get("restaurant") ?? me.data?.restaurants[0]?.id;
  const closures = useClosures(restaurantId);
  const create = useCreateClosure(restaurantId);
  const remove = useDeleteClosure(restaurantId);

  const [mode, setMode] = useState<"partial" | "days">("partial");
  const [date, setDate] = useState(params.get("date") ?? todayKey());
  const [start, setStart] = useState(params.get("start") ?? "18:00");
  const [end, setEnd] = useState(params.get("end") ?? "22:00");
  const [from, setFrom] = useState(todayKey());
  const [to, setTo] = useState(todayKey());
  const [reason, setReason] = useState("");
  const [affected, setAffected] = useState<AffectedBooking[] | null>(null);
  const [changed, setChanged] = useState(false); // ได้ 409 ตอนยืนยัน = มีการจองเข้ามาใหม่ระหว่างดูรายการ
  const [error, setError] = useState<ApiError | null>(null);
  const [done, setDone] = useState("");

  // แก้ฟอร์มแล้วรายการที่เห็นอาจไม่ตรงช่วงใหม่ → ต้องบันทึกใหม่ให้ API ตรวจอีกรอบ
  const edit = (set: (v: string) => void) => (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement>) => {
    set(e.target.value);
    setAffected(null);
  };

  const body = (): ClosureInput =>
    mode === "partial" ? { date, start_time: start, end_time: end, reason } : { from_date: from, to_date: to, reason };

  const submit = async (confirm?: AffectedBooking[]) => {
    setError(null);
    setDone("");
    try {
      await create.mutateAsync({ ...body(), confirm_booking_ids: confirm?.map((b) => b.id) });
      setDone(confirm?.length ? `ปิดร้านแล้ว — ยกเลิก ${confirm.length} การจองและแจ้งลูกค้าแล้ว` : "ปิดร้านแล้ว");
      setAffected(null);
      setChanged(false);
      setReason("");
    } catch (err) {
      const e = apiError(err);
      if (e?.code === "CLOSURE_AFFECTS_BOOKINGS") {
        setChanged(!!confirm);
        setAffected((e.details?.bookings as AffectedBooking[]) ?? []);
        return;
      }
      setError(e ?? { code: "NETWORK", message: "" });
    }
  };

  const reopen = (id: string) => {
    if (window.confirm("เปิดร้านกลับช่วงนี้? การจองที่ถูกยกเลิกไปแล้วจะไม่กลับมา")) remove.mutate(id);
  };

  if (me.data && me.data.restaurants.length === 0) {
    return <p className="rounded-md border border-dashed border-border-strong bg-surface p-6 text-center">ยังไม่มีร้าน — เพิ่มร้านที่หน้า “ร้านของฉัน” ก่อน</p>;
  }
  const guests = affected?.reduce((sum, b) => sum + b.party_size, 0) ?? 0;

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center gap-3">
        <h1 className="text-lg font-semibold">ปิดร้านชั่วคราว</h1>
        <select aria-label="ร้าน" value={restaurantId ?? ""} onChange={(e) => router.replace(`/owner/closures?restaurant=${e.target.value}`)} className={input}>
          {me.data?.restaurants.map((r) => <option key={r.id} value={r.id}>{r.name}</option>)}
        </select>
      </div>

      <section aria-labelledby="list-h" className="border border-border bg-surface">
        <h2 id="list-h" className="border-b border-border px-3.5 py-2.5 font-semibold">ช่วงปิดที่ยังไม่จบ</h2>
        {closures.data?.length === 0 && <p className="p-4 text-muted">ไม่มีช่วงปิด — ร้านเปิดตามเวลาปกติ</p>}
        <ul>
          {closures.data?.map((c) => (
            <li key={c.id} className="flex flex-wrap items-center gap-3 border-t border-border px-3.5 py-2 first:border-0">
              <span className="font-medium tabular">{closureRangeLabel(c)}</span>
              <span className="text-muted">{c.reason}</span>
              <button type="button" onClick={() => reopen(c.id)} disabled={remove.isPending} className="obtn obtn-sm ml-auto">เปิดร้านกลับ</button>
            </li>
          ))}
        </ul>
      </section>

      <form onSubmit={(e) => { e.preventDefault(); void submit(); }} className="flex flex-col gap-4 border border-border bg-surface p-4">
        <fieldset className="flex gap-2">
          <legend className="sr-only">รูปแบบการปิด</legend>
          {([["partial", "บางช่วงของวัน"], ["days", "ทั้งวันหรือหลายวัน"]] as const).map(([key, label]) => (
            <label key={key} className={`inline-flex min-h-9 items-center gap-1.5 rounded-md border px-3 ${mode === key ? "border-text font-semibold" : "border-border-strong"}`}>
              <input type="radio" name="mode" checked={mode === key} onChange={() => { setMode(key); setAffected(null); }} />
              {label}
            </label>
          ))}
        </fieldset>

        {mode === "partial" ? (
          <div className="grid gap-3 sm:grid-cols-3">
            <label className="flex flex-col gap-1 font-medium">วันทำการ<input type="date" value={date} onChange={edit(setDate)} className={input} /></label>
            <label className="flex flex-col gap-1 font-medium">ตั้งแต่<select value={start} onChange={edit(setStart)} className={input}>{clocks.map((c) => <option key={c}>{c}</option>)}</select></label>
            <label className="flex flex-col gap-1 font-medium">ถึง<select value={end} onChange={edit(setEnd)} className={input}>{clocks.map((c) => <option key={c}>{c}</option>)}</select></label>
            <p className="text-muted sm:col-span-3">ร้านเปิดข้ามเที่ยงคืน: เวลาก่อนเวลาเปิด = หลังเที่ยงคืนของรอบนั้น (เหมือนตอนลูกค้าจอง)</p>
          </div>
        ) : (
          <div className="grid gap-3 sm:grid-cols-2">
            <label className="flex flex-col gap-1 font-medium">จากวัน<input type="date" value={from} onChange={edit(setFrom)} className={input} /></label>
            <label className="flex flex-col gap-1 font-medium">ถึงวัน<input type="date" value={to} onChange={edit(setTo)} className={input} /></label>
          </div>
        )}

        <label className="flex flex-col gap-1 font-medium">
          เหตุผล * <span className="font-normal text-muted">ลูกค้าจะเห็นข้อความนี้</span>
          <textarea required maxLength={200} rows={2} value={reason} onChange={edit(setReason)} className={`${input} h-auto py-2`} placeholder="เช่น ไฟดับทั้งซอย, ปิดปรับปรุงร้าน" />
        </label>

        {!affected && (
          <button type="submit" disabled={create.isPending} aria-busy={create.isPending || undefined} className="obtn obtn-primary self-start">บันทึกช่วงปิด</button>
        )}

        {affected && (
          <section aria-labelledby="affected-h" className="flex flex-col gap-3 rounded-md border border-full-line bg-full-weak p-3.5">
            <h2 id="affected-h" className="font-semibold text-full">การจองที่จะถูกยกเลิก {affected.length} รายการ (รวม {guests} คน)</h2>
            {changed && <p role="status">มีการจองเข้ามาใหม่ระหว่างนี้ — รายการด้านล่างเป็นข้อมูลล่าสุด</p>}
            <table className="w-full border-collapse bg-surface tabular">
              <thead className="text-muted"><tr><th className="px-2 py-1.5 text-left font-medium">เลขที่</th><th className="px-2 text-left font-medium">ลูกค้า</th><th className="px-2 text-left font-medium">เวลา</th><th className="px-2 text-left font-medium">คน</th></tr></thead>
              <tbody>
                {affected.map((b) => (
                  <tr key={b.id} className="border-t border-border">
                    <td className="px-2 py-1.5">{b.code}</td><td className="px-2">{b.customer_name}</td>
                    <td className="px-2">{fmtShortDate(b.start_at)} {fmtTime(b.start_at)}–{fmtTime(b.end_at)}</td><td className="px-2">{b.party_size}</td>
                  </tr>
                ))}
              </tbody>
            </table>
            <p className="text-muted">ลูกค้าจะได้รับแจ้งเตือนพร้อมเหตุผล — ควรโทรแจ้งด้วยถ้าเป็นการปิดกะทันหัน</p>
            <div className="flex gap-2">
              <button type="button" onClick={() => void submit(affected)} disabled={create.isPending} aria-busy={create.isPending || undefined} className="obtn obtn-danger">
                ยืนยันปิดร้านและยกเลิก {affected.length} การจอง
              </button>
              <button type="button" onClick={() => setAffected(null)} className="obtn">ไม่ปิดแล้ว</button>
            </div>
          </section>
        )}

        {error && <p role="alert" className="rounded-md border border-full-line bg-full-weak p-3 text-full">✕ {errorMessage(error.code === "NETWORK" ? null : error)} {error.code !== "NETWORK" && `(${error.code})`}</p>}
        {done && <p role="status" className="rounded-md border border-ok/40 bg-ok-weak p-3 text-ok">✓ {done}</p>}
      </form>
    </div>
  );
}
```

- [ ] **Step 2: ลิงก์เข้า**

`OwnerNavbar.tsx` ต่อจากเมนู "การจองรายวัน":
```tsx
          {item("/owner/closures", "ปิดร้านชั่วคราว", pathname.startsWith("/owner/closures"))}
```

`owner/bookings/page.tsx` — import `Link from "next/link"` และ `clockToMinutes, minutesToClock` จาก format; ต่อท้าย `{board.data && ( <span …>รอบ… </span> )}` ใน div ส่วนหัว:
```tsx
        {board.data && date === todayKey() && !board.data.closed && (
          // ปิดกะทันหัน: กรอกวันนี้ + เวลาปัจจุบันปัดลง :00/:30 ถึงเวลาปิดร้านไว้ให้
          <Link className="obtn obtn-sm ml-auto"
            href={`/owner/closures?${new URLSearchParams({ restaurant: restaurantId ?? "", date,
              start: minutesToClock(Math.floor(clockToMinutes(fmtTime(new Date())) / 30) * 30), end: fmtTime(board.data.closes_at) })}`}>
            ปิดร้านตอนนี้
          </Link>
        )}
```

- [ ] **Step 3: lint + type + test**

Run: `cd web && npm run lint && npx tsc --noEmit && npm test 2>&1 | grep -E "Tests "`
Expected: ผ่านทั้งหมด

- [ ] **Step 4: Commit**

```bash
git add web
git commit -m "เว็บ: หน้าปิดร้านชั่วคราวของเจ้าของร้าน (ยืนยันรายการจองที่จะถูกยกเลิก) + ปุ่มปิดร้านตอนนี้บนบอร์ด

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 11: E2E + เอกสาร

**Files:**
- Modify: `e2e/tests/helpers.ts`, `e2e/tests/owner.spec.ts`, `CLAUDE.md`, `ARCHITECTURE.md`, `README.md`

- [ ] **Step 1: helper** — `e2e/tests/helpers.ts` ต่อจาก `tomorrow()`:
```ts
/** วันที่ n วันจากวันนี้ตามเวลาไทย (YYYY-MM-DD) */
export function daysFromToday(n: number) {
  return new Intl.DateTimeFormat("en-CA", { timeZone: "Asia/Bangkok" }).format(new Date(Date.now() + n * 86_400_000));
}
```

- [ ] **Step 2: E2E** — ต่อท้าย `test.describe` ใน `owner.spec.ts` (import `daysFromToday`, `AUTH`; ใช้ fixture `browser`):
```ts
  test("ปิดร้านทับการจอง → เห็นรายการ → ยืนยัน → ลูกค้าเห็นแจ้งเตือนและป้ายร้านยกเลิก", async ({ page, browser, request }) => {
    // seed: customer1 จองซูชิ (ร้านของ owner2) ไว้ 3 วันข้างหน้า 19:00–20:30
    const sushi = await restaurantId(request, "ซูชิ");
    await page.goto(`/owner/closures?restaurant=${sushi}&date=${daysFromToday(3)}&start=18:00&end=21:00`);
    await page.getByLabel(/เหตุผล/).fill("ทดสอบปิดร้าน");
    await page.getByRole("button", { name: "บันทึกช่วงปิด" }).click();
    await expect(page.getByRole("heading", { name: /การจองที่จะถูกยกเลิก 1 รายการ/ })).toBeVisible();
    await expect(page.getByRole("table")).toContainText("มะลิ วงศ์ดี");
    await page.getByRole("button", { name: "ยืนยันปิดร้านและยกเลิก 1 การจอง" }).click();
    await expect(page.getByRole("status").filter({ hasText: "ปิดร้านแล้ว" })).toBeVisible();

    const customer = await (await browser.newContext({ storageState: AUTH.customer })).newPage();
    await customer.goto("/");
    await customer.getByRole("button", { name: /การแจ้งเตือน \d+ รายการที่ยังไม่อ่าน/ }).click();
    await customer.getByRole("button", { name: /ซูชิ.*ยกเลิกการจองของคุณ.*ทดสอบปิดร้าน/ }).click();
    await expect(customer).toHaveURL(/\/bookings\/[0-9a-f-]{36}$/);
    await expect(customer.getByText("ร้านยกเลิกการจองนี้")).toBeVisible();
    await expect(customer.getByText("เหตุผลจากร้าน: ทดสอบปิดร้าน")).toBeVisible();
  });
```

- [ ] **Step 3: รัน E2E** — ⚠️ `./dev.sh e2e` ล้าง seed (`--reset`) = ลบข้อมูลร้านที่ผู้ใช้แก้ไว้ใน DB ของเครื่อง **ถามผู้ใช้ก่อนรัน**
Run (หลังได้รับอนุญาต): `./dev.sh up && ./dev.sh e2e`
Expected: ทุกเทสต์ผ่าน รวมเทสต์ใหม่ และเทสต์ mobile 375px (navbar ที่มีกระดิ่งไม่ล้นจอ)

- [ ] **Step 4: เอกสาร**

CLAUDE.md:
- ข้อ 4 schema: เพิ่มตาราง `restaurant_closures`, คอลัมน์ `bookings.cancelled_by/cancel_reason`, ตาราง `notifications` (ย่อจาก migration 00008–00010 พร้อมคอมเมนต์ snapshot)
- ข้อ 4.1 seed: เพิ่ม "การจองของ customer1 ที่ร้านยกเลิก 1 รายการ + แจ้งเตือนที่ยังไม่อ่าน, ร้านบุฟเฟ่ต์ปิดปรับปรุง 5 วันข้างหน้า, owner2 มีแจ้งเตือนการจองใหม่"
- ข้อ 5.1 ตาราง: `| ปิดร้านชั่วคราว (บางช่วง/ทั้งวัน/หลายวัน) ต้องมีเหตุผล ≤ 200 ตัว, ไม่ย้อนหลัง, ≤ 90 วัน | ผิด → 400 INVALID_CLOSURE |` และ `| ช่วงปิดทับการจองที่ยังไม่เริ่ม | 409 CLOSURE_AFFECTS_BOOKINGS + details.bookings → ส่งซ้ำพร้อม confirm_booking_ids ที่ตรงกันพอดีจึงยกเลิก + แจ้งลูกค้าในทรานแซกชันเดียว |`; แถว "ลบร้าน…" เพิ่ม "ยกเลิกด้วย `cancelled_by='restaurant'` + แจ้งลูกค้า"
- ข้อ 5.2 ตาราง เพิ่มกฎ `| 5b | ช่วงจองไม่ทับช่วงที่ร้านปิดชั่วคราว (ตรวจในล็อก ก่อนกฎข้อ 8) | 400 RESTAURANT_CLOSED |`
- ข้อ 5.4 ต่อท้าย: "**ช่วงปิดชั่วคราว:** เก็บเป็นช่วงเวลาจริง ตรวจด้วย `closureAt` ตัวเดียว (checkSlot, Slots, SlotsAround); แปลงคำขอด้วย `Hours.Span`/`Hours.Days`"
- ข้อ 5.5: ยกเลิกโดยลูกค้า = `cancelled_by='customer'`
- ข้อ 6: ตาราง API เพิ่ม `GET/POST /restaurants/:id/closures`, `DELETE /restaurants/:id/closures/:closureId`, `GET /me/notifications`, `PUT /me/notifications/:id/read`, `PUT /me/notifications/read-all`; ตาราง error code เพิ่ม 3 ตัว
- ข้อ 7: หน้าเพิ่ม `/owner/closures`; บรรทัด "กระดิ่งแจ้งเตือนใน navbar ทั้งสองโหมด (poll 60 วินาที)"
- ข้อ 9: เพิ่มเทสต์ช่วงปิด/แจ้งเตือนตามที่ทำจริง
- ข้อ 11 ตารางคำถาม เพิ่ม 3 แถว:
  - "ทำไมยืนยันด้วยรายการ id ไม่ใช่ confirm: true" → "กันยกเลิกการจองที่เจ้าของร้านยังไม่เคยเห็น — มีคนจองแทรกระหว่างดูรายการจะได้ 409 ใหม่"
  - "ทำไมสร้าง notification ในทรานแซกชันเดียวกัน" → "จองไม่สำเร็จไม่มีแจ้งเตือนหลง, ยกเลิกสำเร็จลูกค้าได้รับแจ้งแน่นอน; ต่อยอดอีเมล/LINE ด้วย outbox + worker"
  - "ทำไม polling ไม่ใช่ WebSocket" → "ง่าย อธิบายได้ ช้ากว่ากันไม่เกิน 60 วินาที ซึ่งพอสำหรับการแจ้งยกเลิกล่วงหน้า"

ARCHITECTURE.md: เพิ่ม `notification/` ในโครงโฟลเดอร์ + ย่อหน้า "booking/restaurant เขียนตาราง notifications ผ่าน `notification.Insert(ctx, tx, …)` ในทรานแซกชันของตัวเองโดยตั้งใจ (เหตุผลเดียวกับการแตะตาราง restaurants)" และเพิ่มคอลัมน์ใหม่ในรายการคอลัมน์ข้ามโดเมน

README.md: ฟีเจอร์ใหม่ในตาราง seed/ความสามารถ + หัวข้อ "งานต่อยอด": อีเมล/LINE ผ่าน outbox, เตือนก่อนถึงเวลาจอง (ต้องมี scheduler), WebSocket

- [ ] **Step 5: ตรวจรวม + Commit**

Run: `./dev.sh lint && ./dev.sh test`
Expected: ผ่านทั้งหมด

```bash
git add e2e CLAUDE.md ARCHITECTURE.md README.md
git commit -m "E2E ปิดร้าน→ลูกค้าได้รับแจ้งเตือน + อัปเดตเอกสารช่วงปิดชั่วคราวและแจ้งเตือน

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```
