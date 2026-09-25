# เฟส 1: แกนกติกาการจอง (booking core) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** สร้าง package `internal/booking` ที่เป็น pure Go (ไม่แตะ DB/HTTP) ครอบกติกาวันทำการ เวลาเปิด–ปิด การนับที่นั่งแบบ sweep line ตารางเวลาว่าง และกฎตรวจคำขอจอง พร้อม test ข้อ 1–16 ของ CLAUDE.md ข้อ 9

**Architecture:** ฟังก์ชันล้วนที่รับเวลา `now` เป็นพารามิเตอร์ (ไม่เรียก `time.Now()` เอง) เพื่อให้เทสต์ boundary ได้แน่นอน ทุกเวลาเป็น `time.Time` ที่ timezone `Asia/Bangkok` ตอนคำนวณ ชั้น service/repository ในเฟสถัดไปจะเรียกฟังก์ชันเหล่านี้ภายในทรานแซกชัน

**Tech Stack:** Go 1.26, `github.com/google/uuid`, `github.com/stretchr/testify`

**Spec:** `CLAUDE.md` (ข้อ 4, 5.2–5.5, 6 availability, 9)

## Global Constraints

- Go module ชื่อ `jongyoung` อยู่ที่ `api/`
- timezone ของทุกร้าน `Asia/Bangkok` (ไม่มี DST); import `time/tzdata` เพื่อให้หา zone เจอใน container distroless
- ช่วงจองเป็นช่วงละ 30 นาที (:00/:30), ยาว 30 นาที – 4 ชม., ล่วงหน้าไม่เกิน 90 วัน, lead time 30 นาที
- `duration = (close - open + 1440) % 1440`; ได้ 0 → 1440 (เปิด 24 ชม.)
- `date` = วันทำการ = รอบเปิดที่เริ่มในวันปฏิทินนั้น; เวลาที่เลือก < `open_minute` → บวก 1 วัน
- ห้ามใช้ naive SUM — ใช้ `maxConcurrent` ตัวเดียว คืน `(peak, at)`
- ข้อความ commit เป็นภาษาไทย + บรรทัด `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`

## Review Focus

1. booking ที่จบพอดีเวลาปิด (`end == closesAt`) ต้องจองได้ — ไม่ใช่ถูกปัดทิ้งเพราะเทียบ `<` ผิดข้าง → เทสต์ใน Task 2
2. booking ที่ติดกันพอดี (A จบ 12:30, B เริ่ม 12:30) ต้องไม่นับว่าทับกัน → เทสต์ใน Task 3 (เคส 2)
3. ยกเลิกตรงเวลาเส้นตายพอดี (`now == start - cancel_before`) ต้องยกเลิกได้ → เทสต์ใน Task 4
4. ตารางเวลาว่างของ "วันนี้" ต้องตัดช่วงที่เริ่มก่อน `now + 30 นาที` ออก แต่ช่วงที่เริ่มตรง `now + 30 นาที` พอดีต้องอยู่ → เทสต์ใน Task 5
5. container ไม่มี tzdata → `LoadLocation("Asia/Bangkok")` พังตอนเริ่มโปรแกรม → import `time/tzdata` ใน Task 1 และเทสต์ว่า offset = +07:00

---

## แผนภาพรวมทุกเฟส (แต่ละเฟสเขียนแผนละเอียดแยกเมื่อเริ่มเฟสนั้น)

| เฟส | ส่งมอบ | ทดสอบได้โดย |
|---|---|---|
| **1 (แผนนี้)** | `internal/booking` pure logic + test ข้อ 1–16 | `go test ./internal/booking/...` |
| 2 | infra: compose (postgres, keycloak, caddy), realm export, migrations ทั้งหมด, config, `cmd/api` + `/healthz` | `docker compose up` แล้ว `curl api.jongyoung.localhost/healthz` |
| 3 | auth: `middleware.JWT` + JIT `EnsureExists`, `reqctx`, `httputil` error, `GET /me` | handler test 401 + integration test upsert |
| 4 | restaurant domain: CRUD, images, ลดที่นั่ง/ย่นเวลา (409), soft delete ยกเลิก booking, list + Bayesian + `slots`, availability, next-available | service + repository test |
| 5 | booking HTTP: POST/PUT/DELETE/GET, `FOR UPDATE` + กฎ 7/8, owner board, concurrency test | integration test 2 goroutine |
| 6 | review domain + atomic `rating_sum` | service + repository test |
| 7 | seed (ข้อ 4.1) + swagger | `go run ./cmd/seed` |
| 8 | web scaffold: Next.js, next-auth, axios, TanStack, design tokens, Navbar/สลับโหมด | Vitest + เปิดหน้าเว็บ |
| 9 | web ฝั่งลูกค้า: ค้นหา+time chip, หน้าร้าน+แผงจอง, หน้ายืนยัน, การจองของฉัน, สถานะทั้งหมด | Vitest ข้อ 27–30 |
| 10 | web ฝั่ง owner: ตารางร้าน, ฟอร์ม, บอร์ดรายวัน + seat bar | เปิดหน้าเว็บ |
| 11 | `.gitlab-ci.yml` + Dockerfile api/web | pipeline ผ่าน |
| 12 | README + docs/ + ลิงก์ design | อ่านทวน |

