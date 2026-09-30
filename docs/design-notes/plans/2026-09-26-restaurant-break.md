# ช่วงพักร้าน Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** ร้านตั้ง "ช่วงพัก" ได้ 1 ช่วงภายในรอบเปิด (เช่น 11:00–22:00 พัก 14:00–17:00) และจองทับช่วงพักไม่ได้ทั้งที่ API และหน้าเว็บ

**Architecture:** เพิ่มช่วงพักเข้า `booking.Hours` แล้วให้ `Fits` / `Slots` / `OpenAtMinute` ใน `businessday.go` + `availability.go` รู้จักช่วงพัก — ทุกที่ที่เรียกฟังก์ชันเหล่านี้ (จอง, แก้, การ์ด, next-available, ย่นเวลา, บอร์ด owner) ได้กติกาใหม่เองโดยไม่ต้องเขียนซ้ำ นิยามวันทำการ (`Window`, `At`, `BusinessDate`) ไม่เปลี่ยน หน้าเว็บรู้ว่าตรงไหนเป็นช่วงพักจาก slot สองตัวที่ต่อกันไม่สนิท

**Tech Stack:** Go + Gin + GORM + goose (api), Next.js + TanStack Query + react-hook-form + zod + Vitest (web)

**Spec:** `docs/design-notes/specs/2026-09-26-restaurant-break-design.md`

## Global Constraints

- เก็บเป็น `break_start_minute`, `break_end_minute` (int, not null, default 0, check 0–1439); `start == end` = ไม่มีช่วงพัก — ไม่ใช้ NULL
- ช่วงพักต้องอยู่ข้างในรอบ: นับนาทีจากเวลาเปิด `0 < offset(start) < offset(end) < duration`; ผิด → 400 `INVALID_BREAK`
- API: `break_start` / `break_end` เป็น `"HH:MM"` ลง :00/:30, ส่งคู่หรือไม่ส่งเลย; response เป็น `""` เมื่อไม่มีช่วงพัก
- รูปแบบ response ของ availability / บอร์ด owner ไม่เปลี่ยน
- กติกาเวลาเปิดอยู่ใน `businessday.go` ที่เดียว (CLAUDE.md ข้อ 12) — ห้ามเขียนเช็คช่วงพักที่อื่น
- ข้อความ commit เป็นภาษาไทย ลงท้ายด้วย `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`
- ห้ามใช้อีโมจิเป็นไอคอน, ตัวอักษรไทย ≥ 13px (CLAUDE.md ข้อ 8)

## Review Focus

1. **ค้นหาด้วยเวลาที่ตกช่วงพัก** (ค้น 15:30 ที่ร้านพัก 15:00–17:00) → next-available ต้องเสนอช่วงว่างแรก ๆ ไม่ใช่ตอบ "เต็ม 14 วัน" — เพราะ `NextAvailable` ใช้ `OpenAtMinute` ตัดสินว่าร้านเปิดตอนนั้นไหม → เทสต์ใน Task 1 (`TestOpenAtMinuteBreak`)
2. **แผงจองเลือกเวลาก่อนพักแล้วระยะเวลาคร่อมช่วงพัก** (เลือก 13:30 นาน 2 ชม.) → ต้องขึ้นเตือนและกดจองไม่ได้ ไม่ใช่ส่ง 13:30–18:30 ไป — เพราะโค้ดเดิมตัด slot ด้วย index ต่อกัน → เทสต์ใน Task 4 (`contiguousFrom`)
3. **แก้ร้านแล้วช่วงพักไม่ถูกบันทึก / ลบช่วงพักไม่ได้** — `repository.Update` ใช้ `Select` รายคอลัมน์ ลืมใส่ = ค่าหายเงียบ ๆ → เทสต์ integration ใน Task 3
4. **จองในช่วงพักผ่าน API ตรง** — `booking.RestaurantInfo` อ่านคอลัมน์ผ่าน `restaurantColumns`; ลืมใส่ = `Hours` ไม่มีช่วงพักแล้วจองผ่าน → เทสต์ integration ใน Task 3
5. **ร้าน 24 ชม. จองคร่อมเที่ยงคืนเข้าช่วงพักของรอบถัดไป** → ต้องไม่ผ่าน → เทสต์ใน Task 1 (`TestFitsWithBreak`)

---

## File Structure

| ไฟล์ | หน้าที่ในงานนี้ |
|---|---|
| `api/internal/booking/businessday.go` | `Hours` + ช่วงพัก: `HasBreak`, `ValidBreak`, `Break`, `overlapsBreak`, `offset`; แก้ `Fits`, `OpenAtMinute` |
| `api/internal/booking/availability.go` | `Slots` ข้ามช่วงที่ทับช่วงพัก |
| `api/internal/booking/repository.go` | `RestaurantInfo` + `restaurantColumns` อ่านช่วงพัก |
| `api/internal/booking/error.go` | ข้อความ `ErrOutsideHours` รวมช่วงพัก |
| `api/migrations/00007_add_break.sql` | คอลัมน์ใหม่ (Up/Down) |
| `api/internal/restaurant/model.go`, `service.go`, `repository.go` | เก็บ/อัปเดตช่วงพัก |
| `api/internal/restaurant/dto.go`, `handler.go` | รับ/ส่ง `break_start`/`break_end`, `INVALID_BREAK` |
| `api/cmd/seed/main.go` | ซูชิ ทาคุมิ พัก 15:00–17:00 |
| `web/lib/format.ts` | `hoursLabel`, `isGap`, `contiguousFrom` |
| `web/lib/types.ts`, `web/services/restaurants.ts`, `web/lib/errors.ts` | type + ข้อความ error |
| `web/components/owner/RestaurantForm.tsx` | ช่องช่วงพัก |
| `web/components/booking/BookingPanel.tsx` | เส้นแบ่งช่วงพัก + ไม่ให้เลือกคร่อมพัก |
| `web/app/owner/bookings/page.tsx` | แถว "พักร้าน" ในบอร์ด |
| `web/components/restaurant/RestaurantCard.tsx`, `web/app/(customer)/restaurants/[id]/page.tsx`, `web/app/owner/restaurants/page.tsx` | ป้ายเวลาเปิดผ่าน `hoursLabel` |
| `CLAUDE.md`, `ARCHITECTURE.md`, `README.md` | เอกสาร |

---

### Task 1: กติกาช่วงพักใน `Hours`

**Files:**
- Modify: `api/internal/booking/businessday.go`
- Test: `api/internal/booking/businessday_test.go`

**Interfaces:**
- Produces (package `booking`):
  - `Hours.BreakStartMinute int`, `Hours.BreakEndMinute int`
  - `func (h Hours) HasBreak() bool`
  - `func (h Hours) ValidBreak() bool`
  - `func (h Hours) Break(date time.Time) (start, end time.Time, ok bool)`
  - `func (h Hours) overlapsBreak(date, start, end time.Time) bool` (unexported — Task 2 ใช้)
  - `func (h Hours) offset(minuteOfDay int) int` (unexported)
  - `Fits` และ `OpenAtMinute` รู้จักช่วงพัก

- [ ] **Step 1: เขียนเทสต์ที่ยังไม่ผ่าน** — ต่อท้าย `api/internal/booking/businessday_test.go`

