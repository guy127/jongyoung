# Architecture Overview

เอกสารนี้สรุปสถาปัตยกรรมของ **จองยัง (jongyoung)** ให้คนหรือ agent ที่เพิ่งเข้ามาเข้าใจระบบและหาไฟล์ที่ต้องแก้ได้เร็ว
กติกาธุรกิจฉบับเต็มอยู่ใน [CLAUDE.md](CLAUDE.md) ส่วนวิธีรันอยู่ใน [README.md](README.md) — แก้เอกสารนี้ทุกครั้งที่โครงสร้างเปลี่ยน

## 1. Project Structure

```
jongyoung/
├── compose.yml                # postgres, keycloak, migrate, seed, api, web, caddy
├── .env.example               # ทุก key ที่ compose ใช้ (ค่า dev เท่านั้น)
├── caddy/Caddyfile            # reverse proxy *.jongyoung.localhost → web / api / keycloak
├── keycloak/
│   ├── import/jongyoung-realm.json   # realm + client + user ทดสอบ (import ตอนเริ่ม)
│   └── README.md              # วิธี export realm ใหม่
├── api/                       # Backend — Go module "jongyoung"
│   ├── cmd/
│   │   ├── api/               # main.go (config → DB → OIDC → router), router.go (route + middleware)
│   │   ├── migrate/           # รัน goose up/down/status จาก migration ที่ embed ไว้
│   │   └── seed/              # ข้อมูลตัวอย่าง (--reset ล้างแล้วใส่ใหม่)
│   ├── internal/
│   │   ├── config/            # อ่าน env ครั้งเดียวเป็น struct + validate
│   │   ├── middleware/        # jwt.go (verify + JIT user), oidc.go, devauth.go
│   │   ├── reqctx/            # เก็บ/อ่าน userID ใน context
│   │   ├── httputil/          # error response รูปแบบเดียว, pagination
│   │   ├── platform/          # database (เปิด GORM), testdb (Postgres ใน testcontainers)
│   │   ├── user/              # ผู้ใช้ + EnsureExists (upsert จาก claims)
│   │   ├── restaurant/        # CRUD ร้าน, รูป, ค้นหา + Bayesian, ปุ่มเวลาบนการ์ด, availability
│   │   ├── booking/           # ⭐ businessday.go, availability.go, rules.go, changes.go + จอง/แก้/ยกเลิก/บอร์ด
│   │   └── review/            # รีวิว + คะแนนรวมแบบ atomic
│   ├── migrations/            # goose SQL (Up + Down) — ห้ามมี seed data
│   ├── docs/                  # Swagger 2 + OpenAPI 3 ที่ generate จาก comment (./dev.sh docs)
│   └── Dockerfile
├── web/                       # Frontend — Next.js 16 (App Router)
│   ├── app/
│   │   ├── (customer)/        # หน้าแรก, restaurants/[id], bookings/[id], me/bookings
│   │   ├── owner/             # restaurants (จัดการร้าน), bookings (บอร์ดรายวันทำการ)
│   │   ├── login/             # ส่งต่อไป Keycloak ทันที (แทนหน้า sign-in สำเร็จรูปของ next-auth)
│   │   ├── api/auth/          # [...nextauth], logout
│   │   ├── layout.tsx, providers.tsx, globals.css (design tokens)
│   ├── components/            # bases/ (layout: PageLayout/Section/Card/Field, Choice, ปุ่ม, TimeChip, Alert…),
│   │                          #   restaurant/, booking/, review/, owner/
│   ├── containers/            # Navbar (ลูกค้า), OwnerNavbar
│   ├── services/              # hook TanStack Query ต่อ resource (restaurants, bookings, reviews, me)
│   ├── lib/                   # api.ts (axios), auth/, format.ts (เวลาไทย + ป้ายข้ามวัน), ics.ts, errors.ts, types.ts
│   ├── proxy.ts               # กันหน้า /me, /owner, /bookings ถ้ายังไม่ login
│   └── Dockerfile
├── docs/                      # แผนงาน, design
├── e2e/                       # Playwright E2E (tests/*.spec.ts) — รันใน container ด้วย ./dev.sh e2e
├── dev.sh                     # คำสั่งที่ใช้บ่อย: up / reset / lint / test / e2e / check
├── .gitlab-ci.yml
├── CLAUDE.md                  # spec + กติกาธุรกิจทั้งหมด
├── README.md
└── ARCHITECTURE.md            # เอกสารนี้
```

