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
| Styling | Tailwind (token ใน `globals.css`) + component เขียนเองใน `components/bases/` | ไม่ใช้ shadcn — ชิ้นส่วนน้อย เขียนเองอธิบายได้ทุกบรรทัด |
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

- `proxy.ts` (middleware ของ next-auth) กันหน้า `/me/*`, `/owner/*` และ `/bookings/*` ถ้ายังไม่ login
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
- `EnsureExists` = **upsert ไม่ใช่ get-or-create ครั้งเดียวจบ** — ถ้า `email` หรือ `display_name` ใน claims
  ต่างจากที่เก็บไว้ ต้องอัปเดตให้ตรง (ผู้ใช้เปลี่ยนชื่อ/อีเมลใน Keycloak แล้วระบบเราต้องตามทัน)
  ```sql
  INSERT INTO users (id, keycloak_uid, email, display_name) VALUES (...)
  ON CONFLICT (keycloak_uid) WHERE deleted_at IS NULL
  DO UPDATE SET email = EXCLUDED.email, display_name = EXCLUDED.display_name, updated_at = now()
  RETURNING *;
  ```
- `users.keycloak_uid` มี unique index — แต่ถ้าใช้ soft delete ต้องเป็น **partial unique index** (`WHERE deleted_at IS NULL`) ไม่งั้นคนที่ถูกลบแล้วล็อกอินใหม่จะชนกับแถวเก่า
  (`ON CONFLICT ... WHERE deleted_at IS NULL` ต้องเขียน predicate ให้ตรงกับ index นี้ ไม่งั้น Postgres หา index ไม่เจอ)
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
│   │   │   ├── businessday.go    # ⭐ แปลงวันทำการ ↔ ช่วงเวลาจริง (ใช้ร่วมกันทั้งระบบ)
│   │   │   ├── availability.go   # ⭐ หัวใจของโจทย์ (maxConcurrent, checkSeats, slots)
│   │   │   └── availability_test.go
│   │   └── review/
│   ├── migrations/               # goose (Up + Down ทุกไฟล์) — ห้ามใส่ seed data
│   ├── docs/                     # swagger generated (ภาพรวมระบบอยู่ที่ ARCHITECTURE.md ที่ root)
│   ├── .mockery.yml              # mock ลง mocks_test.go ข้างไฟล์ interface
│   └── Dockerfile
├── web/                          # Next.js
│   ├── app/
│   │   ├── page.tsx              # รายการร้าน + แถบค้นหา (วันที่/เวลา/จำนวนคน)
│   │   ├── restaurants/[id]/
│   │   ├── bookings/[id]/        # หน้ายืนยันการจอง
│   │   ├── me/bookings/
│   │   ├── owner/restaurants/
│   │   └── api/auth/             # [...nextauth], logout
│   ├── components/bases/         # ปุ่ม, input, badge, TimeChip ฯลฯ
│   ├── containers/               # Navbar (สลับโหมด), OwnerNavbar, Footer
│   ├── services/                 # hook TanStack Query ต่อ resource
│   ├── lib/                      # axios.ts, auth/auth-options.ts, formatters (เวลาไทย + ป้ายข้ามวัน), ics.ts
│   ├── proxy.ts                  # กันหน้า /me, /owner, /bookings
│   └── Dockerfile
├── docs/                         # spec, UI design, ER diagram
├── .gitlab-ci.yml
├── CLAUDE.md
├── ARCHITECTURE.md               # ภาพรวมสถาปัตยกรรม (อัปเดตเมื่อโครงสร้างเปลี่ยน)
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
  cuisine               text                          -- ใช้ filter + แนะนำ "ร้านประเภทเดียวกัน" ตอนร้านเต็ม
  address               text not null                 -- แสดงในหน้ายืนยัน + ลิงก์ Google Maps (ไม่เก็บพิกัด)
  map_url               text not null default ''      -- ลิงก์ Google Maps ที่เจ้าของร้านวางเอง (ไม่บังคับ); ว่าง = ค้นจาก address
  seats                 int  not null check (seats > 0)
  open_minute           int  not null check (open_minute between 0 and 1439)
  close_minute          int  not null check (close_minute between 0 and 1439)
  closed_weekdays       smallint not null default 0 check (closed_weekdays between 0 and 126)
                                                -- วันปิดประจำสัปดาห์: บิตที่ n = ปิดทุก time.Weekday(n) (0 = อาทิตย์); 127 = ปิดทั้งสัปดาห์ ไม่อนุญาต
  break_start_minute    int  not null default 0 check (break_start_minute between 0 and 1439)
  break_end_minute      int  not null default 0 check (break_end_minute between 0 and 1439)
                                                -- ช่วงพักภายในรอบ; เท่ากัน = ไม่มีช่วงพัก; ต้องอยู่ข้างในรอบ ไม่ติดขอบ (Hours.ValidBreak)
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
  cancelled_by  text check (cancelled_by in ('customer', 'restaurant'))  -- null = ยังไม่ยกเลิก/ข้อมูลเก่าก่อนมีคอลัมน์นี้ (ไม่ backfill)
  cancel_reason text not null default ''         -- เหตุผลจากร้าน ลูกค้าเห็นได้
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

restaurant_closures                   -- ช่วงที่ร้านปิดชั่วคราว (ไฟดับ, ปิดปรับปรุง, หยุดยาว) เป็นช่วงเวลาจริง [start_at, end_at)
  id            uuid pk               -- ปิดทั้งวัน/หลายวันเก็บรูปแบบเดียวกัน (= ช่วงที่ครอบทั้งรอบของวันนั้น)
  restaurant_id uuid not null references restaurants(id) on delete cascade
  start_at      timestamptz not null
  end_at        timestamptz not null
  reason        text not null check (reason <> '')
  created_at    timestamptz not null default now()
  check (end_at > start_at)
  index on (restaurant_id, end_at)

notifications                         -- แจ้งเตือนในเว็บ; ไม่เก็บข้อความ — หน้าเว็บประกอบข้อความไทยจาก kind เอง
  id              uuid pk
  user_id         uuid not null references users(id)   -- ผู้รับ
  kind            text not null check (kind in ('booking_created', 'booking_updated', 'booking_cancelled', 'booking_cancelled_by_restaurant'))
  booking_id      uuid not null references bookings(id)
  restaurant_id   uuid not null references restaurants(id)
  restaurant_name text not null      -- snapshot ตอนเกิดเหตุ: booking อาจถูกแก้ภายหลัง แต่แจ้งเตือนต้องบอกสิ่งที่เกิดตอนนั้น
  customer_name   text not null
  business_date   date not null      -- ใช้ทำลิงก์ไปบอร์ด owner
  start_at        timestamptz not null
  end_at          timestamptz not null
  party_size      int not null
  reason          text not null default ''
  read_at         timestamptz
  created_at      timestamptz not null default now()
  index on (user_id, created_at desc)