```go
// ช่วงพักร้าน: 11:00–22:00 พัก 14:00–17:00 / ข้ามคืน 18:00–02:00 พัก 23:00–00:00 / 24 ชม. พัก 03:00–05:00
var (
	lunchDinner    = Hours{OpenMinute: 11 * 60, CloseMinute: 22 * 60, BreakStartMinute: 14 * 60, BreakEndMinute: 17 * 60}
	overnightBreak = Hours{OpenMinute: 18 * 60, CloseMinute: 2 * 60, BreakStartMinute: 23 * 60, BreakEndMinute: 0}
	allDayBreak    = Hours{OpenMinute: 0, CloseMinute: 0, BreakStartMinute: 3 * 60, BreakEndMinute: 5 * 60}
)

func TestValidBreak(t *testing.T) {
	withBreak := func(h Hours, start, end int) Hours { h.BreakStartMinute, h.BreakEndMinute = start, end; return h }
	cases := []struct {
		name string
		h    Hours
		want bool
	}{
		{"ไม่มีช่วงพัก", normal, true},
		{"พักกลางรอบ", lunchDinner, true},
		{"ข้ามคืน พัก 23:00–00:00", overnightBreak, true},
		{"24 ชม. พัก 03:00–05:00", allDayBreak, true},
		{"เริ่มพักพร้อมเวลาเปิด = แค่เปิดช้าลง", withBreak(normal, 11*60, 12*60), false},
		{"พักจนถึงเวลาปิด = แค่ปิดเร็วขึ้น", withBreak(normal, 21*60, 22*60), false},
		{"พักนอกเวลาเปิด", withBreak(normal, 8*60, 9*60), false},
		{"เวลาจบพักมาก่อนเวลาเริ่มพัก", withBreak(normal, 17*60, 14*60), false},
		{"ข้ามคืน พักหลังตีสอง (นอกรอบ)", withBreak(overnight, 3*60, 4*60), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, c.h.ValidBreak())
		})
	}
}

func TestBreak(t *testing.T) {
	_, _, ok := normal.Break(bkk(2026, 10, 10, 0, 0))
	assert.False(t, ok, "ไม่มีช่วงพัก")

	start, end, ok := overnightBreak.Break(bkk(2026, 10, 10, 0, 0))
	require.True(t, ok)
	assert.True(t, start.Equal(bkk(2026, 10, 10, 23, 0)), "start = %s", start)
	assert.True(t, end.Equal(bkk(2026, 10, 11, 0, 0)), "end = %s — จบพักเที่ยงคืนของวันถัดไป", end)
}

func TestFitsWithBreak(t *testing.T) {
	cases := []struct {
		name       string
		h          Hours
		start, end time.Time
		want       bool
	}{
		{"จบพอดีตอนเริ่มพัก", lunchDinner, bkk(2026, 10, 10, 13, 0), bkk(2026, 10, 10, 14, 0), true},
		{"คร่อมเข้าช่วงพัก", lunchDinner, bkk(2026, 10, 10, 13, 30), bkk(2026, 10, 10, 14, 30), false},
		{"อยู่ในช่วงพัก", lunchDinner, bkk(2026, 10, 10, 14, 30), bkk(2026, 10, 10, 15, 0), false},
		{"ครอบทั้งช่วงพัก", lunchDinner, bkk(2026, 10, 10, 13, 0), bkk(2026, 10, 10, 18, 0), false},
		{"เริ่มพอดีตอนจบพัก", lunchDinner, bkk(2026, 10, 10, 17, 0), bkk(2026, 10, 10, 18, 0), true},
		{"ข้ามคืน ก่อนพัก", overnightBreak, bkk(2026, 10, 10, 22, 0), bkk(2026, 10, 10, 23, 0), true},
		{"ข้ามคืน คร่อมพัก 23:30–00:30", overnightBreak, bkk(2026, 10, 10, 23, 30), bkk(2026, 10, 11, 0, 30), false},
		{"ข้ามคืน หลังพัก 00:00–01:00", overnightBreak, bkk(2026, 10, 11, 0, 0), bkk(2026, 10, 11, 1, 0), true},
		{"24 ชม. อยู่ในช่วงพัก", allDayBreak, bkk(2026, 10, 10, 4, 0), bkk(2026, 10, 10, 4, 30), false},
		{"24 ชม. คร่อมเที่ยงคืน จบพอดีตอนรอบถัดไปเริ่มพัก", allDayBreak, bkk(2026, 10, 10, 23, 0), bkk(2026, 10, 11, 3, 0), true},
		{"24 ชม. คร่อมเที่ยงคืน เข้าช่วงพักของรอบถัดไป", allDayBreak, bkk(2026, 10, 10, 23, 0), bkk(2026, 10, 11, 3, 30), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, c.h.Fits(c.start, c.end))
		})
	}
}

// NextAvailable ใช้ OpenAtMinute ตัดสินว่า "ร้านเปิดตอนเวลาที่ค้นไหม" — ค้นตอนพักต้องได้ false
// ไม่งั้นจะไปหาช่วงรอบเวลาพักของวันถัด ๆ ไป (ปิดหมด) แล้วตอบว่าเต็ม 14 วัน
func TestOpenAtMinuteBreak(t *testing.T) {
	assert.True(t, lunchDinner.OpenAtMinute(13*60+30))
	assert.False(t, lunchDinner.OpenAtMinute(14*60), "เริ่มพัก")
	assert.False(t, lunchDinner.OpenAtMinute(15*60+30))
	assert.True(t, lunchDinner.OpenAtMinute(17*60), "จบพักแล้ว")
	assert.False(t, overnightBreak.OpenAtMinute(23*60+30))
	assert.True(t, overnightBreak.OpenAtMinute(0))
}
```

- [ ] **Step 2: รันให้เห็นว่าไม่ผ่าน**

Run: `cd api && go test ./internal/booking/ -run 'TestValidBreak|TestBreak|TestFitsWithBreak|TestOpenAtMinuteBreak'`
Expected: FAIL — compile error `unknown field BreakStartMinute in struct literal of type Hours`

- [ ] **Step 3: แก้ `businessday.go`**

เพิ่มฟิลด์ใน `Hours` (ต่อจาก `ClosedWeekdays`):

```go
	// ช่วงพักภายในรอบ (นาทีจากเที่ยงคืนเวลาไทย) เช่น 11:00–22:00 พัก 14:00–17:00
	// start == end แปลว่าไม่มีช่วงพัก — ใช้ 0/0 เป็นค่าเริ่มต้น ไม่ต้องใช้ pointer และ Hours ยังเทียบด้วย != ได้
	BreakStartMinute int
	BreakEndMinute   int
```

แทน `OpenAtMinute` เดิมทั้งฟังก์ชันด้วยชุดนี้ (วางต่อจาก `Is24h`):

