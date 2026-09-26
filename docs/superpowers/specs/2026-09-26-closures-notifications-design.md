# ปิดร้านชั่วคราว + Notification — design

## ที่มา
ร้านต้องปิดได้ทั้งแบบแจ้งล่วงหน้า (หยุดยาว, ปิดปรับปรุง) และแบบกะทันหัน (ไฟดับ, เชฟป่วย)
ตอนนี้มีแค่วันปิดประจำสัปดาห์ (`closed_weekdays`) และช่วงพัก — ปิดเฉพาะวัน/ช่วงเวลาไม่ได้
และเมื่อร้านยกเลิกการจอง (ตอนลบร้าน) ลูกค้าไม่มีทางรู้เลยว่าใครยกเลิกและเพราะอะไร

สิ่งที่ตัดสินแล้ว:
- ปิดได้ **เป็นช่วงเวลา** (บางช่วงของวัน, ทั้งวัน, หลายวัน) — เก็บเป็นช่วงเวลาจริงรูปแบบเดียว
- ช่วงปิดทับการจองที่มีอยู่ → **แสดงรายการก่อน แล้วเจ้าของร้านยืนยัน** จึงยกเลิก
- **Notification ในเว็บ** (กระดิ่ง) — แจ้งลูกค้าเมื่อร้านยกเลิก + แจ้งเจ้าของร้านเมื่อลูกค้าจอง/แก้/ยกเลิก
- สร้าง notification **ใน service ทรานแซกชันเดียวกับงานนั้น** (ไม่ใช้ DB trigger หรือ event bus)

ไม่ทำ (บอกไว้ใน README เป็นงานต่อยอด): อีเมล/LINE, WebSocket, เตือนก่อนถึงเวลาจอง (ต้องมี scheduler)

## 1. ข้อมูล

migration ใหม่ 3 ไฟล์ (Up + Down ทุกไฟล์):

```sql
-- 00008_create_restaurant_closures.sql
CREATE TABLE restaurant_closures (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    restaurant_id uuid NOT NULL REFERENCES restaurants(id) ON DELETE CASCADE,
    start_at      timestamptz NOT NULL,
    end_at        timestamptz NOT NULL,
    reason        text NOT NULL CHECK (reason <> ''),
    created_at    timestamptz NOT NULL DEFAULT now(),
    CHECK (end_at > start_at)
);
CREATE INDEX ON restaurant_closures (restaurant_id, end_at);

-- 00009_add_booking_cancel_info.sql
ALTER TABLE bookings
    ADD COLUMN cancelled_by  text CHECK (cancelled_by IN ('customer', 'restaurant')),  -- null = ยังไม่ยกเลิก (หรือข้อมูลเก่าก่อนมีคอลัมน์นี้)
    ADD COLUMN cancel_reason text NOT NULL DEFAULT '';                                 -- เหตุผลจากร้าน

-- 00010_create_notifications.sql
CREATE TABLE notifications (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         uuid NOT NULL REFERENCES users(id),        -- ผู้รับ
    kind            text NOT NULL CHECK (kind IN ('booking_created', 'booking_updated', 'booking_cancelled', 'booking_cancelled_by_restaurant')),
    booking_id      uuid NOT NULL REFERENCES bookings(id),
    restaurant_id   uuid NOT NULL REFERENCES restaurants(id),
    -- snapshot ตอนเกิดเหตุ: การจองอาจถูกแก้ภายหลัง แต่ notification ต้องบอกสิ่งที่เกิดตอนนั้น
    restaurant_name text NOT NULL,
    customer_name   text NOT NULL,
    business_date   date NOT NULL,         -- ใช้ทำลิงก์ไปบอร์ด owner (ร้านข้ามคืน วันทำการ ≠ วันปฏิทิน)
    start_at        timestamptz NOT NULL,
    end_at          timestamptz NOT NULL,
    party_size      int NOT NULL,
    reason          text NOT NULL DEFAULT '',
    read_at         timestamptz,
    created_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ON notifications (user_id, created_at DESC);
```

- booking ที่ยกเลิกไปก่อนมีคอลัมน์ `cancelled_by` คงเป็น `null` → หน้าเว็บแสดง "ยกเลิกแล้ว" แบบเดิม (ไม่ backfill เพราะไม่รู้จริงว่าใครยกเลิก)
- ข้อความ notification **ไม่เก็บในตาราง** — หน้าเว็บประกอบข้อความไทยจาก `kind` (แนวเดียวกับแปล error code)