**กฎการแบ่งชั้นใน Go:** ทุก domain มี `handler.go` (HTTP ↔ struct) → `service.go` (business logic ทั้งหมด) → `repository.go` (คุย DB)
และ `model.go` `dto.go` `error.go` — service รับ repository เป็น interface จึง mock ด้วย mockery ได้

## 2. High-Level System Diagram

```
                         ┌──────────── Docker Compose network ────────────┐
                         │                                                │
[Browser] ──HTTP:80──▶ [Caddy] ──▶ jongyoung.localhost ──────▶ [web: Next.js :3000]
    │                    │                                         │  (SSR หน้า list/detail เรียก API ตรง)
    │                    ├──▶ api.jongyoung.localhost ──────▶ [api: Go/Gin :8080] ──▶ [PostgreSQL]
    │                    │                                         │
    │                    └──▶ keycloak.jongyoung.localhost ──▶ [Keycloak :8080]
    │                                                              ▲
    └─ login redirect / Bearer token ─────────────────────────────┘  api ดึง JWKS มาตรวจ token
```

Caddy มี network alias เป็นทั้งสามชื่อ → container อื่นเรียก `keycloak.jongyoung.localhost` ได้ชื่อเดียวกับ browser
จึงได้ `iss` ใน token ตรงกันทุกฝั่ง (ไม่งั้น browser เห็น `localhost:xxxx` แต่ api เห็น `keycloak:8080` แล้ว verify ไม่ผ่าน)

**Flow การจอง (เส้นทางสำคัญที่สุด)**
```
Browser ─POST /api/v1/bookings (Bearer)─▶ middleware.JWT: verify (go-oidc) → EnsureExists(user) → userID ลง context
   ▶ booking.Handler: bind body → Choice{date, start_time, end_time, party_size}
   ▶ booking.Service: แปลงวันทำการเป็นเวลาจริง (businessday.go) → ValidateRequest (กฎ 2–6, 9–10)
   ▶ ทรานแซกชันเดียว:
        1. SELECT restaurant FOR UPDATE        ← คำขอพร้อมกันต่อคิวที่นี่
        2. หา booking ของ user เดียวกันที่ทับช่วง → 409 DUPLICATE_BOOKING
        3. อ่าน booking ที่ทับช่วง → maxConcurrent (sweep line) → 409 NOT_ENOUGH_SEATS
        4. INSERT booking
   ◀ 201 BookingResponse → หน้าเว็บ router.push(/bookings/:id)
```

## 3. Core Components

### 3.1. Frontend

**Name:** web (Next.js)

**Description:** หน้าเว็บสำหรับลูกค้าและเจ้าของร้านในบัญชีเดียว
- ลูกค้า: ค้นหาร้านพร้อม "ปุ่มเวลา" บนการ์ด → หน้าร้าน + แผงจอง → หน้ายืนยันการจอง (.ics / Google Calendar) → การจองของฉัน → รีวิว
- เจ้าของร้าน (`/owner/*`): จัดการร้าน/รูป และบอร์ดการจองรายวันทำการพร้อมแถบที่นั่งต่อช่วง 30 นาที
- หน้า list/detail เป็น Server Component เรียก API ตรง (ไม่ต้องใช้ token); หน้าที่ต้อง login และฟอร์มเป็น Client Component (TanStack Query + axios)
- next-auth เก็บ access/refresh/id token ใน session cookie ที่เข้ารหัส และต่ออายุ token ใน callback `jwt` เมื่อเหลือไม่ถึง 30 วินาที
- axios interceptor แนบ `Authorization: Bearer` และพาไป login เมื่อได้ 401
- แปลงเวลาเป็น `Asia/Bangkok` ที่ฝั่งนี้เท่านั้น และติดป้าย "(เช้าวันที่ n)" ให้เวลาหลังเที่ยงคืนของรอบ