```go
// offset นับนาทีจากเวลาเปิดวนรอบ 24 ชม. — ใช้เทียบทุกอย่างในรอบ (เวลาเปิด, ช่วงพัก) ด้วยวิธีเดียวกัน
// ร้าน 18:00–02:00: 23:00 → 300, 01:00 → 420
func (h Hours) offset(minuteOfDay int) int {
	return (minuteOfDay - h.OpenMinute + minutesPerDay) % minutesPerDay
}

func (h Hours) HasBreak() bool { return h.BreakStartMinute != h.BreakEndMinute }

// ValidBreak: ช่วงพักต้องอยู่ข้างในรอบจริง ๆ — ติดขอบเวลาเปิดหรือปิด = แค่ย่นเวลาเปิด ให้ไปแก้เวลาเปิด–ปิดแทน
func (h Hours) ValidBreak() bool {
	if !h.HasBreak() {
		return true
	}
	start, end := h.offset(h.BreakStartMinute), h.offset(h.BreakEndMinute)
	return 0 < start && start < end && end < h.DurationMinutes()
}

// Break คืนช่วงพักจริงของรอบวันทำการ date — ok=false ถ้าร้านไม่มีช่วงพัก
func (h Hours) Break(date time.Time) (start, end time.Time, ok bool) {
	if !h.HasBreak() {
		return time.Time{}, time.Time{}, false
	}
	opensAt, _ := h.Window(date)
	start = opensAt.Add(time.Duration(h.offset(h.BreakStartMinute)) * time.Minute)
	end = opensAt.Add(time.Duration(h.offset(h.BreakEndMinute)) * time.Minute)
	return start, end, true
}

// overlapsBreak บอกว่าช่วง [start,end) ทับช่วงพักของรอบวันทำการ date ไหม (จบตอนเริ่มพักพอดี = ไม่ทับ)
func (h Hours) overlapsBreak(date, start, end time.Time) bool {
	breakStart, breakEnd, ok := h.Break(date)
	return ok && start.Before(breakEnd) && end.After(breakStart)
}

// OpenAtMinute บอกว่าเวลาบนนาฬิกานี้ร้านเปิดอยู่ไหม (อยู่ในรอบ ไม่ว่าจะก่อนหรือหลังเที่ยงคืน และไม่ใช่ช่วงพัก)
func (h Hours) OpenAtMinute(minuteOfDay int) bool {
	o := h.offset(minuteOfDay)
	if o >= h.DurationMinutes() {
		return false
	}
	return !h.HasBreak() || o < h.offset(h.BreakStartMinute) || o >= h.offset(h.BreakEndMinute)
}
```

แก้ `Fits` ทั้งฟังก์ชัน:

```go
// Fits บอกว่าช่วง [start,end) อยู่ในเวลาเปิดของร้านไหม: start ต้องอยู่ในรอบที่ไม่ใช่วันปิด ไม่ทับช่วงพัก และจบไม่เกินรอบนั้น
// ยกเว้นร้าน 24 ชม. ที่รอบถัดไปเริ่มทันทีที่รอบนี้จบ → คร่อมได้ถ้ารอบถัดไปไม่ใช่วันปิดและไม่ทับช่วงพักของรอบถัดไป
func (h Hours) Fits(start, end time.Time) bool {
	date, ok := h.BusinessDate(start)
	if !ok || h.ClosedOn(date) || h.overlapsBreak(date, start, end) {
		return false
	}
	_, closesAt := h.Window(date)
	if !end.After(closesAt) {
		return true
	}
	next := date.AddDate(0, 0, 1)
	return h.Is24h() && !h.ClosedOn(next) && !h.overlapsBreak(next, start, end) && !end.After(closesAt.Add(24*time.Hour))
}
```

- [ ] **Step 4: รันเทสต์ทั้ง package ให้ผ่าน** (รวมเทสต์เดิม — `TestOpenAtMinute` เดิมต้องยังผ่าน)

Run: `cd api && go test ./internal/booking/`
Expected: `ok  	jongyoung/internal/booking`

- [ ] **Step 5: Commit**

```bash
git add api/internal/booking/businessday.go api/internal/booking/businessday_test.go
git commit -m "เพิ่มช่วงพักร้านใน Hours: Fits และ OpenAtMinute ไม่นับช่วงพักเป็นเวลาเปิด

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 2: ตารางเวลาว่างข้ามช่วงพัก

**Files:**
- Modify: `api/internal/booking/availability.go` (ฟังก์ชัน `Slots`)
- Test: `api/internal/booking/availability_test.go`, `api/internal/booking/changes_test.go`

**Interfaces:**
- Consumes: `Hours.overlapsBreak(date, start, end time.Time) bool`, fixture `lunchDinner` (Task 1, อยู่ใน `businessday_test.go` package เดียวกัน)
- Produces: `Slots` ไม่คืนช่วงที่ทับช่วงพัก (availability, next-available, บอร์ด owner ได้ผลนี้เอง)

- [ ] **Step 1: เขียนเทสต์**

ต่อท้าย `availability_test.go`:

```go
func TestSlotsSkipBreak(t *testing.T) {
	longAgo := bkk(2026, 10, 1, 0, 0)
	s := Slots(lunchDinner, 10, nil, bkk(2026, 10, 10, 0, 0), longAgo)
	require.Len(t, s, 16, "11:00–14:00 = 6 ช่วง + 17:00–22:00 = 10 ช่วง")
	assert.True(t, s[5].StartAt.Equal(bkk(2026, 10, 10, 13, 30)))
	assert.True(t, s[5].EndAt.Equal(bkk(2026, 10, 10, 14, 0)))
	assert.True(t, s[6].StartAt.Equal(bkk(2026, 10, 10, 17, 0)), "ถัดจาก 13:30 คือ 17:00 — ช่วงพักไม่อยู่ในลิสต์")
}
```

ต่อท้าย `changes_test.go` (สองเทสต์นี้ผ่านตั้งแต่ Task 1 เพราะใช้ `Fits` — ใส่ไว้ตรึงพฤติกรรม):

```go
func TestSlotsAroundBreak(t *testing.T) {
	longAgo := bkk(2026, 10, 1, 0, 0)
	s := SlotsAround(lunchDinner, 10, nil, bkk(2026, 10, 10, 0, 0), 14*60, longAgo)
	require.Len(t, s, 5)
	closed := []bool{s[0].Closed, s[1].Closed, s[2].Closed, s[3].Closed, s[4].Closed}
	assert.Equal(t, []bool{false, false, true, true, true}, closed, "13:00 13:30 เปิด / 14:00 14:30 15:00 พัก")
}

func TestCheckHoursChangeAddBreak(t *testing.T) {
	lunch := bk(2, bkk(2026, 10, 10, 14, 30), bkk(2026, 10, 10, 15, 30))
	lunch.ID = uuid.New()
	dinner := bk(2, bkk(2026, 10, 10, 18, 0), bkk(2026, 10, 10, 19, 0))
	dinner.ID = uuid.New()

	err := CheckHoursChange([]Booking{lunch, dinner}, lunchDinner)
	var e *HoursConflictError
	require.True(t, errors.As(err, &e), "ตั้งช่วงพักทับ booking ในอนาคต → 409 HOURS_CONFLICT_EXISTING_BOOKINGS")
	assert.Equal(t, []uuid.UUID{lunch.ID}, e.BookingIDs)
}
```

- [ ] **Step 2: รันให้เห็นว่าไม่ผ่าน**

Run: `cd api && go test ./internal/booking/ -run 'TestSlotsSkipBreak|TestSlotsAroundBreak|TestCheckHoursChangeAddBreak'`
Expected: `TestSlotsSkipBreak` FAIL (`"[22 elements]" should have 16 item(s)`); อีกสองตัว PASS

- [ ] **Step 3: แก้ loop ใน `Slots`** (`availability.go`)

```go
// Slots คืนทุกช่วง 30 นาทีของรอบวันทำการ date
// bookings = booking 'active' ที่ทับกับรอบนั้น; ช่วงที่เริ่มก่อน now+LeadTime จองไม่ได้แล้ว และช่วงพักของร้าน จึงไม่อยู่ในลิสต์
// วันปิดประจำสัปดาห์ไม่มีรอบ → คืนลิสต์ว่าง
func Slots(h Hours, seats int, bookings []Booking, date, now time.Time) []Slot {
	if h.ClosedOn(date) {
		return nil
	}
	opensAt, closesAt := h.Window(date)
	earliest := now.Add(LeadTime)
	var slots []Slot
	for t := opensAt; t.Before(closesAt); t = t.Add(SlotLength) {
		end := t.Add(SlotLength)
		if t.Before(earliest) || h.overlapsBreak(date, t, end) {
			continue
		}
		peak, _ := maxConcurrent(bookings, t, end)
		slots = append(slots, Slot{StartAt: t, EndAt: end, Available: seats - peak})
	}
	return slots
}
```

- [ ] **Step 4: รันทั้ง package**

Run: `cd api && go test ./internal/booking/`
Expected: `ok`

- [ ] **Step 5: Commit**

```bash
git add api/internal/booking/availability.go api/internal/booking/availability_test.go api/internal/booking/changes_test.go
git commit -m "ตารางเวลาว่างข้ามช่วงพักร้าน + เทสต์ปุ่มเวลาบนการ์ดและการตั้งช่วงพักทับ booking

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 3: เก็บ/รับ/ส่งช่วงพักผ่าน DB และ API