## 2. กติกาช่วงปิด

### 2.1 ตรวจร่วมกันทั้งระบบ (package `booking`)
```go
type Closure struct { ID, RestaurantID uuid.UUID; StartAt, EndAt time.Time; Reason string }

// closureAt คืนช่วงปิดตัวแรกที่ทับ [start,end) — nil ถ้าไม่ทับ (จบตอนเริ่มปิดพอดี = ไม่ทับ)
func closureAt(closures []Closure, start, end time.Time) *Closure
```
อยู่ใน `businessday.go` ข้างกติกาเวลาเปิดอื่น — ห้ามเขียนเช็คช่วงปิดที่อื่น

| จุด | ทำอะไร |
|---|---|
| `checkSlot` (จอง/แก้ ภายใต้ `FOR UPDATE`) | อ่านช่วงปิดที่ทับช่วงที่ขอ → ทับ = 400 `RESTAURANT_CLOSED` `details: { reason, start_at, end_at }` — ตรวจหลัง `ValidateRequest` ก่อนกฎข้อ 8 |
| `Slots(h, seats, bookings, closures, date, now)` | ข้ามช่วงที่ทับช่วงปิด (availability, บอร์ด owner, next-available) |
| `SlotsAround(h, seats, bookings, closures, date, minute, now)` | ช่วงที่ทับช่วงปิด → `closed: true` |
| list ร้าน `?date=` | ดึงช่วงปิดของทุกร้านในหน้าด้วย **query เดียว** (`restaurant_id IN (...)`) แล้วแบ่งตามร้านใน Go — แบบเดียวกับ booking |
| next-available | ดึงช่วงปิดทั้งช่วง 15 วันด้วย query เดียว; วันที่ถูกปิดได้ปุ่ม `closed` แล้วไปหาวันถัดไปเอง |

### 2.2 ตั้งช่วงปิด — `POST /restaurants/:id/closures` (owner)
body รับ **แบบใดแบบหนึ่ง** (ส่งผสม/ไม่ครบ → 400 `INVALID_CLOSURE`):

| แบบ | body | แปลงเป็นช่วงจริง |
|---|---|---|
| บางช่วงของวันทำการ | `{ date, start_time, end_time, reason }` | `Hours.At(date, start)` ถึง `Hours.At(date, end)` — รองรับร้านข้ามคืน (00:30 ของรอบวันที่ 10 = เช้าวันที่ 11) |
| ทั้งวัน / หลายวัน | `{ from_date, to_date, reason }` | `opensAt` ของ `from_date` ถึง `closesAt` ของ `to_date` (ผ่าน `Hours.Window`) |

- ทุกเวลาลง :00/:30, `end > start`, `to_date >= from_date`
- `start_at >= now ปัดลงเป็น :00/:30` (ปิดกะทันหัน "ตอนนี้" หน้าเว็บส่งเวลาปัจจุบันปัดลง)
- ยาวไม่เกิน 90 วัน, `reason` ต้องไม่ว่าง (ลูกค้าจะเห็น) ยาวไม่เกิน 200 ตัวอักษร
- ช่วงปิดซ้อนกันได้ — ระบบถือว่าปิดทั้งหมด
- แปลงวันทำการผ่าน `businessday.go` เท่านั้น — ห้ามคำนวณเวลาเอง

### 2.3 flow ยืนยัน (ภายในทรานแซกชันที่ล็อกแถวร้าน `FOR UPDATE`)
```
affected = booking active ของร้านนี้ ที่ start_at > now และทับ [start_at, end_at)   -- คนที่นั่งอยู่แล้วไม่แตะ

ถ้า affected ว่าง                                   → บันทึกช่วงปิด → 201
ถ้า set(affected.id) == set(body.confirm_booking_ids) → บันทึกช่วงปิด + ยกเลิก affected + notification → 201
ไม่งั้น                                              → 409 CLOSURE_AFFECTS_BOOKINGS
                                                      details.bookings = [{ id, code, customer_name, start_at, end_at, party_size }]
```
- ไม่ใช้ `confirm: true` — ใช้รายการ id ที่เจ้าของร้านเห็น เพื่อ **ไม่ยกเลิกการจองที่เจ้าของยังไม่เคยเห็น**
  (มีคนจองแทรกระหว่างที่ดูรายการ → ได้ 409 อีกรอบพร้อมรายการใหม่)