---

## File Structure (เฟส 1)

| ไฟล์ | หน้าที่ |
|---|---|
| `api/go.mod` | module `jongyoung` |
| `api/internal/booking/model.go` | struct `Booking` + ค่าคงที่สถานะ |
| `api/internal/booking/errors.go` | `RuleError` + ตัวแปร error ของกฎ, `NotEnoughSeatsError` |
| `api/internal/booking/businessday.go` | `Bangkok`, `Hours`, `ParseDate`, `Window`, `At`, `Fits` |
| `api/internal/booking/availability.go` | `maxConcurrent`, `checkSeats`, `Slot`, `Slots` |
| `api/internal/booking/rules.go` | `Request`, `ValidateRequest`, `CancelDeadline`, `CanCancel` |
| `api/internal/booking/*_test.go` | เทสต์ของแต่ละไฟล์ + `helpers_test.go` |

---

### Task 1: ตั้ง module + model + errors + timezone

**Files:**
- Create: `api/go.mod`, `api/internal/booking/model.go`, `api/internal/booking/errors.go`, `api/internal/booking/businessday.go` (เฉพาะส่วน timezone), `api/internal/booking/helpers_test.go`, `api/internal/booking/businessday_test.go`

**Interfaces:**
- Produces: `booking.Bangkok *time.Location`, `booking.Booking`, `booking.StatusActive`, `booking.StatusCancelled`, `*booking.RuleError`, `*booking.NotEnoughSeatsError`

- [ ] **Step 1: สร้าง module**

```bash
mkdir -p api/internal/booking && cd api && go mod init jongyoung && go get github.com/google/uuid github.com/stretchr/testify
```

- [ ] **Step 2: เขียนเทสต์ timezone ที่ต้อง fail**

`api/internal/booking/helpers_test.go`
```go
package booking

import "time"

// bkk สร้างเวลาไทยแบบย่อ ใช้ในเทสต์เท่านั้น
func bkk(y int, m time.Month, d, h, min int) time.Time {
	return time.Date(y, m, d, h, min, 0, 0, Bangkok)
}
```

`api/internal/booking/businessday_test.go`
```go
package booking

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBangkokOffsetIsPlus7(t *testing.T) {
	_, offset := bkk(2026, 10, 10, 12, 0).Zone()
	assert.Equal(t, 7*60*60, offset)
}
```

- [ ] **Step 3: รันให้ fail** — `cd api && go test ./internal/booking/...` → FAIL `undefined: Bangkok`

- [ ] **Step 4: เขียนโค้ด**

`api/internal/booking/businessday.go`
```go
package booking

import (
	"time"
	_ "time/tzdata" // ฝัง tzdata ในไบนารี — container distroless หา Asia/Bangkok เจอเสมอ
)

// Bangkok คือ timezone ของทุกร้าน (ไม่มี DST)
var Bangkok = mustLoadLocation("Asia/Bangkok")

func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}
```

`api/internal/booking/model.go`
```go
package booking

import (
	"time"

	"github.com/google/uuid"
)

const (
	StatusActive    = "active"
	StatusCancelled = "cancelled"
)

// Booking คือการจองหนึ่งรายการ (ตาราง bookings)
type Booking struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	RestaurantID uuid.UUID  `gorm:"type:uuid;not null"`
	UserID       uuid.UUID  `gorm:"type:uuid;not null"`
	PartySize    int        `gorm:"not null"`
	StartAt      time.Time  `gorm:"not null"`
	EndAt        time.Time  `gorm:"not null"`
	Status       string     `gorm:"not null;default:active"`
	CancelledAt  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
```

`api/internal/booking/errors.go`
```go
package booking

import (
	"fmt"
	"time"
)

// RuleError คือกฎที่ตรวจได้โดยไม่ต้องใช้ฐานข้อมูล Code ส่งต่อให้หน้าเว็บแปลเป็นข้อความไทย
type RuleError struct {
	Code    string
	Message string
}

func (e *RuleError) Error() string { return e.Code + ": " + e.Message }

var (
	ErrInvalidRange   = &RuleError{"INVALID_TIME_RANGE", "เวลาสิ้นสุดต้องหลังเวลาเริ่ม"}
	ErrNotOnHalfHour  = &RuleError{"NOT_ON_HALF_HOUR", "เวลาเริ่มและสิ้นสุดต้องเป็น :00 หรือ :30"}
	ErrBadDuration    = &RuleError{"INVALID_DURATION", "ระยะเวลาจองต้องอยู่ระหว่าง 30 นาทีถึง 4 ชั่วโมง"}
	ErrInPast         = &RuleError{"BOOKING_IN_PAST", "เวลาที่เลือกผ่านไปแล้ว"}
	ErrTooLateToBook  = &RuleError{"TOO_LATE_TO_BOOK", "ต้องจองล่วงหน้าอย่างน้อย 30 นาที"}
	ErrTooFarAhead    = &RuleError{"TOO_FAR_AHEAD", "จองล่วงหน้าได้ไม่เกิน 90 วัน"}
	ErrInvalidParty   = &RuleError{"INVALID_PARTY_SIZE", "จำนวนคนต้องอย่างน้อย 1 คน"}
	ErrPartyTooLarge  = &RuleError{"PARTY_TOO_LARGE", "จำนวนคนมากกว่าที่นั่งทั้งร้าน"}
	ErrOutsideHours   = &RuleError{"OUTSIDE_OPENING_HOURS", "ช่วงที่เลือกอยู่นอกเวลาเปิด–ปิดของร้าน"}
)

// NotEnoughSeatsError → 409 NOT_ENOUGH_SEATS; At = จุดที่แน่นที่สุด, Available = ที่ว่าง ณ จุดนั้น
type NotEnoughSeatsError struct {
	At        time.Time
	Available int
}

func (e *NotEnoughSeatsError) Error() string {
	return fmt.Sprintf("NOT_ENOUGH_SEATS: %d seats available at %s", e.Available, e.At.Format(time.RFC3339))
}
```