**Files:**
- Create: `api/migrations/00007_add_break.sql`
- Modify: `api/internal/restaurant/model.go`, `api/internal/restaurant/service.go` (`apply`), `api/internal/restaurant/repository.go` (`Update`), `api/internal/restaurant/dto.go`, `api/internal/restaurant/handler.go` (`bindInput`), `api/internal/booking/repository.go` (`RestaurantInfo`, `restaurantColumns`), `api/internal/booking/error.go`, `api/cmd/seed/main.go`
- Test: `api/internal/restaurant/dto_test.go`, `api/internal/restaurant/repository_integration_test.go`, `api/internal/booking/concurrency_integration_test.go`
- Regenerate: `api/docs/*` ด้วย `./dev.sh docs`

**Interfaces:**
- Consumes: `booking.Hours{…, BreakStartMinute, BreakEndMinute}`, `Hours.HasBreak()`, `Hours.ValidBreak()` (Task 1)
- Produces: JSON `break_start` / `break_end` (string `"HH:MM"` หรือ `""`) ใน request และ response ของร้าน; error code `INVALID_BREAK` (400) — Task 5 ใช้

- [ ] **Step 1: เขียนเทสต์ DTO** — ต่อท้าย `api/internal/restaurant/dto_test.go`

```go
func TestToInputBreak(t *testing.T) {
	req := RestaurantRequest{OpenTime: "11:00", CloseTime: "22:00"}

	in, err := req.ToInput()
	require.NoError(t, err)
	assert.False(t, in.Hours().HasBreak(), "ไม่ส่งมา = ไม่มีช่วงพัก")

	req.BreakStart, req.BreakEnd = "14:00", "17:00"
	in, err = req.ToInput()
	require.NoError(t, err)
	assert.Equal(t, 14*60, in.BreakStartMinute)
	assert.Equal(t, 17*60, in.BreakEndMinute)

	for _, bad := range [][2]string{
		{"14:00", ""},      // ส่งตัวเดียว
		{"", "17:00"},      // ส่งตัวเดียว
		{"14:00", "14:00"}, // เท่ากัน
		{"11:00", "12:00"}, // ติดเวลาเปิด
		{"21:00", "22:00"}, // ติดเวลาปิด
		{"08:00", "09:00"}, // นอกเวลาเปิด
		{"14:15", "17:00"}, // ไม่ลง :00/:30
	} {
		req.BreakStart, req.BreakEnd = bad[0], bad[1]
		_, err := req.ToInput()
		assert.ErrorIs(t, err, errInvalidBreak, "%v", bad)
	}
}

func TestNewRestaurantResponseBreak(t *testing.T) {
	r := Restaurant{OpenMinute: 11 * 60, CloseMinute: 22 * 60}
	resp := NewRestaurantResponse(r)
	assert.Empty(t, resp.BreakStart)
	assert.Empty(t, resp.BreakEnd)

	r.BreakStartMinute, r.BreakEndMinute = 15*60, 17*60
	resp = NewRestaurantResponse(r)
	assert.Equal(t, "15:00", resp.BreakStart)
	assert.Equal(t, "17:00", resp.BreakEnd)
}
```

- [ ] **Step 2: เขียนเทสต์ integration** (Postgres จริง — Review Focus ข้อ 3, 4)

ต่อท้าย `api/internal/restaurant/repository_integration_test.go`:

```go
func TestRepositoryUpdateBreak(t *testing.T) {
	db := testdb.New(t)
	repo := NewRepository(db)
	ctx := context.Background()
	r := createRestaurant(t, repo, createOwner(t, db), "ร้านมีช่วงพัก", 0, 0)

	r.BreakStartMinute, r.BreakEndMinute = 15*60, 17*60
	require.NoError(t, repo.Update(ctx, &r))
	got, err := repo.FindByID(ctx, r.ID)
	require.NoError(t, err)
	assert.Equal(t, 15*60, got.BreakStartMinute, "ช่วงพักต้องถูกบันทึกตอนแก้ไข")
	assert.Equal(t, 17*60, got.BreakEndMinute)

	r.BreakStartMinute, r.BreakEndMinute = 0, 0
	require.NoError(t, repo.Update(ctx, &r))
	got, err = repo.FindByID(ctx, r.ID)
	require.NoError(t, err)
	assert.False(t, got.Hours().HasBreak(), "ลบช่วงพัก (0/0) ต้องบันทึกค่าศูนย์ได้ด้วย")
}
```

ต่อท้าย `api/internal/booking/concurrency_integration_test.go`:

```go
// ช่วงพักต้องถูกอ่านจาก DB ตอนจองด้วย (restaurantColumns) — ไม่งั้นจองช่วงพักผ่าน API ได้
func TestBookingInBreakRejected(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	now := func() time.Time { return bkk(2026, 10, 1, 12, 0) }
	svc := NewService(NewRepository(db), now)
	rid := insertRestaurant(t, db, insertUser(t, db), 10) // 11:00–22:00
	require.NoError(t, db.Exec(`UPDATE restaurants SET break_start_minute = 900, break_end_minute = 1020 WHERE id = ?`, rid).Error)

	inBreak := Choice{Date: bkk(2026, 10, 10, 0, 0), StartMinute: 15 * 60, EndMinute: 16 * 60, PartySize: 2}
	_, err := svc.Create(ctx, insertUser(t, db), rid, inBreak)
	assert.ErrorIs(t, err, ErrOutsideHours)

	afterBreak := Choice{Date: bkk(2026, 10, 10, 0, 0), StartMinute: 17 * 60, EndMinute: 18 * 60, PartySize: 2}
	_, err = svc.Create(ctx, insertUser(t, db), rid, afterBreak)
	assert.NoError(t, err)
}
```

- [ ] **Step 3: รันให้เห็นว่าไม่ผ่าน**

Run: `cd api && go test ./internal/restaurant/ ./internal/booking/ -run 'Break'`
Expected: FAIL — compile error `req.BreakStart undefined` / `r.BreakStartMinute undefined`