- ยกเลิก = `status='cancelled', cancelled_at=now, cancelled_by='restaurant', cancel_reason=reason`
- ล็อกแถวร้านก่อนอ่าน affected → คำขอจองที่มาพร้อมกันต่อคิวที่ล็อกเดียวกัน จึงไม่มีการจองหลุดเข้าไปในช่วงปิด

### 2.4 endpoint อื่น
| Method | Path | Auth | หมายเหตุ |
|---|---|---|---|
| GET | `/restaurants/:id/closures` | – | ช่วงปิดที่ `end_at > now` เรียงตามเวลาเริ่ม (หน้าร้าน + หน้า owner) |
| DELETE | `/restaurants/:id/closures/:closureId` | ✓ owner | เปิดร้านกลับ → 204; การจองที่ยกเลิกไปแล้ว **ไม่ฟื้น** (ลูกค้าได้รับแจ้งและอาจจองที่อื่นแล้ว); ไม่ใช่ของร้านนี้ → 404 |

- ไม่ใช่เจ้าของร้าน → 403 (เหมือน endpoint owner อื่น)
- **ลบร้าน** (`DELETE /restaurants/:id`): `CancelFutureBookings` เปลี่ยนเป็นยกเลิกด้วย `cancelled_by='restaurant', cancel_reason='ร้านปิดให้บริการ'` + notification ให้ลูกค้าทุกราย ในทรานแซกชันเดิม
- ลูกค้ายกเลิกเอง (`DELETE /bookings/:id`): `cancelled_by='customer'`

## 3. Notification

### 3.1 จุดที่สร้าง (ทุกจุดอยู่ในทรานแซกชันเดิมของงานนั้น)
| เหตุการณ์ | ที่ไหน | ผู้รับ | kind |
|---|---|---|---|
| ลูกค้าจองใหม่ | `booking.Service.Create` | เจ้าของร้าน | `booking_created` |
| ลูกค้าแก้การจอง | `booking.Service.Update` | เจ้าของร้าน | `booking_updated` (snapshot = เวลาใหม่) |
| ลูกค้ายกเลิก | `booking.Service.Cancel` | เจ้าของร้าน | `booking_cancelled` |
| ร้านตั้งช่วงปิดทับการจอง | `restaurant.Service.CreateClosure` | ลูกค้าแต่ละราย | `booking_cancelled_by_restaurant` |
| ลบร้าน | `restaurant.Service.Delete` | ลูกค้าแต่ละราย | `booking_cancelled_by_restaurant` |

- **ไม่แจ้งคนที่เป็นคนทำเอง**: ผู้รับ == ผู้กระทำ → ข้าม (เจ้าของร้านจองร้านตัวเอง / ยกเลิกการจองของตัวเองด้วยการปิดร้าน)
- booking และ restaurant **เขียนลงตาราง `notifications` ผ่าน repository ของตัวเอง** ในทรานแซกชันนั้น
  (แนวเดียวกับที่ booking แตะตาราง `restaurants` — บันทึกใน ARCHITECTURE.md) — ถ้าเรียกผ่าน package `notification`
  ต้องส่ง `*gorm.DB` ของทรานแซกชันข้ามแพ็กเกจ
- struct `notification.Notification` ใช้ร่วมกัน; package `notification` ไม่ import `booking`/`restaurant` → ไม่มี import cycle
- ทำไมไม่ใช้ DB trigger: logic ซ่อนใน SQL อธิบาย/เทสต์ยาก และ trigger ไม่รู้ว่าใครเป็นคนยกเลิก
- ทำไมไม่ใช้ event bus/outbox: เกินจำเป็นตอนนี้ — แต่เป็นทางต่อยอดไปอีเมล/LINE (worker อ่านตาราง notifications แล้วส่งต่อ)

### 3.2 API — domain ใหม่ `internal/notification` (handler / service / repository / model / dto / error)
| Method | Path | Auth | หมายเหตุ |
|---|---|---|---|
| GET | `/me/notifications` | ✓ | `{ items: [...20 ล่าสุด], unread_count }` |
| PUT | `/me/notifications/:id/read` | ✓ | ของคนอื่น/ไม่มี → **404** (ไม่บอกว่ามีอยู่จริง) → 204 |
| PUT | `/me/notifications/read-all` | ✓ | → 204 |

`user_id` มาจาก token เท่านั้น (กฎข้อ 5 ข้อศูนย์)

item: `{ id, kind, booking_id, restaurant_id, restaurant_name, customer_name, business_date, start_at, end_at, party_size, reason, read_at, created_at }`

