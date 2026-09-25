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
- `EnsureExists` = upsert จริง: ไม่มี → สร้าง; มีแล้วแต่ `email`/`display_name` ใน claims ต่างจากที่เก็บ → อัปเดต
  (ผู้ใช้แก้โปรไฟล์ใน Keycloak แล้วต้องเห็นชื่อใหม่ในระบบเรา)
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
  cuisine               text                          -- ใช้ filter + แนะนำร้านประเภทเดียวกันตอนร้านเต็ม
  address               text not null                 -- แสดงในหน้ายืนยัน + ลิงก์ Google Maps (ไม่เก็บพิกัด)
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
- seed ต้องมีร้านหลายแบบเพื่อโชว์ว่ารองรับทุกเคส อย่างน้อย:
  1. ร้านเวลาปกติ (11:00–22:00) ที่มีการจองคืนนี้จนเกือบเต็ม/เต็มบางช่วง
  2. ร้านเปิดข้ามเที่ยงคืน (18:00–02:00)
  3. ร้านเปิด 24 ชม. (`open_minute = close_minute`)
  4. ร้านที่ยังไม่มีรีวิว
  5. ร้าน ★5.0 จาก 1 รีวิว คู่กับร้าน ★4.8 จากรีวิวเยอะ (โชว์ Bayesian sort)

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
- **ลดที่นั่ง:** อ่าน booking ที่ `active` และยังไม่จบ (`end_at > now`) ของร้าน → `peak, at := maxConcurrent(bookings)` (ตัวเดียวกับ 5.3 ห้ามเขียนใหม่)
  → `peak > newSeats` → 409 `SEATS_BELOW_BOOKED` พร้อม `details: { at, booked: peak }`
- **ย่นเวลาเปิด–ปิด:** ทุก booking ในอนาคตต้องยังผ่าน `fitsOpeningHours` (5.4) ด้วยเวลาใหม่ → ไม่ผ่านตัวไหน → 409 `HOURS_CONFLICT` พร้อมช่วงที่ชน

### 5.2 การจอง — เงื่อนไขครบทุกข้อ
ลูกค้ากรอก 3 อย่าง: **จำนวนคน / วันที่ / เวลาเริ่ม–สิ้นสุด**

| # | เงื่อนไข | ผิดแล้วตอบ |
|---|---|---|
| 1 | ร้านมีอยู่จริงและไม่ถูกลบ | 404 |
| 2 | `end_at > start_at` | 400 |
| 3 | จองล่วงหน้าอย่างน้อย 30 นาที: `start_at >= now + 30 นาที` (เทียบ `time.Now()` ที่ server) — กันจองรอบที่อีก 2 นาทีจะเริ่ม | 400 |
| 4 | ช่วงจองอยู่ในเวลาเปิด–ปิดของร้าน (รองรับเปิดข้ามเที่ยงคืน) | 400 |
| 5 | `party_size <= restaurant.seats` (ขอเกินความจุร้านไปเลย) | 400 |
| 6 | **ที่นั่งไม่เกินในทุกวินาที** — ดู 5.3 | **409** `NOT_ENOUGH_SEATS` |
| 7 | ผู้ใช้คนเดียวกันจองร้านเดียวกันซ้อนเวลากันเองไม่ได้ — **ตรวจใน FOR UPDATE block เดียวกับข้อ 6** (ดู 5.3) | **409** `DUPLICATE_BOOKING` |
| 8 | จองล่วงหน้าไม่เกิน 90 วัน และช่วงละไม่เกิน 4 ชม. (กฎเราเอง — กัน abuse, อธิบายได้) | 400 |
| 9 | เวลาเริ่ม/สิ้นสุดเป็นช่วงละ 30 นาที (:00 หรือ :30) | 400 |

**กดจองซ้ำ / เน็ตกระตุก:** ไม่ใช้ `Idempotency-Key` — กฎข้อ 7 กันการจองซ้ำให้อยู่แล้ว
(คำขอที่สองรอล็อก → เห็นการจองของคำขอแรก → 409 `DUPLICATE_BOOKING`) + หน้าเว็บ disable ปุ่มระหว่างยิง
กฎ 7 กันซ้ำได้ **ก็ต่อเมื่อ** อยู่หลัง `FOR UPDATE` — ถ้าเช็คก่อนเข้าทรานแซกชัน สองคำขอจะอ่านเจอ "ยังไม่มี" ทั้งคู่แล้วเขียนทั้งคู่

**409 สองแบบ หน้าเว็บต้องทำต่างกัน:**
| code | ความหมาย | หน้าเว็บทำอะไร |
|---|---|---|
| `NOT_ENOUGH_SEATS` | ที่นั่งไม่พอจริง | refetch availability, บอกที่นั่งที่เหลือจริง, เสนอช่วงที่ยังว่างพอ |
| `DUPLICATE_BOOKING` | ผู้ใช้มีการจองช่วงนี้อยู่แล้ว (มักเพราะกดซ้ำแล้วคำขอแรกสำเร็จ) | **ห้ามบอกว่าร้านเต็ม** → แจ้ง "คุณจองช่วงนี้ไว้แล้ว" แล้วพาไป `/me/bookings` |

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