- [ ] **Step 4: Migration** — สร้าง `api/migrations/00007_add_break.sql`

```sql
-- +goose Up
-- ช่วงพักร้านภายในรอบเปิด (เช่น 11:00–22:00 พัก 14:00–17:00) เป็นนาทีจากเที่ยงคืนเวลาไทย
-- start == end (ค่าเริ่มต้น 0/0) = ไม่มีช่วงพัก
-- เงื่อนไข "อยู่ข้างในรอบ" ตรวจที่ Go (Hours.ValidBreak) เพราะต้องรู้ความยาวรอบซึ่งขึ้นกับเปิดข้ามคืน/24 ชม.
ALTER TABLE restaurants
    ADD COLUMN break_start_minute int NOT NULL DEFAULT 0 CHECK (break_start_minute BETWEEN 0 AND 1439),
    ADD COLUMN break_end_minute   int NOT NULL DEFAULT 0 CHECK (break_end_minute BETWEEN 0 AND 1439);

-- +goose Down
ALTER TABLE restaurants DROP COLUMN break_start_minute, DROP COLUMN break_end_minute;
```

- [ ] **Step 5: Model / service / repository ของ restaurant**

`api/internal/restaurant/model.go` — ใน `Restaurant` และ `Input` เพิ่มต่อจาก `ClosedWeekdays`:

```go
	BreakStartMinute    int // ช่วงพัก (ดู booking.Hours) — เท่ากัน = ไม่มีช่วงพัก
	BreakEndMinute      int
```

และแก้ทั้งสอง `Hours()`:

```go
func (r Restaurant) Hours() booking.Hours {
	return booking.Hours{OpenMinute: r.OpenMinute, CloseMinute: r.CloseMinute, ClosedWeekdays: r.ClosedWeekdays,
		BreakStartMinute: r.BreakStartMinute, BreakEndMinute: r.BreakEndMinute}
}
```

```go
func (in Input) Hours() booking.Hours {
	return booking.Hours{OpenMinute: in.OpenMinute, CloseMinute: in.CloseMinute, ClosedWeekdays: in.ClosedWeekdays,
		BreakStartMinute: in.BreakStartMinute, BreakEndMinute: in.BreakEndMinute}
}
```

`api/internal/restaurant/service.go` — ใน `apply` ต่อจาก `rest.ClosedWeekdays = in.ClosedWeekdays`:

```go
	rest.BreakStartMinute = in.BreakStartMinute
	rest.BreakEndMinute = in.BreakEndMinute
```

`api/internal/restaurant/repository.go` — `Update`:

```go
func (r *repository) Update(ctx context.Context, rest *Restaurant) error {
	return r.db.WithContext(ctx).Model(rest).Select(
		"name", "description", "cuisine", "address", "map_url", "seats",
		"open_minute", "close_minute", "closed_weekdays", "break_start_minute", "break_end_minute",
		"cancel_before_minutes", "updated_at",
	).Updates(rest).Error
}
```

- [ ] **Step 6: booking repository อ่านช่วงพัก** (`api/internal/booking/repository.go`)

ใน `RestaurantInfo` ต่อจาก `ClosedWeekdays int`:

```go
	BreakStartMinute    int
	BreakEndMinute      int
```

```go
func (r RestaurantInfo) Hours() Hours {
	return Hours{OpenMinute: r.OpenMinute, CloseMinute: r.CloseMinute, ClosedWeekdays: r.ClosedWeekdays,
		BreakStartMinute: r.BreakStartMinute, BreakEndMinute: r.BreakEndMinute}
}
```

```go
const restaurantColumns = "id, owner_id, name, address, seats, open_minute, close_minute, closed_weekdays, break_start_minute, break_end_minute, cancel_before_minutes"
```

`api/internal/booking/error.go` — ข้อความของ `ErrOutsideHours`:

```go
	ErrOutsideHours  = &RuleError{"OUTSIDE_OPENING_HOURS", "ช่วงที่เลือกอยู่นอกเวลาเปิด–ปิด หรือตรงกับช่วงพักของร้าน"}
```

- [ ] **Step 7: DTO** (`api/internal/restaurant/dto.go`)

ใน `RestaurantRequest` ต่อจาก `CloseTime`:

```go
	BreakStart          string   `json:"break_start" example:"14:00"` // ไม่บังคับ — ส่งคู่กับ break_end หรือไม่ส่งเลย
	BreakEnd            string   `json:"break_end" example:"17:00"`
```

ใน `var (...)` เพิ่ม:

```go
	errInvalidBreak  = errors.New("ช่วงพักต้องกรอกทั้งเวลาเริ่มและเวลาจบ (ลง :00 หรือ :30) และอยู่ภายในเวลาเปิด–ปิด ไม่ติดขอบ")
```

ใน `ToInput` — ต่อจากบรรทัดที่ parse `shut` เพิ่ม:

```go
	breakStart, breakEnd, err := parseBreak(req.BreakStart, req.BreakEnd)
	if err != nil {
		return Input{}, err
	}
```

และแทน `return Input{...}, nil` ท้ายฟังก์ชันด้วย:

```go
	in := Input{
		Name: req.Name, Description: req.Description, Cuisine: req.Cuisine, Address: req.Address, MapURL: mapURL,
		Seats: req.Seats, OpenMinute: open, CloseMinute: shut, ClosedWeekdays: closed,
		BreakStartMinute: breakStart, BreakEndMinute: breakEnd,
		CancelBeforeMinutes: req.CancelBeforeMinutes,
	}
	if !in.Hours().ValidBreak() {
		return Input{}, errInvalidBreak
	}
	return in, nil
}

// parseBreak: ไม่ส่งทั้งคู่ = ไม่มีช่วงพัก (0, 0); ส่งต้องครบคู่ ลง :00/:30 และไม่เท่ากัน
// (ส่วน "อยู่ข้างในรอบ" ตรวจด้วย Hours.ValidBreak ตัวเดียวกับที่ booking ใช้)
func parseBreak(start, end string) (int, int, error) {
	if start == "" && end == "" {
		return 0, 0, nil
	}
	s, errStart := ParseClock(start)
	e, errEnd := ParseClock(end)
	if errStart != nil || errEnd != nil || s == e {
		return 0, 0, errInvalidBreak
	}
	return s, e, nil
}
```

ใน `RestaurantResponse` ต่อจาก `CloseTime`:

```go
	BreakStart          string          `json:"break_start" example:"15:00"` // "" = ไม่มีช่วงพัก
	BreakEnd            string          `json:"break_end" example:"17:00"`
```

ใน `NewRestaurantResponse` ก่อน `return`:

```go
	breakStart, breakEnd := "", ""
	if r.Hours().HasBreak() {
		breakStart, breakEnd = formatClock(r.BreakStartMinute), formatClock(r.BreakEndMinute)
	}
```

และใน struct literal ต่อจาก `CloseTime: formatClock(r.CloseMinute),` เพิ่ม `BreakStart: breakStart, BreakEnd: breakEnd,`

- [ ] **Step 8: Handler** (`api/internal/restaurant/handler.go` ใน `bindInput`) ต่อจากบล็อก `errInvalidMapURL`:

```go
	if errors.Is(err, errInvalidBreak) {
		httputil.Abort(c, http.StatusBadRequest, "INVALID_BREAK", err.Error(), nil)
		return uuid.Nil, Input{}, req, false
	}
```