- [ ] **Step 5: รันให้ผ่าน** — `cd api && go test ./internal/booking/...` → PASS

- [ ] **Step 6: Commit**
```bash
git add api/ && git commit -m "เริ่ม Go module และโครง package booking (model, errors, timezone)"
```

---

### Task 2: วันทำการและเวลาเปิด–ปิด (`businessday.go`)

**Files:**
- Modify: `api/internal/booking/businessday.go`
- Test: `api/internal/booking/businessday_test.go`

**Interfaces:**
- Consumes: `Bangkok` (Task 1)
- Produces:
  - `type Hours struct{ OpenMinute, CloseMinute int }`
  - `func (h Hours) DurationMinutes() int`
  - `func (h Hours) Is24h() bool`
  - `func ParseDate(s string) (time.Time, error)` — เที่ยงคืนเวลาไทย
  - `func (h Hours) Window(date time.Time) (opensAt, closesAt time.Time)`
  - `func (h Hours) At(date time.Time, minuteOfDay int) time.Time`
  - `func (h Hours) Fits(start, end time.Time) bool`

- [ ] **Step 1: เขียนเทสต์ (ข้อ 9 เคส 6–8, 13–16 + Review Focus 1)**

เพิ่มใน `businessday_test.go`
```go
var (
	normal    = Hours{OpenMinute: 11 * 60, CloseMinute: 22 * 60} // 11:00–22:00
	overnight = Hours{OpenMinute: 18 * 60, CloseMinute: 2 * 60}  // 18:00–02:00
	allDay    = Hours{OpenMinute: 0, CloseMinute: 0}             // 24 ชม.
)

func TestDurationMinutes(t *testing.T) {
	assert.Equal(t, 11*60, normal.DurationMinutes())
	assert.Equal(t, 8*60, overnight.DurationMinutes())
	assert.Equal(t, 1440, allDay.DurationMinutes(), "open == close ต้องเป็น 24 ชม. ไม่ใช่ 0")
}

func TestParseDate(t *testing.T) {
	d, err := ParseDate("2026-10-10")
	require.NoError(t, err)
	assert.True(t, d.Equal(bkk(2026, 10, 10, 0, 0)))

	_, err = ParseDate("10/10/2026")
	assert.Error(t, err)
}

func TestWindow(t *testing.T) {
	date := bkk(2026, 10, 10, 0, 0)
	cases := []struct {
		name       string
		h          Hours
		open, shut time.Time
	}{
		{"ร้านปกติ วันทำการ = วันปฏิทิน", normal, bkk(2026, 10, 10, 11, 0), bkk(2026, 10, 10, 22, 0)},
		{"ข้ามคืน date=10 → 18:00 วันที่ 10 ถึง 02:00 วันที่ 11", overnight, bkk(2026, 10, 10, 18, 0), bkk(2026, 10, 11, 2, 0)},
		{"24 ชม.", allDay, bkk(2026, 10, 10, 0, 0), bkk(2026, 10, 11, 0, 0)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			open, shut := c.h.Window(date)
			assert.True(t, open.Equal(c.open), "opensAt = %s", open)
			assert.True(t, shut.Equal(c.shut), "closesAt = %s", shut)
		})
	}
}

func TestAt(t *testing.T) {
	date := bkk(2026, 10, 10, 0, 0)
	assert.True(t, overnight.At(date, 30).Equal(bkk(2026, 10, 11, 0, 30)), "00:30 ของวันทำการ 10 = เช้าวันที่ 11")
	assert.True(t, overnight.At(date, 19*60).Equal(bkk(2026, 10, 10, 19, 0)))
	assert.True(t, normal.At(date, 12*60).Equal(bkk(2026, 10, 10, 12, 0)))
	assert.True(t, allDay.At(date, 0).Equal(bkk(2026, 10, 10, 0, 0)))
}

func TestFits(t *testing.T) {
	closesAt23 := Hours{OpenMinute: 11 * 60, CloseMinute: 23 * 60}
	cases := []struct {
		name       string
		h          Hours
		start, end time.Time
		want       bool
	}{
		{"ปกติ อยู่ในเวลา", normal, bkk(2026, 10, 10, 12, 0), bkk(2026, 10, 10, 13, 0), true},
		{"ปกติ จบพอดีเวลาปิด", normal, bkk(2026, 10, 10, 21, 0), bkk(2026, 10, 10, 22, 0), true},
		{"ปกติ ก่อนเปิด", normal, bkk(2026, 10, 10, 10, 30), bkk(2026, 10, 10, 11, 30), false},
		{"คาบเกี่ยวเวลาปิด 22:30–23:30 ร้านปิด 23:00", closesAt23, bkk(2026, 10, 10, 22, 30), bkk(2026, 10, 10, 23, 30), false},
		{"ข้ามคืน คร่อมเที่ยงคืน", overnight, bkk(2026, 10, 10, 23, 30), bkk(2026, 10, 11, 0, 30), true},
		{"ข้ามคืน หลังเที่ยงคืนจบพอดีตีสอง", overnight, bkk(2026, 10, 11, 1, 0), bkk(2026, 10, 11, 2, 0), true},
		{"ข้ามคืน เลยตีสอง", overnight, bkk(2026, 10, 11, 1, 30), bkk(2026, 10, 11, 2, 30), false},
		{"ข้ามคืน ก่อนเปิด", overnight, bkk(2026, 10, 10, 17, 30), bkk(2026, 10, 10, 18, 30), false},
		{"24 ชม. คร่อมเที่ยงคืน", allDay, bkk(2026, 10, 10, 23, 30), bkk(2026, 10, 11, 0, 30), true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, c.h.Fits(c.start, c.end))
		})
	}
}
```
เพิ่ม import `"time"` และ `"github.com/stretchr/testify/require"`