// maxConcurrent คืนจำนวนคนสูงสุดที่อยู่ในร้านพร้อมกัน และเวลาที่เกิด
// ใช้ทั้งตอนจอง/แก้ไข (5.3) และตอนเจ้าของลดที่นั่ง (5.1) — มีฟังก์ชันเดียว ห้ามเขียนซ้ำ
func maxConcurrent(bookings []Booking) (peak int, at time.Time) {
    // จุดที่ต้องตรวจ = เวลาเริ่มของทุก booking
    // เพราะจำนวนคนในร้านเพิ่มขึ้นได้เฉพาะตอนมีคนเริ่มเข้า ระหว่างสองจุดค่าคงที่ (จุดที่คนออก ค่าลดลง ไม่ต้องตรวจ)
    for _, p := range bookings {
        t := p.StartAt
        n := 0
        for _, b := range bookings {
            // อยู่ในร้าน ณ เวลา t คือ start_at <= t < end_at
            if !b.StartAt.After(t) && b.EndAt.After(t) {
                n += b.PartySize
            }
        }
        if n > peak {
            peak, at = n, t
        }
    }
    return peak, at
}

// checkSeats: existing = booking 'active' ที่ทับกับช่วงของ req (ไม่รวมตัวที่กำลังแก้)
// booking ที่มีอยู่ถูกต้องอยู่แล้ว (ไม่เกิน seats) → ถ้า peak รวม req เกิน ต้องเกินในช่วงของ req แน่นอน
func checkSeats(existing []Booking, seats int, req Booking) error {
    all := append(slices.Clone(existing), req) // clone กัน append ไปเขียนทับ slice ของผู้เรียก
    if peak, at := maxConcurrent(all); peak > seats {
        return &NotEnoughSeatsError{At: at, Available: seats - (peak - req.PartySize)} // → 409 NOT_ENOUGH_SEATS
    }
    return nil
}
```

**การกันสองคนกดพร้อมกัน — กฎ 6 และกฎ 7 ต้องอยู่ในทรานแซกชันเดียวกัน หลังล็อก:**
```go
err := db.Transaction(func(tx *gorm.DB) error {
    // 1) ล็อกแถวร้าน — คำขอที่สอง (ร้านเดียวกัน) จะรอที่บรรทัดนี้จนคำขอแรก commit
    if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
        First(&r, "id = ? AND deleted_at IS NULL", restaurantID).Error; err != nil { return err }

    // 2) กฎ 7: ผู้ใช้คนนี้มีการจองร้านนี้ที่ทับช่วงนี้อยู่แล้วไหม — ต้องอยู่หลังล็อก และตรวจก่อนกฎ 6
    //    (กดซ้ำ: คำขอที่สองต้องได้ DUPLICATE_BOOKING ไม่ใช่ NOT_ENOUGH_SEATS)
    var dup int64
    tx.Model(&Booking{}).
       Where("restaurant_id = ? AND user_id = ? AND status = 'active'", restaurantID, userID).
       Where("start_at < ? AND end_at > ?", end, start).
       Where("id <> ?", excludeID).      // จองใหม่ใช้ uuid.Nil
       Count(&dup)
    if dup > 0 { return ErrDuplicateBooking } // → 409 DUPLICATE_BOOKING

    // 3) กฎ 6: อ่าน booking ที่ทับกัน "ข้างใน" ล็อก
    var existing []Booking
    tx.Where("restaurant_id = ? AND status = 'active'", restaurantID).
       Where("start_at < ? AND end_at > ?", end, start).
       Where("id <> ?", excludeID).      // ⚠️ ตอนแก้ไข: ห้ามนับที่นั่งเดิมของตัวเองซ้ำ
       Find(&existing)
    if err := checkSeats(existing, r.Seats, booking); err != nil { return err }

    // 4) เขียน — ทั้งหมดในล็อกเดียว
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
```go
// ความยาวของรอบเปิด (นาที)
duration := (close - open + 1440) % 1440
if duration == 0 {
    duration = 1440 // open == close = เปิด 24 ชม. — ⚠️ ถ้าไม่มีบรรทัดนี้ ร้าน 24 ชม. จะจองไม่ได้เลย
}
```
`fitsOpeningHours(r, start, end)` = หารอบเปิดที่ `start` อยู่ แล้วเช็ค `offset(start) >= 0 && offset(end) <= duration`
ใช้ทั้งตอนจอง (กฎ 4) และตอนเจ้าของย่นเวลา (5.1)
เขียน test ครอบทั้งร้านปกติ, ข้ามคืน, 24 ชม. + เคสจองคาบเกี่ยวเวลาปิด (22:30–23:30 ที่ร้านปิด 23:00 → 400)

**`date` = วันทำการ (รอบที่เริ่มเปิดในวันนั้น)** — ใช้ความหมายเดียวกันทุกที่:
`GET /restaurants/:id/availability?date=`, `GET /restaurants/:id/bookings?date=` (ของ owner) และฟอร์มจอง
```
opens_at  = date + open_minute (เวลาไทย)
closes_at = opens_at + duration
รอบของ date = [opens_at, closes_at)
```
- ร้านเปิด 18:00–02:00 ขอ `date=2026-10-10` → slot 18:00 ของวันที่ 10 ถึง 02:00 ของวันที่ 11 ต่อเนื่องในรอบเดียว
- เลือกวันที่ 10 แล้วใส่เวลา 00:30 → หมายถึง 00:30 ของวันที่ 11
- ⚠️ ขอ `date=2026-10-11` ต้อง **ไม่มี** slot 00:00–02:00 ของเช้าวันที่ 11 — ช่วงนั้นเป็นของรอบวันที่ 10 ไปแล้ว (เคสที่พลาดง่ายสุด ต้องมีเทสต์)
- owner ดูการจอง `?date=` → เอา booking ที่ `start_at` อยู่ใน `[opens_at, closes_at)` ของรอบนั้น
  (คนจองตี 1 คืนวันเสาร์ต้องอยู่ในบอร์ดของวันเสาร์ ไม่ใช่วันอาทิตย์)

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
  `score` ไม่ใช่คอลัมน์ในตาราง — ต้องคำนวณใน query (raw SQL ผ่าน GORM `Raw`):
  ```sql
  WITH g AS (
    SELECT COALESCE(SUM(rating_sum)::float / NULLIF(SUM(rating_count), 0), 0) AS m
    FROM restaurants WHERE deleted_at IS NULL
  )
  SELECT r.*, (5 * g.m + r.rating_sum) / (5 + r.rating_count) AS score
  FROM restaurants r CROSS JOIN g
  WHERE r.deleted_at IS NULL
  ORDER BY (r.rating_count = 0), score DESC, r.rating_count DESC, r.id
  LIMIT :limit OFFSET :offset
  ```
  `::float` สำคัญ — ไม่งั้น `SUM(int) / SUM(int)` เป็นหารจำนวนเต็ม ค่าเฉลี่ย 4.37 จะกลายเป็น 4
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
| GET | `/restaurants` | – | `?q=&cuisine=&sort=rating\|reviews\|newest&page=&limit=` + ตัวเลือก `&date=&time=&party=` → แต่ละร้านมี `slots` 5 ช่วงรอบ `time` (ดูด้านล่าง) |
| GET | `/restaurants/:id/next-available` | – | `?date=&time=&party=` → วันทำการถัดไป (ไม่เกิน 14 วัน) ที่มีช่วงว่างพอ ใช้ทำ empty state |
| GET | `/restaurants/:id` | – | + คะแนนเฉลี่ย + จำนวนรีวิว + รูปทั้งหมด |
| GET | `/restaurants/:id/availability` | – | `?date=YYYY-MM-DD` (วันทำการ) → ที่นั่งคงเหลือทุกช่วง 30 นาที |
| POST | `/restaurants` | ✓ | |
| PUT / DELETE | `/restaurants/:id` | ✓ owner | |
| POST / DELETE | `/restaurants/:id/images` | ✓ owner | รับ URL |
| GET | `/restaurants/:id/bookings` | ✓ owner | `?date=YYYY-MM-DD` (**วันทำการ** เหมือน availability) — Owner ดูการจองของร้าน |
| POST | `/bookings` | ✓ | 409 `NOT_ENOUGH_SEATS` หรือ `DUPLICATE_BOOKING` |
| GET | `/me/bookings` | ✓ | `?status=upcoming\|past\|cancelled` |
| GET | `/bookings/:id` | ✓ เจ้าของ booking หรือ owner ร้าน | |
| PUT | `/bookings/:id` | ✓ เจ้าของ booking | แก้จำนวนคน/วัน/เวลา |
| DELETE | `/bookings/:id` | ✓ เจ้าของ booking | ยกเลิก → `status=cancelled` (ไม่ลบจริง เก็บประวัติ) |
| GET | `/restaurants/:id/reviews` | – | `?page=&limit=` |
| POST | `/restaurants/:id/reviews` | ✓ | สร้าง; มีอยู่แล้ว → 409; เจ้าของร้าน → 403 |
| PUT | `/restaurants/:id/reviews` | ✓ | แก้ของตัวเอง; ยังไม่มี → 404 |
| DELETE | `/restaurants/:id/reviews` | ✓ | ลบของตัวเอง |

**Response ของ availability** — เวลาเป็น timestamp เต็ม (RFC 3339, UTC) ทุกตัว ห้ามส่ง `"00:30"` ลอย ๆ
เพราะร้านข้ามคืน "00:30" อาจเป็นวันถัดไป — ให้ web แปลงเป็นเวลาไทยเอง
```json
{
  "business_date": "2026-10-10",
  "opens_at":  "2026-10-10T11:00:00Z",
  "closes_at": "2026-10-10T19:00:00Z",
  "seats": 20,
  "slots": [
    { "start_at": "2026-10-10T11:00:00Z", "end_at": "2026-10-10T11:30:00Z", "booked": 7, "available": 13 },
    { "start_at": "2026-10-10T17:30:00Z", "end_at": "2026-10-10T18:00:00Z", "booked": 20, "available": 0 }
  ]
}
```
(ตัวอย่างคือร้าน 18:00–02:00 เวลาไทย = 11:00Z–19:00Z; `booked` ของแต่ละ slot = คนที่อยู่ในร้าน ณ ต้น slot)

**`slots` บนการ์ดร้าน (`GET /restaurants?date=&time=&party=`)** — รูปแบบ slot เดียวกับ availability แต่เอาแค่ 5 ช่วงรอบ `time`
และคำนวณจาก **query เดียว** ที่ดึง booking ของทุกร้านในหน้านั้นในรอบวันนั้น (`restaurant_id IN (...)`) แล้วแบ่งใน Go
— ห้ามยิง availability ทีละร้าน (N+1); slot ที่ร้านปิดส่ง `"closed": true`

**Status code ที่ต้องใช้ให้ถูก**
`200/201/204` สำเร็จ · `400` รูปแบบ/เงื่อนไขเวลาผิด · `401` ไม่มี/token ไม่ถูก/หมดอายุ ·
`403` สิทธิ์ไม่ถึง (ไม่ใช่เจ้าของ, รีวิวร้านตัวเอง, ยกเลิกช้าเกิน) · `404` ไม่มีข้อมูล ·
`409` ชนกับสถานะปัจจุบัน (ที่นั่งไม่พอ, จองซ้อน, ยกเลิกซ้ำ, ลดที่นั่งไม่ได้, รีวิวซ้ำ) · `500` ที่เหลือ

**Error response รูปแบบเดียวทั้งระบบ** (`internal/httputil`)
```json
{ "error": { "code": "NOT_ENOUGH_SEATS", "message": "ช่วง 12:30–13:00 เหลือ 3 ที่นั่ง", "details": { "available": 3 } } }
```
ฝั่ง web แปล `code` เป็นข้อความไทย — ไม่พึ่ง `message` จาก API ในการตัดสินใจ

error code ของ 409 (ต้องแยกกัน เพราะหน้าเว็บทำต่างกัน):
`NOT_ENOUGH_SEATS` · `DUPLICATE_BOOKING` · `BOOKING_CANCELLED` · `BOOKING_ALREADY_STARTED` ·
`SEATS_BELOW_BOOKED` · `HOURS_CONFLICT` · `REVIEW_EXISTS`

**Middleware order:** Recovery → RequestID/Logger → CORS (`http://jongyoung.localhost` เท่านั้น) → RateLimit(เบา ๆ) → JWT (verify + JIT provisioning) → เช็คความเป็นเจ้าของใน service

---

## 7. หน้าเว็บ (Next.js)

| Path | เนื้อหา | Rendering |
|---|---|---|
| `/` | ค้นหา (ร้าน/เมนู + **วันที่ / เวลา / จำนวนคน** อยู่บนสุด) → การ์ดร้านมี **ปุ่มเวลาว่าง 5 ปุ่ม** + เรียงลำดับ | server + client island (ตัวกรอง) |
| `/restaurants/[id]` | รายละเอียด + รูป + ปุ่มเวลา + แผงจอง + รีวิว | server + client island (แผงจอง, รีวิว) |
| `/bookings/[id]` | **ยืนยันการจอง**: เลขที่จอง, ร้าน + ที่อยู่ + ลิงก์แผนที่, วันเวลา, จำนวนคน, "ยกเลิกได้ถึง …", เพิ่มลงปฏิทิน | client |
| `/me/bookings` | การจองของฉัน (tab กำลังจะถึง/ผ่านมาแล้ว/ยกเลิกแล้ว) แก้-ยกเลิกในหน้านี้ | client (TanStack + token) |
| `/owner/restaurants` | ร้านของฉัน (ตาราง + ฟอร์มแก้ไข/ลบ) | client |
| `/owner/restaurants/[id]/bookings` | บอร์ดรายวัน: seat bar ต่อช่วง 30 นาที + ตารางการจอง | client |
| `/api/auth/[...nextauth]`, `/api/auth/logout` | next-auth + logout | – |

**Flow การจอง = 2 แท็ป:** กดปุ่มเวลาบนการ์ดร้าน → หน้าร้านเปิดพร้อมเวลาที่เลือกไว้ในแผงจอง → กด "ยืนยันการจอง" → `/bookings/[id]`

- ฟอร์มใช้ react-hook-form + zod
- **การจอง: ห้ามใช้ optimistic update** — ต้องรอผลจริง; ได้ 409 ให้ดู `error.code` (ตาราง 5.2):
  `NOT_ENOUGH_SEATS` → refetch availability บอกที่นั่งที่เหลือจริง และเสนอช่วงที่ยังว่างพอ /
  `DUPLICATE_BOOKING` → "คุณจองช่วงนี้ไว้แล้ว" (สีฟ้า/กลาง ไม่ใช่แดง) แล้วพาไป `/me/bookings` (ห้ามบอกว่าร้านเต็ม)
  (รีวิวใช้ optimistic ได้)
- ปุ่มกดจองต้อง disable ระหว่างยิง กันกดรัว
- **กติกายกเลิกต้องเห็นก่อนกดจอง** — แผงจองแสดง "ยกเลิกหรือแก้ไขได้ถึง HH:MM น." เหนือปุ่มยืนยันเสมอ
- **หน้ายืนยันการจอง**: เพิ่มลงปฏิทินได้ 2 ทาง — ลิงก์ Google Calendar (template URL) และไฟล์ `.ics` ที่สร้างฝั่ง web
  แผนที่ = ลิงก์ Google Maps ค้นหาจาก `address` (ไม่ต้องใช้ API key)
- **ร้านเต็ม/ไม่มีเวลาว่าง** ห้ามเป็นจอเปล่า: เสนอ "วันถัดไปที่ว่าง" (`/next-available`) + "ร้านประเภทเดียวกันที่ว่างเวลานี้"
  (ไม่มีข้อมูลพิกัด จึงไม่ทำ "ร้านใกล้เคียง")
- **ร้านเปิดข้ามคืน:** เวลาหลังเที่ยงคืนต้องมีป้าย "(เช้าวันที่ 11)" **ทุกที่ที่แสดง** — ปุ่มเวลา, สรุปในแผงจอง, หน้ายืนยัน, การจองของฉัน, บอร์ด owner
- **สองบทบาทบัญชีเดียว:** navbar มีตัวสลับ `ลูกค้า | เจ้าของร้าน` ชัดเจน (แสดงเมื่อ `/me` มีร้านที่เป็นเจ้าของ)
  โหมดเจ้าของร้านใช้ navbar ทึบสีเข้ม + ป้าย "โหมดเจ้าของร้าน" ให้รู้ทันทีว่าอยู่โหมดไหน
- **เปิดหน้าร้านตัวเอง:** ไม่แสดงปุ่ม "เขียนรีวิว" แต่ขึ้นข้อความว่า "นี่คือร้านของคุณ — เจ้าของร้านรีวิวร้านตัวเองไม่ได้" + ลิงก์ไปจัดการร้าน
  (ไม่ใช่ให้กดแล้วเด้ง 403 — API ยังต้องกัน 403 เหมือนเดิม)
- ซ่อนปุ่มแก้/ลบเมื่อไม่ใช่เจ้าของ — เป็นเรื่อง UX เท่านั้น API ต้องกันซ้ำเสมอ
- ปุ่มที่กดไม่ได้เพราะกติกา (เช่น เลยเวลายกเลิก) ให้ disabled พร้อมข้อความบอกเหตุผล ไม่ใช่ซ่อน
- แสดงเวลาเป็นเวลาไทยทุกที่ (`Intl.DateTimeFormat('th-TH', { timeZone: 'Asia/Bangkok' })`)
- **ทุกหน้าต้องมี** loading skeleton / empty state / error state (+ 401 เซสชันหมดอายุ → login แล้วกลับมาหน้าเดิม)

---

## 8. Design system

**ต้นแบบคาแรกเตอร์:** [guy127/introduce_myself](https://github.com/guy127/introduce_myself) — เอาแค่โทนสี/แบรนด์ ไม่เอา design language ของเว็บพอร์ตโฟลิโอ
**Mockup ทุกหน้า (Claude Design):** https://claude.ai/artifact/Cb7eGqYexzQd73X6AiqvKR
(ฝั่งลูกค้า 4 หน้า, หน้ารวมสถานะ empty/error/409, ฝั่งเจ้าของร้าน 2 หน้า, มือถือ 2 หน้า)

**หลักคิด:** นี่คือระบบที่คนกดจองและผูกพันตามกติกา — อ่านเร็ว เชื่อถือได้ มาก่อนความประทับใจ
- **Day (สว่าง) เป็นค่าเริ่มต้น**, Night เป็นตัวเลือกจากปุ่มบน navbar
- **กระจกขุ่น (backdrop-filter) ใช้แค่ 3 ที่:** navbar, กล่องค้นหา hero, แผงจอง — ที่เหลือ (การ์ดร้าน, รายการ, ตาราง) เป็น **พื้นทึบ + เส้นขอบ**
  เหตุผล: blur บนการ์ดหลายสิบใบทำให้มือถือ Android ระดับกลางกระตุก และพื้นหลังที่เปลี่ยนตามตำแหน่งทำให้รับประกัน contrast AA ไม่ได้
- radial glow เบา ๆ เฉพาะบนสุดของหน้า ไม่มีเส้นกริด/orb ลอย

### 8.1 Font
```
ทั้งระบบ: "Prompt" 400 / 500 / 600
"Kanit" 600 ใช้แค่หัวข้อ hero และโลโก้ (โหลดน้ำหนักเดียว)
Google Fonts css2 + display=swap (subset ไทยแยกตาม unicode-range ให้อยู่แล้ว)
ตัวเลข/เวลา/จำนวนที่นั่ง: font-variant-numeric: tabular-nums
```
ขนาด (ลูกค้า): hero 36–44px / หัวข้อ section 20–22px / ชื่อร้านบนการ์ด 19px / เนื้อหา 16px / กำกับ 14px / เล็กสุด 13px
ขนาด (เจ้าของร้าน): ทั้งหน้า 14px, หัวข้อ 22px
**ห้ามต่ำกว่า 13px สำหรับภาษาไทย** · `line-height: 1.7` สำหรับเนื้อหา · หัวข้อไทย line-height ≥ 1.2

### 8.2 สี — token สองโหมด (Day เป็นค่าเริ่มต้น)

| Token | Day | Night | ใช้กับ |
|---|---|---|---|
| `--page` | `#fbf6f1` | `#11070a` | พื้นหลัง |
| `--surface` | `#ffffff` | `#1b0e12` | การ์ด/ตาราง (ทึบ) |
| `--line` / `--line-strong` | `#eadcd4` / `#d9c5bb` | `#3a262c` / `#4d333a` | เส้นขอบ, ขอบ input |
| `--chip` | `#f4ece7` | `#26151a` | พื้นปุ่มรอง, ปุ่มเวลาที่เต็ม |
| `--text` | `#261419` | `#fff7ed` | ตัวอักษรหลัก |
| `--muted` / `--soft` | `#6f4c4d` / `#7d625f` | `#d7b8b4` / `#a88b8d` | กำกับ / meta |
| `--accent` | `#c5162e` | `#ff5a6e` | แบรนด์ (โลโก้) |
| `--cta` | `#c5162e → #b3361a` | `#e11d48 → #c2410c` | ปุ่มหลัก gradient 135° — **หนึ่งปุ่มต่อหน้าจอ** |
| `--link` | `#007c91` | `#43e8ff` | ลิงก์ |
| `--sel` / `--sel-weak` | `#007c91` / `#e3f3f5` | `#43e8ff` / `rgba(67,232,255,.12)` | เวลาที่เลือก (ทึบ) / ช่วงที่ครอบ (เส้นประ) |
| `--ok` (ข้อความ / พื้น / ขอบ) | `#006e40` / `#e6f5ec` / `#a7d9bd` | `#27e88b` / 10% / 35% | ปุ่มเวลาว่าง, ยืนยันแล้ว |
| `--warn` (ข้อความ / พื้น / ขอบ) | `#8a5a00` / `#fdf1dc` / `#efcf93` | `#ffb703` / 10% / 35% | เหลือน้อย (ว่าง ≤ 30%) |
| `--full` (ข้อความ / พื้น / ขอบ) | `#b0132a` / `#fcebed` / `#f0b5bd` | `#ff6b7f` / 12% / 40% | error, ยกเลิก, ลบ |
| `--eyebrow` | `#8a5a00` | `#ffb703` | ป้ายเล็กเหนือหัวข้อ |
| `--star` | `#b77900` | `#ffb703` | ไอคอนดาว |

กฎการใช้สี:
- **หนึ่งหน้าจอมีปุ่ม gradient (`--cta`) ได้ปุ่มเดียว** ที่เหลือเป็นปุ่มขอบบาง/พื้นทึบ
- เขียว-อำพัน-แดง ใช้สื่อสถานะที่นั่ง/การจอง/error เท่านั้น; เวลาที่เลือกใช้ `--sel` (ฟ้าอมเขียว) ไม่ปนกับสีสถานะ
- ข้อความต้องผ่าน contrast **AA (4.5:1)** บนพื้นทึบ — เป็นเหตุผลที่การ์ดต้องทึบ
- **ห้ามสื่อความหมายด้วยสีอย่างเดียว** — ปุ่มเวลามีสัญลักษณ์ ✓ / ! / ✕ + ข้อความ, เต็มมีขีดฆ่า
- ไอคอนใช้ SVG (lucide) — ไม่ใช้อีโมจิ

### 8.3 Component ฝั่งลูกค้า
```
max-width หน้า: 1120px, padding ขอบ 16px (mobile) / 24px (desktop)
spacing: 4 8 12 16 24 32 48
radius: 999px (ปุ่มหลัก/nav/pill), 18–24px (การ์ด/แผง), 12px (ปุ่มเวลา, input)
focus: outline 2px solid currentColor, outline-offset 2px (สีเดียวกับตัวอักษร — ไม่ชนกับสีเวลาที่เลือก)
ไม่มี hover ลอย (translateY) บนการ์ด
```
- **Navbar**: แคปซูลกระจกลอย — โลโก้ "จ" + "จองยัง", ลิงก์ ค้นหาร้าน / การจองของฉัน, ตัวสลับ `ลูกค้า | เจ้าของร้าน`, ปุ่ม Day/Night, avatar
- **Hero + ค้นหา**: กล่องกระจก หัวข้อ 40px + แถวตัวกรอง ร้าน/เมนู · วันที่ · เวลาประมาณ · จำนวนคน · ปุ่มค้นหา (CTA ของหน้านี้)
- **Restaurant card** (ทึบ): รูป 16:8 + badge ประเภท → ชื่อ 19px → คะแนน → เวลาเปิด → **แถวปุ่มเวลา 5 ปุ่ม** รอบเวลาที่ค้น
  - คะแนน: รีวิว ≥ 5 → `★ 4.6 · 128 รีวิว`; 1–4 รีวิว → ป้าย "รีวิวน้อย (n)" แทนดาว; 0 → "ยังไม่มีรีวิว" (สอดคล้องกับ Bayesian ที่ backend ใช้)
  - ไม่มีเวลาว่างเลย → กล่องเส้นประ "เต็มทั้งคืนนี้ · ว่างวันถัดไป" หรือ "ร้านปิด 18:00 · ช่วงที่ยังว่าง" + ปุ่มทางเลือก
- **ปุ่มเวลา (time chip)** — แทน seat bar ฝั่งลูกค้า เพราะลูกค้าถาม "ทุ่มนึงจองได้ไหม" ไม่ได้ถาม "ร้านเต็มแค่ไหน"
  | สถานะ | หน้าตา |
  |---|---|
  | ว่าง | พื้น/ขอบ `--ok` + "✓ ว่าง" |
  | เหลือน้อย (ว่าง ≤ 30% ของที่นั่ง) | พื้น/ขอบ `--warn` + "! เหลือ n" |
  | เต็มสำหรับจำนวนคนนี้ / ร้านปิด | พื้น `--chip`, ขีดฆ่า, disabled + "✕ เต็ม" / "ปิด" |
  | เลือกแล้ว | พื้นทึบ `--sel` ตัวขาว; ช่วงที่ครอบด้วยระยะเวลา = เส้นประ `--sel` |
  สถานะคำนวณจาก `available` ของ slot เทียบกับ **จำนวนคนที่ผู้ใช้เลือก** ไม่ใช่เทียบกับ 1
- **แผงจอง** (กระจก, sticky ขวา desktop / แผงล่าง mobile): วันที่, จำนวนคน (stepper), ระยะเวลา, ข้อความตรวจสอบ,
  บรรทัดสรุป "ส. 10 ต.ค. · 18:00–19:00 · 2 คน" + **กติกายกเลิก** แล้วจึงเป็นปุ่ม "ยืนยันการจอง"
- **หน้ายืนยัน**: ไอคอนสำเร็จ, เลขที่จองตัวใหญ่, ตาราง ร้าน/วันเวลา/จำนวนคน/ที่อยู่+แผนที่/ยกเลิกได้ถึง, ปุ่ม เพิ่มลงปฏิทิน (CTA) · ดูการจองของฉัน · แก้ไข
- **Booking card** (ทึบ): กล่องวันที่ + ชื่อร้าน + badge สถานะ (✓/✕) + ปุ่มแก้/ยกเลิก; เลยเวลายกเลิก → disabled + เหตุผล; ยกเลิกต้องมีกล่องยืนยัน
- แตะได้ต้องใหญ่ ≥ 44×44px บนมือถือ · mobile-first ทดสอบที่ 375px ก่อน

### 8.4 Component ฝั่งเจ้าของร้าน (`/owner/*`) — คนละภาษาการออกแบบ
เครื่องมือทำงานที่เปิดวันละหลายรอบ — ความเร็วในการสแกนสำคัญกว่าความประทับใจ
- navbar ทึบสีเข้ม (`#261419` / Night `#000`) + ป้าย "โหมดเจ้าของร้าน" + ตัวสลับโหมด
- ตัวอักษร 14px ทั้งหน้า, พื้นทึบ, ตารางความหนาแน่นสูง (แถว ~36–40px), radius 6px
- **ไม่มี** gradient, กระจก, glow, hover ลอย; ปุ่มหลักสีทึบ `--accent`
- **Seat bar อยู่ที่นี่**: บอร์ดรายวันแสดงแถบ "คนในร้าน/ที่นั่ง" ทุกช่วง 30 นาทีของรอบ (เขียว < 70% · อำพัน ≥ 70% · แดง เต็ม) + peak ของรอบ
- error ในฟอร์มแสดง code จาก API (เช่น `SEATS_BELOW_BOOKED`) พร้อมเวลาที่ชน

### 8.5 สิ่งที่ต้องส่ง
ออกแบบใน Claude Design ก่อนแล้วค่อยทำจริงบน Next.js (โจทย์บังคับ) — ทำแล้วที่ลิงก์ด้านบน
ต้องมีทั้ง happy path **และ** empty / error / 409 / confirmation state (กรรมการดูข้อนี้)
เก็บลิงก์/ภาพไว้ที่ `docs/` และต้องกด Share ลิงก์ก่อนส่งให้ผู้ตรวจ

---

## 9. Test ที่ต้องมี

`internal/booking/availability_test.go` — table-driven ครอบ:
1. ร้าน 10 ที่ มีคนจอง 7 ขอเพิ่ม 5 ในช่วงทับกัน → **ปฏิเสธ**
2. A 7 (12:00–12:30), B 7 (12:30–13:00), C ขอ 3 (12:00–13:00) → **ผ่าน** ⭐ เคสที่ naive SUM พลาด
3. A 7 + B 3 เต็มพอดี → A แก้เป็น 8 = ปฏิเสธ / A แก้เป็น 5 = ผ่าน (ทดสอบ `excludeID`)
4. จองเวลาที่ผ่านมาแล้ว → ปฏิเสธ; จองรอบที่เริ่มในอีก 29 นาที → ปฏิเสธ / อีก 30 นาทีพอดี → ผ่าน (lead time boundary)
5. จองนอกเวลาเปิด + ร้านเปิดข้ามเที่ยงคืน 18:00–02:00 (ทั้งเคสผ่านและไม่ผ่าน)
6. จองคาบเกี่ยวเวลาปิดร้าน → ปฏิเสธ
7. ยกเลิกช้ากว่า `cancel_before_minutes` → ปฏิเสธ; ตรงเวลาพอดี → ผ่าน (boundary)
8. `party_size > seats` → ปฏิเสธ
9. booking ที่ยกเลิกแล้วไม่ถูกนับเป็นที่นั่งที่ถูกใช้
10. ร้าน 24 ชม. (`open = close`) → จองได้ทุกเวลา รวมช่วงคร่อมเที่ยงคืน
11. วันทำการ: ร้าน 18:00–02:00 ขอ `date=11` → ไม่มี slot 00:00–02:00 ของเช้าวันที่ 11 (เป็นของรอบวันที่ 10); ขอ `date=10` → มีครบถึง 02:00 ของวันที่ 11
12. `maxConcurrent`: ไม่มี booking → 0; A/B/C ของเคส 2 → peak 10 ที่ 12:00

service test (booking / restaurant):
- กฎ 7: ผู้ใช้เดิมจองร้านเดิมซ้อนเวลา → `DUPLICATE_BOOKING`; คนละร้าน/ไม่ทับกัน → ผ่าน; ตอนแก้ไขไม่นับตัวเอง (`excludeID`)
- ลดที่นั่งต่ำกว่า `maxConcurrent` ของ booking ในอนาคต → `SEATS_BELOW_BOOKED`; เท่ากับ peak พอดี → ผ่าน
- ย่นเวลาเปิด–ปิดจน booking ในอนาคตตกนอกเวลา → `HOURS_CONFLICT`; booking ในอดีตไม่นับ
- owner `?date=`: booking ตี 1 คืนวันเสาร์อยู่ในบอร์ดวันเสาร์ ไม่ใช่วันอาทิตย์
- `EnsureExists`: ครั้งแรกสร้าง; ครั้งถัดไป email/ชื่อเปลี่ยน → อัปเดต

- service test: mock repository ด้วย mockery (`mocks_test.go`)
- repository test: testcontainers (Postgres จริง) รัน migrations/ ทุกไฟล์
- handler test: `httptest` ยิงใส่ Gin router — เช็ค 401 (ไม่มี token), 403 (ไม่ใช่เจ้าของ / รีวิวร้านตัวเอง), 409 (ที่นั่งไม่พอ, รีวิวซ้ำ)
- integration test การจองพร้อมกัน: ยิง 2 goroutine จองพร้อมกันบนที่นั่งที่เหลือพอสำหรับคนเดียว → ต้องสำเร็จ 1 ล้มเหลว 1
- integration test กดซ้ำ: ผู้ใช้คนเดียวยิงคำขอเดียวกัน 2 goroutine → สำเร็จ 1, อีกอันได้ `DUPLICATE_BOOKING` (ไม่ใช่ `NOT_ENOUGH_SEATS`)
- Bayesian sort: ★5.0 จาก 1 รีวิวต้องไม่อยู่เหนือ ★4.8 จาก 46 รีวิว; ร้านไม่มีรีวิวอยู่ท้ายสุด
- `slots` บนการ์ด: ใช้ query booking เดียวต่อหน้า; slot นอกเวลาเปิดได้ `closed`
- `next-available`: ร้านเต็มทั้งคืนวันที่ 10 → คืนวันที่ 11; เต็มครบ 14 วัน → คืน `null`

web (Vitest):
- สถานะปุ่มเวลา: `available >= party` และ ≤ 30% ของที่นั่ง → เหลือน้อย; `available < party` → เต็ม
- ป้ายคะแนน: รีวิว 0 / 1–4 / ≥ 5 → "ยังไม่มีรีวิว" / "รีวิวน้อย" / ดาว
- ป้าย "(เช้าวันที่ n)" ขึ้นเมื่อ slot อยู่หลังเที่ยงคืนของวันทำการ
- 409 `DUPLICATE_BOOKING` ไม่แสดงข้อความ "เต็ม"

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
| กดจองซ้ำ/เน็ตกระตุก | กฎ "คนเดียวกันจองร้านเดียวกันซ้อนเวลาไม่ได้" ตรวจหลัง `FOR UPDATE` → คำขอที่สองรอล็อกแล้วเห็นการจองแรก → 409 `DUPLICATE_BOOKING` (หน้าเว็บพาไปการจองของฉัน ไม่บอกว่าร้านเต็ม) |
| ทำไมลูกค้าไม่เห็นแถบ % ที่นั่ง | ลูกค้าถาม "เวลานี้จองได้ไหม" → ปุ่มเวลา 3 สถานะตอบตรงกว่า; แถบ % เป็นมุมมองของเจ้าของร้าน (peak ของคืน) จึงอยู่ในบอร์ด owner |
| การ์ดร้านแสดงเวลาว่างโดยไม่ N+1 ยังไง | query booking ของทุกร้านในหน้านั้นครั้งเดียว (`IN`) แล้วคำนวณ slot ใน Go |
| ร้านเปิด 24 ชม. | `open == close` → `duration` ได้ 0 จาก modulo ต้องตีความเป็น 1440 |
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
- ห้ามเช็คที่นั่ง (กฎ 6) และการจองซ้อนตัวเอง (กฎ 7) นอกทรานแซกชัน / ก่อน `FOR UPDATE`
- ห้ามเขียน logic หาค่าสูงสุดของคนในร้านซ้ำ — ใช้ `maxConcurrent` ตัวเดียว
- ห้ามส่งเวลาใน API เป็น `"HH:MM"` ลอย ๆ — ใช้ timestamp เต็มเสมอ
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