### 3.3 BookingResponse
เพิ่ม `cancelled_by` (`"customer"` | `"restaurant"` | `null`) และ `cancel_reason`

## 4. Error code ใหม่
| code | status | details |
|---|---|---|
| `RESTAURANT_CLOSED` | 400 | `reason`, `start_at`, `end_at` |
| `INVALID_CLOSURE` | 400 | – |
| `CLOSURE_AFFECTS_BOOKINGS` | 409 | `bookings` |

## 5. Web

**กระดิ่ง (Navbar + OwnerNavbar, เมื่อ login)**
- lucide `Bell` + ตัวเลขยังไม่อ่าน, `aria-label="การแจ้งเตือน n รายการที่ยังไม่อ่าน"`, ขนาดแตะ ≥ 44×44px
- `useNotifications()` — `refetchInterval: 60_000` + `refetchOnWindowFocus`
- กดเปิดแผงรายการ + ปุ่ม "อ่านทั้งหมด"; ว่าง → "ยังไม่มีการแจ้งเตือน"
- กดรายการ → mark อ่าน → ลูกค้าไป `/bookings/:id`, เจ้าของร้านไป `/owner/bookings?restaurant=&date=<business_date>`
- ข้อความตาม `kind` (ฟังก์ชันเดียวใน `lib/`, มีป้าย "(เช้าวันที่ n)" ตามกฎ 8.4):
  - `booking_cancelled_by_restaurant` → "ร้าน{restaurant_name}ยกเลิกการจอง {วันเวลา} · {reason}"
  - `booking_created` → "{customer_name} จอง {party_size} คน {วันเวลา}"
  - `booking_updated` → "{customer_name} แก้การจองเป็น {party_size} คน {วันเวลา}"
  - `booking_cancelled` → "{customer_name} ยกเลิกการจอง {วันเวลา}"

**ลูกค้า**
- การจองที่ `cancelled_by='restaurant'` → badge "✕ ร้านยกเลิก" + เหตุผล (การจองของฉัน + หน้ายืนยัน) แทน "ยกเลิกแล้ว"
- หน้าร้าน: แถบแจ้ง "ร้านปิด {ช่วง} · {reason}" ถ้ามีช่วงปิดที่ยังไม่จบ (สี `--warn` + ไอคอน)
- แผงจอง: ช่วงปิดเป็น "ช่องว่าง" ระหว่าง slot (ใช้ `isGap`/`contiguousFrom` ตัวเดิม — ป้ายเขียนว่า "ปิด" ไม่ใช่ "พักร้าน")
- `RESTAURANT_CLOSED` ตอนจอง → กล่องสีกลาง "ร้านปิด {ช่วง} · {reason}" + refetch availability (ไม่ใช่สีแดง ไม่ใช่ "เต็ม")

**เจ้าของร้าน — หน้าใหม่ `/owner/closures?restaurant=`** (ภาษาออกแบบฝั่ง owner ข้อ 8.5)
- เลือกร้านจาก dropdown; ตารางช่วงปิดที่ยังไม่จบ + ปุ่ม "เปิดร้านกลับ" (มีกล่องยืนยัน บอกว่าการจองที่ยกเลิกไปแล้วไม่ฟื้น)
- ฟอร์มสองแท็บ: "บางช่วงของวัน" (วันทำการ + เวลาเริ่ม–จบ) / "ทั้งวันหรือหลายวัน" (จากวัน–ถึงวัน) + เหตุผล (บังคับ)
- บันทึกแล้วได้ 409 → แสดงตารางการจองที่จะถูกยกเลิก (เลขที่, ลูกค้า, เวลา, คน) + ยอดรวม
  + ปุ่มสีแดง "ยืนยันปิดร้านและยกเลิก n การจอง" (แดงเพราะทำลายข้อมูล) → ส่งซ้ำพร้อม `confirm_booking_ids`
  ได้ 409 อีกรอบ → แทนตารางด้วยรายการใหม่ + ข้อความ "มีการจองเข้ามาใหม่ระหว่างนี้"
- บอร์ดรายวัน: ปุ่ม "ปิดร้านตอนนี้" → ไปหน้านี้พร้อมกรอก วันทำการ = วันที่ดูอยู่, เวลาเริ่ม = ตอนนี้ปัดลง, เวลาจบ = เวลาปิดร้าน
- OwnerNavbar มีลิงก์ "ปิดร้านชั่วคราว"

**`proxy.ts`**: `/owner/*` กันไว้อยู่แล้ว ไม่ต้องแก้