- [ ] **Step 2: รันให้ fail** — `go test ./internal/booking/...` → FAIL `undefined: Hours`

- [ ] **Step 3: เขียนโค้ด** ต่อท้าย `businessday.go`
```go
const minutesPerDay = 1440

// Hours คือเวลาเปิด–ปิดของร้าน เป็นนาทีนับจากเที่ยงคืนเวลาไทย
// close <= open แปลว่าเปิดข้ามเที่ยงคืน, open == close แปลว่าเปิด 24 ชม.
type Hours struct {
	OpenMinute  int
	CloseMinute int
}

// DurationMinutes คือความยาวของรอบเปิดหนึ่งรอบ
func (h Hours) DurationMinutes() int {
	d := (h.CloseMinute - h.OpenMinute + minutesPerDay) % minutesPerDay
	if d == 0 {
		d = minutesPerDay // ⚠️ ร้าน 24 ชม. — ถ้าไม่มีบรรทัดนี้จะจองไม่ได้เลย
	}
	return d
}

func (h Hours) Is24h() bool { return h.OpenMinute == h.CloseMinute }

// ParseDate แปลง "YYYY-MM-DD" เป็นเที่ยงคืนของวันนั้นเวลาไทย
func ParseDate(s string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02", s, Bangkok)
}

// Window คืนรอบเปิดของวันทำการ date เป็นช่วง [opensAt, closesAt)
func (h Hours) Window(date time.Time) (opensAt, closesAt time.Time) {
	y, m, d := date.In(Bangkok).Date()
	opensAt = time.Date(y, m, d, 0, h.OpenMinute, 0, 0, Bangkok)
	closesAt = opensAt.Add(time.Duration(h.DurationMinutes()) * time.Minute)
	return opensAt, closesAt
}

// At แปลงเวลาบนนาฬิกา (นาทีจากเที่ยงคืน) ที่ผู้ใช้เลือกในวันทำการ date เป็นเวลาจริง
// เวลาที่น้อยกว่าเวลาเปิด = ส่วนหลังเที่ยงคืนของรอบนั้น → บวก 1 วัน
func (h Hours) At(date time.Time, minuteOfDay int) time.Time {
	y, m, d := date.In(Bangkok).Date()
	if minuteOfDay < h.OpenMinute {
		d++
	}
	return time.Date(y, m, d, 0, minuteOfDay, 0, 0, Bangkok)
}

// Fits บอกว่าช่วง [start,end) อยู่ภายในรอบเปิดรอบเดียวของร้านไหม
// start อาจอยู่ในรอบของวันเดียวกัน หรือส่วนหลังเที่ยงคืนของรอบเมื่อวาน จึงลองสองรอบ
func (h Hours) Fits(start, end time.Time) bool {
	if h.Is24h() {
		return true
	}
	for _, back := range []int{0, -1} {
		opensAt, closesAt := h.Window(start.In(Bangkok).AddDate(0, 0, back))
		if !start.Before(opensAt) && start.Before(closesAt) && !end.After(closesAt) {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: รันให้ผ่าน** — `go test ./internal/booking/...` → PASS

- [ ] **Step 5: Commit** — `git commit -m "เพิ่มวันทำการและการตรวจเวลาเปิด–ปิด (รองรับข้ามคืนและ 24 ชม.)"`

---

### Task 3: sweep line (`maxConcurrent`, `checkSeats`)

**Files:**
- Create: `api/internal/booking/availability.go`, `api/internal/booking/availability_test.go`

**Interfaces:**
- Consumes: `Booking`, `StatusCancelled`, `NotEnoughSeatsError`
- Produces:
  - `func maxConcurrent(bookings []Booking, start, end time.Time) (peak int, at time.Time)`
  - `func checkSeats(existing []Booking, seats, partySize int, start, end time.Time) error`

- [ ] **Step 1: เขียนเทสต์ (ข้อ 9 เคส 1, 2, 3, 11, 12 + Review Focus 2)**

`availability_test.go`
```go
package booking

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func bk(party int, start, end time.Time) Booking {
	return Booking{PartySize: party, StartAt: start, EndAt: end, Status: StatusActive}
}