**Technologies:** Next.js 16 (App Router, Turbopack, `output: standalone`), TypeScript, TanStack Query, axios, next-auth v4 (KeycloakProvider),
react-hook-form + zod, Tailwind v4, lucide-react

**Deployment:** Docker image (node:24-alpine) ใน compose; CI build image แล้ว push ขึ้น GitLab Container Registry

### 3.2. Backend Services

#### 3.2.1. API

**Name:** api (Go)

**Description:** REST API `/api/v1` ที่บังคับกติกาธุรกิจ **ทุกข้อ** — หน้าเว็บตรวจซ้ำเพื่อ UX เท่านั้น
- `userID` มาจาก token เท่านั้น ไม่รับจาก body/query
- สร้างผู้ใช้ในตาราง `users` ตอนเห็น `sub` ครั้งแรก (JIT) และอัปเดต email/ชื่อเมื่อ claims เปลี่ยน
- หัวใจอยู่ที่ `internal/booking`:
  - `businessday.go` — แปลง "วันทำการ" (รอบที่เปิดในวันนั้น รองรับข้ามเที่ยงคืนและ 24 ชม.) ↔ เวลาจริง ใช้ร่วมกันทุกที่
    `Hours` มีช่วงพักได้ 1 ช่วงภายในรอบ — `Fits`/`Slots`/`OpenAtMinute` ไม่นับช่วงพักเป็นเวลาเปิด ทุกที่ที่เรียกฟังก์ชันเหล่านี้ได้กติกาเดียวกัน
  - `availability.go` — `maxConcurrent()` sweep line ใช้ทั้งตอนจอง แก้ไข ลดที่นั่ง และคำนวณช่วงว่าง
- ปุ่มเวลาบนการ์ดร้าน: query booking ครั้งเดียวต่อหน้า (`restaurant_id IN (...)`) แล้วคำนวณใน Go — ไม่มี N+1
- middleware: Recovery → RequestLog (`X-Request-ID` + log JSON ต่อคำขอ) → CORS (`http://jongyoung.localhost` เท่านั้น)
  → RateLimit (POST/PUT/DELETE 30 ครั้ง/นาที/IP → 429 `RATE_LIMITED`) → JWT (เฉพาะ route ที่ต้อง login)
- Swagger ที่ `/swagger/index.html`, health check ที่ `/healthz`
- **แตะตาราง `restaurants` ข้ามโดเมนโดยตั้งใจ:** `booking` ล็อกแถวร้าน (`LockRestaurant`) และ `review` อ่าน `owner_id` + ปรับ `rating_sum/rating_count` ด้วย SQL ของตัวเอง
  เพราะงานเหล่านี้ต้องอยู่ในทรานแซกชันเดียวกับการเขียน booking/review — ถ้าเรียกผ่าน package `restaurant` ต้องส่ง `*gorm.DB` ของทรานแซกชันข้ามแพ็กเกจ
  ทำให้ขอบเขตทรานแซกชันรั่วออกนอก service (และ `booking` import `restaurant` ไม่ได้อยู่แล้ว เพราะ `restaurant` import `booking` → import cycle)
  คอลัมน์ที่แตะมีแค่ `id, owner_id, name, address, map_url, seats, open_minute, close_minute, closed_weekdays, break_start_minute, break_end_minute, cancel_before_minutes, rating_sum, rating_count`
  — ถ้าเปลี่ยน schema ส่วนนี้ต้องแก้ทั้งสามแพ็กเกจ