## 6. Seed
- ร้านบุฟเฟ่ต์: ช่วงปิดทั้งวัน 5 วันข้างหน้า "ปิดปรับปรุงร้าน" (ร้านนี้มีเฉพาะ booking ที่ยกเลิกแล้ว จึงไม่ทับการจอง active)
- `customer1`: booking ที่ร้านยกเลิก 1 รายการ (`cancelled_by='restaurant'`, เหตุผล "ไฟดับทั้งซอย") + notification ยังไม่อ่าน
- `owner1`/`owner2`: notification `booking_created` ของ booking ล่วงหน้าที่ seed มีอยู่แล้ว (ยังไม่อ่าน 1–2 รายการ)
→ login แล้วเห็นตัวเลขบนกระดิ่งทันที

## 7. Test

Go unit (ไม่ใช้ DB)
- `closureAt`: ทับ / จบตอนเริ่มปิดพอดีไม่ทับ / หลายช่วงเลือกตัวแรกที่ทับ / ว่าง
- `Slots` ข้ามช่วงปิด; `SlotsAround` ได้ `closed`
- แปลง input: บางช่วงของร้านข้ามคืน (00:30 → วันถัดไป), ทั้งวันหลายวัน (`opensAt` วันแรก → `closesAt` วันสุดท้าย), ผสมสองแบบ/ไม่ครบ/ไม่ลง :30/ยาวเกิน 90 วัน/เหตุผลว่าง → `INVALID_CLOSURE`

service (mock ด้วย mockery)
- ไม่มีการจองทับ → บันทึกเลย; มีการจองทับไม่ส่ง confirm → `CLOSURE_AFFECTS_BOOKINGS` พร้อมรายการ
- confirm ตรง → ยกเลิก + notification ครบทุกราย; confirm ไม่ตรง (มีการจองใหม่แทรก) → 409 ใหม่ ไม่ยกเลิกอะไร
- การจองที่เริ่มแล้วไม่อยู่ใน affected
- จอง/แก้/ยกเลิก → notification ถึงเจ้าของร้าน; เจ้าของร้านจองร้านตัวเอง → ไม่มี notification
- ลบร้าน → ยกเลิกด้วย `cancelled_by='restaurant'` + notification

integration (testcontainers)
- ปิดร้านพร้อม confirm → ช่วงปิด + booking ถูกยกเลิก + notification อยู่ครบ (ทรานแซกชันเดียว)
- จองทับช่วงปิด → `RESTAURANT_CLOSED`
- list ร้าน `?date=` ที่มีช่วงปิด → ปุ่มเวลาได้ `closed` (query ช่วงปิดครั้งเดียวต่อหน้า)
- `ListMine` notification เรียงล่าสุดก่อน + `unread_count` ถูก; `read-all` แตะเฉพาะของตัวเอง

handler (httptest)
- `/me/notifications` ไม่มี token → 401; mark อ่านของคนอื่น → 404
- POST closures โดยไม่ใช่เจ้าของ → 403

web (Vitest)
- ข้อความตาม `kind` ครบ 4 แบบ + ป้าย "(เช้าวันที่ n)"
- badge การจอง: `cancelled_by` = restaurant / customer / null
- กระดิ่ง: ตัวเลขยังไม่อ่าน + aria-label

E2E
- เจ้าของร้านปิดช่วงที่มีการจอง → เห็นรายการ → ยืนยัน → ลูกค้า login เห็นกระดิ่ง + badge "ร้านยกเลิก"

## 8. เอกสาร
- CLAUDE.md: ข้อ 4 (schema 3 ตาราง/คอลัมน์), 4.1 (seed), 5.1 (กติกาปิดร้าน), 5.4 (ช่วงปิดเป็นส่วนของกติกาเวลาเปิด), 5.5 (`cancelled_by`),
  6 (endpoint + error code), 7 (หน้าใหม่ + กระดิ่ง), 9 (test), 11 (คำถามสัมภาษณ์: ทำไมยืนยันด้วยรายการ id, ทำไม notification อยู่ในทรานแซกชัน, ทำไม polling ไม่ใช่ WebSocket)
- ARCHITECTURE.md: domain `notification` + การเขียนตาราง notifications ข้ามโดเมนโดยตั้งใจ
- README: ฟีเจอร์ใหม่ + งานต่อยอด (อีเมล/LINE ผ่าน outbox, เตือนก่อนถึงเวลา, WebSocket)