func TestMaxConcurrent(t *testing.T) {
	a := bk(7, bkk(2026, 10, 10, 12, 0), bkk(2026, 10, 10, 12, 30))
	b := bk(7, bkk(2026, 10, 10, 12, 30), bkk(2026, 10, 10, 13, 0))
	c := bk(3, bkk(2026, 10, 10, 12, 0), bkk(2026, 10, 10, 13, 0))

	peak, _ := maxConcurrent(nil, bkk(2026, 10, 10, 12, 0), bkk(2026, 10, 10, 13, 0))
	assert.Equal(t, 0, peak)

	peak, at := maxConcurrent([]Booking{a, b, c}, bkk(2026, 10, 10, 12, 0), bkk(2026, 10, 10, 13, 0))
	assert.Equal(t, 10, peak)
	assert.True(t, at.Equal(bkk(2026, 10, 10, 12, 0)), "peak แรกเกิดตอน 12:00")
}

func TestCheckSeats(t *testing.T) {
	const seats = 10
	s1200, s1230, s1300, s1330 := bkk(2026, 10, 10, 12, 0), bkk(2026, 10, 10, 12, 30), bkk(2026, 10, 10, 13, 0), bkk(2026, 10, 10, 13, 30)

	cases := []struct {
		name       string
		existing   []Booking
		party      int
		start, end time.Time
		wantErr    bool
		available  int
	}{
		{"1) มีคนจอง 7 ขอเพิ่ม 5 ทับกัน → ปฏิเสธ",
			[]Booking{bk(7, s1200, s1300)}, 5, s1230, s1330, true, 3},
		{"2) A7 12:00–12:30, B7 12:30–13:00, C3 12:00–13:00 → ผ่าน (naive SUM พลาด)",
			[]Booking{bk(7, s1200, s1230), bk(7, s1230, s1300)}, 3, s1200, s1300, false, 0},
		{"3a) A7+B3 เต็ม, A แก้เป็น 8 (existing ไม่รวม A) → ปฏิเสธ",
			[]Booking{bk(3, s1200, s1300)}, 8, s1200, s1300, true, 7},
		{"3b) A แก้เป็น 5 → ผ่าน",
			[]Booking{bk(3, s1200, s1300)}, 5, s1200, s1300, false, 0},
		{"11) booking ที่ยกเลิกแล้วไม่ถูกนับ",
			[]Booking{{PartySize: 10, StartAt: s1200, EndAt: s1300, Status: StatusCancelled}}, 10, s1200, s1300, false, 0},
		{"จองเต็มพอดี 10/10 → ผ่าน",
			[]Booking{bk(4, s1200, s1300)}, 6, s1200, s1300, false, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := checkSeats(c.existing, seats, c.party, c.start, c.end)
			if !c.wantErr {
				assert.NoError(t, err)
				return
			}
			var nes *NotEnoughSeatsError
			require.True(t, errors.As(err, &nes), "ต้องได้ NotEnoughSeatsError แต่ได้ %v", err)
			assert.Equal(t, c.available, nes.Available)
		})
	}
}
```

- [ ] **Step 2: รันให้ fail** — FAIL `undefined: maxConcurrent`

- [ ] **Step 3: เขียนโค้ด** `availability.go`
```go
package booking

import "time"

// maxConcurrent คืนจำนวนคนสูงสุดที่อยู่ในร้านพร้อมกันภายในช่วง [start,end) และเวลาที่เกิดค่าสูงสุด
// ใช้ทั้งตอนจอง, แก้ไข, ลดที่นั่ง และคำนวณ slot — มีฟังก์ชันเดียว ห้ามเขียนซ้ำ
func maxConcurrent(bookings []Booking, start, end time.Time) (peak int, at time.Time) {
	// จุดที่ต้องตรวจ = จุดเริ่มของช่วง + ทุกจุดที่มีคนเข้าร้านเพิ่มภายในช่วงนั้น
	// (จำนวนคนเพิ่มได้เฉพาะตอนมีคนเริ่ม ระหว่างสองจุดค่าคงที่ จุดที่คนออกค่าลดลงจึงไม่ต้องตรวจ)
	points := []time.Time{start}
	for _, b := range bookings {
		if b.StartAt.After(start) && b.StartAt.Before(end) {
			points = append(points, b.StartAt)
		}
	}
	at = start
	for _, t := range points {
		occupied := 0
		for _, b := range bookings {
			if b.Status == StatusCancelled {
				continue // กันไว้อีกชั้น — ปกติ query ก็กรองเฉพาะ active อยู่แล้ว
			}
			// อยู่ในร้าน ณ เวลา t คือ start_at <= t < end_at
			if !b.StartAt.After(t) && b.EndAt.After(t) {
				occupied += b.PartySize
			}
		}
		if occupied > peak {
			peak, at = occupied, t
		}
	}
	return peak, at
}