**Technologies:** Go 1.26, Gin, GORM (pgx) + raw SQL, goose, coreos/go-oidc, caarlos0/env + validator, swaggo, log/slog

**Deployment:** Docker image (distroless) ใน compose; binary `migrate` และ `seed` อยู่ใน image เดียวกัน รันเป็น service แยกก่อน `api`

#### 3.2.2. Identity Provider

**Name:** Keycloak

**Description:** เก็บตัวตนผู้ใช้ (login, สมัครสมาชิก, refresh token, logout) แยกจาก business logic
- realm `jongyoung`, client `jongyoung-web` แบบ confidential (ใช้กับ next-auth), audience mapper ใส่ `aud = jongyoung-api` ให้ access token
- **ไม่มี role owner/customer** — ความเป็นเจ้าของคือ `restaurants.owner_id` (ข้อมูล ไม่ใช่สิทธิ์ระดับบัญชี)

**Technologies:** Keycloak 26.4 (`start-dev --import-realm`)

**Deployment:** container ใน compose, import realm จาก `keycloak/import/` ทุกครั้งที่เริ่ม;
หน้า login ใช้ธีม `keycloak/themes/jongyoung` (ต่อยอด keycloak.v2 ด้วย CSS อย่างเดียว, ภาษาไทยเป็นค่าเริ่มต้น)

#### 3.2.3. Reverse Proxy

**Name:** Caddy

**Description:** รับพอร์ต 80 แล้วแยกตาม host ไป web / api / keycloak และทำให้ทุกฝ่ายเห็น Keycloak ด้วยชื่อเดียวกัน

**Technologies:** Caddy 2

## 4. Data Stores

### 4.1. Primary Database

**Name:** jongyoung

**Type:** PostgreSQL 17

**Purpose:** เก็บ domain data ทั้งหมด เลือก Postgres เพราะต้องใช้ทรานแซกชัน + `SELECT ... FOR UPDATE` กันจองชน,
`timestamptz`, partial index และ `ON CONFLICT` สำหรับ upsert ผู้ใช้

**Key Schemas/Collections:**
| ตาราง | หมายเหตุ |
|---|---|
| `users` | ผูกกับ Keycloak ด้วย `keycloak_uid` (= `sub`) — partial unique index `WHERE deleted_at IS NULL` |
| `restaurants` | `seats`, `open_minute`/`close_minute` (นาทีจากเที่ยงคืนเวลาไทย), `cancel_before_minutes`, `rating_sum`/`rating_count`, soft delete |
| `restaurant_images` | URL รูป, `sort_order` 0 = รูปปก |
| `bookings` | `start_at`/`end_at` (UTC), `status` active/cancelled (ไม่ลบจริง), partial index สำหรับ booking ที่ active |
| `reviews` | `unique (restaurant_id, user_id)` — 1 คน 1 รีวิวต่อร้าน |

ไม่มีตารางโต๊ะ (นับที่นั่งรวม) และไม่เก็บพิกัด (แผนที่เป็นลิงก์ Google Maps: `map_url` ที่เจ้าของร้านวาง หรือค้นจาก `address`)
Migration อยู่ที่ `api/migrations/` (goose, มี Down ทุกไฟล์) — seed data แยกไว้ที่ `api/cmd/seed/`

### 4.2. Cache / Queue

ไม่มี — ไม่ใช้ Redis เพราะ Go ไม่มี login flow ที่ต้องเก็บ state (next-auth จัดการ) และไม่มีข้อมูลที่ต้อง cache ข้ามคำขอ
ค่าเฉลี่ยรีวิวเก็บเป็นผลรวมที่ตารางร้านแทนการ cache

## 5. External Integrations / APIs