```

- **ไม่มีตารางโต๊ะ** — โจทย์นับเป็นที่นั่งรวม ร้าน 10 ที่ = รับพร้อมกันได้ 10 คน
- **ไม่เก็บพิกัด** → แผนที่เป็นลิงก์ Google Maps (`map_url` ถ้าเจ้าของร้านวางไว้ ไม่งั้นค้นจาก `address`) และไม่มีฟีเจอร์ "ร้านใกล้เคียง"
- เวลาเก็บเป็น `timestamptz` (UTC) ทั้งหมด; `open_minute/close_minute` เก็บเป็นนาทีจากเที่ยงคืน **เวลาไทย**
- แปลงเป็นเวลาไทยที่ฝั่ง web เท่านั้น (`Asia/Bangkok` ไม่มี DST — พูดถึงได้ตอนสัมภาษณ์ว่าถ้ามี DST ต้องเก็บ timezone ของร้านด้วย)
- migration มีทั้ง Up และ Down และทดสอบ `goose down` แล้ว
- seed data อยู่ที่ `cmd/seed` **ห้ามใส่ใน migrations/** (integration test รันทุกไฟล์ใน migrations/)

### 4.1 Seed data ที่ต้องมี (ต้องครบ 5 แบบเพื่อโชว์ว่ารองรับ)

| ร้าน | เวลา | open/close_minute | โชว์อะไร |
|---|---|---|---|
| ร้านอาหารตามสั่ง | 10:00–21:00 | 600 / 1260 | เคสปกติ + มีช่วงเกือบเต็มคืนนี้ (โชว์ "เหลือน้อย") |
| ร้านในห้าง | 11:00–22:00 | 660 / 1320 | เคสปกติ + รีวิวเยอะ (46 รีวิว ★4.8) + ช่วงพัก 15:00–17:00 |
| ร้านบุฟเฟ่ต์ | 17:00–23:00 | 1020 / 1380 | เปิดเฉพาะเย็น + มีรีวิวเดียว ★5.0 (ทดสอบ Bayesian + ป้าย "รีวิวน้อย") |
| ร้านซีฟู้ด/บาร์ | 18:00–02:00 | 1080 / 120 | **ข้ามเที่ยงคืน** |
| ร้านโจ๊ก 24 ชม. | 00:00–00:00 | 0 / 0 | **เปิด 24 ชม.** |

ต้องมีร้านที่ยังไม่มีรีวิวอย่างน้อย 1 ร้าน (ทดสอบลำดับท้ายสุด), ร้านที่เต็มทั้งรอบคืนนี้อย่างน้อย 1 ร้าน (โชว์ empty state)
และ booking ล่วงหน้าของ `customer1` อย่างน้อย 2 รายการ (รายการหนึ่งอยู่ในรอบข้ามเที่ยงคืน) + booking ที่ผ่านมาแล้ว 1 รายการ
ทุกร้านต้องมี `address` และรูปอย่างน้อย 1 รูป

ต้องมีด้วย: การจองของ `customer1` ที่ร้านยกเลิก 1 รายการ + แจ้งเตือนที่ยังไม่อ่าน, ร้านบุฟเฟ่ต์ปิดปรับปรุง 5 วันข้างหน้า, owner2 มีแจ้งเตือนการจองใหม่

---

## 5. กฎธุรกิจทั้งหมด

> **กฎข้อที่ศูนย์: ทุกกฎในหัวข้อนี้ต้องบังคับที่ Go** หน้าเว็บตรวจซ้ำได้เพื่อ UX แต่ห้ามตรวจแค่ที่หน้าเว็บ
> `userID` มาจาก token เท่านั้น **ห้ามรับจาก body/query/header อื่น**

### 5.1 ร้าน (Owner)
| กฎ | ผล |
|---|---|
| สร้างร้านได้ทุกคนที่ล็อกอิน | 201 |
| แก้/ลบได้เฉพาะร้านตัวเอง | ไม่ใช่เจ้าของ → **403** |
| ต้องมีรูปอย่างน้อย 1 รูป (URL) และมี `address` | ไม่มี → 400 |
| `map_url` (ไม่บังคับ) ต้องเป็นลิงก์ Google Maps แบบ https (`maps.app.goo.gl`, `google.com/maps`, …) — ลิงก์นี้ไปอยู่ใน `<a href>` ที่ลูกค้ากด ห้ามรับ URL อะไรก็ได้ | ผิด → 400 `INVALID_MAP_URL` |
| `seats > 0` | |
| `cancel_before_minutes >= 30` | ตั้งต่ำกว่า → 400 |
| ช่วงพัก (ไม่บังคับ) ส่งคู่ `break_start`/`break_end` และต้องอยู่ข้างในเวลาเปิด–ปิด ไม่ติดขอบ | ผิด → 400 `INVALID_BREAK` |
| **ลดจำนวนที่นั่งต่ำกว่าที่มีคนจองไว้แล้ว** | ปฏิเสธ **409** `SEATS_BELOW_EXISTING_BOOKINGS` พร้อม `details: { at, peak }` |
| **แก้เวลาเปิด–ปิดให้แคบกว่า booking ที่มีอยู่** | ปฏิเสธ 409 `HOURS_CONFLICT_EXISTING_BOOKINGS` พร้อม booking ที่ตกนอกเวลา |
| ปิดร้านชั่วคราว (บางช่วง/ทั้งวัน/หลายวัน) ต้องมีเหตุผล ≤ 200 ตัว, ≤ 90 วัน; ปิดบางช่วงต้องอยู่ในรอบเปิดของวันทำการนั้นและห้ามย้อนหลัง; ปิดทั้งวัน/หลายวันถ้ารอบเปิดไปแล้ว clamp ให้เริ่มจากตอนนี้แทนการปฏิเสธ | ผิด → 400 `INVALID_CLOSURE` |
| ช่วงปิดทับการจองที่ยังไม่เริ่ม | 409 `CLOSURE_AFFECTS_BOOKINGS` + `details.bookings` → ส่งซ้ำพร้อม `confirm_booking_ids` ที่ตรงกันพอดีจึงยกเลิก + แจ้งลูกค้าในทรานแซกชันเดียว |
| **ลบร้านที่ยังมี booking ในอนาคต** | soft delete ได้ แต่ต้องยกเลิก booking ในอนาคตทั้งหมดในทรานแซกชันเดียวกัน ด้วย `cancelled_by='restaurant'` + แจ้งลูกค้า (อธิบายเหตุผลได้ว่าทำไมเลือกแบบนี้ ไม่ใช่ block การลบ) |
| ร้านที่ถูก soft delete | หายจาก list และจองใหม่ไม่ได้ (404) แต่ประวัติการจอง/รีวิวเดิมยังอ่านได้ |

การตรวจลดที่นั่ง/ย่นเวลาต้องล็อกแถวร้าน (`FOR UPDATE`) เหมือนตอนจอง — ไม่งั้นมีคนจองแทรกระหว่างตรวจ
- ลดที่นั่ง: `peak, at := maxConcurrent(bookingในอนาคต, now, ไกลสุด)` → `peak > newSeats` → 409 (ใช้ **`maxConcurrent()` ตัวเดียวกับการจอง** ดู 5.3)
- ย่นเวลา (รวมถึงการเพิ่ม/ขยายช่วงพัก): booking ในอนาคตทุกตัวต้องยังผ่าน `fitsOpeningHours` (5.4) ด้วยเวลาใหม่
ห้ามเขียน logic นับที่นั่งหรือเช็คเวลาเปิดขึ้นใหม่ที่นี่

### 5.2 การจอง — เงื่อนไขครบทุกข้อ
ลูกค้ากรอก 3 อย่าง: **จำนวนคน / วันทำการ / เวลาเริ่ม–สิ้นสุด**

| # | เงื่อนไข | ผิดแล้วตอบ |
|---|---|---|
| 1 | ร้านมีอยู่จริงและไม่ถูกลบ | 404 |
| 2 | `end_at > start_at` | 400 |
| 3 | `start_at` ยังไม่ผ่านไปแล้ว | 400 |
| 4 | **จองล่วงหน้าอย่างน้อย 30 นาที** (`start_at >= now + 30m`) — กันจองรอบที่กำลังจะเริ่มใน 2 นาที | 400 `TOO_LATE_TO_BOOK` |
| 5 | ช่วงจองอยู่ในเวลาเปิด–ปิดของร้าน (รองรับเปิดข้ามเที่ยงคืน + 24 ชม.) และวันทำการนั้นไม่ใช่วันปิดประจำสัปดาห์ | 400 (`CLOSED_WEEKDAY` ถ้าตกวันปิด) |
| 5b | ช่วงจองไม่ทับช่วงที่ร้านปิดชั่วคราว (ตรวจในล็อก ก่อนกฎข้อ 8) | 400 `RESTAURANT_CLOSED` |
| 6 | `party_size <= restaurant.seats` (ขอเกินความจุร้านไปเลย) | 400 |
| 7 | **ที่นั่งไม่เกินในทุกวินาที** — ดู 5.3 | **409** `NOT_ENOUGH_SEATS` |
| 8 | ผู้ใช้คนเดียวกันจองร้านเดียวกันซ้อนเวลากันเองไม่ได้ | **409** `DUPLICATE_BOOKING` |
| 9 | จองล่วงหน้าไม่เกิน 90 วัน และช่วงละ 30 นาที – 4 ชม. (กฎเราเอง — กัน abuse, อธิบายได้) | 400 |
| 10 | เวลาเริ่ม/สิ้นสุดเป็นช่วงละ 30 นาที (:00 หรือ :30) | 400 |

**กดจองซ้ำ / เน็ตกระตุก:** ไม่ใช้ `Idempotency-Key` — **กฎข้อ 8 กันการจองซ้ำให้อยู่แล้ว**
แต่กันได้ต่อเมื่อกฎข้อ 8 ถูกตรวจ **ในทรานแซกชันที่ล็อกแถวร้านแล้ว และตรวจก่อนกฎข้อ 7** (ดู 5.3)
ถ้าตรวจก่อนเข้าทรานแซกชัน สองคำขอที่มาพร้อมกันจะอ่านเจอว่า "ยังไม่มี" ทั้งคู่แล้วเขียนทั้งคู่
ถ้าตรวจหลังกฎข้อ 7 คำขอที่สองอาจได้ `NOT_ENOUGH_SEATS` ทั้งที่จริงคือกดซ้ำ

409 สองแบบต้องแยกกันให้ชัด เพราะผู้ใช้ต้องทำคนละอย่าง:

| code | ความหมาย | หน้าเว็บทำอะไร |
|---|---|---|
| `NOT_ENOUGH_SEATS` | ร้านเต็มในช่วงนั้น | refetch availability + บอกที่นั่งที่เหลือจริง + เสนอช่วงที่ยังว่าง |
| `DUPLICATE_BOOKING` | ผู้ใช้มีการจองร้านนี้ที่ทับช่วงนี้อยู่แล้ว | แจ้ง **"คุณมีการจองที่ทับช่วงนี้อยู่แล้ว"** + ปุ่มไป `/bookings/:id` ของรายการเดิม (`details.booking_id`) — ใช้สีกลาง ไม่ใช่แดง และ **ห้ามบอกว่าร้านเต็ม** |

ห้ามแสดง `DUPLICATE_BOOKING` เป็น "จองสำเร็จ" — รายการเดิมอาจไม่ได้มาจากการกดซ้ำ
(เช่น จองไว้ 18:00–19:00 เมื่อสัปดาห์ก่อน วันนี้ขอ 18:30–20:00) ให้ผู้ใช้เห็นรายการเดิมแล้วตัดสินใจเองว่าจะแก้ไขรายการนั้นไหม
(ในระบบที่การซ้ำเป็นเรื่องถูกต้องได้ เช่น ตัดเงินมัดจำ กฎธุรกิจกันซ้ำให้ไม่ได้ ต้องใช้ `Idempotency-Key` จริง — เตรียมตอบข้อนี้)

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

**ต้องใช้ sweep line** — ตรวจเฉพาะ "จุดเปลี่ยนแปลง" เขียนเป็นสองฟังก์ชัน ใช้ซ้ำได้ทั้งระบบ:

```go
// internal/booking/availability.go