// checkSeats: existing = booking ที่ทับกับ [start,end) โดยไม่รวมตัวที่กำลังแก้ (excludeID)
func checkSeats(existing []Booking, seats, partySize int, start, end time.Time) error {
	peak, at := maxConcurrent(existing, start, end)
	if peak+partySize > seats {
		return &NotEnoughSeatsError{At: at, Available: seats - peak}
	}
	return nil
}
```

- [ ] **Step 4: รันให้ผ่าน** — PASS

- [ ] **Step 5: Commit** — `git commit -m "เพิ่มการนับที่นั่งแบบ sweep line (maxConcurrent, checkSeats)"`

---

### Task 4: กฎตรวจคำขอจองและยกเลิก (`rules.go`)

**Files:**
- Create: `api/internal/booking/rules.go`, `api/internal/booking/rules_test.go`

**Interfaces:**
- Consumes: `Hours.Fits`, error vars ของ Task 1
- Produces:
  - `const SlotLength, LeadTime, MinDuration, MaxDuration, MaxAdvance time.Duration`
  - `type Request struct{ StartAt, EndAt time.Time; PartySize int }`
  - `func ValidateRequest(req Request, h Hours, seats int, now time.Time) error`
  - `func CancelDeadline(startAt time.Time, cancelBeforeMinutes int) time.Time`
  - `func CanCancel(startAt time.Time, cancelBeforeMinutes int, now time.Time) bool`

- [ ] **Step 1: เขียนเทสต์ (ข้อ 9 เคส 4, 5, 6, 7, 8, 9, 10 + Review Focus 3)**

`rules_test.go`
```go
package booking

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestValidateRequest(t *testing.T) {
	now := bkk(2026, 10, 10, 17, 30)
	req := func(party int, start, end time.Time) Request { return Request{StartAt: start, EndAt: end, PartySize: party} }
	cases := []struct {
		name  string
		h     Hours
		r     Request
		now   time.Time
		want  error
	}{
		{"ปกติผ่าน", normal, req(2, bkk(2026, 10, 10, 19, 0), bkk(2026, 10, 10, 20, 0)), now, nil},
		{"end <= start", normal, req(2, bkk(2026, 10, 10, 19, 0), bkk(2026, 10, 10, 19, 0)), now, ErrInvalidRange},
		{"ไม่ลง :00/:30", normal, req(2, bkk(2026, 10, 10, 19, 15), bkk(2026, 10, 10, 20, 15)), now, ErrNotOnHalfHour},
		{"ยาวเกิน 4 ชม.", normal, req(2, bkk(2026, 10, 10, 12, 0), bkk(2026, 10, 10, 16, 30)), bkk(2026, 10, 10, 9, 0), ErrBadDuration},
		{"4) เวลาที่ผ่านไปแล้ว", normal, req(2, bkk(2026, 10, 10, 17, 0), bkk(2026, 10, 10, 18, 0)), now, ErrInPast},
		{"5) lead time อีก 20 นาที → ปฏิเสธ", overnight, req(2, bkk(2026, 10, 10, 18, 0), bkk(2026, 10, 10, 19, 0)), bkk(2026, 10, 10, 17, 40), ErrTooLateToBook},
		{"5) lead time อีก 30 นาทีพอดี → ผ่าน", overnight, req(2, bkk(2026, 10, 10, 18, 0), bkk(2026, 10, 10, 19, 0)), now, nil},
		{"เกิน 90 วัน", normal, req(2, bkk(2027, 1, 20, 12, 0), bkk(2027, 1, 20, 13, 0)), now, ErrTooFarAhead},
		{"จำนวนคน 0", normal, req(0, bkk(2026, 10, 10, 19, 0), bkk(2026, 10, 10, 20, 0)), now, ErrInvalidParty},
		{"10) party > seats", normal, req(11, bkk(2026, 10, 10, 19, 0), bkk(2026, 10, 10, 20, 0)), now, ErrPartyTooLarge},
		{"party = seats พอดี → ผ่าน", normal, req(10, bkk(2026, 10, 10, 19, 0), bkk(2026, 10, 10, 20, 0)), now, nil},
		{"6) ข้ามคืน คร่อมเที่ยงคืน → ผ่าน", overnight, req(2, bkk(2026, 10, 10, 23, 30), bkk(2026, 10, 11, 0, 30)), now, nil},
		{"6) ข้ามคืน เลยตีสอง → ปฏิเสธ", overnight, req(2, bkk(2026, 10, 11, 1, 30), bkk(2026, 10, 11, 2, 30)), now, ErrOutsideHours},
		{"7) 24 ชม. คร่อมเที่ยงคืน → ผ่าน", allDay, req(2, bkk(2026, 10, 10, 23, 30), bkk(2026, 10, 11, 0, 30)), now, nil},
		{"8) คาบเกี่ยวเวลาปิด → ปฏิเสธ", normal, req(2, bkk(2026, 10, 10, 21, 30), bkk(2026, 10, 10, 22, 30)), now, ErrOutsideHours},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidateRequest(c.r, c.h, 10, c.now)
			if c.want == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, c.want)
			}
		})
	}
}