**Keycloak** — **Purpose:** OIDC identity provider — **Integration Method:** OIDC authorization code flow (next-auth), JWKS + discovery (go-oidc)

**Google Maps** — **Purpose:** ลิงก์แผนที่ของร้าน — **Integration Method:** ลิงก์ที่เจ้าของร้านวาง (`map_url`, รับเฉพาะโดเมน Google Maps) หรือ URL ค้นหาจาก `address` (ไม่ใช้ API key)

**Google Calendar** — **Purpose:** เพิ่มการจองลงปฏิทิน — **Integration Method:** template URL + ไฟล์ `.ics` ที่สร้างฝั่ง web

รูปร้านเป็น URL ภายนอกที่เจ้าของร้านใส่เอง (ไม่มีระบบอัปโหลดไฟล์)

## 6. Deployment & Infrastructure

**Cloud Provider:** ยังไม่มี — ตอนนี้รันด้วย Docker Compose บนเครื่องเดียว (`docker compose up -d --build`)

**Key Services Used:** Docker Compose, GitLab Container Registry

**CI/CD Pipeline:** GitLab CI ([.gitlab-ci.yml](.gitlab-ci.yml))
| Stage | ทำอะไร | รันเมื่อ |
|---|---|---|
| lint | `go vet`, `gofmt -l`, `npm run lint`, `tsc --noEmit` | ทุก push |
| test | `go test ./...` (Docker-in-Docker ให้ testcontainers), `npm test` | ทุก push |
| build | `go build`, `next build` | ทุก push |
| image | build + push image `api`/`web` (tag SHA + latest) ด้วย `CI_JOB_TOKEN` | เข้า `main` |

ยังไม่มี deploy อัตโนมัติ — ถ้าทำ ใช้ deploy key คู่ใหม่เก็บเป็น CI/CD variable แบบ Protected + Masked

**Monitoring & Logging:** log JSON ลง stdout (`log/slog`) หนึ่งบรรทัดต่อคำขอพร้อม `request_id` (ส่งกลับใน header `X-Request-ID` ด้วย) ดูด้วย `docker compose logs`; `/healthz` สำหรับ health check

## 7. Security Considerations

**Authentication:** OIDC ผ่าน Keycloak; Go ตรวจ access token (JWT) ครบ signature + `exp` + `iss` + `aud` ด้วย go-oidc
- `DEV_AUTH` (ข้ามการตรวจ token) เปิดได้เฉพาะ `APP_ENV=development` และมี log เตือน

**Authorization:** ตรวจความเป็นเจ้าของใน service ทีละคำขอ (ไม่ใช่ RBAC)
- เจ้าของร้าน = `restaurants.owner_id == userID`
- เจ้าของการจอง = `bookings.user_id == userID`
- ไม่ใช่เจ้าของ → 403; รีวิวร้านตัวเอง → 403 `OWN_RESTAURANT`

**Data Encryption:**
- local ใช้ HTTP (`*.localhost`) — production ต้องเปิด TLS ที่ Caddy
- session cookie ของ next-auth เข้ารหัสด้วย `NEXTAUTH_SECRET`

**Key Security Tools/Practices:**
- CORS อนุญาตเฉพาะ `http://jongyoung.localhost`
- ไม่เก็บรหัสผ่านเอง; ไม่เก็บ token ใน localStorage; ไม่ใช้ `dangerouslySetInnerHTML` กับข้อมูลผู้ใช้ (ลดความเสี่ยง XSS ต่อ token ที่ browser ถือ)
- ไม่ log token / client secret / DSN เต็ม; `.env` ไม่ถูก commit; secret ใน realm export เป็นค่า dev เท่านั้น
- ทางที่ปลอดภัยกว่าในอนาคต: BFF proxy ใน Next.js ให้ token อยู่ฝั่ง server อย่างเดียว

## 8. Development & Testing Environment

**Local Setup Instructions:** ดู [README.md](README.md) — `cp .env.example .env && docker compose up -d --build` แล้วเปิด http://jongyoung.localhost