// maxConcurrent คืนจำนวนคนสูงสุดที่อยู่ในร้านพร้อมกันภายในช่วง [start,end) และเวลาที่เกิดค่าสูงสุดนั้น
// ใช้ทั้งตอนจอง (5.2), ตอนแก้ไข (5.5), ตอนลดที่นั่ง (5.1) และตอนคำนวณ slot ของ availability (6)
func maxConcurrent(bookings []Booking, start, end time.Time) (peak int, at time.Time) {
    // จุดที่ต้องตรวจ = จุดเริ่มของช่วง + ทุกจุดที่มีคนเข้าร้านเพิ่มภายในช่วงนั้น
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

// existing = booking ที่ status='active' และช่วงทับกับ [start,end) โดยไม่รวมตัวที่กำลังแก้
func checkSeats(existing []Booking, seats, partySize int, start, end time.Time) error {
    peak, at := maxConcurrent(existing, start, end)
    if peak+partySize > seats {
        return &NotEnoughSeatsError{At: at, Available: seats - peak} // → 409 NOT_ENOUGH_SEATS
    }
    return nil
}
```
เหตุผลที่ตรวจแค่จุดเริ่ม: จำนวนคนในร้านเพิ่มขึ้นได้เฉพาะตอนมีคนเริ่มจองใหม่ ระหว่างสองจุดค่าคงที่
(จุดที่คนออกไม่ต้องตรวจ เพราะค่าลดลง)

**ทุกการตรวจต้องอยู่ในทรานแซกชันเดียวที่ล็อกแถวร้านแล้ว — รวมกฎข้อ 8 ด้วย:**
```go
err := db.Transaction(func(tx *gorm.DB) error {
    // 1) ล็อกแถวร้าน — คนที่สองจะรอที่บรรทัดนี้
    if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
        First(&r, "id = ? AND deleted_at IS NULL", restaurantID).Error; err != nil { return err }

    // 2) กฎข้อ 8 — จองซ้อนตัวเอง ต้องเช็ค "ข้างใน" ล็อก และก่อนกฎข้อ 7
    var dup Booking
    err := tx.Where("restaurant_id = ? AND user_id = ? AND status = 'active'", restaurantID, userID).
       Where("start_at < ? AND end_at > ?", end, start).
       Where("id <> ?", excludeID).       // จองใหม่ใช้ uuid.Nil
       First(&dup).Error
    if err == nil { return &DuplicateBookingError{BookingID: dup.ID} }   // → 409 DUPLICATE_BOOKING
    if !errors.Is(err, gorm.ErrRecordNotFound) { return err }

    // 3) กฎข้อ 7 — อ่าน booking ที่ทับกัน "ข้างใน" ล็อก
    var existing []Booking
    tx.Where("restaurant_id = ? AND status = 'active'", restaurantID).
       Where("start_at < ? AND end_at > ?", end, start).
       Where("id <> ?", excludeID).      // ⚠️ ตอนแก้ไข: ห้ามนับที่นั่งเดิมของตัวเองซ้ำ
       Find(&existing)

    // 4) ตัดสิน แล้วเขียน — ทั้งหมดในล็อกเดียว
    if err := checkSeats(existing, r.Seats, partySize, start, end); err != nil { return err }
    return tx.Create(&booking).Error
})
```
ห้ามเช็คที่นั่งนอกทรานแซกชัน — ร้านเหลือ 6 ที่ สองคนกดคนละ 6 ที่พร้อมกัน
ต้องสำเร็จคนเดียว อีกคนได้ 409 (ทางเลือกอื่นที่อธิบายได้: advisory lock ต่อร้าน, `SERIALIZABLE` + retry,
หรือ `EXCLUDE USING gist` บน `(user_id, restaurant_id, tstzrange(start_at,end_at))` ให้ DB กันซ้ำเอง —
เลือก FOR UPDATE เพราะง่ายสุดและอธิบายได้ในหนึ่งประโยค)

### 5.4 เวลาเปิด–ปิด และนิยาม "วันทำการ"

```
ปกติ    open=11:00 close=22:00 → open_minute(660) < close_minute(1320)
ข้ามวัน  open=18:00 close=02:00 → close_minute(120) <= open_minute(1080)
24 ชม.  open=00:00 close=00:00 → เท่ากัน
```

```go
// internal/booking/businessday.go
duration := (closeMin - openMin + 1440) % 1440
if duration == 0 { duration = 1440 }   // ⚠️ ร้าน 24 ชม. — ถ้าไม่ใส่บรรทัดนี้จะจองไม่ได้เลย
```
วิธีตรวจ: แปลงช่วงจองเป็น "นาทีนับจากเวลาเปิดของรอบนั้น" แล้วเทียบกับ `duration`
`fitsOpeningHours(r, start, end)` = `offset(start) >= 0 && offset(end) <= duration` — ใช้ทั้งตอนจอง (กฎ 5) และตอนย่นเวลา (5.1)

**`date` = วันทำการ = รอบเปิดที่เริ่มในวันปฏิทินนั้น** ใช้ความหมายเดียวกันทั้งระบบ
(`GET /availability?date=`, `GET /restaurants?date=`, `GET /restaurants/:id/bookings?date=` ของ owner, และฟอร์มจอง)
```
opens_at  = date + open_minute (เวลาไทย)
closes_at = opens_at + duration
รอบของ date = [opens_at, closes_at)
```
- ร้านเปิด 18:00–02:00 ขอ `date=2026-10-10` → ได้ช่วง 18:00 ของวันที่ 10 ถึง 02:00 ของวันที่ 11 **ต่อเนื่องในรอบเดียว**
- ผู้ใช้เลือกวันทำการ 10 แล้วใส่เวลา 00:30 → server แปลงเป็น `2026-10-11T00:30+07:00`
  กฎแปลง: **ถ้ารอบของร้านข้ามเที่ยงคืน และเวลาที่เลือก < `open_minute` แปลว่าอยู่ในส่วนหลังเที่ยงคืน → บวก 1 วัน**
  (ร้านปกติ 11:00–22:00 เวลาก่อนเปิดคือ "ยังไม่เปิด" ของวันเดียวกัน — ห้ามบวกวัน ไม่งั้นค้น 10:30 แล้วได้ปุ่มเวลาของพรุ่งนี้)
- `date=2026-10-11` ต้อง **ไม่มี** ช่วงตี 0–2 ของเช้าวันที่ 11 เพราะนั่นเป็นของรอบวันที่ 10 ไปแล้ว
  (บั๊กที่เกิดง่ายที่สุด — ต้องมีเทสต์)
- owner `?date=` เอา booking ที่ `start_at` อยู่ใน `[opens_at, closes_at)` — แขกตี 1 คืนวันเสาร์อยู่ในบอร์ดวันเสาร์
- ร้านปกติ วันทำการ = วันปฏิทิน ไม่ต้องทำอะไรเพิ่ม
- **วันปิดประจำสัปดาห์ผูกกับวันทำการ** (`Hours.ClosedOn(date)`): ร้าน 18:00–02:00 ปิดวันจันทร์ → ตี 1 เช้าวันจันทร์ยังจองได้ (รอบวันอาทิตย์)
  แต่ตี 1 เช้าวันอังคารจองไม่ได้ (รอบวันจันทร์); ร้าน 24 ชม. จองคร่อมเที่ยงคืนเข้าวันปิดไม่ได้
  ตั้งวันปิดทับวันที่มี booking ในอนาคต → 409 `HOURS_CONFLICT_EXISTING_BOOKINGS` (ผ่าน `Fits` ตัวเดียวกับย่นเวลา)
- รอบของ "วันนี้" ต้องตัดช่วงที่เลยไปแล้วและช่วงที่เหลือน้อยกว่า lead time 30 นาทีออก

**ช่วงพัก (ร้านเปิด 2 ช่วง):** ร้านมีรอบเดียว + ช่วงพักได้ 1 ช่วง เช่น 11:00–22:00 พัก 14:00–17:00
- วันทำการ/`Window`/`At` ไม่เปลี่ยน — ช่วงพักเป็นแค่ "รูในรอบ"
- `Fits` ปฏิเสธช่วงจองที่ทับช่วงพัก (จบตอนเริ่มพักพอดีได้); ร้าน 24 ชม. ที่คร่อมเข้ารอบถัดไปตรวจช่วงพักของรอบถัดไปด้วย
- `Slots` ข้ามช่วงพัก; `OpenAtMinute` ตอบ false ในช่วงพัก (ใช้ใน next-available)
- หน้าเว็บรู้ว่าตรงไหนเป็นช่วงพักจาก slot ที่ต่อกันไม่สนิท (`isGap`/`contiguousFrom`) — API ไม่ต้องส่งช่วงพักแยก

**ช่วงปิดชั่วคราว:** เก็บเป็นช่วงเวลาจริง ตรวจด้วย `closureAt` ตัวเดียว (`checkSlot`, `Slots`, `SlotsAround`); แปลงคำขอด้วย `Hours.Span`/`Hours.Days`

**ตัวแปลงวันทำการ ↔ เวลาจริง ต้องเป็นฟังก์ชันเดียวใน `businessday.go`** ที่ availability, list, POST /bookings,
PUT /bookings และหน้า owner เรียกใช้ร่วมกัน — ห้ามเขียนซ้ำในแต่ละที่

### 5.5 แก้ไขและยกเลิก
| กฎ | ผล |
|---|---|
| แก้/ยกเลิกได้เฉพาะ booking ของตัวเอง | 403 |
| แก้ไขใช้กติกาเดียวกับจองใหม่ทุกข้อ + `excludeID = booking.ID` | |
| booking ที่ `cancelled` แล้ว แก้/ยกเลิกอีกไม่ได้ | 409 `BOOKING_CANCELLED` |
| booking ที่เลยเวลาไปแล้ว แก้/ยกเลิกไม่ได้ | 409 `BOOKING_ALREADY_STARTED` |
| ยกเลิกได้เมื่อ `now <= start_at - cancel_before_minutes` | ช้ากว่านั้น → **403** `CANCEL_WINDOW_PASSED` พร้อม `details.cancel_until` |
| การแก้ไขแบบเลื่อนเวลา ใช้ cancel window ของเวลา**เดิม** | กันเลี่ยงกฎด้วยการเลื่อนเวลาแทนยกเลิก |

ยกเลิกโดยลูกค้าเอง (`DELETE /bookings/:id`) → `cancelled_by='customer'` (ต่างจากร้านยกเลิกที่ `cancelled_by='restaurant'` ดู 5.1)

ตัวอย่างจากโจทย์: ร้าน 10 ที่ A จอง 7 B จอง 3 (เต็มพอดี) → A แก้เป็น 8 คน = ต้องใช้ 11 ที่ → **ปฏิเสธ**; A แก้เป็น 5 คน → **ผ่าน**

### 5.6 รีวิว
| กฎ | ผล |
|---|---|
| คะแนนเป็นจำนวนเต็ม 1–5 + ข้อความ | ไม่ใช่ → 400 |
| **รีวิวร้านตัวเองไม่ได้** (`owner_id == userID`) | **403** `OWN_RESTAURANT` |
| 1 คน 1 รีวิวต่อร้าน — **แยก create/update ชัดเจน** | `POST` ซ้ำ → **409** `REVIEW_EXISTS`; `PUT` ตอนยังไม่เคยรีวิว → **404** |
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

`sort=rating` → **Highest rated ด้วย Bayesian average**
`score = (C·m + rating_sum) / (C + rating_count)` โดย `C = 5`,
`m` = ค่าเฉลี่ยของทุกรีวิวในระบบ (ร้านที่ไม่ถูกลบ)

`score` **ไม่ใช่คอลัมน์ในตาราง** ต้องคำนวณใน query จริง:
```sql
WITH g AS (
  SELECT COALESCE(SUM(rating_sum)::float / NULLIF(SUM(rating_count), 0), 0) AS m
  FROM restaurants WHERE deleted_at IS NULL
)
SELECT r.*,
       (5 * g.m + r.rating_sum) / (5 + r.rating_count) AS score
FROM restaurants r CROSS JOIN g
WHERE r.deleted_at IS NULL
ORDER BY (r.rating_count = 0), score DESC, r.rating_count DESC, r.id
LIMIT :limit OFFSET :offset;
```
- `::float` สำคัญ — ไม่งั้น `SUM(int) / SUM(int)` เป็นหารจำนวนเต็ม ค่าเฉลี่ย 4.37 จะกลายเป็น 4
- **ร้านที่ยังไม่มีรีวิวอยู่ท้ายสุดเสมอ** (`rating_count = 0` มาก่อนใน ORDER BY) — ไม่งั้นจะได้คะแนน = m พอดีและแซงร้านที่มีรีวิวจริงแต่ต่ำกว่าค่าเฉลี่ย
- `sort=reviews` → เรียงตาม `rating_count DESC, id`
- default → ร้านใหม่สุด (`created_at DESC, id`)
- ต้องมี tie-breaker คงที่ (`id`) ทุก sort ไม่งั้น pagination จะได้ข้อมูลซ้ำ/หาย
- (ทางเลือกที่พิจารณาแล้ว: เกณฑ์ขั้นต่ำ ≥ 5 รีวิว — ง่ายกว่าแต่มีเส้นตัดแข็ง; เขียนเปรียบเทียบไว้ใน README)
- UI แสดง "รีวิวน้อย (n)" แทนดาวเด่นเมื่อรีวิว < 5 ให้สอดคล้องกับเหตุผลเดียวกัน (ดู 8.4)

---

## 6. API

prefix `/api/v1` — JSON ทั้งหมด — base URL `http://api.jongyoung.localhost/api/v1` — Swagger ที่ `/swagger/index.html`

| Method | Path | Auth | หมายเหตุ |
|---|---|---|---|
| GET | `/me` | ✓ | โปรไฟล์ + ร้านที่เป็นเจ้าของ (web ใช้ตัดสินว่าจะโชว์ตัวสลับโหมด owner + ซ่อนปุ่มรีวิวร้านตัวเอง) |
| GET | `/restaurants` | – | `?q=&cuisine=&sort=rating\|reviews\|newest&page=&limit=` + ตัวเลือก `&date=&time=&party_size=` → แต่ละร้านมี `slots` (ดูด้านล่าง) |
| GET | `/restaurants/:id` | – | + คะแนนเฉลี่ย + จำนวนรีวิว + รูปทั้งหมด + `address` + `cancel_before_minutes` |
| GET | `/restaurants/:id/availability` | – | `?date=YYYY-MM-DD` (**วันทำการ**) `&party_size=` → ทุกช่วง 30 นาทีของรอบ |
| GET | `/restaurants/:id/next-available` | – | `?date=&time=&party_size=` → วันทำการถัดไป (ไม่เกิน 14 วัน) ที่มีช่วงว่างพอ + ช่วงเหล่านั้น; ไม่เจอ → `null` |
| POST | `/restaurants` | ✓ | |
| PUT / DELETE | `/restaurants/:id` | ✓ owner | |
| POST | `/restaurants/:id/images` | ✓ owner | รับ `{ url }` |
| DELETE | `/restaurants/:id/images/:imageId` | ✓ owner | ลบรูปสุดท้ายไม่ได้ (400 `IMAGE_REQUIRED`) |
| GET | `/restaurants/:id/closures` | – | ช่วงปิดที่ `end_at > now` เรียงตามเวลาเริ่ม |
| POST | `/restaurants/:id/closures` | ✓ owner | ปิดชั่วคราว (บางช่วง/ทั้งวัน/หลายวัน) — ทับการจองต้องยืนยันด้วย `confirm_booking_ids` → 409 `CLOSURE_AFFECTS_BOOKINGS` |
| DELETE | `/restaurants/:id/closures/:closureId` | ✓ owner | เปิดร้านกลับ; การจองที่ยกเลิกไปแล้วไม่ฟื้น |
| GET | `/restaurants/:id/bookings` | ✓ owner | `?date=` (**วันทำการ เหมือนกัน**) — แขกตอนตี 1 ของคืนวันเสาร์ต้องอยู่ในวันเสาร์ |
| POST | `/bookings` | ✓ | body `{ restaurant_id, date, start_time, end_time, party_size }` (`date` = วันทำการ, เวลาเป็น `HH:MM` แล้ว server แปลงด้วย `businessday.go`) → 409 `NOT_ENOUGH_SEATS` หรือ `DUPLICATE_BOOKING` |
| GET | `/me/bookings` | ✓ | `?status=upcoming\|past\|cancelled` |
| GET | `/bookings/:id` | ✓ เจ้าของ booking หรือ owner ร้าน | ใช้เป็นหน้ายืนยันการจอง |
| PUT | `/bookings/:id` | ✓ เจ้าของ booking | แก้จำนวนคน/วัน/เวลา |
| DELETE | `/bookings/:id` | ✓ เจ้าของ booking | ยกเลิก → `status=cancelled` (ไม่ลบจริง เก็บประวัติ) |
| GET | `/restaurants/:id/reviews` | – | `?page=&limit=` |
| GET | `/restaurants/:id/reviews/mine` | ✓ | รีวิวของฉันในร้านนี้ (ยังไม่มี → 404) ใช้ตัดสินว่าฟอร์มเป็น POST หรือ PUT |
| POST | `/restaurants/:id/reviews` | ✓ | สร้าง; มีอยู่แล้ว → 409; เจ้าของร้าน → 403 |
| PUT | `/restaurants/:id/reviews` | ✓ | แก้ของตัวเอง; ยังไม่มี → 404 |
| DELETE | `/restaurants/:id/reviews` | ✓ | ลบของตัวเอง |
| GET | `/me/notifications` | ✓ | `{ items: [...20 ล่าสุด], unread_count }` |
| PUT | `/me/notifications/:id/read` | ✓ | ของคนอื่น/ไม่มี → 404 (ไม่บอกว่ามีอยู่จริง) |
| PUT | `/me/notifications/read-all` | ✓ | อ่านทั้งหมด |

**Availability response — ต้องส่ง timestamp เต็ม ไม่ใช่ `"00:30"` ลอย ๆ**
เพราะรอบข้ามเที่ยงคืนทำให้เวลาเดียวกันอยู่คนละวัน ฝั่ง web ต้องไม่ต้องเดา:
```json
{
  "business_date": "2026-10-10",
  "opens_at":  "2026-10-10T18:00:00+07:00",
  "closes_at": "2026-10-11T02:00:00+07:00",
  "seats": 10,
  "slots": [
    { "start_at": "2026-10-10T18:00:00+07:00", "end_at": "2026-10-10T18:30:00+07:00", "available": 10 },
    { "start_at": "2026-10-11T00:30:00+07:00", "end_at": "2026-10-11T01:00:00+07:00", "available": 4 }
  ]
}
```
`available` = `seats - peak` ของช่วงนั้น (จาก `maxConcurrent`) และช่วงที่เลย lead time ไปแล้วจะไม่อยู่ในลิสต์

**`slots` บนการ์ดร้าน (`GET /restaurants?date=&time=&party_size=`)**
- 5 ช่วงรอบ `time` รูปแบบเดียวกับ availability (+ `"closed": true` ถ้าช่วงนั้นร้านปิด)
- คำนวณจาก **query booking เดียวต่อหน้า** (`restaurant_id IN (...)` ของร้านในหน้านั้น ในรอบวันทำการนั้น) แล้วแบ่งตามร้านใน Go
- **ห้ามให้หน้าเว็บยิง availability ทีละร้าน** (N+1 — หน้าหนึ่ง 6–12 request)

**Status code ที่ต้องใช้ให้ถูก**
`200/201/204` สำเร็จ · `400` รูปแบบ/เงื่อนไขเวลาผิด · `401` ไม่มี/token ไม่ถูก/หมดอายุ ·
`403` สิทธิ์ไม่ถึง (ไม่ใช่เจ้าของ, รีวิวร้านตัวเอง, ยกเลิกช้าเกิน) · `404` ไม่มีข้อมูล ·
`409` ชนกับสถานะปัจจุบัน (ที่นั่งไม่พอ, จองซ้อน, ยกเลิกซ้ำ, ลดที่นั่งไม่ได้, รีวิวซ้ำ) · `500` ที่เหลือ

**Error response รูปแบบเดียวทั้งระบบ** (`internal/httputil`)
```json
{ "error": { "code": "NOT_ENOUGH_SEATS", "message": "ช่วง 12:30–13:00 เหลือ 3 ที่นั่ง", "details": { "available": 3, "at": "2026-10-10T12:30:00+07:00" } } }
```
ฝั่ง web แปล `code` เป็นข้อความไทย — ไม่พึ่ง `message` จาก API ในการตัดสินใจ

| code | status | details |
|---|---|---|
| `TOO_LATE_TO_BOOK` | 400 | `earliest_start_at` |
| `CLOSED_WEEKDAY` | 400 | – |
| `INVALID_BREAK` | 400 | – |
| `NOT_ENOUGH_SEATS` | 409 | `available`, `at` |
| `DUPLICATE_BOOKING` | 409 | `booking_id` |
| `BOOKING_CANCELLED` / `BOOKING_ALREADY_STARTED` | 409 | – |
| `CANCEL_WINDOW_PASSED` | 403 | `cancel_until` |
| `SEATS_BELOW_EXISTING_BOOKINGS` | 409 | `peak`, `at` |
| `HOURS_CONFLICT_EXISTING_BOOKINGS` | 409 | `booking_ids` |
| `OWN_RESTAURANT` | 403 | – |
| `REVIEW_EXISTS` | 409 | – |
| `RESTAURANT_CLOSED` | 400 | `reason`, `start_at`, `end_at` |
| `INVALID_CLOSURE` | 400 | – |
| `CLOSURE_AFFECTS_BOOKINGS` | 409 | `bookings` |

**Middleware order:** Recovery → RequestID/Logger → CORS (`http://jongyoung.localhost` เท่านั้น) → RateLimit(เบา ๆ) → JWT (verify + JIT provisioning) → เช็คความเป็นเจ้าของใน service

---

## 7. หน้าเว็บ (Next.js)

| Path | เนื้อหา | Rendering |
|---|---|---|
| `/` | แถบค้นหา (วันที่ / เวลา / จำนวนคน) + รายการร้าน พร้อม **ปุ่มเวลาบนการ์ด** | server (ไม่ต้องใช้ token) + client island (ตัวกรอง) |
| `/restaurants/[id]` | รายละเอียด + รูป + ปุ่มเวลา + แผงจอง + กติกายกเลิก + รีวิว | server + client island |
| `/bookings/[id]` | **หน้ายืนยันการจอง** — เลขที่จอง, ร้าน + ที่อยู่ + แผนที่, เวลา, "ยกเลิกได้ถึง …", เพิ่มลงปฏิทิน | client |
| `/me/bookings` | การจองของฉัน (tab กำลังจะถึง/ผ่านมาแล้ว/ยกเลิกแล้ว) แก้-ยกเลิกในหน้านี้ | client |
| `/owner/restaurants` | ร้านของฉัน (ตาราง + สร้าง/แก้/ลบ) | client |
| `/owner/bookings?restaurant=&date=` | ตารางการจองรายวันทำการ + แถบที่นั่งต่อช่วง (เลือกร้านจาก dropdown) | client |
| `/owner/closures?restaurant=` | ปิดร้านชั่วคราว (บางช่วง/ทั้งวัน/หลายวัน) + ตารางช่วงปิดที่ยังไม่จบ + เปิดร้านกลับ | client |
| `/api/auth/[...nextauth]`, `/api/auth/logout` | next-auth + logout | – |

กระดิ่งแจ้งเตือนใน navbar ทั้งโหมดลูกค้าและเจ้าของร้าน (poll 60 วินาที)

**Flow การจอง = 2 แท็ป:** กดปุ่มเวลาบนการ์ดร้าน → หน้าร้านเปิดพร้อมเวลาที่เลือกไว้ในแผงจอง → "ยืนยันการจอง" → `/bookings/[id]`

- ฟอร์มใช้ react-hook-form + zod
- **การจอง: ห้ามใช้ optimistic update** — ต้องรอผลจริง; `NOT_ENOUGH_SEATS` → refetch availability + เสนอช่วงที่ยังว่าง;
  `DUPLICATE_BOOKING` → "คุณมีการจองที่ทับช่วงนี้อยู่แล้ว" + ปุ่มไป `/bookings/:id` ของรายการเดิม (ตาราง 5.2) (รีวิวใช้ optimistic ได้)
- ปุ่มกดจองต้อง disable ตั้งแต่ก่อนยิง และเก็บสถานะ submitting ที่ตัว form (กัน Enter ซ้ำ)
- หน้ายืนยัน: เพิ่มลงปฏิทินได้ 2 ทาง — ลิงก์ Google Calendar (template URL) และไฟล์ `.ics` ที่สร้างฝั่ง web (`lib/ics.ts`)
  แผนที่ = `map_url` ของร้าน ถ้าไม่มีค่อยค้นหาจาก `address` (`mapHref` ใน `lib/format.ts`, ไม่ต้องใช้ API key)
- ซ่อนปุ่มแก้/ลบเมื่อไม่ใช่เจ้าของ — เป็นเรื่อง UX เท่านั้น API ต้องกันซ้ำเสมอ
- ปุ่มที่กดไม่ได้เพราะกติกา (เช่น เลยเวลายกเลิก) ให้ disabled พร้อมข้อความบอกเหตุผล ไม่ใช่ซ่อน
- แสดงเวลาเป็นเวลาไทยทุกที่ (`Intl.DateTimeFormat('th-TH', { timeZone: 'Asia/Bangkok' })`)
- empty state / loading skeleton / error state ทุกหน้า (+ 401 เซสชันหมดอายุ → login แล้วกลับมาหน้าเดิม)

---

## 8. Design system

**Mockup ทุกหน้า (Claude Design):** https://claude.ai/artifact/Cb7eGqYexzQd73X6AiqvKR
(ฝั่งลูกค้า 4 หน้า, หน้ารวมสถานะ empty/error/409, ฝั่งเจ้าของร้าน 2 หน้า, มือถือ 2 หน้า)
**ต้นแบบคาแรกเตอร์:** [guy127/introduce_myself](https://github.com/guy127/introduce_myself) — เอาแค่โทนสี/แบรนด์

> **หลักคิด:** นี่คือระบบที่คนกดแล้วเกิดข้อผูกพัน (มีกติกายกเลิก มีที่นั่งจำกัด) ไม่ใช่เว็บพอร์ตโฟลิโอ
> ความสวยต้องไม่แลกกับความเร็วในการตัดสินใจ งานหลักของหน้าแรกคือ **"ร้านไหนว่างตอนที่ฉันอยากไป"**
> ไม่ใช่การอวดภาพใหญ่ ๆ

### 8.0 สิ่งที่เปลี่ยนจากแนวคิดเดิม (สำคัญ — อย่าลอกแบบเดิม)

| เดิม | ใหม่ | เหตุผล |
|---|---|---|
| Night เป็นค่าเริ่มต้น | **Day เป็นค่าเริ่มต้น** Night เป็นตัวเลือก | หน้าจองคืองาน data entry (วัน เวลา จำนวนคน) โหมดสว่างอ่านง่ายกว่าและดูน่าเชื่อถือกว่า |
| กระจกขุ่น (blur) ทั้งระบบ | blur เฉพาะ **navbar + hero + booking panel** การ์ดร้านใช้พื้นทึบ + เส้นขอบ | `backdrop-filter` บนการ์ดหลายสิบใบทำให้มือถือกลาง ๆ กระตุก และ blur ทับ glow ทำให้ contrast คาดเดาไม่ได้ (เคลม AA ไม่ได้) |
| Seat bar เป็น UI ของลูกค้า | **Time chip** สำหรับลูกค้า / **Seat bar ย้ายไปหน้า owner** | ลูกค้าถามว่า "ทุ่มนึงจองได้ไหม" ไม่ได้ถามว่า "ร้านเต็มกี่เปอร์เซ็นต์" |
| เข้าหน้าร้านก่อนถึงจะเลือกเวลาได้ | เลือกเวลาได้ตั้งแต่หน้าแรก | 2 แท็ปจบ แทน 5 — มาตรฐานของ OpenTable/Resy/Hungry Hub |
| Kanit + Prompt ทั้งหน้า | **Prompt ทั้งระบบ** — Kanit 600 น้ำหนักเดียว เฉพาะโลโก้และหัวข้อ hero | โหลดเบากว่า และไทยสองทรงในเนื้อหาดูไม่นิ่ง |
| hero 56–84px | **36–44px** แล้วดันแถบค้นหาขึ้นมาแทน | พื้นที่ครึ่งจอบนควรเป็นเครื่องมือ ไม่ใช่ป้ายชื่อ |

### 8.1 คาแรกเตอร์และ Font

**คาแรกเตอร์:** อบอุ่นแบบร้านอาหารตอนเย็น แต่คมและอ่านง่ายแบบเครื่องมือจอง —
พื้นสว่างอุ่น การ์ดขอบชัด แสงเรืองแดง/อำพันเฉพาะ hero กับปุ่มหลัก

```
ทั้งระบบ: "Prompt" 400 / 500 / 600   (Google Fonts css2 + display=swap — แยก subset ไทยตาม unicode-range ให้เอง)
โลโก้และหัวข้อ hero: "Kanit" 600       (เฉพาะสองที่นี้เท่านั้น)
ตัวเลข/เวลา/จำนวนที่นั่ง: font-variant-numeric: tabular-nums
```
ขนาด: hero 36–44px / หัวข้อ section 20–28px / ชื่อร้านบนการ์ด 19–20px / เนื้อหา 16px / กำกับ 14px / เล็กสุด 13px
**ห้ามต่ำกว่า 13px สำหรับภาษาไทย** · `line-height: 1.7` เนื้อหา · หัวข้อไทย **≥ 1.2** (สระ/วรรณยุกต์ต้องไม่ชนบรรทัดบน)
· letter-spacing ไม่ติดลบเกิน -0.01em

### 8.2 สี — token สองโหมด (Day = default)

| Token | Day (default) | Night | ใช้กับ |
|---|---|---|---|
| `--bg` | `#fbf6f1` | `#11070a` | พื้นหลัง (glow อ่อน ๆ เฉพาะบนสุดของหน้า) |
| `--surface` | `#ffffff` | `#1b0e12` | พื้นการ์ด/ตาราง (ทึบ) |
| `--chip` | `#f4ece7` | `#26151a` | พื้นปุ่มรอง, ปุ่มเวลาที่เต็ม |
| `--text` | `#261419` | `#fff7ed` | ตัวอักษรหลัก |
| `--muted` | `#6f4c4d` | `#d7b8b4` | กำกับ |
| `--soft` | `#7d625f` | `#a88b8d` | meta เล็ก |
| `--accent` | `#c5162e` | `#ff5a6e` | แบรนด์ (โลโก้, hero) |
| `--eyebrow` / `--star` | `#8a5a00` / `#b77900` | `#ffb703` | ป้ายเล็กเหนือหัวข้อ / ไอคอนดาว |
| `--accent-3` | `#007c91` (อ่อน `#e3f3f5`) | `#43e8ff` (อ่อน 12%) | ลิงก์, **ช่วงเวลาที่เลือก** |
| `--cta` | `#c5162e → #b3361a` | `#e11d48 → #c2410c` | ปุ่มหลัก (gradient 135°) — ตัวขาวต้องผ่าน AA |
| `--ok` | `#006e40` (พื้น `#e6f5ec`) | `#27e88b` | ยืนยันแล้ว, สำเร็จ |
| `--warn` | `#8a5a00` (พื้น `#fdf1dc`, ขอบ `#efcf93`) | `#ffb703` | เหลือน้อย |
| `--full` | `#b0132a` (พื้น `#fcebed`, ขอบ `#f0b5bd`) | `#ff6b7f` | error, ยกเลิก, ลบ |
| `--border` / `--border-strong` | `#eadcd4` / `#d9c5bb` | `#3a262c` / `#4d333a` | เส้นขอบ / ขอบ input |
| `--focus` | `#111827` | `#ffffff` | **focus ring (ไม่ใช้สีเดียวกับช่วงที่เลือก)** |

กฎการใช้สี:
- **หนึ่งหน้าจอมีปุ่ม gradient (`--cta`) ได้ปุ่มเดียว** ที่เหลือเป็นปุ่มขอบบาง/พื้นทึบ
- เขียว-อำพัน-แดง (`--ok/--warn/--full`) ใช้สื่อสถานะที่นั่ง/การจอง/error เท่านั้น
- ช่วงเวลาที่เลือกใช้ `--accent-3`; **focus ring ใช้ `--focus` (outline 2px + offset 2px)** — สองอย่างนี้ต้องแยกออกจากกัน
- ตัวอักษรต้องผ่าน contrast **AA (4.5:1)** บนพื้นทึบเสมอ — ห้ามวางตัวอักษรเนื้อหาบนพื้นที่มี blur ทับ glow
- **ห้ามสื่อความหมายด้วยสีอย่างเดียว** ต้องมีไอคอนหรือข้อความกำกับ
- ไอคอนใช้ SVG (lucide) — **ไม่ใช้อีโมจิ** (รวมถึงตัวสลับโหมด)

### 8.3 Layout พื้นฐาน
```
max-width หน้า: 1120px, padding ขอบ 16px (mobile) / 24px (desktop)
spacing scale: 4 8 12 16 24 32 48
radius: 999px (ปุ่มหลัก/nav/badge), 16–24px (การ์ด/แผง), 12px (ปุ่มเวลา, input)
การ์ดร้าน: background var(--surface); border 1px var(--border); box-shadow 0 1px 2px rgba(38,20,25,.05)
glass (เฉพาะ navbar/hero/booking panel): backdrop-filter blur(14px) + พื้นกึ่งทึบ ≥ 78%
ไม่มี hover ลอยบนการ์ด (เปลี่ยนแค่สีขอบ) — และปิด transition ทั้งหมดเมื่อ prefers-reduced-motion
```

### 8.4 Component ฝั่งลูกค้า

**Search bar (บนสุดของหน้าแรก — งานหลักของหน้านี้)**
กล่องกระจกเดียว: `ร้าน/เมนู` · `วันทำการ` · `เวลาประมาณ` · `จำนวนคน` + ปุ่มค้นหา (CTA ของหน้า)
mobile: กดแล้วเปิด bottom sheet ทีละช่อง · ค่าเริ่มต้น = วันนี้ / ช่วงเวลาถัดไปที่จองได้ / 2 คน
ค่าที่เลือกต้องอยู่ใน URL (`?date=&time=&party_size=`) เพื่อแชร์ลิงก์และ refresh ได้

**Restaurant card + Time chips (ของสำคัญที่สุดของหน้าแรก)**
```
[รูป 16:8 + badge ประเภทอาหาร]
ร้านส้มตำแซ่บ
★ 4.3 · 58 รีวิว · 18:00–02:00
[18:00] [18:30] [19:00] [19:30] [20:00]
 ว่าง    ว่าง   เหลือ 3   เต็ม    ว่าง
```
- แสดง 5 chip ของช่วงรอบเวลาที่ค้นหา ครบทุกสถานะ (เต็ม = disabled) กดแล้วไปหน้าร้านพร้อมเวลาที่เลือกไว้แล้ว
- ไม่มีช่วงว่างเลย → ไม่ซ่อนร้าน แต่เปลี่ยนเป็นกล่อง "เต็มทั้งคืนนี้ · ว่างวันถัดไป" + chip จาก `next-available`
  หรือ "ร้านปิด 18:00 · ช่วงที่ยังว่าง" ถ้าเวลาที่ค้นเลยเวลาปิด
- คะแนน: รีวิว ≥ 5 → `★ 4.6 · 128 รีวิว`; รีวิว 1–4 → ป้าย "รีวิวน้อย (n)" แทนดาวเด่น (สอดคล้องกับ Bayesian); 0 → "ยังไม่มีรีวิว"

**Time chip (แทน seat bar ฝั่งลูกค้า)** — สถานะคิดจาก `available` ของ slot เทียบกับ **จำนวนคนที่ผู้ใช้เลือก** (`party`)
| สถานะ | เงื่อนไข | หน้าตา |
|---|---|---|
| ว่าง | `available >= party` และ `available > 30%` ของที่นั่ง | พื้น `--surface` ขอบ `--border-strong` ตัวอักษร `--text` + "✓ ว่าง" |
| เหลือน้อย | `available >= party` และ `available <= 30%` ของที่นั่ง | พื้น/ขอบ/ตัวอักษร `--warn` + "! เหลือ n" |
| เต็ม | `available < party` หรือร้านปิดช่วงนั้น | พื้น `--chip` ตัวอักษร `--soft` ขีดฆ่า + "✕ เต็ม" / "ปิด" — `disabled` จริง |
| เลือกอยู่ | – | พื้นทึบ `--accent-3` ตัวอักษรขาว; ช่วงที่ครอบด้วยระยะเวลา = เส้นประ `--accent-3` |
ขนาดขั้นต่ำ 44×44px · บนมือถือเรียงเป็นกริด 4 คอลัมน์หรือแถวเลื่อนแนวนอน

**ป้ายรอบข้ามเที่ยงคืน (ห้ามลืม)**
ทุกที่ที่แสดงเวลาหลังเที่ยงคืนของรอบนั้น ต้องมีป้ายกำกับวัน: `00:30 (เช้าวันที่ 11)`
ทั้งใน chip, ฟอร์มจอง, หน้ายืนยัน, การจองของฉัน และหน้า owner — ไม่งั้นลูกค้ามาผิดวันแน่นอน

**Booking panel (หน้าร้าน)**
กระจก sticky ขวาบน desktop / แผงลอยล่างบน mobile
```
ส. 10 ต.ค. · 18:00–19:00 · 2 คน
ยกเลิกหรือแก้ไขได้ถึง 17:30 (ก่อนเวลาจอง 30 นาที)     ← ต้องเห็น "ก่อน" กดจอง
[ ยืนยันการจอง ]
```
กติกายกเลิกต้องอยู่เหนือปุ่มเสมอ ไม่ใช่ไปโผล่หลังจองแล้ว

**หน้ายืนยันการจอง `/bookings/[id]`**
เลขที่จองตัวใหญ่ · ชื่อร้าน + ที่อยู่ + ลิงก์แผนที่ · วันเวลา (มีป้ายข้ามวันถ้าจำเป็น) · จำนวนคน ·
"ยกเลิกได้ถึง 17:30" · ปุ่มเพิ่มลงปฏิทิน (CTA) · ดูการจองของฉัน · แก้ไข
นี่คือหน้าที่ผู้ใช้ตัดสินว่าระบบน่าเชื่อถือ — ต้องออกแบบจริง ไม่ใช่ redirect กลับหน้าแรก

**Error / empty state**
- 409 `NOT_ENOUGH_SEATS` → กล่องแดง "ที่นั่งไม่พอแล้ว — ยังไม่ได้จองให้" + chip ล่าสุดที่โหลดใหม่ + ช่วงที่ยังว่างพอ (mobile: bottom sheet)
- 409 `DUPLICATE_BOOKING` → กล่องสีกลาง "คุณมีการจองที่ทับช่วงนี้อยู่แล้ว" + รายละเอียดรายการเดิม + ปุ่มไปดู
- เต็มทั้งรอบ → เสนอ "วันถัดไปที่ว่าง" และ "ร้านประเภทเดียวกันที่ว่างเวลานี้" (ไม่มีพิกัด จึงไม่ทำ "ร้านใกล้เคียง")
- ค้นหาไม่เจอ → แนะนำให้ลดจำนวนคนหรือขยายช่วงเวลา ไม่ใช่จอเปล่า

**Booking card (การจองของฉัน)**
กล่องวันที่ + ชื่อร้าน + badge สถานะที่มีทั้งสีและข้อความ (✓ ยืนยันแล้ว / ✕ ยกเลิกแล้ว) + ปุ่มแก้/ยกเลิก
เลยเวลายกเลิก → ปุ่ม disabled + ข้อความว่าทำไม · ยกเลิกต้องมีกล่องยืนยันที่ระบุจำนวนที่นั่งที่จะปล่อย

**Review**
- เปิดหน้าร้านของตัวเอง → **ไม่แสดงปุ่ม "เขียนรีวิว"** แต่ขึ้นข้อความ "นี่คือร้านของคุณ — เจ้าของร้านรีวิวร้านตัวเองไม่ได้" + ลิงก์ไปจัดการร้าน
  (ไม่ใช่ให้กดแล้วเด้ง 403 — API ยังต้องกัน 403 เหมือนเดิม)
- ยังไม่ล็อกอิน → ปุ่ม "เข้าสู่ระบบเพื่อรีวิว"
- ดาวสีอำพัน + ตัวเลขทศนิยม 1 ตำแหน่ง + จำนวนรีวิวเสมอ

**สลับบทบาท (หนึ่งบัญชี สองบทบาท)**
navbar มีตัวสลับแบบแบ่งสองช่อง `ลูกค้า | เจ้าของร้าน` ที่เห็นชัดว่าตอนนี้อยู่โหมดไหน (แสดงเมื่อ `/me` มีร้านที่เป็นเจ้าของ)
เข้าโหมดเจ้าของร้านแล้ว navbar เปลี่ยนเป็นแถบทึบสีเข้ม + ป้าย "โหมดเจ้าของร้าน"

### 8.5 ฝั่ง Owner — คนละภาษาการออกแบบ

`/owner/*` เป็น **เครื่องมือทำงาน** ไม่ใช่หน้าขาย เจ้าของร้านเปิดวันละหลายรอบ ความเร็วในการสแกนสำคัญกว่าความประทับใจ

- navbar ทึบสีเข้ม (`#261419` / Night `#000`) + ป้าย "โหมดเจ้าของร้าน" + ตัวสลับโหมด
- ไม่มี gradient ไม่มี glass ไม่มี hover ลอย — พื้นทึบ เส้นแบ่งชัด radius 6px ปุ่มหลักสีหมึกทึบ `--text` (ไม่ใช้แดง — แดงสงวนไว้ให้ลบ/ยกเลิก คนจะเข้าใจผิดว่า "บันทึก" เป็นปุ่มอันตราย)
- ตัวอักษร 14px ความหนาแน่นสูง แถวตาราง 36–40px
- **Seat bar อยู่ที่นี่**: แถวต่อช่วง 30 นาทีตลอดรอบวันทำการ แถบยาว = สัดส่วนที่ถูกจอง
  เขียว < 70% → อำพัน ≥ 70% → แดง 100% พร้อมตัวเลข `7/10` กำกับเสมอ
- ตารางการจองรายวันทำการ: เวลา · ชื่อลูกค้า · จำนวนคน · เลขที่จอง · สถานะ — เรียงตามเวลา
  มีแถบสรุปบนสุด (การจอง · ลูกค้ารวม · peak ของรอบ · ยกเลิก)
- ปุ่มเปลี่ยนวันทำการ ◀ ▶ + date picker
- error ในฟอร์มแสดง code จาก API (เช่น `SEATS_BELOW_EXISTING_BOOKINGS`) พร้อมเวลาที่ชน

### 8.6 Accessibility & performance
- `:focus-visible` outline 2px `--focus` + offset 2px ทุก interactive element
- แตะได้ ≥ 44×44px บนมือถือ · mobile-first ทดสอบที่ 375px ก่อน
- chip ที่เต็มต้องเป็น `disabled` จริง ไม่ใช่แค่สีจาง (screen reader ต้องรู้)
- `prefers-reduced-motion` → ปิด transform/transition ทั้งหมด
- รูปทุกใบมี `alt` + `next/image` + `sizes` ที่ถูกต้อง
- ฟอนต์: Prompt 3 น้ำหนัก + Kanit 1 น้ำหนักเท่านั้น (ผ่าน `next/font/google`)

### 8.7 สิ่งที่ต้องส่ง
Mockup ใน Claude Design ต้องมี **ไม่ใช่แค่ happy path** — ทำแล้วที่ลิงก์ด้านบน:
หน้าแรก (มี time chip), หน้าร้าน + booking panel, **หน้ายืนยันการจอง**, การจองของฉัน,
**หน้ารวมสถานะ** (ร้านเต็มทั้งรอบ, 409 ทั้งสองแบบ, ร้านของตัวเอง, ป้ายข้ามวัน, loading, error/401, ว่าง),
หน้า owner (ตาราง + seat bar), มือถือ 2 หน้า
เก็บลิงก์/ภาพไว้ที่ `docs/` และต้องกด Share ลิงก์ก่อนส่งให้ผู้ตรวจ

---

## 9. Test ที่ต้องมี

`internal/booking/availability_test.go` — table-driven ครอบ:
1. ร้าน 10 ที่ มีคนจอง 7 ขอเพิ่ม 5 ในช่วงทับกัน → **ปฏิเสธ**
2. A 7 (12:00–12:30), B 7 (12:30–13:00), C ขอ 3 (12:00–13:00) → **ผ่าน** ⭐ เคสที่ naive SUM พลาด
3. A 7 + B 3 เต็มพอดี → A แก้เป็น 8 = ปฏิเสธ / A แก้เป็น 5 = ผ่าน (ทดสอบ `excludeID`)
4. จองเวลาที่ผ่านมาแล้ว → ปฏิเสธ
5. **lead time**: จองรอบที่เริ่มอีก 20 นาที → ปฏิเสธ; อีก 30 นาทีพอดี → ผ่าน (boundary)
6. จองนอกเวลาเปิด + ร้านเปิดข้ามเที่ยงคืน 18:00–02:00 (ทั้งเคสผ่านและไม่ผ่าน)
7. **ร้าน 24 ชม. (`open=close=0`)** จองได้ทุกช่วง รวมช่วงคร่อมเที่ยงคืน — กัน `duration = 0`
8. จองคาบเกี่ยวเวลาปิดร้าน → ปฏิเสธ
9. ยกเลิกช้ากว่า `cancel_before_minutes` → ปฏิเสธ; ตรงเวลาพอดี → ผ่าน (boundary)
10. `party_size > seats` → ปฏิเสธ
11. booking ที่ยกเลิกแล้วไม่ถูกนับเป็นที่นั่งที่ถูกใช้
12. `maxConcurrent`: ไม่มี booking → `0`; เคส A/B/C → peak 10 ที่ 12:00 (คืนเวลาที่ถูกต้อง)

`internal/booking/businessday_test.go`
13. ร้าน 18:00–02:00 `date=10` → ได้ช่วง 18:00 (10 ต.ค.) ถึง 02:00 (11 ต.ค.)
14. ร้าน 18:00–02:00 `date=11` → **ไม่มี** ช่วงตี 0–2 ของเช้าวันที่ 11 (เป็นของรอบวันที่ 10)
15. เลือกวันทำการ 10 + เวลา 00:30 → แปลงเป็น `2026-10-11T00:30+07:00`
16. ร้านปกติ วันทำการ = วันปฏิทิน
16b. ช่วงพัก: จบตอนเริ่มพักผ่าน / คร่อมหรืออยู่ในพักไม่ผ่าน / ข้ามคืน / 24 ชม. คร่อมเข้าพักของรอบถัดไป / `ValidBreak` ติดขอบไม่ผ่าน

service test (mock repository ด้วย mockery `mocks_test.go`)
17. **กฎข้อ 8**: ผู้ใช้เดิมจองร้านเดิมซ้อนเวลา → `DUPLICATE_BOOKING` พร้อม `booking_id` (ไม่ใช่ `NOT_ENOUGH_SEATS`); คนละร้าน/ไม่ทับ → ผ่าน; ตอนแก้ไขไม่นับตัวเอง
18. **ลดที่นั่ง** ต่ำกว่า peak ของ booking ในอนาคต → `SEATS_BELOW_EXISTING_BOOKINGS`; เท่ากับ peak พอดี → ผ่าน
19. **ย่นเวลาเปิด** จน booking ในอนาคตตกนอกเวลา → `HOURS_CONFLICT_EXISTING_BOOKINGS`; booking ในอดีตไม่นับ
20. owner `?date=`: booking ตี 1 คืนวันเสาร์อยู่ในบอร์ดวันเสาร์ ไม่ใช่วันอาทิตย์
21. `EnsureExists`: ครั้งแรกสร้าง; ครั้งถัดไป email/ชื่อเปลี่ยน → อัปเดต
22. `next-available`: เต็มทั้งรอบวันที่ 10 → คืนวันที่ 11; เต็มครบ 14 วัน → `null`

repository / integration test (testcontainers — Postgres จริง รัน migrations/ ทุกไฟล์)
23. Bayesian sort: ★5.0 จาก 1 รีวิวต้องไม่อยู่เหนือ ★4.8 จาก 46 รีวิว; ร้านไม่มีรีวิวอยู่ท้ายสุด
24. `slots` บนการ์ด: ใช้ query booking เดียวต่อหน้า; ช่วงนอกเวลาเปิดได้ `closed`
25. การจองพร้อมกัน: ยิง 2 goroutine (คนละผู้ใช้) บนที่นั่งที่เหลือพอสำหรับคนเดียว → สำเร็จ 1, `NOT_ENOUGH_SEATS` 1
26. กดซ้ำ: ผู้ใช้เดียวกันยิงคำขอเดียวกัน 2 goroutine → สำเร็จ 1, `DUPLICATE_BOOKING` 1

handler test: `httptest` ยิงใส่ Gin router — เช็ค 401 (ไม่มี token), 403 (ไม่ใช่เจ้าของ / รีวิวร้านตัวเอง), 409 (ที่นั่งไม่พอ, จองซ้อน, รีวิวซ้ำ)

E2E (Playwright, `e2e/` — รันด้วย `./dev.sh e2e` กับระบบจริง, ล้าง seed ก่อนทุกครั้ง)
- flow ลูกค้า: ค้นหา → กดปุ่มเวลา → จอง → หน้ายืนยัน → การจองของฉัน → ยกเลิก; 409 ทั้งสองแบบที่หน้าเว็บ
- flow เจ้าของร้าน: บอร์ดรายวัน, ลดที่นั่งถูกปฏิเสธ, ร้านตัวเองรีวิวไม่ได้; หน้าค้นหาที่มือถือ 375px

web (Vitest + Testing Library)
27. สถานะ time chip: ว่าง / เหลือน้อย / เต็ม ตามกติกา 8.4 (เทียบกับ `party` ไม่ใช่ 1)
28. ป้ายคะแนน: รีวิว 0 / 1–4 / ≥ 5 → "ยังไม่มีรีวิว" / "รีวิวน้อย" / ดาว
29. ป้าย "(เช้าวันที่ n)" ขึ้นเมื่อ slot อยู่หลังเที่ยงคืนของวันทำการ
30. 409 `DUPLICATE_BOOKING` ไม่แสดงคำว่า "เต็ม" และมีลิงก์ไปรายการเดิม

### ปิดร้านชั่วคราว + Notification (`internal/booking`, `internal/restaurant`, `internal/notification`)

Go unit (`businessday_test.go`, `availability_test.go`) — `TestClosureAt`: ทับ / จบตอนเริ่มปิดพอดีไม่ทับ / หลายช่วงเลือกตัวแรกที่ทับ / ว่าง;
`TestSlotsSkipClosure`: `Slots` ข้ามช่วงปิด, `SlotsAround` ได้ `closed`

service test (mock ด้วย mockery `mocks_test.go`)
- `booking.TestServiceClosuresAndNotify`: ทับช่วงปิด → `ClosedError` (ไม่ตรวจกฎ 8/7 ต่อ); จองสำเร็จ → แจ้งเจ้าของร้าน; เจ้าของร้านจองร้านตัวเอง → ไม่แจ้งตัวเอง
- `restaurant.TestCreateClosure`: ไม่มีการจองทับ → บันทึกเลย; มีการจองทับไม่ส่ง confirm → `CLOSURE_AFFECTS_BOOKINGS` พร้อมรายการ; confirm ตรง → ยกเลิก + notification ครบ; confirm ไม่ตรง (มีการจองใหม่แทรก) → 409 ใหม่
- `notification.TestServiceMarkRead`: mark อ่านของตัวเองสำเร็จ; ของคนอื่น → 404

integration (testcontainers)
- `restaurant.TestClosures`: ปิดร้านพร้อม confirm → ช่วงปิด + booking ถูกยกเลิก (ทรานแซกชันเดียว); list ร้าน `?date=` ที่มีช่วงปิด → ปุ่มเวลาได้ `closed`
- `notification.TestInsertSnapshot` / `TestInsertMissingBooking` / `TestRepositoryReadState`: snapshot ถูกต้องตอนสร้าง; อ้าง booking ที่ไม่มีจริง → error; `ListMine` เรียงล่าสุดก่อน + `unread_count` ถูก, `read-all` แตะเฉพาะของตัวเอง

handler test (`httptest`)
- `restaurant.TestHandlerCreateClosure`: POST closures โดยไม่ใช่เจ้าของ → 403
- `notification.TestHandler`: `/me/notifications` ไม่มี token → 401; mark อ่านของคนอื่น → 404

web (Vitest) — `lib/notifications.test.ts`: ข้อความตาม `kind` ครบ 4 แบบ + ป้าย "(เช้าวันที่ n)"; `bellLabel` ตัวเลขยังไม่อ่าน + ข้อความไม่มีตัวเลขตอนเป็น 0

E2E (`owner.spec.ts`) — เจ้าของร้านปิดช่วงที่มีการจอง → เห็นรายการ → ยืนยัน → ลูกค้า login เห็นกระดิ่ง + badge "ร้านยกเลิก" + เหตุผล

---

## 10. สิ่งที่ต้องส่ง (11 ต.ค. — ไม่มีเลื่อน)

1. **Git repo** — Next.js + Go พร้อม README ที่มี:
   - วิธีรันทีละคำสั่ง (`cp .env.example .env` → `docker compose up -d --build` — migrate + seed รันอัตโนมัติใน compose → เปิด http://jongyoung.localhost)
   - บัญชีทดสอบของ Keycloak (owner 2 คน, ลูกค้า 1 คน)
   - seed data ที่เปิดมาแล้ว **เห็นร้านหลายร้าน + การจอง + รีวิวทันที** (ข้อ 4.1)
   - เหตุผลที่เลือก DB / auth / library แต่ละตัว + เปรียบเทียบ Bayesian กับเกณฑ์ ≥ 5 รีวิว
2. **วิดีโอ ≤ 5 นาที** — เดโมครบทุกฟีเจอร์ + อธิบายโค้ดจุดตรวจกติกาที่นั่งและจุดล็อกกันชนกัน
3. **UI Design** — ลิงก์ Claude Design (ข้อ 8)

---

## 11. คำถามวันสัมภาษณ์ (17–18 ต.ค.) และคำตอบที่โค้ดนี้ให้

| คำถาม | คำตอบ |
|---|---|
| สองคนกดพร้อมกัน ร้านเหลือ 6 ที่ | `SELECT ... FOR UPDATE` บนแถวร้าน + นับ + เขียน ในทรานแซกชันเดียว → รับคนเดียว อีกคน 409 |
| ทำไมไม่ใช้ SUM ตรวจที่นั่ง | เคส A 7 / B 7 / C 3 — ช่วงทับกันไม่ได้แปลว่าอยู่พร้อมกัน ต้อง sweep line ตรวจทุกจุดที่มีคนเข้าร้าน |
| ร้านเปิด 18:00–02:00 เก็บยังไง | นาทีจากเที่ยงคืน; `close <= open` = ข้ามวัน; `duration = 0` = เปิด 24 ชม.; `date` = วันทำการ |
| ★5.0 จาก 1 รีวิว vs ★4.8 จาก 300 | Bayesian average ดึงร้านที่รีวิวน้อยเข้าหาค่าเฉลี่ยรวม (C = 5) ร้านไม่มีรีวิวอยู่ท้าย และ UI แสดง "รีวิวน้อย" |
| ค่าเฉลี่ยคำนวณเมื่อไหร่ | เก็บ `rating_sum`/`rating_count` อัปเดตแบบ atomic ในทรานแซกชันเดียวกับรีวิว → หน้า list ไม่ต้อง AVG/JOIN, ไม่มี N+1 |
| กดจองซ้ำ/เน็ตกระตุก | กฎ "คนเดียวกันจองร้านเดียวกันซ้อนเวลาไม่ได้" ตรวจในล็อกเดียวกันก่อนนับที่นั่ง → คำขอที่สองได้ `DUPLICATE_BOOKING` แล้วพาไปดูรายการเดิม; ถ้ามีการตัดเงินมัดจำต้องใช้ `Idempotency-Key` จริง |
| ทำไมลูกค้าไม่เห็นแถบ % ที่นั่ง | ลูกค้าถาม "เวลานี้จองได้ไหม" → time chip ตอบตรงกว่า; แถบ % เป็นมุมมองของเจ้าของร้าน (peak ของคืน) จึงอยู่ในบอร์ด owner |
| การ์ดร้านแสดงเวลาว่างโดยไม่ N+1 ยังไง | query booking ของทุกร้านในหน้านั้นครั้งเดียว (`IN`) แล้วคำนวณ slot ใน Go ด้วย `maxConcurrent` ตัวเดิม |
| Server vs Client Component | หน้า list/detail = server (ไม่ต้องใช้ token, โหลดเร็ว); หน้าที่ต้อง login และฟอร์ม = client (TanStack + axios) |
| session/token เก็บที่ไหน | next-auth เก็บ session ใน cookie; axios ดึง access token จาก `getSession()` แนบ Bearer — ความเสี่ยง XSS รับมือด้วย token อายุสั้น + ไม่ render HTML จากผู้ใช้; ทางที่ปลอดภัยกว่าคือ BFF proxy |
| issuer ไม่ตรงระหว่าง browser กับ container | Caddy + โดเมน `*.jongyoung.localhost` ให้ทุกฝ่ายเห็น Keycloak ชื่อเดียวกัน |
| ทำไมต้องมี users ในฐานข้อมูลเราอีก ทั้งที่มี Keycloak | Keycloak เก็บ identity, DB เราเก็บ domain data + FK; ผูกด้วย `sub` ไม่ใช่ email; upsert ตอนเจอ token และอัปเดตเมื่อ claims เปลี่ยน |
| timezone | เก็บ UTC ที่ DB, `Asia/Bangkok` แสดงที่ web; ถ้าร้านอยู่หลาย timezone ต้องเก็บ timezone ต่อร้าน และคำนวณวันทำการด้วยปฏิทินของ timezone นั้น |
| ทำไมไม่ใช้ realm role แยก owner/customer | บทบาทผูกกับร้านแต่ละร้าน ไม่ใช่ผูกกับบัญชี — เป็นข้อมูล ไม่ใช่สิทธิ์ระดับ realm |
| ออกแบบ UI ยังไงให้คนจองสำเร็จ | เลือกเวลาได้ตั้งแต่หน้าแรก (time chip), กติกายกเลิกเห็นก่อนกดจอง, มีหน้ายืนยัน, error บอกทางออกไม่ใช่บอกว่าพัง |
| ทำไมยืนยันด้วยรายการ id ไม่ใช่ `confirm: true` | กันยกเลิกการจองที่เจ้าของร้านยังไม่เคยเห็น — มีคนจองแทรกระหว่างดูรายการจะได้ 409 ใหม่ |
| ทำไมสร้าง notification ในทรานแซกชันเดียวกัน | จองไม่สำเร็จไม่มีแจ้งเตือนหลง, ยกเลิกสำเร็จลูกค้าได้รับแจ้งแน่นอน; ต่อยอดอีเมล/LINE ด้วย outbox + worker |
| ทำไม polling ไม่ใช่ WebSocket | ง่าย อธิบายได้ ช้ากว่ากันไม่เกิน 60 วินาที ซึ่งพอสำหรับการแจ้งยกเลิกล่วงหน้า |

---

## 12. ข้อห้าม

- ห้ามใช้ naive `SUM` ตรวจที่นั่ง (ดู 5.3)
- ห้ามลืม `excludeID` ตอนแก้ไขการจอง
- ห้ามเช็คที่นั่ง **หรือกฎจองซ้อนตัวเอง** นอกทรานแซกชันที่ล็อกแถวร้าน
- ห้ามเขียน logic นับที่นั่ง, เช็คเวลาเปิด หรือแปลงวันทำการซ้ำในหลายที่ — ใช้ `maxConcurrent()`, `fitsOpeningHours()` และ `businessday.go` ตัวเดียว
- ห้ามให้หน้าเว็บยิง availability ทีละร้านในหน้า list (N+1)
- ห้ามอัปเดต `rating_sum/rating_count` ด้วยการอ่านค่ามาบวกใน Go แล้วเขียนกลับ
- ห้ามรับ `user_id` จาก request body/query — เอาจาก token เท่านั้น
- ห้ามตรวจกติกาแค่ที่หน้าเว็บ
- ห้ามส่งเวลาเป็นสตริง `"00:30"` ลอย ๆ ใน API — ต้องเป็น timestamp เต็มพร้อม offset
- ห้ามเก็บ token ใน localStorage เอง และห้ามใช้ `dangerouslySetInnerHTML` กับข้อมูลผู้ใช้
- ห้าม log token, client_secret, DSN เต็ม
- ห้าม commit `.env` — ต้องมี `.env.example` ครบทุก key; secret ใน realm export ต้องเป็นค่า dev เท่านั้น
- ห้ามเขียน secret ลงใน `compose.yml` ตรง ๆ — อ่านจาก `.env`
- ห้ามลบข้อมูลจริงเมื่อยกเลิกการจอง (ใช้ `status`) — ต้องเก็บประวัติ
- ห้ามใส่ seed data ใน `migrations/`
- ห้ามใช้ `backdrop-filter` กับการ์ดในลิสต์ และห้ามวางตัวอักษรเนื้อหาบนพื้นที่ blur ทับ glow
- ห้ามใช้อีโมจิเป็นไอคอน
- ห้ามใส่ library ที่อธิบายไม่ได้ว่าทำอะไร

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
- ลำดับที่แนะนำ: `businessday.go` + `availability.go` + test (ข้อ 9 ข้อ 1–16) ให้ผ่านก่อน
  แล้วค่อยต่อ HTTP layer → หน้าเว็บ → owner → รีวิว
