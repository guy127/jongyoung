# jongyoung

ระบบจองโต๊ะร้านอาหาร + community รีวิวร้าน — โจทย์สอบปลายภาค PEA DevPool 2026
ส่งงาน **11 ต.ค. 2026** / Technical Interview **17–18 ต.ค.** (คนละ 15 นาที)

เกณฑ์ตัดสินคือ **foundation** — ต้องอธิบายโค้ดของตัวเองได้ทุกบรรทัด
เพราะฉะนั้นเลือกทางที่ **ตรงไปตรงมาและอธิบายได้** เสมอ ห้ามใส่ abstraction หรือ library ที่อธิบายไม่ได้

โครงและรูปแบบโค้ดอ้างอิงจาก [pea-wongnok-2026 branch `06-auth-keycloak`](https://github.com/joeyprint/pea-wongnok-2026/tree/06-auth-keycloak)
แต่ **หน้าตา UI ต้องไม่เหมือน wongnok** — ใช้สไตล์ตามข้อ 8

---

## 1. Tech stack

| ส่วน | ใช้ | เหตุผล (ต้องตอบได้ในห้องสัมภาษณ์) |
|---|---|---|
| Frontend | Next.js (App Router) + TypeScript | บังคับตามโจทย์ |
| Data fetching (web) | TanStack Query + axios (`services/`) | cache/refetch/สถานะ loading ในที่เดียว; axios interceptor แนบ token |
| Auth (web) | next-auth v4 + KeycloakProvider | จัดการ OIDC code flow, session cookie, callback ให้ ไม่ต้องเขียน flow เอง |
| Backend | Go + Gin | บังคับตามโจทย์; Gin มี router + middleware + binding ครบ |
| Config (Go) | caarlos0/env + go-playground/validator + godotenv | อ่าน env ครั้งเดียวเป็น struct แล้ว validate ตอนเริ่ม |
| Database | PostgreSQL | ต้องใช้ transaction + row lock กันการจองชนกัน |
| ORM | GORM | ใช้ตอน CRUD ธรรมดา และ drop ลง raw SQL ตอน query ซับซ้อน |
| Migration | goose | เวอร์ชันคุมได้ มีทั้ง Up/Down |
| **Auth** | **Keycloak (OIDC)** | แยกเรื่อง identity ออกจาก business logic ไม่ต้องเก็บรหัสผ่านเอง ได้ refresh token/logout/social login ฟรี |
| Token verify (Go) | coreos/go-oidc | discovery + JWKS + cache + rotate key ให้หมด |
| API docs | swaggo (Swagger) | สร้างจาก comment ใน handler |
| Styling | Tailwind + shadcn/ui | |
| Test | testify + mockery + httptest + testcontainers (Go), Vitest + Testing Library (web) | repository test รันกับ Postgres จริงใน container |
| Local infra | Docker Compose (postgres, keycloak, api, web, caddy) | ผู้ตรวจรัน `docker compose up` คำสั่งเดียวได้ทั้งระบบ |
| Reverse proxy | Caddy (`*.jongyoung.localhost`) | browser และ container เห็น Keycloak ด้วยชื่อเดียวกัน → issuer ตรงกัน (ดู 2.3) |
| CI/CD | GitLab CI | ดูข้อ 13 |

**ไม่ใช้:** Redis (ไม่มี login flow ฝั่ง Go ให้ต้องเก็บ state), Redux, library ที่อธิบายไม่ได้

---

## 2. สถาปัตยกรรม Auth ด้วย Keycloak

### 2.1 การตั้งค่า Keycloak

```
Realm: jongyoung
├── Clients
│   └── jongyoung-web   — confidential client (ใช้กับ next-auth), standard flow
│                         redirect URI: http://jongyoung.localhost/api/auth/callback/keycloak
│                         post logout redirect: http://jongyoung.localhost
├── Client scope: audience mapper → access token มี aud = jongyoung-api
├── Users: บัญชีทดสอบ 3 ตัว (owner1, owner2, customer1 — ดู README) เปิด self-registration ไว้
└── Roles: ไม่ต้องมี realm role สำหรับ owner/customer — ดูข้อ 2.4
```

Export realm เป็น `keycloak/import/jongyoung-realm.json` แล้วให้ compose ใช้ `start-dev --import-realm`
→ ผู้ตรวจ `docker compose up` แล้วได้ realm + client + user ทดสอบครบทันที **สำคัญมากต่อคะแนน**
client secret ใน realm export เป็น **ค่าสำหรับ dev เท่านั้น** และตรงกับ `.env.example` — เขียนบอกใน README ชัด ๆ
วิธี export ใหม่อยู่ใน `keycloak/README.md`

### 2.2 Flow (next-auth + Bearer token จาก browser)

```
1. ผู้ใช้กด "เข้าสู่ระบบ" → signIn('keycloak') ของ next-auth
   → next-auth พาไป Keycloak (authorization code flow, จัดการ state ให้)

2. Keycloak redirect กลับ /api/auth/callback/keycloak
   → next-auth แลก code เป็น token
   → callback jwt เก็บ accessToken / refreshToken / idToken / expiresAt ใน session (cookie ของ next-auth)

3. เรียก API
   → axios interceptor: getSession() → Authorization: Bearer <accessToken>
   → ยิงตรงไป http://api.jongyoung.localhost/api/v1
   → Go ตรวจ token ด้วย go-oidc (ดู 2.3)

4. token ใกล้หมดอายุ
   → ต่ออายุใน callback jwt ของ next-auth: ถ้าเหลือ < 30 วินาที ใช้ refresh token ขอใหม่จาก Keycloak
   → refresh ไม่ผ่าน → ใส่ session.error → หน้าเว็บเรียก signOut()

5. ออกจากระบบ
   → GET /api/auth/logout (route handler) → redirect ไป Keycloak end_session พร้อม id_token_hint + ลบ cookie ของ next-auth
```

- `proxy.ts` (middleware ของ next-auth) กันหน้า `/me/*` และ `/owner/*` ถ้ายังไม่ login
- Go เปิด CORS ให้ **เฉพาะ** origin `http://jongyoung.localhost`

**ทำไมให้ browser ถือ access token (เลือกแบบนี้แทนการเก็บ token ฝั่ง server ล้วน):**
ง่ายและตรงกับแนวที่เรียนมา (next-auth + axios interceptor) — เตรียมคำตอบเรื่อง XSS ไว้:
access token อายุสั้น, React escape ข้อความให้เสมอ, ห้ามใช้ `dangerouslySetInnerHTML` กับข้อมูลผู้ใช้,
และถ้าต้องการปิดช่องนี้ทั้งหมด ทางถัดไปคือทำ BFF proxy ใน Next.js ให้ token อยู่ฝั่ง server อย่างเดียว

### 2.3 ฝั่ง Go — ตรวจ token

```go
provider, _ := oidc.NewProvider(ctx, cfg.Keycloak.RealmURL())
verifier := provider.Verifier(&oidc.Config{ClientID: "jongyoung-api"})
// ใน middleware.JWT: verifier.Verify(ctx, rawToken) → ได้ claims
```
ต้องตรวจครบ: **signature + exp + iss + aud** (go-oidc ทำให้ทั้งหมดถ้าตั้ง ClientID ตรงกับ aud)

⚠️ **issuer mismatch:** browser เห็น Keycloak ที่ `localhost:<port>` แต่ Go ในคอนเทนเนอร์เห็น `keycloak:8080`
→ `iss` ไม่ตรง → verify ไม่ผ่าน
**ทางแก้ที่ใช้: Caddy + โดเมนกลาง** `*.jongyoung.localhost` (`.localhost` resolve เป็น 127.0.0.1 เองตาม RFC 6761)
- `jongyoung.localhost` → web, `api.jongyoung.localhost` → api, `keycloak.jongyoung.localhost` → keycloak
- ใน compose ให้ caddy มี network alias ชื่อเดียวกัน → container อื่นเรียก `keycloak.jongyoung.localhost` ได้เหมือน browser
- `KEYCLOAK_BASE_URL=http://keycloak.jongyoung.localhost` ทั้งฝั่ง web และ api → issuer ตรงกันเสมอ

### 2.4 การผูก Keycloak user กับ users ในฐานข้อมูลเรา (JIT provisioning)

Keycloak เก็บ identity (ใครคือใคร) — **ฐานข้อมูลเราเก็บ domain data** (ใครเป็นเจ้าของร้านไหน)
แปลว่าเราต้องมีตาราง `users` ของตัวเอง เพราะ FK ของ restaurants/bookings/reviews ต้องชี้ไปที่นั่น

**สร้าง user ใน middleware.JWT ตอนเจอ `sub` ครั้งแรก** — ไม่ใช่ตอน login
(เพราะ login เกิดที่ next-auth ฝั่ง Go ไม่เห็น callback; wongnok สร้าง user ตอน callback ของ Go จึงเจอ 401 "user not found")

```go
// middleware.JWT หลังตรวจ token ผ่าน: upsert ผู้ใช้จาก claims แล้วใส่ local user id ลง context
user, err := userSvc.EnsureExists(ctx, claims.Subject, claims.Email, claims.Name)
rctx := reqctx.WithUserID(ctx.Request.Context(), user.ID)
```
- ผูกด้วย `sub` **ห้ามผูกด้วย email** เพราะ email เปลี่ยนได้
- `users.keycloak_uid` มี unique index — แต่ถ้าใช้ soft delete ต้องเป็น **partial unique index** (`WHERE deleted_at IS NULL`) ไม่งั้นคนที่ถูกลบแล้วล็อกอินใหม่จะชนกับแถวเก่า
- `middleware.DevAuth` (ข้าม token ตอน dev) ใช้ได้เฉพาะ `APP_ENV=development` และต้องมี log เตือน

**บทบาท owner/customer ไม่ต้องอยู่ใน Keycloak เลย** — โจทย์บอก "หนึ่งบัญชี สองบทบาท"
ความเป็นเจ้าของคือ `restaurants.owner_id == userID` ซึ่งเป็น **ข้อมูล ไม่ใช่ role** ตรวจตอน runtime ทีละคำขอ
(ถ้าจะเพิ่ม `admin` ทีหลัง ค่อยใช้ realm role — ตอนนี้ยังไม่ต้อง)

---

## 3. โครงโฟลเดอร์

```
jongyoung/
├── compose.yml                   # postgres, keycloak, api, web, caddy (secret อ่านจาก .env)
├── .env.example                  # ทุก key ที่ compose ใช้
├── caddy/Caddyfile               # *.jongyoung.localhost → web / api / keycloak
├── keycloak/
│   ├── import/jongyoung-realm.json
│   └── README.md                 # วิธี export realm ใหม่
├── api/                          # Go module "jongyoung"
│   ├── cmd/
│   │   ├── api/main.go           # run() + os.Exit(1) ถ้า config พัง
│   │   └── seed/                 # ข้อมูลตัวอย่าง (แยกจาก migration)
│   ├── internal/
│   │   ├── config/               # อ่าน env ครั้งเดียว + Validate()
│   │   ├── middleware/           # jwt.go (verify + JIT), devauth.go
│   │   ├── reqctx/               # เก็บ/อ่าน userID ใน context
│   │   ├── httputil/             # error response กลาง, pagination helper
│   │   ├── platform/database/    # เปิด GORM connection
│   │   ├── user/                 # handler, service, repository, model, dto, error
│   │   ├── restaurant/
│   │   ├── booking/
│   │   │   ├── availability.go   # ⭐ หัวใจของโจทย์
│   │   │   └── availability_test.go
│   │   └── review/
│   ├── migrations/               # goose (Up + Down ทุกไฟล์) — ห้ามใส่ seed data
│   ├── docs/                     # swagger generated + ARCHITECTURE/TESTING
│   ├── .mockery.yml              # mock ลง mocks_test.go ข้างไฟล์ interface
│   └── Dockerfile
├── web/                          # Next.js
│   ├── app/
│   │   ├── page.tsx              # รายการร้าน
│   │   ├── restaurants/[id]/
│   │   ├── me/bookings/
│   │   ├── owner/restaurants/
│   │   └── api/auth/             # [...nextauth], logout
│   ├── components/bases/         # ปุ่ม, input, badge ฯลฯ
│   ├── containers/               # Navbar, Footer
│   ├── services/                 # hook TanStack Query ต่อ resource
│   ├── lib/                      # axios.ts, auth/auth-options.ts, formatters
│   ├── proxy.ts                  # กันหน้า /me, /owner
│   └── Dockerfile
├── docs/                         # spec, UI design, ER diagram
├── .gitlab-ci.yml
├── CLAUDE.md
└── README.md
```

กฎเหล็ก: ทุก domain มีไฟล์ **handler (HTTP) → service (business logic) → repository (DB)** + `model.go` `dto.go` `error.go`
business logic อยู่ที่ service เท่านั้น handler แปลง HTTP↔struct, repository คุย DB
service รับ repository เป็น interface → mock ด้วย mockery ได้

---

## 4. Database schema

```sql
users
  id            uuid pk default gen_random_uuid()
  keycloak_uid  text not null              -- claims.sub
  email         text not null
  display_name  text not null
  created_at, updated_at, deleted_at
  -- partial unique index กัน soft delete ชนกัน
  create unique index on users(keycloak_uid) where deleted_at is null;

restaurants
  id                    uuid pk
  owner_id              uuid not null references users(id)
  name                  text not null
  description           text
  cuisine               text                          -- ใช้ filter
  seats                 int  not null check (seats > 0)
  open_minute           int  not null check (open_minute between 0 and 1439)
  close_minute          int  not null check (close_minute between 0 and 1439)
  cancel_before_minutes int  not null default 30 check (cancel_before_minutes >= 30)
  rating_sum            int  not null default 0
  rating_count          int  not null default 0
  created_at, updated_at, deleted_at
  index on (owner_id)

restaurant_images                     -- แยกตาราง เพื่อรองรับหลายรูป (เก็บเป็น URL ไม่มีระบบอัปโหลดไฟล์)
  id            uuid pk
  restaurant_id uuid not null references restaurants(id) on delete cascade
  url           text not null
  sort_order    int  not null default 0   -- 0 = รูปปก

bookings
  id            uuid pk
  restaurant_id uuid not null references restaurants(id)
  user_id       uuid not null references users(id)
  party_size    int  not null check (party_size > 0)
  start_at      timestamptz not null
  end_at        timestamptz not null
  status        text not null default 'active'   -- 'active' | 'cancelled'
  cancelled_at  timestamptz
  created_at, updated_at
  check (end_at > start_at)
  index on (restaurant_id, start_at, end_at) where status = 'active'
  index on (user_id, start_at desc)

reviews
  id            uuid pk
  restaurant_id uuid not null references restaurants(id)
  user_id       uuid not null references users(id)
  score         int  not null check (score between 1 and 5)
  body          text
  created_at, updated_at
  unique (restaurant_id, user_id)      -- 1 คน 1 รีวิวต่อร้าน
```

- **ไม่มีตารางโต๊ะ** — โจทย์นับเป็นที่นั่งรวม ร้าน 10 ที่ = รับพร้อมกันได้ 10 คน
- เวลาเก็บเป็น `timestamptz` (UTC) ทั้งหมด; `open_minute/close_minute` เก็บเป็นนาทีจากเที่ยงคืน **เวลาไทย**
- แปลงเป็นเวลาไทยที่ฝั่ง web เท่านั้น (`Asia/Bangkok` ไม่มี DST — พูดถึงได้ตอนสัมภาษณ์ว่าถ้ามี DST ต้องเก็บ timezone ของร้านด้วย)
- migration มีทั้ง Up และ Down และทดสอบ `goose down` แล้ว
- seed data อยู่ที่ `cmd/seed` **ห้ามใส่ใน migrations/** (integration test รันทุกไฟล์ใน migrations/)

---

## 5. กฎธุรกิจทั้งหมด

> **กฎข้อที่ศูนย์: ทุกกฎในหัวข้อนี้ต้องบังคับที่ Go** หน้าเว็บตรวจซ้ำได้เพื่อ UX แต่ห้ามตรวจแค่ที่หน้าเว็บ
> `userID` มาจาก token เท่านั้น **ห้ามรับจาก body/query/header อื่น**

### 5.1 ร้าน (Owner)
| กฎ | ผล |
|---|---|
| สร้างร้านได้ทุกคนที่ล็อกอิน | 201 |
| แก้/ลบได้เฉพาะร้านตัวเอง | ไม่ใช่เจ้าของ → **403** |
| ต้องมีรูปอย่างน้อย 1 รูป (URL) | ไม่มี → 400 |
| `seats > 0` | |
| `cancel_before_minutes >= 30` | ตั้งต่ำกว่า → 400 |
| **ลดจำนวนที่นั่งต่ำกว่าที่มีคนจองไว้แล้ว** | ปฏิเสธ **409** พร้อมบอกว่าช่วงไหนชน (ไม่ทำให้ booking ที่มีอยู่กลายเป็น invalid) |
| **แก้เวลาเปิด–ปิดให้แคบกว่า booking ที่มีอยู่** | ปฏิเสธ 409 ด้วยเหตุผลเดียวกัน |
| **ลบร้านที่ยังมี booking ในอนาคต** | soft delete ได้ แต่ต้องยกเลิก booking ในอนาคตทั้งหมดในทรานแซกชันเดียวกัน (อธิบายเหตุผลได้ว่าทำไมเลือกแบบนี้ ไม่ใช่ block การลบ) |
| ร้านที่ถูก soft delete | หายจาก list และจองใหม่ไม่ได้ (404) แต่ประวัติการจอง/รีวิวเดิมยังอ่านได้ |

การตรวจลดที่นั่ง/ย่นเวลาต้องล็อกแถวร้าน (`FOR UPDATE`) เหมือนตอนจอง — ไม่งั้นมีคนจองแทรกระหว่างตรวจ

### 5.2 การจอง — เงื่อนไขครบทุกข้อ
ลูกค้ากรอก 3 อย่าง: **จำนวนคน / วันที่ / เวลาเริ่ม–สิ้นสุด**

| # | เงื่อนไข | ผิดแล้วตอบ |
|---|---|---|
| 1 | ร้านมีอยู่จริงและไม่ถูกลบ | 404 |
| 2 | `end_at > start_at` | 400 |
| 3 | `start_at` ยังไม่ผ่านไปแล้ว (เทียบ `time.Now()` ที่ server) | 400 |
| 4 | ช่วงจองอยู่ในเวลาเปิด–ปิดของร้าน (รองรับเปิดข้ามเที่ยงคืน) | 400 |
| 5 | `party_size <= restaurant.seats` (ขอเกินความจุร้านไปเลย) | 400 |
| 6 | **ที่นั่งไม่เกินในทุกวินาที** — ดู 5.3 | **409** |
| 7 | ผู้ใช้คนเดียวกันจองร้านเดียวกันซ้อนเวลากันเองไม่ได้ | 409 |
| 8 | จองล่วงหน้าไม่เกิน 90 วัน และช่วงละไม่เกิน 4 ชม. (กฎเราเอง — กัน abuse, อธิบายได้) | 400 |
| 9 | เวลาเริ่ม/สิ้นสุดเป็นช่วงละ 30 นาที (:00 หรือ :30) | 400 |

**กดจองซ้ำ / เน็ตกระตุก:** ไม่ใช้ `Idempotency-Key` — กฎข้อ 7 กันการจองซ้ำให้อยู่แล้ว
(คำขอที่สองทับเวลาคำขอแรก → 409) + หน้าเว็บ disable ปุ่มระหว่างยิง และเมื่อได้ 409 ให้ refetch "การจองของฉัน"

### 5.3 ⭐ กติกาที่นั่ง — จุดสำคัญที่สุดของโจทย์

> "ไม่ว่าเวลาไหน จำนวนคนที่จองรวมกันต้องไม่เกินจำนวนที่นั่งของร้าน"

**ห้ามเขียนแบบนี้ (ผิด):**
```sql
-- ❌ SUM ของทุก booking ที่ช่วงทับกัน
SELECT SUM(party_size) FROM bookings WHERE start_at < :newEnd AND end_at > :newStart
```
เคสที่พิสูจน์ว่าผิด (มาจากสไลด์โจทย์): ร้าน 10 ที่
A จอง 7 คน 12:00–12:30, B จอง 7 คน 12:30–13:00, C ขอ 3 คน 12:00–13:00
naive SUM = 7+7+3 = 17 → ปฏิเสธ **แต่คำตอบที่ถูกคือจองได้** เพราะ A ไม่ทับ B → ไม่มีวินาทีใดเกิน 10

**ต้องใช้ sweep line** — ตรวจเฉพาะ "จุดเปลี่ยนแปลง":

```go
// internal/booking/availability.go

// existing = booking ที่ status='active' และช่วงทับกับ [start,end) โดยไม่รวมตัวที่กำลังแก้
func checkSeats(existing []Booking, seats, partySize int, start, end time.Time) error {
    // จุดที่ต้องตรวจ = จุดเริ่มของช่วงใหม่ + ทุกจุดที่มีคนเข้าร้านเพิ่มภายในช่วงใหม่
    points := []time.Time{start}
    for _, b := range existing {
        if b.StartAt.After(start) && b.StartAt.Before(end) {
            points = append(points, b.StartAt)
        }
    }
    for _, t := range points {
        occupied := 0
        for _, b := range existing {
            // อยู่ในร้าน ณ เวลา t คือ start_at <= t < end_at
            if !b.StartAt.After(t) && b.EndAt.After(t) {
                occupied += b.PartySize
            }
        }
        if occupied+partySize > seats {
            return ErrNotEnoughSeats // → 409
        }
    }
    return nil
}
```
เหตุผลที่ตรวจแค่จุดเริ่ม: จำนวนคนในร้านเพิ่มขึ้นได้เฉพาะตอนมีคนเริ่มจองใหม่ ระหว่างสองจุดค่าคงที่
(จุดที่คนออกไม่ต้องตรวจ เพราะค่าลดลง)

**การกันสองคนกดพร้อมกัน — ต้องอยู่ในทรานแซกชันเดียวกับการนับ:**
```go
err := db.Transaction(func(tx *gorm.DB) error {
    // 1) ล็อกแถวร้าน — คนที่สองจะรอที่บรรทัดนี้
    if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
        First(&r, "id = ?", restaurantID).Error; err != nil { return err }

    // 2) อ่าน booking ที่ทับกัน "ข้างใน" ล็อก
    var existing []Booking
    tx.Where("restaurant_id = ? AND status = 'active'", restaurantID).
       Where("start_at < ? AND end_at > ?", end, start).
       Where("id <> ?", excludeID).      // ⚠️ ตอนแก้ไข: ห้ามนับที่นั่งเดิมของตัวเองซ้ำ
       Find(&existing)

    // 3) ตัดสิน แล้วเขียน — ทั้งหมดในล็อกเดียว
    if err := checkSeats(existing, r.Seats, partySize, start, end); err != nil { return err }
    return tx.Create(&booking).Error
})
```
ห้ามเช็คก่อนแล้วค่อย insert นอกทรานแซกชัน — ร้านเหลือ 6 ที่ สองคนกดคนละ 6 ที่พร้อมกัน
ต้องสำเร็จคนเดียว อีกคนได้ 409 (ทางเลือกอื่นที่อธิบายได้: advisory lock ต่อร้าน, `SERIALIZABLE` + retry — เลือก FOR UPDATE เพราะง่ายสุดและเข้าใจได้)

### 5.4 เวลาเปิด–ปิด รวมร้านที่เปิดข้ามเที่ยงคืน
```
ปกติ   open=11:00 close=22:00 → open_minute(660) < close_minute(1320)
ข้ามวัน open=18:00 close=02:00 → close_minute(120) <= open_minute(1080)
```
วิธีตรวจ: แปลงช่วงจองเป็น "นาทีนับจากเวลาเปิดของรอบนั้น" แล้วเทียบกับความยาวรอบเปิด
`duration = (close - open + 1440) % 1440` (ถ้าได้ 0 = เปิด 24 ชม.)
เขียน test ครอบทั้งสองแบบ + เคสจองคาบเกี่ยวเวลาปิด (22:30–23:30 ที่ร้านปิด 23:00 → 400)

**`date` = วันทำการ (รอบที่เริ่มเปิดในวันนั้น)** — ทั้ง `GET /availability?date=` และฟอร์มจอง
ร้านเปิด 18:00–02:00 ขอ `date=2026-10-10` → ได้ slot 18:00 ของวันที่ 10 ถึง 02:00 ของวันที่ 11 ต่อเนื่องในรอบเดียว
เลือกวันที่ 10 แล้วใส่เวลา 00:30 → หมายถึง 00:30 ของวันที่ 11

### 5.5 แก้ไขและยกเลิก
| กฎ | ผล |
|---|---|
| แก้/ยกเลิกได้เฉพาะ booking ของตัวเอง | 403 |
| แก้ไขใช้กติกาเดียวกับจองใหม่ทุกข้อ + `excludeID = booking.ID` | |
| booking ที่ `cancelled` แล้ว แก้/ยกเลิกอีกไม่ได้ | 409 |
| booking ที่เลยเวลาไปแล้ว แก้/ยกเลิกไม่ได้ | 409 |
| ยกเลิกได้เมื่อ `now <= start_at - cancel_before_minutes` | ช้ากว่านั้น → **403** พร้อมบอกเวลาที่ยกเลิกได้ถึง |
| การแก้ไขแบบเลื่อนเวลา ใช้ cancel window ของเวลา**เดิม** | กันเลี่ยงกฎด้วยการเลื่อนเวลาแทนยกเลิก |

ตัวอย่างจากโจทย์: ร้าน 10 ที่ A จอง 7 B จอง 3 (เต็มพอดี) → A แก้เป็น 8 คน = ต้องใช้ 11 ที่ → **ปฏิเสธ**; A แก้เป็น 5 คน → **ผ่าน**

### 5.6 รีวิว
| กฎ | ผล |
|---|---|
| คะแนนเป็นจำนวนเต็ม 1–5 + ข้อความ | ไม่ใช่ → 400 |
| **รีวิวร้านตัวเองไม่ได้** (`owner_id == userID`) | **403** |
| 1 คน 1 รีวิวต่อร้าน — **แยก create/update ชัดเจน** | `POST` ซ้ำ → **409**; `PUT` ตอนยังไม่เคยรีวิว → **404** |
| แก้/ลบได้เฉพาะรีวิวตัวเอง | 403 |
| ต้องเคยจองก่อนไหม | **ไม่ต้อง** — โจทย์ไม่ได้บังคับ |
| `rating_sum`/`rating_count` | อัปเดต **ในทรานแซกชันเดียวกับ**การเพิ่ม/แก้/ลบรีวิวเสมอ |
| แสดงค่าเฉลี่ย | ทศนิยม 1 ตำแหน่ง + จำนวนรีวิว; ร้านที่ยังไม่มีรีวิว → คืน `null` แล้วแสดง "ยังไม่มีรีวิว" ทุกที่ |

อัปเดตผลรวมแบบ atomic เสมอ (กันสองคนรีวิวพร้อมกันแล้วค่าหาย) — ห้ามอ่านค่ามาบวกใน Go แล้วเขียนกลับ:
```sql
-- create                         -- update (ผลต่าง)                          -- delete
rating_sum = rating_sum + :new    rating_sum = rating_sum + (:new - :old)     rating_sum = rating_sum - :old
rating_count = rating_count + 1                                               rating_count = rating_count - 1
```

### 5.7 การเรียงลำดับร้าน
- `sort=rating` → **Highest rated ด้วย Bayesian average**
  `score = (C·m + rating_sum) / (C + rating_count)` โดย `C = 5`,
  `m` = ค่าเฉลี่ยของทุกรีวิวในระบบ = `SUM(rating_sum) / SUM(rating_count)` ของร้านที่ไม่ถูกลบ (ไม่มีรีวิวเลย → `COALESCE`)
  **ร้านที่ยังไม่มีรีวิว (`rating_count = 0`) อยู่ท้ายสุดเสมอ** — ไม่งั้นจะได้คะแนน = m พอดีและแซงร้านที่มีรีวิวจริงแต่ต่ำกว่าค่าเฉลี่ย
  ```sql
  ORDER BY (rating_count = 0), score DESC, rating_count DESC, id
  ```
  (ทางเลือกที่พิจารณาแล้ว: เกณฑ์ขั้นต่ำ ≥ 5 รีวิว — ง่ายกว่าแต่มีเส้นตัดแข็ง; เขียนเปรียบเทียบไว้ใน README)
- `sort=reviews` → **Most reviewed**: เรียงตาม `rating_count`
- default → ร้านใหม่สุด
- ต้องมี tie-breaker คงที่ (`id`) ไม่งั้น pagination จะได้ข้อมูลซ้ำ/หาย

---

## 6. API

prefix `/api/v1` — JSON ทั้งหมด — base URL `http://api.jongyoung.localhost/api/v1` — Swagger ที่ `/swagger/index.html`

| Method | Path | Auth | หมายเหตุ |
|---|---|---|---|
| GET | `/me` | ✓ | โปรไฟล์ + ร้านที่เป็นเจ้าของ (web ใช้ตัดสินว่าจะโชว์เมนู owner ไหม) |
| GET | `/restaurants` | – | `?q=&cuisine=&sort=rating\|reviews\|newest&page=&limit=` |
| GET | `/restaurants/:id` | – | + คะแนนเฉลี่ย + จำนวนรีวิว + รูปทั้งหมด |
| GET | `/restaurants/:id/availability` | – | `?date=YYYY-MM-DD` (วันทำการ) → ที่นั่งคงเหลือทุกช่วง 30 นาที |
| POST | `/restaurants` | ✓ | |
| PUT / DELETE | `/restaurants/:id` | ✓ owner | |
| POST / DELETE | `/restaurants/:id/images` | ✓ owner | รับ URL |
| GET | `/restaurants/:id/bookings` | ✓ owner | `?date=` — Owner ดูการจองของร้าน |
| POST | `/bookings` | ✓ | 409 ถ้าที่นั่งไม่พอหรือซ้อนการจองของตัวเอง |
| GET | `/me/bookings` | ✓ | `?status=upcoming\|past\|cancelled` |
| GET | `/bookings/:id` | ✓ เจ้าของ booking หรือ owner ร้าน | |
| PUT | `/bookings/:id` | ✓ เจ้าของ booking | แก้จำนวนคน/วัน/เวลา |
| DELETE | `/bookings/:id` | ✓ เจ้าของ booking | ยกเลิก → `status=cancelled` (ไม่ลบจริง เก็บประวัติ) |
| GET | `/restaurants/:id/reviews` | – | `?page=&limit=` |
| POST | `/restaurants/:id/reviews` | ✓ | สร้าง; มีอยู่แล้ว → 409; เจ้าของร้าน → 403 |
| PUT | `/restaurants/:id/reviews` | ✓ | แก้ของตัวเอง; ยังไม่มี → 404 |
| DELETE | `/restaurants/:id/reviews` | ✓ | ลบของตัวเอง |

**Status code ที่ต้องใช้ให้ถูก**
`200/201/204` สำเร็จ · `400` รูปแบบ/เงื่อนไขเวลาผิด · `401` ไม่มี/token ไม่ถูก/หมดอายุ ·
`403` สิทธิ์ไม่ถึง (ไม่ใช่เจ้าของ, รีวิวร้านตัวเอง, ยกเลิกช้าเกิน) · `404` ไม่มีข้อมูล ·
`409` ชนกับสถานะปัจจุบัน (ที่นั่งไม่พอ, จองซ้อน, ยกเลิกซ้ำ, ลดที่นั่งไม่ได้, รีวิวซ้ำ) · `500` ที่เหลือ

**Error response รูปแบบเดียวทั้งระบบ** (`internal/httputil`)
```json
{ "error": { "code": "NOT_ENOUGH_SEATS", "message": "ช่วง 12:30–13:00 เหลือ 3 ที่นั่ง", "details": { "available": 3 } } }
```
ฝั่ง web แปล `code` เป็นข้อความไทย — ไม่พึ่ง `message` จาก API ในการตัดสินใจ

**Middleware order:** Recovery → RequestID/Logger → CORS (`http://jongyoung.localhost` เท่านั้น) → RateLimit(เบา ๆ) → JWT (verify + JIT provisioning) → เช็คความเป็นเจ้าของใน service

---

## 7. หน้าเว็บ (Next.js)

| Path | เนื้อหา | Rendering |
|---|---|---|
| `/` | รายการร้าน + ค้นหา + ปุ่มเรียง ใหม่ล่าสุด / คะแนนสูงสุด / รีวิวมากที่สุด | server (ไม่ต้องใช้ token) |
| `/restaurants/[id]` | รายละเอียด + รูป + ที่นั่งว่าง + ฟอร์มจอง + รีวิว | server + client island (ฟอร์มจอง, รีวิว) |
| `/me/bookings` | การจองของฉัน (tab กำลังจะถึง/ผ่านมาแล้ว/ยกเลิกแล้ว) แก้-ยกเลิกในหน้านี้ | client (TanStack + token) |
| `/owner/restaurants` | ร้านของฉัน (สร้าง/แก้/ลบ) | client |
| `/owner/restaurants/[id]/bookings` | ตารางการจองรายวันของร้าน + กราฟที่นั่งต่อช่วง | client |
| `/api/auth/[...nextauth]`, `/api/auth/logout` | next-auth + logout | – |

- ฟอร์มใช้ react-hook-form + zod
- **การจอง: ห้ามใช้ optimistic update** — ต้องรอผลจริง; ได้ 409 ให้ refetch availability พร้อมบอกจำนวนที่นั่งที่เหลือจริง และเสนอช่วงที่ยังว่างพอ
  (รีวิวใช้ optimistic ได้)
- ปุ่มกดจองต้อง disable ระหว่างยิง กันกดรัว
- ซ่อนปุ่มแก้/ลบเมื่อไม่ใช่เจ้าของ — เป็นเรื่อง UX เท่านั้น API ต้องกันซ้ำเสมอ
- ปุ่มที่กดไม่ได้เพราะกติกา (เช่น เลยเวลายกเลิก) ให้ disabled พร้อมข้อความบอกเหตุผล ไม่ใช่ซ่อน
- แสดงเวลาเป็นเวลาไทยทุกที่ (`Intl.DateTimeFormat('th-TH', { timeZone: 'Asia/Bangkok' })`)
- empty state / loading skeleton / error state ทุกหน้า

---

## 8. Design system

**ต้นแบบหน้าตา:** [guy127/introduce_myself](https://github.com/guy127/introduce_myself) — **ไม่ใช่ wongnok**
**Mockup ทุกหน้า (Claude Design):** https://claude.ai/artifact/Cb7eGqYexzQd73X6AiqvKR

**คาแรกเตอร์:** กลางคืนอบอุ่น + กระจกขุ่น — พื้นมืดโทนอุ่นมีแสงเรืองแดง/อำพัน/ฟ้า, การ์ดกระจก (blur),
หัวข้อใหญ่ไล่สี, ป้ายเล็กมีหมายเลข "01 • …", ปุ่มแคปซูลไล่สีมีแสงเรือง, จุดเขียวกะพริบแบบ live
มีโหมด **Night (ค่าเริ่มต้น) / Day** สลับได้จากปุ่มบน navbar

### 8.1 Font
```
หัวข้อ/ตัวเลขเด่น: "Kanit" 500–700
เนื้อหา/UI:        "Prompt" 400–600
ตัวเลข/เวลา/จำนวนที่นั่ง: font-variant-numeric: tabular-nums
```
ขนาด: hero 56–84px / หัวข้อ section 44–48px / หัวการ์ด 22px / เนื้อหา 16px / กำกับ 14px / เล็กสุด 13px
**ห้ามต่ำกว่า 13px สำหรับภาษาไทย** · `line-height: 1.7` สำหรับเนื้อหา · หัวข้อไทยใช้ line-height ≥ 1.1
(ต้นแบบใช้ 0.92 ซึ่งทำให้สระ/วรรณยุกต์ชนกัน — ห้ามลอก) · letter-spacing ไม่ติดลบเกิน -0.02em

### 8.2 สี — token สองโหมด

| Token | Night | Day | ใช้กับ |
|---|---|---|---|
| `--bg` | `#11070a` | `#fff8f2` | พื้นหลัง (+ radial glow แดง/ฟ้า/อำพัน + เส้นกริด 64px จาง ๆ) |
| `--text` | `#fff7ed` | `#261419` | ตัวอักษรหลัก |
| `--muted` | `#d7b8b4` | `#6f4c4d` | กำกับ |
| `--soft` | `#a88b8d` | `#7d625f` | meta เล็ก |
| `--accent` | `#ff354f` | `#c5162e` | แบรนด์ (โลโก้, glow, gradient) |
| `--accent-2` | `#ffb703` | `#b77900` | eyebrow, ตัวเลขสถิติ, ดาว |
| `--accent-3` | `#43e8ff` | `#007c91` | ลิงก์, ช่วงเวลาที่เลือก, focus ring |
| `--cta` | `#e11d48 → #c2410c` | `#c5162e → #b3361a` | ปุ่มหลัก (gradient 135°) — เข้มกว่าต้นแบบเพื่อให้ตัวขาวผ่าน contrast |
| `--ok` | `#27e88b` | `#008f53` (ข้อความ `#007a47`) | ที่นั่งว่าง, ยืนยันแล้ว |
| `--warn` | `#ffb703` | `#d08a00` (ข้อความ `#8a5a00`) | ใกล้เต็ม (จองแล้ว ≥ 70%) |
| `--full` | `#ff354f` (ข้อความ `#ff6b7f`) | `#c5162e` (ข้อความ `#b0132a`) | เต็ม, ยกเลิก, error, ลบ |
| `--border` | `rgba(255,255,255,.16)` | `rgba(120,36,36,.14)` | เส้นขอบกระจก |
| `--panel` / `--panel-strong` | `rgba(255,255,255,.08/.12)` | `rgba(255,255,255,.76/.92)` | พื้นการ์ดกระจก |

กฎการใช้สี:
- **หนึ่งหน้าจอมีปุ่ม gradient (`--cta`) ได้ปุ่มเดียว** ที่เหลือเป็นแคปซูลขอบบาง/โปร่ง
- เขียว-เหลือง-แดง (`--ok/--warn/--full`) ใช้สื่อสถานะที่นั่ง/การจองเท่านั้น
- ช่วงเวลาที่เลือกใช้ `--accent-3` (ฟ้า) เพื่อไม่ปนกับสีสถานะ
- ตัวอักษรต้องผ่าน contrast **AA (4.5:1)** — โหมด Day ใช้สีข้อความเข้มตามตาราง
- **ห้ามสื่อความหมายด้วยสีอย่างเดียว** ต้องมีไอคอนหรือข้อความกำกับ
- ไอคอนใช้ SVG (lucide) — ไม่ใช้อีโมจิ

### 8.3 Layout & component
```
max-width หน้า: 1120px, padding ขอบ 16px (mobile) / 24px (desktop)
spacing scale: 4 8 12 16 24 32 48
radius: 999px (ปุ่ม/nav/badge/pill), 20–28px (การ์ดกระจก), 14–16px (input, แถว slot)
glass: background linear-gradient(145deg, panel-strong, panel); border 1px var(--border);
       box-shadow 0 22px 70px var(--shadow); backdrop-filter blur(18px)
hover การ์ด: translateY(-6px) — ปิดเมื่อ prefers-reduced-motion
```
- **Navbar**: แคปซูลกระจกลอยด้านบน โลโก้วงกลม gradient "จ" + "จองยัง", ลิงก์เป็น pill, ปุ่ม Night/Day, avatar
- **Section heading**: eyebrow (อำพัน, 13px, letter-spacing .14em, "01 • …") → หัวข้อ Kanit → คำอธิบาย muted
- **Restaurant card**: การ์ดกระจก รูป 16:9 มี badge ประเภทอาหารลอยบนรูป → ชื่อ Kanit 22px → `★ 4.3 · 58 รีวิว · 10 ที่นั่ง` → เวลาเปิด + "ดูที่นั่งว่าง →"
- **Seat bar** (หัวใจของ UI): แถวต่อช่วง 30 นาที แถบยาว = สัดส่วนที่ถูกจอง
  เขียว (< 70%) → อำพัน + "เหลือ n ที่" (≥ 70%) → แดง + "เต็ม" (100%, กดไม่ได้); ช่วงที่เลือกขอบฟ้า
- **Booking form**: การ์ดกระจกทึบกว่า sticky ขวาบน desktop / แผงลอยล่างบน mobile; บรรทัดสรุปตลอด
  "ส. 10 ต.ค. · 18:00–19:00 · 2 คน · ยกเลิกได้ถึง 17:30"
- **409 ที่นั่งไม่พอ**: bottom sheet (mobile) บอกที่นั่งที่เหลือจริง + เสนอช่วงที่ยังว่างพอ
- **Booking card**: กล่องวันที่ gradient + ชื่อร้าน Kanit + badge สถานะมีจุดสี + ปุ่มแก้/ยกเลิก
  เลยเวลายกเลิก → ปุ่ม disabled + ข้อความบอกเหตุผล; ยกเลิกต้องมีกล่องยืนยัน
- **Star rating**: ดาวสีอำพัน + ตัวเลขทศนิยม 1 ตำแหน่ง + จำนวนรีวิวเสมอ
- ทุก interactive element มี `:focus-visible` ring 2px สี `--accent-3`
- แตะได้ต้องใหญ่ ≥ 44×44px บนมือถือ · mobile-first ทดสอบที่ 375px ก่อน

### 8.4 สิ่งที่ต้องส่ง
ออกแบบหน้าจอใน Claude Design ก่อนแล้วค่อยทำจริงบน Next.js (โจทย์บังคับ) — ทำแล้วที่ลิงก์ด้านบน
(หน้า list, หน้าร้าน+ฟอร์มจอง, การจองของฉัน, จัดการร้าน, การจองรายวันของร้าน, มือถือ 2 หน้า)
เก็บลิงก์/ภาพไว้ที่ `docs/` และต้องกด Share ลิงก์ก่อนส่งให้ผู้ตรวจ

---

## 9. Test ที่ต้องมี

`internal/booking/availability_test.go` — table-driven ครอบ:
1. ร้าน 10 ที่ มีคนจอง 7 ขอเพิ่ม 5 ในช่วงทับกัน → **ปฏิเสธ**
2. A 7 (12:00–12:30), B 7 (12:30–13:00), C ขอ 3 (12:00–13:00) → **ผ่าน** ⭐ เคสที่ naive SUM พลาด
3. A 7 + B 3 เต็มพอดี → A แก้เป็น 8 = ปฏิเสธ / A แก้เป็น 5 = ผ่าน (ทดสอบ `excludeID`)
4. จองเวลาที่ผ่านมาแล้ว → ปฏิเสธ
5. จองนอกเวลาเปิด + ร้านเปิดข้ามเที่ยงคืน 18:00–02:00 (ทั้งเคสผ่านและไม่ผ่าน)
6. จองคาบเกี่ยวเวลาปิดร้าน → ปฏิเสธ
7. ยกเลิกช้ากว่า `cancel_before_minutes` → ปฏิเสธ; ตรงเวลาพอดี → ผ่าน (boundary)
8. `party_size > seats` → ปฏิเสธ
9. booking ที่ยกเลิกแล้วไม่ถูกนับเป็นที่นั่งที่ถูกใช้

- service test: mock repository ด้วย mockery (`mocks_test.go`)
- repository test: testcontainers (Postgres จริง) รัน migrations/ ทุกไฟล์
- handler test: `httptest` ยิงใส่ Gin router — เช็ค 401 (ไม่มี token), 403 (ไม่ใช่เจ้าของ / รีวิวร้านตัวเอง), 409 (ที่นั่งไม่พอ, รีวิวซ้ำ)
- integration test การจองพร้อมกัน: ยิง 2 goroutine จองพร้อมกันบนที่นั่งที่เหลือพอสำหรับคนเดียว → ต้องสำเร็จ 1 ล้มเหลว 1
- Bayesian sort: ★5.0 จาก 1 รีวิวต้องไม่อยู่เหนือ ★4.8 จาก 46 รีวิว; ร้านไม่มีรีวิวอยู่ท้ายสุด

---

## 10. สิ่งที่ต้องส่ง (11 ต.ค. — ไม่มีเลื่อน)

1. **Git repo** — Next.js + Go พร้อม README ที่มี:
   - วิธีรันทีละคำสั่ง (`cp .env.example .env` → `docker compose up` → migrate → seed → เปิด http://jongyoung.localhost)
   - บัญชีทดสอบของ Keycloak (owner 2 คน, ลูกค้า 1 คน)
   - seed data ที่เปิดมาแล้ว **เห็นร้านหลายร้าน + การจอง + รีวิวทันที**
   - เหตุผลที่เลือก DB / auth / library แต่ละตัว + เปรียบเทียบ Bayesian กับเกณฑ์ ≥ 5 รีวิว
2. **วิดีโอ ≤ 5 นาที** — เดโมครบทุกฟีเจอร์ + อธิบายโค้ดจุดตรวจกติกาที่นั่งและจุดล็อกกันชนกัน
3. **UI Design** — ลิงก์ Claude Design (ข้อ 8)

---

## 11. คำถามวันสัมภาษณ์ (17–18 ต.ค.) และคำตอบที่โค้ดนี้ให้

| คำถาม | คำตอบ |
|---|---|
| สองคนกดพร้อมกัน ร้านเหลือ 6 ที่ | `SELECT ... FOR UPDATE` บนแถวร้าน + นับ + เขียน ในทรานแซกชันเดียว → รับคนเดียว อีกคน 409 |
| ทำไมไม่ใช้ SUM ตรวจที่นั่ง | เคส A 7 / B 7 / C 3 — ช่วงทับกันไม่ได้แปลว่าอยู่พร้อมกัน ต้อง sweep line ตรวจทุกจุดที่มีคนเข้าร้าน |
| ร้านเปิด 18:00–02:00 เก็บยังไง | นาทีจากเที่ยงคืน; `close <= open` = ข้ามวัน; ตรวจด้วย offset จากเวลาเปิด; `date` = วันทำการ |
| ★5.0 จาก 1 รีวิว vs ★4.8 จาก 300 | Bayesian average ดึงร้านที่รีวิวน้อยเข้าหาค่าเฉลี่ยรวม (C = 5) ร้านไม่มีรีวิวอยู่ท้าย |
| ค่าเฉลี่ยคำนวณเมื่อไหร่ | เก็บ `rating_sum`/`rating_count` อัปเดตแบบ atomic ในทรานแซกชันเดียวกับรีวิว → หน้า list ไม่ต้อง AVG/JOIN, ไม่มี N+1 |
| กดจองซ้ำ/เน็ตกระตุก | กฎ "คนเดียวกันจองร้านเดียวกันซ้อนเวลาไม่ได้" → คำขอที่สอง 409 + ปุ่ม disable ระหว่างยิง |
| Server vs Client Component | หน้า list/detail = server (ไม่ต้องใช้ token, โหลดเร็ว); หน้าที่ต้อง login และฟอร์ม = client (TanStack + axios) |
| session/token เก็บที่ไหน | next-auth เก็บ session ใน cookie; axios ดึง access token จาก `getSession()` แนบ Bearer — ความเสี่ยง XSS รับมือด้วย token อายุสั้น + ไม่ render HTML จากผู้ใช้; ทางที่ปลอดภัยกว่าคือ BFF proxy |
| issuer ไม่ตรงระหว่าง browser กับ container | Caddy + โดเมน `*.jongyoung.localhost` ให้ทุกฝ่ายเห็น Keycloak ชื่อเดียวกัน |
| ทำไมต้องมี users ในฐานข้อมูลเราอีก ทั้งที่มี Keycloak | Keycloak เก็บ identity, DB เราเก็บ domain data + FK; ผูกด้วย `sub` ไม่ใช่ email; สร้างตอนเจอ token ครั้งแรก |
| timezone | เก็บ UTC ที่ DB, `Asia/Bangkok` แสดงที่ web; ถ้าร้านอยู่หลาย timezone ต้องเก็บ timezone ต่อร้าน |
| ทำไมไม่ใช้ realm role แยก owner/customer | บทบาทผูกกับร้านแต่ละร้าน ไม่ใช่ผูกกับบัญชี — เป็นข้อมูล ไม่ใช่สิทธิ์ระดับ realm |

---

## 12. ข้อห้าม

- ห้ามใช้ naive `SUM` ตรวจที่นั่ง (ดู 5.3)
- ห้ามลืม `excludeID` ตอนแก้ไขการจอง
- ห้ามเช็คที่นั่งนอกทรานแซกชัน
- ห้ามอัปเดต `rating_sum/rating_count` ด้วยการอ่านค่ามาบวกใน Go แล้วเขียนกลับ
- ห้ามรับ `user_id` จาก request body/query — เอาจาก token เท่านั้น
- ห้ามตรวจกติกาแค่ที่หน้าเว็บ
- ห้ามเก็บ token ใน localStorage เอง และห้ามใช้ `dangerouslySetInnerHTML` กับข้อมูลผู้ใช้
- ห้าม log token, client_secret, DSN เต็ม
- ห้าม commit `.env` — ต้องมี `.env.example` ครบทุก key; secret ใน realm export ต้องเป็นค่า dev เท่านั้น
- ห้ามเขียน secret ลงใน `compose.yml` ตรง ๆ — อ่านจาก `.env`
- ห้ามลบข้อมูลจริงเมื่อยกเลิกการจอง (ใช้ `status`) — ต้องเก็บประวัติ
- ห้ามใส่ seed data ใน `migrations/`
- ห้ามใส่ library ที่อธิบายไม่ได้ว่าทำอะไร
- ห้ามลอกหน้าตา wongnok

---

## 13. CI/CD (GitLab)

ไฟล์ `.gitlab-ci.yml` — ไม่ต้องใช้ SSH key (runner ดึงโค้ดด้วย job token, push image ด้วย `CI_JOB_TOKEN`)

| Stage | ทำอะไร | รันเมื่อ |
|---|---|---|
| lint | `go vet` + `gofmt -l` (api), `npm run lint` (web) | ทุก push |
| test | `go test ./...` (ใช้ service Docker-in-Docker ให้ testcontainers), `npm test` (web) | ทุก push |
| build | `next build` เช็คว่า build ผ่าน | ทุก push |
| image | build Docker image ของ `api` และ `web` แล้ว push ขึ้น GitLab Container Registry | merge เข้า `main` |

deploy อัตโนมัติยังไม่ทำ — ถ้าจะทำทีหลัง ให้สร้าง deploy key คู่ใหม่ (ห้ามใช้ key ส่วนตัว)
เก็บ private key เป็น CI/CD variable แบบ Protected + Masked แล้วให้เซิร์ฟเวอร์ pull image จาก registry

---

## 14. วิธีทำงานใน repo นี้

- commit เป็นระยะทุกครั้งที่ทำเสร็จเป็นชิ้นและรันได้ — **ข้อความ commit เป็นภาษาไทย**
- ทำ test ก่อนสำหรับกติกาธุรกิจ (โดยเฉพาะ 5.2–5.6)