func TestCanCancel(t *testing.T) {
	start := bkk(2026, 10, 10, 18, 0)
	assert.True(t, CancelDeadline(start, 30).Equal(bkk(2026, 10, 10, 17, 30)))
	assert.True(t, CanCancel(start, 30, bkk(2026, 10, 10, 17, 0)), "ก่อนเส้นตาย")
	assert.True(t, CanCancel(start, 30, bkk(2026, 10, 10, 17, 30)), "9) ตรงเส้นตายพอดี → ยังยกเลิกได้")
	assert.False(t, CanCancel(start, 30, bkk(2026, 10, 10, 17, 31)), "9) ช้ากว่าเส้นตาย → ปฏิเสธ")
	assert.False(t, CanCancel(start, 60, bkk(2026, 10, 10, 17, 30)), "ร้านที่ให้ยกเลิกก่อน 60 นาที")
}
```

- [ ] **Step 2: รันให้ fail** — FAIL `undefined: ValidateRequest`

- [ ] **Step 3: เขียนโค้ด** `rules.go`
```go
package booking

import "time"

const (
	SlotLength  = 30 * time.Minute    // ช่วงเวลาจองละ 30 นาที
	LeadTime    = 30 * time.Minute    // ต้องจองล่วงหน้าอย่างน้อย 30 นาที
	MinDuration = 30 * time.Minute
	MaxDuration = 4 * time.Hour       // กฎเราเอง — กันจองยาวผิดปกติ
	MaxAdvance  = 90 * 24 * time.Hour // จองล่วงหน้าได้ไม่เกิน 90 วัน
)

// Request คือคำขอจองที่แปลงเป็นเวลาจริงแล้ว (ผ่าน Hours.At)
type Request struct {
	StartAt   time.Time
	EndAt     time.Time
	PartySize int
}

// ValidateRequest ตรวจกฎข้อ 5.2 ที่ไม่ต้องใช้ฐานข้อมูล (ข้อ 2–6, 9, 10)
// กฎที่ต้องอ่าน booking อื่น (ข้อ 7, 8) ตรวจใน service ภายใต้ FOR UPDATE
func ValidateRequest(req Request, h Hours, seats int, now time.Time) error {
	if !req.EndAt.After(req.StartAt) {
		return ErrInvalidRange
	}
	if !onHalfHour(req.StartAt) || !onHalfHour(req.EndAt) {
		return ErrNotOnHalfHour
	}
	if d := req.EndAt.Sub(req.StartAt); d < MinDuration || d > MaxDuration {
		return ErrBadDuration
	}
	if !req.StartAt.After(now) {
		return ErrInPast
	}
	if req.StartAt.Before(now.Add(LeadTime)) {
		return ErrTooLateToBook
	}
	if req.StartAt.After(now.Add(MaxAdvance)) {
		return ErrTooFarAhead
	}
	if req.PartySize < 1 {
		return ErrInvalidParty
	}
	if req.PartySize > seats {
		return ErrPartyTooLarge
	}
	if !h.Fits(req.StartAt, req.EndAt) {
		return ErrOutsideHours
	}
	return nil
}

// onHalfHour: เวลาไทยต่างจาก UTC เป็นชั่วโมงเต็ม นาทีจึงเท่ากันทุก timezone ที่เราใช้
func onHalfHour(t time.Time) bool {
	return t.Minute()%30 == 0 && t.Second() == 0 && t.Nanosecond() == 0
}

// CancelDeadline คือเวลาสุดท้ายที่ยังยกเลิก/แก้ไขได้
func CancelDeadline(startAt time.Time, cancelBeforeMinutes int) time.Time {
	return startAt.Add(-time.Duration(cancelBeforeMinutes) * time.Minute)
}