**Testing Frameworks:**
- Go: testify, mockery (mock repository ลง `mocks_test.go`), httptest (handler), testcontainers-go (Postgres จริง — รวมเทสต์จองพร้อมกัน/กดซ้ำ)
- Web: Vitest + Testing Library (jsdom)
- E2E: Playwright (`e2e/`) ทดสอบผ่าน browser จริง desktop + มือถือ 375px — `./dev.sh e2e` ล้าง seed ก่อนทุกครั้ง

**Code Quality Tools:** `go vet`, `gofmt`, ESLint (eslint-config-next + React Compiler rules), TypeScript strict

## 9. Future Considerations / Roadmap

- rate limit เก็บตัวนับในหน่วยความจำของ process — ถ้ารัน api หลาย instance ต้องย้ายไปเก็บที่ Redis
- ย้าย token ไปอยู่ฝั่ง server ทั้งหมดด้วย BFF proxy
- ถ้ามีร้านหลาย timezone ต้องเก็บ timezone ต่อร้าน และคำนวณวันทำการตามปฏิทินของ timezone นั้น (ตอนนี้ `Asia/Bangkok` ไม่มี DST)
- ถ้ามีการตัดเงินมัดจำ ต้องใช้ `Idempotency-Key` จริง (ตอนนี้กันกดซ้ำด้วยกฎ DUPLICATE_BOOKING)
- ถ้ามีภาระสูง: ทางเลือกแทน row lock คือ `EXCLUDE USING gist` หรือ `SERIALIZABLE` + retry
- deploy อัตโนมัติจาก registry ไปยังเซิร์ฟเวอร์จริง + TLS

## 10. Project Identification

**Project Name:** จองยัง (jongyoung) — ระบบจองโต๊ะร้านอาหาร + รีวิวร้าน (โจทย์สอบปลายภาค PEA DevPool 2026)

**Repository URL:** https://gitlab.com/guy127/jongyoung

**Primary Contact/Team:** Pongsatorn Deekratok

**Date of Last Update:** 2026-09-26

## 11. Glossary / Acronyms

**วันทำการ (business date / `date`):** รอบเปิดร้านที่ *เริ่ม* ในวันปฏิทินนั้น — ร้าน 18:00–02:00 วันที่ 10 = 18:00 วันที่ 10 ถึง 02:00 วันที่ 11

**Sweep line / `maxConcurrent`:** นับคนในร้าน ณ ทุกจุดที่มีคนเข้าร้าน เพื่อหาจำนวนคนพร้อมกันสูงสุด (แทน `SUM` ที่นับเกินเมื่อการจองไม่ได้ทับกันจริง)

**Lead time:** ต้องจองล่วงหน้าอย่างน้อย 30 นาที (`TOO_LATE_TO_BOOK`)

**Cancel window:** ยกเลิก/แก้ไขได้ถึง `start_at - cancel_before_minutes`

**Time chip (ปุ่มเวลา):** ปุ่มช่วง 30 นาทีบนการ์ดร้าน/หน้าร้าน สถานะ ว่าง / เหลือน้อย / เต็ม / ปิด

**Seat bar:** แถบสัดส่วนที่นั่งที่ถูกจองต่อช่วง — แสดงเฉพาะฝั่งเจ้าของร้าน

**JIT provisioning:** สร้าง/อัปเดตแถวใน `users` ตอนเห็น token ครั้งแรก แทนการสร้างตอน login

**Bayesian average:** `(C·m + rating_sum) / (C + rating_count)` โดย C = 5 — กันร้านรีวิวน้อยแซงร้านรีวิวเยอะ

**OIDC / JWKS:** OpenID Connect / JSON Web Key Set (public key ที่ใช้ตรวจลายเซ็น token)

**BFF:** Backend for Frontend — server กลางที่ถือ token แทน browser