- [ ] **Step 9: รันเทสต์ทั้งหมดของ api** (ต้องมี Docker สำหรับ testcontainers)

Run: `cd api && go test ./...`
Expected: ทุก package `ok`

- [ ] **Step 10: Seed** (`api/cmd/seed/main.go`) ต่อจากบรรทัดสร้าง `sushi`:

```go
	// ร้านนี้มีช่วงพัก 15:00–17:00 (booking ใน seed ของร้านนี้คือ 19:00–20:30 จึงไม่ทับ)
	s.exec(`UPDATE restaurants SET break_start_minute = ?, break_end_minute = ? WHERE id = ?`, 15*60, 17*60, sushi)
```

แก้คอมเมนต์ของร้านนี้เป็น `// 2) เคสปกติ + รีวิวเยอะ ★4.8 จาก 46 รีวิว + มีช่วงพักบ่าย`

Run: `cd api && go build ./... && go vet ./...`
Expected: ไม่มี output

- [ ] **Step 11: Swagger**

Run: `./dev.sh docs`
Expected: `api/docs/docs.go`, `swagger.yaml`, `openapi.yaml` มี `break_start` / `break_end`
(ตรวจ: `grep -c break_start api/docs/openapi.yaml` ได้ ≥ 2)

- [ ] **Step 12: Commit**

```bash
git add api/migrations/00007_add_break.sql api/internal api/cmd/seed/main.go api/docs
git commit -m "เก็บช่วงพักร้านใน DB + รับ/ส่ง break_start, break_end ผ่าน API (INVALID_BREAK) + seed ร้านซูชิพักบ่าย

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 4: ตัวช่วยฝั่งเว็บ (`lib/format.ts`)

**Files:**
- Modify: `web/lib/format.ts`, `web/lib/types.ts`
- Test: `web/lib/format.test.ts`

**Interfaces:**
- Produces:
  - `Restaurant.break_start: string`, `Restaurant.break_end: string` ใน `lib/types.ts`
  - `hoursLabel(r: Pick<Restaurant, "open_time" | "close_time" | "open_24h" | "break_start" | "break_end">): string`
  - `isGap(prev: { end_at: string } | undefined, next: { start_at: string }): boolean`
  - `contiguousFrom<T extends { start_at: string; end_at: string }>(slots: T[], start: number, n: number): T[]`

- [ ] **Step 1: เพิ่ม type** — `web/lib/types.ts` ใน `Restaurant` ต่อจาก `close_time`:

```ts
  break_start: string; // "" = ไม่มีช่วงพัก
  break_end: string;
```

- [ ] **Step 2: เขียนเทสต์** — ต่อท้าย `web/lib/format.test.ts` และเพิ่ม `contiguousFrom, hoursLabel, isGap` ใน import บรรทัดบนสุด

```ts
describe("hoursLabel", () => {
  const base = { open_time: "11:00", close_time: "22:00", open_24h: false, break_start: "", break_end: "" };
  it("ไม่มีช่วงพัก", () => {
    expect(hoursLabel(base)).toBe("11:00–22:00");
  });
  it("มีช่วงพัก", () => {
    expect(hoursLabel({ ...base, break_start: "15:00", break_end: "17:00" })).toBe("11:00–22:00 · พัก 15:00–17:00");
  });
  it("24 ชม.", () => {
    expect(hoursLabel({ ...base, open_time: "00:00", close_time: "00:00", open_24h: true })).toBe("เปิด 24 ชม.");
  });
});

describe("ช่วงพักในลิสต์ slot", () => {
  const slot = (start: string, end: string) => ({ start_at: `2026-10-10T${start}:00+07:00`, end_at: `2026-10-10T${end}:00+07:00` });
  // 13:00 13:30 | พัก | 17:00 17:30
  const slots = [slot("13:00", "13:30"), slot("13:30", "14:00"), slot("17:00", "17:30"), slot("17:30", "18:00")];

  it("isGap: ต่อกันสนิท = ไม่ใช่ช่วงพัก, ไม่ต่อ = ช่วงพัก, ตัวแรก = ไม่ใช่", () => {
    expect(isGap(slots[0], slots[1])).toBe(false);
    expect(isGap(slots[1], slots[2])).toBe(true);
    expect(isGap(undefined, slots[0])).toBe(false);
  });
  it("contiguousFrom: เลือก 13:00 นาน 1 ชม. → 2 ช่วงครบ", () => {
    expect(contiguousFrom(slots, 0, 2)).toEqual([slots[0], slots[1]]);
  });
  it("contiguousFrom: เลือก 13:30 นาน 2 ชม. → หยุดที่ช่วงพัก ได้ช่วงเดียว (ห้ามกระโดดไป 17:00)", () => {
    expect(contiguousFrom(slots, 1, 4)).toEqual([slots[1]]);
  });
  it("contiguousFrom: เลยท้ายลิสต์ → ได้เท่าที่มี", () => {
    expect(contiguousFrom(slots, 3, 2)).toEqual([slots[3]]);
  });
});
```

- [ ] **Step 3: รันให้เห็นว่าไม่ผ่าน**

Run: `cd web && npx vitest run lib/format.test.ts`
Expected: FAIL — `hoursLabel is not a function` (หรือ import error)

- [ ] **Step 4: เขียนฟังก์ชัน** — ใน `web/lib/format.ts` ต่อจาก `closedDaysLabel`:

```ts
/** "11:00–22:00", "11:00–22:00 · พัก 15:00–17:00", "เปิด 24 ชม." — ป้ายเวลาเปิดของร้านทุกที่ใช้ตัวนี้ */
export function hoursLabel(r: { open_time: string; close_time: string; open_24h: boolean; break_start: string; break_end: string }): string {
  const base = r.open_24h ? "เปิด 24 ชม." : `${r.open_time}–${r.close_time}`;
  return r.break_start ? `${base} · พัก ${r.break_start}–${r.break_end}` : base;
}

/**
 * slot ถัดไปต่อจาก slot ก่อนหน้าสนิทไหม — ไม่ต่อ = มีช่วงพักร้านคั่น
 * (API ตัดช่วงพักออกจากลิสต์; ช่วงที่เลย lead time หายจากต้นลิสต์เท่านั้น จึงไม่ทำให้เกิดช่องว่างกลางลิสต์)
 */
export const isGap = (prev: { end_at: string } | undefined, next: { start_at: string }) =>
  !!prev && new Date(prev.end_at).getTime() !== new Date(next.start_at).getTime();

/** slot ที่ต่อกันสนิทนับจาก index start ไม่เกิน n ช่วง — หยุดเมื่อเจอช่วงพักหรือหมดลิสต์ */
export function contiguousFrom<T extends { start_at: string; end_at: string }>(slots: T[], start: number, n: number): T[] {
  const out: T[] = [];
  for (let i = start; i < slots.length && out.length < n; i++) {
    if (out.length > 0 && isGap(slots[i - 1], slots[i])) break;
    out.push(slots[i]);
  }
  return out;
}
```

- [ ] **Step 5: รันเทสต์ web ทั้งหมด**

Run: `cd web && npm test`
Expected: ผ่านทั้งหมด

- [ ] **Step 6: Commit**

```bash
git add web/lib/format.ts web/lib/format.test.ts web/lib/types.ts
git commit -m "เว็บ: ป้ายเวลาเปิดพร้อมช่วงพัก + ตัวช่วยหาช่วงพักจาก slot ที่ต่อกันไม่สนิท

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 5: หน้าเว็บใช้ช่วงพัก