// CanCancel: ยกเลิกได้เมื่อ now <= start - cancel_before (ตรงเส้นตายพอดียังได้)
func CanCancel(startAt time.Time, cancelBeforeMinutes int, now time.Time) bool {
	return !now.After(CancelDeadline(startAt, cancelBeforeMinutes))
}
```

- [ ] **Step 4: รันให้ผ่าน** — PASS

- [ ] **Step 5: Commit** — `git commit -m "เพิ่มกฎตรวจคำขอจองและเส้นตายการยกเลิก"`

---

### Task 5: ตารางเวลาว่างของวันทำการ (`Slots`)

**Files:**
- Modify: `api/internal/booking/availability.go`
- Test: `api/internal/booking/availability_test.go`

**Interfaces:**
- Consumes: `Hours.Window`, `maxConcurrent`, `SlotLength`, `LeadTime`
- Produces:
  - `type Slot struct{ StartAt, EndAt time.Time; Available int }`
  - `func Slots(h Hours, seats int, bookings []Booking, date, now time.Time) []Slot`

- [ ] **Step 1: เขียนเทสต์ (ข้อ 9 เคส 13, 14 + Review Focus 4)**

ต่อท้าย `availability_test.go`
```go
func TestSlots(t *testing.T) {
	longAgo := bkk(2026, 10, 1, 0, 0)

	t.Run("13) ข้ามคืน date=10 → 18:00 วันที่ 10 ถึงช่วงสุดท้าย 01:30 วันที่ 11", func(t *testing.T) {
		s := Slots(overnight, 10, nil, bkk(2026, 10, 10, 0, 0), longAgo)
		require.Len(t, s, 16)
		assert.True(t, s[0].StartAt.Equal(bkk(2026, 10, 10, 18, 0)))
		assert.True(t, s[15].StartAt.Equal(bkk(2026, 10, 11, 1, 30)))
		assert.True(t, s[15].EndAt.Equal(bkk(2026, 10, 11, 2, 0)))
	})

	t.Run("14) ข้ามคืน date=11 → ไม่มีช่วงตี 0–2 ของเช้าวันที่ 11", func(t *testing.T) {
		s := Slots(overnight, 10, nil, bkk(2026, 10, 11, 0, 0), longAgo)
		for _, slot := range s {
			assert.False(t, slot.StartAt.Before(bkk(2026, 10, 11, 18, 0)), "เจอช่วง %s ซึ่งเป็นของรอบวันที่ 10", slot.StartAt)
		}
	})

	t.Run("ที่ว่างต่อช่วงคิดจาก peak ภายในช่วง", func(t *testing.T) {
		b := bk(4, bkk(2026, 10, 11, 0, 30), bkk(2026, 10, 11, 1, 30))
		s := Slots(overnight, 10, []Booking{b}, bkk(2026, 10, 10, 0, 0), longAgo)
		byStart := map[string]int{}
		for _, slot := range s {
			byStart[slot.StartAt.Format("02 15:04")] = slot.Available
		}
		assert.Equal(t, 10, byStart["11 00:00"])
		assert.Equal(t, 6, byStart["11 00:30"])
		assert.Equal(t, 6, byStart["11 01:00"])
		assert.Equal(t, 10, byStart["11 01:30"])
	})

	t.Run("วันนี้ตัดช่วงที่เริ่มก่อน now+30 นาที แต่เก็บช่วงที่ตรงพอดี", func(t *testing.T) {
		s := Slots(overnight, 10, nil, bkk(2026, 10, 10, 0, 0), bkk(2026, 10, 10, 18, 30))
		require.NotEmpty(t, s)
		assert.True(t, s[0].StartAt.Equal(bkk(2026, 10, 10, 19, 0)))
	})

	t.Run("24 ชม. มี 48 ช่วง", func(t *testing.T) {
		assert.Len(t, Slots(allDay, 10, nil, bkk(2026, 10, 10, 0, 0), longAgo), 48)
	})
}
```

- [ ] **Step 2: รันให้ fail** — FAIL `undefined: Slots`

- [ ] **Step 3: เขียนโค้ด** ต่อท้าย `availability.go`
```go
// Slot คือช่วงเวลา 30 นาทีหนึ่งช่วงในตารางเวลาว่าง
type Slot struct {
	StartAt   time.Time
	EndAt     time.Time
	Available int // ที่นั่งว่าง = seats - คนสูงสุดภายในช่วงนี้
}

// Slots คืนทุกช่วง 30 นาทีของรอบวันทำการ date
// bookings = booking 'active' ที่ทับกับรอบนั้น; ช่วงที่เริ่มก่อน now+LeadTime จองไม่ได้แล้วจึงไม่อยู่ในลิสต์
func Slots(h Hours, seats int, bookings []Booking, date, now time.Time) []Slot {
	opensAt, closesAt := h.Window(date)
	earliest := now.Add(LeadTime)
	var slots []Slot
	for t := opensAt; t.Before(closesAt); t = t.Add(SlotLength) {
		if t.Before(earliest) {
			continue
		}
		end := t.Add(SlotLength)
		peak, _ := maxConcurrent(bookings, t, end)
		slots = append(slots, Slot{StartAt: t, EndAt: end, Available: seats - peak})
	}
	return slots
}
```

- [ ] **Step 4: รันทั้ง package + vet** — `go vet ./... && go test ./internal/booking/... -v` → PASS ทุกเทสต์

- [ ] **Step 5: Commit** — `git commit -m "เพิ่มตารางเวลาว่างต่อวันทำการ (Slots)"`