**Files:**
- Modify: `web/services/restaurants.ts`, `web/lib/errors.ts`, `web/components/owner/RestaurantForm.tsx`, `web/components/booking/BookingPanel.tsx`, `web/app/owner/bookings/page.tsx`, `web/components/restaurant/RestaurantCard.tsx`, `web/app/(customer)/restaurants/[id]/page.tsx`, `web/app/owner/restaurants/page.tsx`

**Interfaces:**
- Consumes: `hoursLabel`, `isGap`, `contiguousFrom` (Task 4); JSON `break_start`/`break_end`, code `INVALID_BREAK` (Task 3)

> ก่อนแก้: อ่าน `web/AGENTS.md` — Next.js เวอร์ชันนี้ต่างจากที่คุ้น ไฟล์ที่แก้ในงานนี้เป็น client component/JSX ธรรมดา ไม่แตะ API ของ Next

- [ ] **Step 1: input type + ข้อความ error**

`web/services/restaurants.ts` ใน `RestaurantInput` ต่อจาก `close_time`:

```ts
  break_start: string; // "" = ไม่มีช่วงพัก
  break_end: string;
```

`web/lib/errors.ts` ต่อจาก `INVALID_OPENING_HOURS`:

```ts
  INVALID_BREAK: "ช่วงพักต้องกรอกทั้งเวลาเริ่มและจบ และอยู่ภายในเวลาเปิด–ปิด ไม่ติดขอบ",
```

และแก้ `OUTSIDE_OPENING_HOURS` (บรรทัด 15) เป็น:

```ts
  OUTSIDE_OPENING_HOURS: "ช่วงที่เลือกอยู่นอกเวลาเปิด–ปิด หรือตรงกับช่วงพักของร้าน",
```

- [ ] **Step 2: ฟอร์มร้าน** (`RestaurantForm.tsx`)

ใน `schema` ต่อจาก `close_time: z.string(),`:

```ts
  break_start: z.string(), // "" = ไม่มีช่วงพัก
  break_end: z.string(),
```

แล้วต่อท้าย `z.object({...})` ด้วย refine (ก่อน `;`):

```ts
}).refine((v) => (v.break_start === "") === (v.break_end === ""), { path: ["break_end"], message: "กรอกช่วงพักให้ครบทั้งเวลาเริ่มและจบ หรือเว้นว่างทั้งคู่" });
```

ใน `defaultValues` ของร้านใหม่เพิ่ม `break_start: "", break_end: ""` (ของร้านเดิมได้จาก `...restaurant` อยู่แล้ว)

ต่อจากช่อง "ปิด *" เพิ่ม:

```tsx
        {field("พักตั้งแต่ (ไม่บังคับ)", <select {...register("break_start")} className={input}><option value="">ไม่มีช่วงพัก</option>{clocks.map((c) => <option key={c}>{c}</option>)}</select>)}
        {field("ถึง", <select {...register("break_end")} className={input}><option value="">ไม่มีช่วงพัก</option>{clocks.map((c) => <option key={c}>{c}</option>)}</select>, errors.break_end?.message)}
```

และแก้ข้อความใต้ช่องเวลาเป็น:

```tsx
        <p className="text-muted sm:col-span-2">ปิดน้อยกว่าเปิด = ข้ามเที่ยงคืน (เช่น 18:00–02:00) · เปิดเท่ากับปิด = 24 ชม. · ช่วงพัก = รับจองไม่ได้ช่วงนั้น เช่น เปิด 11:00–22:00 พัก 14:00–17:00</p>
```

- [ ] **Step 3: ป้ายเวลาเปิด 3 ที่** — import `hoursLabel` จาก `@/lib/format` ในแต่ละไฟล์

`RestaurantCard.tsx`:
```tsx
            <span>· {hoursLabel(r)}</span>
```

`app/(customer)/restaurants/[id]/page.tsx`:
```tsx
  const hours = `${hoursLabel(restaurant)}${restaurant.overnight ? " (ถึงเช้าวันถัดไป)" : ""}`;
```

`app/owner/restaurants/page.tsx` (บรรทัด 54):
```tsx
                  <td className={td}>{hoursLabel(r)}{r.overnight && <span className="text-muted"> (ข้ามคืน)</span>}
```

- [ ] **Step 4: แผงจอง** (`BookingPanel.tsx`) — import `contiguousFrom, isGap` เพิ่ม และ `Fragment` จาก `react`

แทนบรรทัด `covered` / `beyondClose` เดิม:

```tsx
  // เอาเฉพาะ slot ที่ต่อกันสนิท — ถ้าคร่อมช่วงพัก ห้ามกระโดดข้ามไปนับหลังพัก (ไม่งั้นส่งเวลาจบผิดไปหลายชั่วโมง)
  const covered = startIndex >= 0 ? contiguousFrom(slots, startIndex, duration) : [];
  const cut = startIndex >= 0 && covered.length < duration;
  const hitsBreak = cut && startIndex + covered.length < slots.length; // ยังมี slot ต่อ แต่ต่อไม่สนิท = ชนช่วงพัก
  const beyondClose = cut && !hitsBreak;
```

แก้ `ready`:

```tsx
  const ready = covered.length === duration && !short;
```

แทน loop ของปุ่มเวลา:

```tsx
            {slots.map((s, i) => {
              const state = chipState(s, r.seats, party);
              return (
                <Fragment key={s.start_at}>
                  {isGap(slots[i - 1], s) && (
                    <p className="col-span-4 border-t border-dashed border-border-strong pt-2 text-[13px] text-muted tabular">
                      พักร้าน {fmtTime(slots[i - 1].end_at)}–{fmtTime(s.start_at)}
                    </p>
                  )}
                  <TimeChip time={fmtTime(s.start_at)} state={state} note={chipNote(state, s.available)}
                    dayLabel={nextDayLabel(s.start_at, date)} selected={i === startIndex}
                    inRange={startIndex >= 0 && i > startIndex && i < startIndex + covered.length}
                    onClick={() => { setStart(fmtTime(s.start_at)); setError(null); }} />
                </Fragment>
              );
            })}
```

ต่อจาก Alert `beyondClose` เพิ่ม:

```tsx
      {hitsBreak && <Alert tone="warn" title="ชนช่วงพักร้าน">ลดระยะเวลา หรือเลือกเวลาเริ่มหลังช่วงพัก</Alert>}
```

และ Alert `short` เปลี่ยนเงื่อนไขเป็น `{short && !cut && (`

- [ ] **Step 5: บอร์ด owner** (`app/owner/bookings/page.tsx`) — import `isGap` และ `Fragment`

แทน loop ของ seat bar:

```tsx
                {board.data.slots.map((s, i, all) => {
                  const ratio = seats ? s.booked / seats : 0;
                  const color = ratio >= 1 ? "bg-full" : ratio >= 0.7 ? "bg-warn" : "bg-ok";
                  return (
                    <Fragment key={s.start_at}>
                      {isGap(all[i - 1], s) && (
                        <li className="grid h-6 grid-cols-[92px_1fr] items-center gap-2.5 text-muted tabular">
                          <span>{fmtTime(all[i - 1].end_at)}–{fmtTime(s.start_at)}</span><span>พักร้าน</span>
                        </li>
                      )}
                      <li className="grid h-6 grid-cols-[92px_1fr_48px] items-center gap-2.5 tabular">
                        <span className="text-muted">{fmtTime(s.start_at)} <span className="text-[12px]">{nextDayLabel(s.start_at, date) ? "+1" : ""}</span></span>
                        <span className="flex h-3 bg-chip"><span className={color} style={{ width: `${Math.min(ratio, 1) * 100}%` }} /></span>
                        <span className={`text-right ${ratio >= 0.7 ? "font-semibold" : "text-muted"}`}>{s.booked}/{seats}</span>
                      </li>
                    </Fragment>
                  );
                })}
```

- [ ] **Step 6: lint + type + test**

Run: `cd web && npm run lint && npx tsc --noEmit && npm test`
Expected: ไม่มี error, เทสต์ผ่านทั้งหมด

- [ ] **Step 7: ลองกับระบบจริง**

Run: `./dev.sh up && ./dev.sh reset`
ตรวจที่ http://jongyoung.localhost:
- การ์ด "ซูชิ ทาคุมิ" แสดง `11:00–22:00 · พัก 15:00–17:00`; ค้นเวลา 15:30 → ปุ่มเวลาช่วงพักขึ้น "ปิด" และมีกล่องช่วงที่ยังว่าง (ไม่ใช่เต็ม 14 วัน)
- หน้าร้าน: มีเส้น "พักร้าน 15:00–17:00" ในแผงจอง; เลือก 14:30 นาน 1 ชม. → เตือน "ชนช่วงพักร้าน" ปุ่มจองกดไม่ได้
- owner2 → บอร์ดการจองของร้านซูชิ มีแถว "พักร้าน"; แก้ร้านตั้งช่วงพัก 19:00–20:00 (ทับ booking 19:00–20:30 ของ customer1) → เห็น `HOURS_CONFLICT_EXISTING_BOOKINGS`

- [ ] **Step 8: Commit**

```bash
git add web
git commit -m "เว็บ: ฟอร์มตั้งช่วงพัก, ป้ายเวลาเปิด, เส้นแบ่งช่วงพักในแผงจองและบอร์ดเจ้าของร้าน

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 6: เอกสาร

**Files:**
- Modify: `CLAUDE.md`, `ARCHITECTURE.md`, `README.md`

- [ ] **Step 1: CLAUDE.md**
  - ข้อ 4 schema `restaurants` ต่อจาก `closed_weekdays`:
    ```
      break_start_minute    int  not null default 0 check (break_start_minute between 0 and 1439)
      break_end_minute      int  not null default 0 check (break_end_minute between 0 and 1439)
                                                -- ช่วงพักภายในรอบ; เท่ากัน = ไม่มีช่วงพัก; ต้องอยู่ข้างในรอบ ไม่ติดขอบ (Hours.ValidBreak)
    ```
  - ข้อ 4.1 ตาราง seed แถว "ร้านในห้าง": โชว์อะไร → `เคสปกติ + รีวิวเยอะ (46 รีวิว ★4.8) + ช่วงพัก 15:00–17:00`
  - ข้อ 5.1 ตารางเพิ่มแถว: `| ช่วงพัก (ไม่บังคับ) ต้องอยู่ข้างในเวลาเปิด–ปิด ไม่ติดขอบ ส่งคู่ start/end | ผิด → 400 INVALID_BREAK |` และในย่อหน้า "ย่นเวลา" เพิ่ม "(รวมถึงการเพิ่ม/ขยายช่วงพัก)"
  - ข้อ 5.4 ต่อท้ายหัวข้อ เพิ่ม:
    ```
    **ช่วงพัก (เปิด 2 ช่วง):** ร้านมีรอบเดียว + ช่วงพักได้ 1 ช่วง เช่น 11:00–22:00 พัก 14:00–17:00
    - วันทำการ/`Window`/`At` ไม่เปลี่ยน — ช่วงพักเป็นแค่ "รูในรอบ"
    - `Fits` ปฏิเสธช่วงจองที่ทับช่วงพัก (จบตอนเริ่มพักพอดีได้); ร้าน 24 ชม. ที่คร่อมเข้ารอบถัดไปตรวจช่วงพักของรอบถัดไปด้วย
    - `Slots` ข้ามช่วงพัก; `OpenAtMinute` ตอบ false ในช่วงพัก (ใช้ใน next-available)
    - หน้าเว็บรู้ว่าตรงไหนเป็นช่วงพักจาก slot ที่ต่อกันไม่สนิท (`isGap`) — API ไม่ต้องส่งช่วงพักแยก
    ```
  - ข้อ 6 ตาราง error code เพิ่ม `| INVALID_BREAK | 400 | – |`
  - ข้อ 9 เพิ่มในกลุ่ม `businessday_test.go`: `13b. ช่วงพัก: จบตอนเริ่มพักผ่าน / คร่อมหรืออยู่ในพักไม่ผ่าน / ข้ามคืน / 24 ชม. คร่อมเข้าพักของรอบถัดไป / ValidBreak ติดขอบไม่ผ่าน`

- [ ] **Step 2: ARCHITECTURE.md** — หาส่วนที่พูดถึงเวลาเปิด–ปิด/`businessday.go` (`grep -n "businessday\|เวลาเปิด" ARCHITECTURE.md`) แล้วเพิ่มหนึ่งประโยค: "`Hours` มีช่วงพักได้ 1 ช่วงภายในรอบ — `Fits`/`Slots`/`OpenAtMinute` ไม่นับช่วงพักเป็นเวลาเปิด ทุกที่ที่เรียกฟังก์ชันเหล่านี้ได้กติกาเดียวกัน"

- [ ] **Step 3: README.md** — หาส่วนเหตุผล/การตัดสินใจออกแบบ (`grep -n "Bayesian\|เหตุผล" README.md`) แล้วเพิ่ม:
  ```
  ### ช่วงพักร้าน vs ตารางเวลาเปิดหลายช่วง
  เลือก "รอบเดียว + ช่วงพัก 1 ช่วง" แทนตาราง `restaurant_hours` หลายช่วง/ต่อวัน เพราะครอบเคสร้านเปิดกลางวัน–เย็นที่พบบ่อยที่สุด
  โดยนิยามวันทำการไม่เปลี่ยนเลย ถ้าต้องรองรับเวลาเปิดต่างกันรายวันค่อยย้ายไปแบบตาราง

  ### เจ้าของร้านจองร้านตัวเองได้
  ต่างจากรีวิวที่ห้ามเพราะผลประโยชน์ทับซ้อน — การจองไม่มีใครเสียประโยชน์ และเจ้าของลดที่นั่งเองได้อยู่แล้ว
  ข้อจำกัด: กฎ "คนเดียวกันจองซ้อนเวลาไม่ได้" ทำให้จองแทนลูกค้าที่โทรมาสองรายในเวลาทับกันไม่ได้
  งานต่อยอด: เพิ่มช่อง "ชื่อผู้มา" ในการจอง
  ```

- [ ] **Step 4: Commit**

```bash
git add CLAUDE.md ARCHITECTURE.md README.md
git commit -m "อัปเดตเอกสาร: ช่วงพักร้าน และเหตุผลที่เจ้าของร้านจองร้านตัวเองได้

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

- [ ] **Step 5: ตรวจรวม**

Run: `./dev.sh lint && ./dev.sh test`
Expected: ผ่านทั้งหมด
