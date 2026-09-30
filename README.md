# จองยัง (jongyoung)

ระบบจองโต๊ะร้านอาหาร + รีวิวร้าน — งานสอบปลายภาค PEA DevPool 2026

- **Frontend:** Next.js 16 (App Router) + TypeScript + TanStack Query + next-auth
- **Backend:** Go + Gin + GORM + PostgreSQL
- **Auth:** Keycloak (OIDC)
- **รันทั้งระบบด้วย** Docker Compose คำสั่งเดียว

## สิ่งที่ส่ง

| # | อย่าง | อยู่ที่ |
|---|---|---|
| 1 | Git repository | repo นี้ — วิธีรันอยู่ข้อ 1 |
| 2 | วิดีโอเดโม ≤ 5 นาที | <ใส่ลิงก์> |
| 3 | UI Design | [docs/ui-design.md](docs/ui-design.md) |

---

## 1. วิธีรัน

ต้องมี Docker (Docker Desktop / Rancher Desktop) และพอร์ต **80** ว่าง

```bash
cp .env.example .env
docker compose up -d --build
```

compose จะทำให้ครบตามลำดับ: `postgres` → `migrate` (goose up) → `seed` (ข้อมูลตัวอย่าง) → `api`, `keycloak` (import realm) → `web` → `caddy`
รอ Keycloak พร้อมประมาณ 30–60 วินาที แล้วเปิด:

| URL | คืออะไร |
|---|---|
| http://jongyoung.localhost | หน้าเว็บ |
| http://api.jongyoung.localhost/swagger/index.html | Swagger ของ API |
| http://api.jongyoung.localhost/healthz | health check |
| http://keycloak.jongyoung.localhost/admin | Keycloak admin (`admin` / `admin`) |

> `*.localhost` ชี้มาที่เครื่องตัวเองอัตโนมัติ (RFC 6761) ใน Chrome / Firefox / Edge ไม่ต้องแก้อะไร
> ถ้า browser หรือโปรแกรมไหนหาชื่อไม่เจอ ให้เพิ่มบรรทัดนี้ใน `/etc/hosts` (Windows: `C:\Windows\System32\drivers\etc\hosts`)
> ```
> 127.0.0.1 jongyoung.localhost api.jongyoung.localhost keycloak.jongyoung.localhost
> ```

คำสั่งที่ใช้บ่อยรวมไว้ใน [`dev.sh`](dev.sh) (bash ธรรมดา อ่านได้ในไฟล์เดียวว่าแต่ละคำสั่งทำอะไร)
```bash
./dev.sh up       # สร้างและรันทั้งระบบ
./dev.sh reset    # ล้างข้อมูลแล้วใส่ข้อมูลตัวอย่างใหม่ (เวลาในข้อมูลอิงจากวันนี้)
./dev.sh check    # lint + unit/integration test + E2E — ตรวจทุกอย่างในคำสั่งเดียว
./dev.sh help     # ดูคำสั่งทั้งหมด

docker compose run --rm migrate status # ดูว่ารัน migration ไหนไปแล้ว (down = ย้อนทีละขั้น)
docker compose down -v                 # ปิดและลบข้อมูลทั้งหมด
```

ต่อฐานข้อมูลด้วย TablePlus / DBeaver / psql: `127.0.0.1:5433` user `jongyoung` password `jongyoung-dev-password` db `jongyoung`

### บัญชีทดสอบ (Keycloak)

| username | password | ชื่อ | บทบาทในข้อมูลตัวอย่าง |
|---|---|---|---|
| `owner1` | `owner1-pass` | สมศรี ใจงาม | เจ้าของร้าน ครัวป้าแดง, บ้านชาบู, โจ๊กสามย่าน |
| `owner2` | `owner2-pass` | วีระ ทองดี | เจ้าของร้าน ซูชิ ทาคุมิ, ท่าเรือซีฟู้ดบาร์, ครัวบ้านสวน |
| `customer1` | `customer1-pass` | มะลิ วงศ์ดี | ลูกค้า มีการจองอยู่แล้ว 4 รายการ |

สมัครบัญชีใหม่ได้จากหน้า login ของ Keycloak (เปิด self-registration ไว้)
ทุกบัญชีเป็นได้ทั้งลูกค้าและเจ้าของร้าน ("หนึ่งบัญชี สองบทบาท") — สร้างร้านแรกแล้วตัวสลับ `ลูกค้า | เจ้าของร้าน` จะขึ้นที่ navbar

> ⚠️ **secret ทั้งหมดใน repo นี้เป็นค่าสำหรับ dev เท่านั้น** — `.env.example` และ client secret ใน
> `keycloak/import/jongyoung-realm.json` (`dev-only-jongyoung-web-secret`) ต้องตรงกัน ห้ามนำไปใช้จริง
> วิธี export realm ใหม่ดูที่ [keycloak/README.md](keycloak/README.md)

### ข้อมูลตัวอย่าง (seed)

| ร้าน | เวลา | สิ่งที่โชว์ |
|---|---|---|
| ครัวป้าแดง ตามสั่ง | 10:00–21:00 | เคสปกติ + พรุ่งนี้ 18:00–19:30 เหลือ 2 ที่ (ปุ่มเวลา "เหลือน้อย") |
| ซูชิ ทาคุมิ | 11:00–22:00 พัก 15:00–17:00 | 46 รีวิว ★4.8, ยกเลิกได้ก่อน 60 นาที, **ช่วงพักบ่าย** (จองช่วงพักไม่ได้) |
| บ้านชาบู บุฟเฟ่ต์ | 17:00–23:00 | เปิดเฉพาะเย็น + รีวิวเดียว ★5.0 → ป้าย "รีวิวน้อย (1)" และไม่แซงร้าน ★4.8 + **ปิดปรับปรุงทั้งวัน 5 วันข้างหน้า** (ทดสอบปิดร้านชั่วคราว) |
| ท่าเรือซีฟู้ดบาร์ | 18:00–02:00 | **ข้ามเที่ยงคืน** + ป้าย "(เช้าวันที่ n)" |
| โจ๊กสามย่าน 24 ชม. | 00:00–00:00 | **เปิด 24 ชม.** + ยังไม่มีรีวิว (อยู่ท้ายสุดของ "คะแนนสูงสุด") |
| ครัวบ้านสวน | 11:00–22:00 | **เต็มทั้งรอบพรุ่งนี้** → กล่อง "เต็มทั้งรอบ · ว่างวันถัดไป" |

การจองของ `customer1`: กำลังจะถึง 2 รายการ (หนึ่งรายการคร่อมเที่ยงคืนที่ร้านซีฟู้ด), ผ่านมาแล้ว 1, ยกเลิกเอง 1
และร้านยกเลิกให้อีก 1 รายการพร้อมเหตุผล (แจ้งเตือนยังไม่อ่านบนกระดิ่ง) — `owner1`/`owner2` ก็มีแจ้งเตือนการจองใหม่ยังไม่อ่านเช่นกัน

---

## 2. โครงสร้าง

```
compose.yml            postgres, keycloak, migrate, seed, api, web, caddy
caddy/Caddyfile        *.jongyoung.localhost → web / api / keycloak
keycloak/import/       realm + client + user ทดสอบ (import ตอนเริ่ม)
api/                   Go — handler → service → repository ต่อ domain
  internal/booking/    ⭐ businessday.go (วันทำการ), availability.go (sweep line), rules.go, service.go
  internal/restaurant/ CRUD, ลดที่นั่ง/ย่นเวลา, ค้นหา + Bayesian, ปุ่มเวลาบนการ์ด
  internal/review/     รีวิว + คะแนนรวมแบบ atomic
  internal/middleware/ ตรวจ token (go-oidc) + สร้าง user ครั้งแรกที่เห็น (JIT)
  migrations/          goose (Up/Down)     cmd/seed/  ข้อมูลตัวอย่าง
web/                   Next.js
  app/(customer)/      หน้าแรก, หน้าร้าน, หน้ายืนยันการจอง, การจองของฉัน
  app/owner/           ร้านของฉัน, บอร์ดการจองรายวันทำการ
  services/            hook TanStack Query ต่อ resource     lib/  axios, auth, format เวลาไทย, .ics
e2e/                   Playwright — ทดสอบระบบจริงทั้งก้อนผ่าน browser (desktop + มือถือ 375px)
dev.sh                 คำสั่งที่ใช้บ่อย (up / reset / lint / test / e2e / check)
.gitlab-ci.yml         lint → test → build → image (main)
```

กติกาธุรกิจทุกข้อบังคับที่ Go เสมอ หน้าเว็บตรวจซ้ำเพื่อ UX เท่านั้น — รายละเอียดทั้งหมดอยู่ใน [CLAUDE.md](CLAUDE.md)

### Use case diagram

หนึ่งบัญชี สองบทบาท: เจ้าของร้านคือลูกค้าที่มีร้าน (`restaurants.owner_id`) ไม่ใช่ role ใน Keycloak
ส่วนการตรวจที่นั่ง (`maxConcurrent`) และเวลาเปิด-ปิด (`businessday.go`) เป็น use case ย่อยตัวเดียวที่ทุกทางเรียกใช้ร่วมกัน

```mermaid
flowchart LR
    %% ===== Actors =====
    Guest["ผู้เยี่ยมชม<br/>(ยังไม่ล็อกอิน)"]
    Customer["ลูกค้า<br/>(ล็อกอินแล้ว)"]
    Owner["เจ้าของร้าน<br/>(ผู้ใช้ที่มีร้าน)"]
    KC["Keycloak<br/>(ระบบภายนอก)"]

    %% หนึ่งบัญชี สองบทบาท: Owner เป็น Customer ที่มีร้าน, Customer ทำทุกอย่างที่ Guest ทำได้
    Customer -. "generalization" .-> Guest
    Owner -. "generalization" .-> Customer

    subgraph SYS["ระบบ jongyoung"]
        direction TB

        %% --- ทุกคน ---
        UC_Search(["ค้นหาร้าน<br/>วัน / เวลา / จำนวนคน"])
        UC_Chips(["ดูเวลาว่างบนการ์ด (time chip)"])
        UC_Detail(["ดูรายละเอียดร้าน + รูป + ที่อยู่"])
        UC_Avail(["ดูช่วงว่างของวันทำการ"])
        UC_Next(["ดูวันถัดไปที่ว่าง"])
        UC_Reviews(["อ่านรีวิว"])
        UC_Login(["เข้าสู่ระบบ / สมัครสมาชิก"])

        %% --- ลูกค้า ---
        UC_Book(["จองโต๊ะ"])
        UC_Confirm(["ดูหน้ายืนยันการจอง"])
        UC_Cal(["เพิ่มลงปฏิทิน<br/>Google Calendar / .ics"])
        UC_MyBk(["ดูการจองของฉัน"])
        UC_EditBk(["แก้ไขการจอง"])
        UC_CancelBk(["ยกเลิกการจอง"])
        UC_Review(["เขียน / แก้ / ลบรีวิว"])
        UC_Logout(["ออกจากระบบ"])
        UC_CreateR(["สร้างร้าน"])

        %% --- เจ้าของร้าน ---
        UC_EditR(["แก้ไขร้าน<br/>ที่นั่ง / เวลาเปิด-ปิด / วันปิด"])
        UC_DelR(["ลบร้าน"])
        UC_Img(["จัดการรูปร้าน"])
        UC_Board(["ดูบอร์ดการจองรายวันทำการ<br/>+ seat bar"])

        %% --- use case ย่อย (include) ---
        UC_Seats(["ตรวจที่นั่งทุกช่วงเวลา<br/>(maxConcurrent)"])
        UC_Dup(["ตรวจการจองซ้อนของตัวเอง"])
        UC_Hours(["ตรวจเวลาเปิด-ปิด + วันปิด<br/>(businessday)"])
        UC_Window(["ตรวจเวลาที่ยังยกเลิกได้"])
        UC_CascadeCancel(["ยกเลิก booking ในอนาคตของร้าน"])
    end

    %% ===== Actor ↔ Use case =====
    Guest --- UC_Search
    Guest --- UC_Chips
    Guest --- UC_Detail
    Guest --- UC_Avail
    Guest --- UC_Next
    Guest --- UC_Reviews
    Guest --- UC_Login

    Customer --- UC_Book
    Customer --- UC_Confirm
    Customer --- UC_MyBk
    Customer --- UC_EditBk
    Customer --- UC_CancelBk
    Customer --- UC_Review
    Customer --- UC_Logout
    Customer --- UC_CreateR

    Owner --- UC_EditR
    Owner --- UC_DelR
    Owner --- UC_Img
    Owner --- UC_Board

    UC_Login --- KC
    UC_Logout --- KC

    %% ===== include / extend =====
    UC_Book -. "«include»" .-> UC_Dup
    UC_Book -. "«include»" .-> UC_Seats
    UC_Book -. "«include»" .-> UC_Hours
    UC_EditBk -. "«include»" .-> UC_Dup
    UC_EditBk -. "«include»" .-> UC_Seats
    UC_EditBk -. "«include»" .-> UC_Hours
    UC_EditBk -. "«include»" .-> UC_Window
    UC_CancelBk -. "«include»" .-> UC_Window
    UC_EditR -. "«include»" .-> UC_Seats
    UC_EditR -. "«include»" .-> UC_Hours
    UC_DelR -. "«include»" .-> UC_CascadeCancel
    UC_Chips -. "«include»" .-> UC_Seats
    UC_Avail -. "«include»" .-> UC_Seats
    UC_Cal -. "«extend»" .-> UC_Confirm
    UC_Next -. "«extend»" .-> UC_Chips
```

---

## 3. จุดสำคัญของโจทย์

### ที่นั่งไม่เกินในทุกวินาที — sweep line ไม่ใช่ SUM
ร้าน 10 ที่: A จอง 7 คน 12:00–12:30, B จอง 7 คน 12:30–13:00, C ขอ 3 คน 12:00–13:00
`SUM` ของทุกการจองที่ทับช่วงของ C = 17 → ปฏิเสธ **แต่ที่ถูกคือต้องจองได้** เพราะ A กับ B ไม่ได้อยู่พร้อมกัน

`maxConcurrent()` ใน [availability.go](api/internal/booking/availability.go) นับคนในร้านเฉพาะ "จุดที่มีคนเข้าร้าน"
(จำนวนคนเพิ่มขึ้นได้เฉพาะตอนมีคนเริ่ม ระหว่างสองจุดค่าคงที่) — ฟังก์ชันเดียวนี้ใช้ทั้งตอนจอง แก้ไข ลดที่นั่ง และคำนวณเวลาว่าง

### สองคนกดพร้อมกัน — ล็อกแถวร้าน
ทุกการจอง/แก้ไขทำในทรานแซกชันเดียว: `SELECT ... FOR UPDATE` แถวร้าน → ตรวจ "จองซ้อนตัวเอง" → ตรวจที่นั่ง → เขียน
คำขอที่สองรอที่บรรทัดล็อกจนคนแรกเสร็จ จึงเห็นข้อมูลล่าสุดเสมอ → สำเร็จคนเดียว อีกคนได้ `409 NOT_ENOUGH_SEATS`
(มี integration test ยิงสอง goroutine ใส่ Postgres จริง)

กดจองซ้ำ/เน็ตกระตุก: กฎ "ผู้ใช้คนเดิมจองร้านเดิมซ้อนเวลาไม่ได้" ถูกตรวจในล็อกเดียวกัน **ก่อน** นับที่นั่ง
→ คำขอที่สองได้ `409 DUPLICATE_BOOKING` พร้อมลิงก์ไปรายการเดิม (ไม่ใช่ "ร้านเต็ม")

### ร้านเปิดข้ามเที่ยงคืน และ "วันทำการ"
เก็บเวลาเปิด–ปิดเป็นนาทีจากเที่ยงคืน: `close < open` = ข้ามวัน, `open == close` = 24 ชม.
`date` ทุกที่ในระบบหมายถึง **รอบที่เปิดในวันนั้น** — ร้าน 18:00–02:00 วันที่ 10 = 18:00 วันที่ 10 ถึง 02:00 วันที่ 11
API ส่ง timestamp เต็มพร้อม offset เสมอ หน้าเว็บจึงไม่ต้องเดาวัน และติดป้าย "(เช้าวันที่ 11)" ให้เวลาหลังเที่ยงคืน

### ร้านที่เปิดสองช่วง — ช่วงพักภายในรอบ
ร้านเปิดกลางวัน–เย็นเก็บเป็น "รอบเดียว + ช่วงพัก 1 ช่วง" เช่น 11:00–22:00 พัก 14:00–17:00 (`break_start_minute`/`break_end_minute`, เท่ากัน = ไม่มีพัก)
นิยามวันทำการไม่เปลี่ยนเลย — `Fits` ปฏิเสธช่วงจองที่ทับช่วงพัก, `Slots` ข้ามช่วงพัก, ตั้งช่วงพักทับการจองที่มีอยู่ → 409 `HOURS_CONFLICT_EXISTING_BOOKINGS`
ไม่เลือกตารางเวลาเปิดหลายช่วง/รายวัน (`restaurant_hours`) เพราะต้องเขียนการแปลงวันทำการใหม่ทั้งชุด — ถ้าต้องการเวลาเปิดต่างกันรายวันค่อยย้ายไปแบบนั้น

### เจ้าของร้านจองร้านตัวเองได้
ต่างจากรีวิวที่ห้ามเพราะผลประโยชน์ทับซ้อน — การจองไม่มีใครเสียประโยชน์ และเจ้าของลดที่นั่งเองได้อยู่แล้ว
ข้อจำกัด: กฎ "คนเดียวกันจองซ้อนเวลาไม่ได้" ทำให้จองแทนลูกค้าที่โทรมาสองรายในเวลาทับกันไม่ได้ → งานต่อยอด: ช่อง "ชื่อผู้มา" ในการจอง

---

## 4. เหตุผลที่เลือกเครื่องมือ

| เลือก | เพราะ |
|---|---|
| **PostgreSQL** | ต้องใช้ทรานแซกชัน + row lock (`FOR UPDATE`) กันจองชน, `timestamptz`, partial index, และ `ON CONFLICT` สำหรับ upsert ผู้ใช้ |
| **Keycloak (OIDC)** | แยกเรื่องตัวตนออกจาก business logic — ไม่ต้องเก็บรหัสผ่านเอง ได้ refresh token / logout / สมัครสมาชิกมาเลย |
| **go-oidc** | ตรวจ signature (JWKS + หมุน key) + `exp` + `iss` + `aud` ให้ครบในที่เดียว |
| **next-auth** | จัดการ authorization code flow + state + PKCE + session cookie ไม่ต้องเขียน flow เอง |
| **Caddy + `*.jongyoung.localhost`** | browser และ container เรียก Keycloak ด้วยชื่อเดียวกัน → `iss` ใน token ตรงกันทั้งสองฝั่ง |
| **GORM + raw SQL** | CRUD ธรรมดาใช้ GORM; query ที่ต้องคุมเอง (Bayesian, upsert, อัปเดตคะแนนแบบ atomic) เขียน SQL ตรง |
| **goose** | migration มีเวอร์ชัน มีทั้ง Up/Down และ embed ลง binary |
| **TanStack Query + axios** | cache / refetch / loading อยู่ที่เดียว; interceptor แนบ Bearer token |
| **testcontainers** | repository test รันกับ Postgres จริง — ทดสอบ lock และ SQL ได้จริง ไม่ใช่ mock |

**ไม่ใช้ Redis** — Go ไม่มี login flow ที่ต้องเก็บ state (next-auth จัดการ) และไม่มีอะไรที่ต้อง cache ข้ามคำขอ

owner/customer **ไม่ใช่ role ใน Keycloak** — ความเป็นเจ้าของผูกกับร้านแต่ละร้าน (`restaurants.owner_id`) เป็นข้อมูล ไม่ใช่สิทธิ์ระดับบัญชี
ตาราง `users` ของเราผูกกับ Keycloak ด้วย `sub` (ไม่ใช่ email เพราะเปลี่ยนได้) และ upsert ทุกครั้งที่เห็น token → ชื่อ/อีเมลตามทัน Keycloak เสมอ

---

## 5. การเรียง "คะแนนสูงสุด": Bayesian average vs เกณฑ์ขั้นต่ำ ≥ 5 รีวิว

ปัญหา: ร้าน ★5.0 จาก 1 รีวิว ไม่ควรอยู่เหนือร้าน ★4.8 จาก 46 รีวิว

```
score = (C·m + rating_sum) / (C + rating_count)     C = 5, m = ค่าเฉลี่ยทุกรีวิวในระบบ
```

| | Bayesian average (ที่เลือก) | ตัดร้านที่รีวิว < 5 ออก / ไปไว้ท้าย |
|---|---|---|
| วิธีคิด | เหมือนทุกร้านมี "รีวิวสมมติ" C ใบที่ได้คะแนนเฉลี่ยของระบบ รีวิวจริงยิ่งเยอะ ยิ่งดึงคะแนนออกจากค่ากลางได้ | ร้านที่รีวิวไม่ถึงเกณฑ์ไม่ถูกจัดอันดับด้วยคะแนน |
| ร้านรีวิว 4 ใบ vs 5 ใบ | ต่างกันนิดเดียว (ต่อเนื่อง) | ต่างกันสุดขั้ว (เส้นตัดแข็ง) |
| ร้านใหม่ที่ดีจริง | ค่อย ๆ ไต่ขึ้นเมื่อมีรีวิวเพิ่ม | ไม่มีทางขึ้นจนกว่าจะครบ 5 |
| อธิบายให้ผู้ใช้ | ยากกว่าเล็กน้อย | ง่าย |

ตัวอย่างจาก seed (รีวิวทั้งระบบ 364 คะแนน / 78 รีวิว → m ≈ 4.67):
บุฟเฟ่ต์ ★5.0 จาก 1 → (5·4.67 + 5)/6 ≈ **4.72** · ซูชิ ★4.8 จาก 46 → (5·4.67 + 221)/51 ≈ **4.79** → ซูชิอยู่เหนือ

เพิ่มเติม:
- `score` คำนวณใน query (ไม่ใช่คอลัมน์) และใช้ `::float` กันหารจำนวนเต็ม
- ร้านที่ยังไม่มีรีวิวอยู่ท้ายสุดเสมอ ไม่งั้นจะได้คะแนน = m พอดีแล้วแซงร้านที่มีรีวิวจริงแต่ต่ำกว่าค่าเฉลี่ย
- UI ใช้เหตุผลเดียวกัน: รีวิว < 5 แสดงป้าย "รีวิวน้อย (n)" แทนดาวเด่น
- ค่าเฉลี่ยไม่ต้อง `AVG()` ทุกครั้ง: เก็บ `rating_sum` / `rating_count` ที่ร้าน และอัปเดตแบบ atomic
  (`rating_sum = rating_sum + ?`) ในทรานแซกชันเดียวกับรีวิว

---

## 6. Test

```bash
./dev.sh test     # Go (unit + integration กับ Postgres จริงใน testcontainers) + web (Vitest)
./dev.sh e2e      # E2E กับระบบที่รันอยู่ (ล้าง seed ก่อนทุกครั้ง ผลจึงคงที่) — ./dev.sh e2e --project=mobile
./dev.sh check    # lint + test + e2e
```

ทดสอบ 3 ชั้น แต่ละชั้นตอบคำถามต่างกัน:

| ชั้น | อยู่ที่ | ตอบคำถาม |
|---|---|---|
| unit | `api/internal/**/_test.go`, `web/**/*.test.ts(x)` | กติกาแต่ละข้อถูกไหม (เร็ว ไม่ต้องมี DB) |
| integration | `*_integration_test.go` (testcontainers) | SQL, lock, การจองพร้อมกันถูกไหม บน Postgres จริง |
| E2E | `e2e/tests/*.spec.ts` (Playwright) | ผู้ใช้ทำงานได้จริงไหม ตั้งแต่ login ที่ Keycloak จนจองเสร็จ |

- `api/internal/booking` — กติกาที่นั่ง (รวมเคส A/B/C), lead time, ข้ามเที่ยงคืน, 24 ชม., วันทำการ, cancel window, แก้ไขโดยไม่นับตัวเอง, จองพร้อมกัน / กดซ้ำ (Postgres จริง)
- `api/internal/restaurant` — ลดที่นั่ง/ย่นเวลาชนการจองที่มีอยู่, Bayesian sort, ปุ่มเวลาบนการ์ด (query เดียวต่อหน้า), วันถัดไปที่ว่าง
- `api/internal/review` — รีวิวร้านตัวเอง 403, รีวิวซ้ำ 409, คะแนนรวมถูกต้องเมื่อรีวิวพร้อมกัน
- `api/internal/middleware`, `user` — 401, สร้าง/อัปเดตผู้ใช้จาก token
- `web` — สถานะปุ่มเวลา, ป้ายคะแนน, ป้าย "(เช้าวันที่ n)", 409 `DUPLICATE_BOOKING` ไม่บอกว่าร้านเต็ม, ไฟล์ .ics
- `e2e` (18 เทสต์) — ค้นหา + ปุ่มเวลา, Bayesian sort, ร้านเต็มทั้งรอบ, ป้ายข้ามเที่ยงคืน, login ผ่าน Keycloak,
  จอง → หน้ายืนยัน → การจองของฉัน → ยกเลิก, จองซ้อนตัวเอง (409), ปุ่มเวลาที่เต็มกดไม่ได้,
  บอร์ดเจ้าของร้าน, ลดที่นั่งต่ำกว่าที่จอง (409), ร้านของตัวเองรีวิวไม่ได้ — หน้าค้นหารันซ้ำที่มือถือ 375px

CI (`.gitlab-ci.yml`): lint + test + build ทุก push, build และ push image `api`/`web` ขึ้น GitLab Container Registry เมื่อเข้า `main`
(job `api:test` ใช้ Docker-in-Docker — runner ต้องเปิด privileged) · E2E รันบนเครื่องด้วย `./dev.sh e2e` (ยังไม่อยู่ใน CI)

---

## 7. งานต่อยอด (ยังไม่ทำ)

- **แจ้งเตือนนอกเว็บ (อีเมล/LINE):** ต่อยอดจากตาราง `notifications` เดิมด้วย outbox pattern — worker อ่านแถวที่ยังไม่ส่งแล้วส่งต่อ ไม่ต้องเปลี่ยน schema
- **เตือนก่อนถึงเวลาจอง:** ต้องมี scheduler (เช่น cron ในตัว Go หรือ job แยก) มาสแกน booking ที่ใกล้ถึงเวลาแล้วสร้างแจ้งเตือน — ตอนนี้ notification เกิดตอนมีเหตุการณ์เท่านั้น (จอง/แก้/ยกเลิก) ไม่มีอะไรมาจุดชนวนล่วงหน้า
- **WebSocket แทน polling:** กระดิ่งตอนนี้ poll ทุก 60 วินาทีเพราะง่าย อธิบายได้ในประโยคเดียว และช้ากว่ากันไม่เกิน 60 วินาทีก็พอสำหรับการแจ้งยกเลิกล่วงหน้า

---

## 8. พัฒนานอก Docker (ไม่บังคับ)

```bash
docker compose up -d postgres keycloak caddy migrate seed
cd api && POSTGRES_DSN="postgres://jongyoung:jongyoung-dev-password@127.0.0.1:5433/jongyoung?sslmode=disable" go run ./cmd/api
cd web && npm install && npm run dev
```
Node/Go บางเครื่อง (เช่น WSL) resolve `*.localhost` เองไม่ได้ ให้เพิ่ม `/etc/hosts` ตามข้อ 1 ก่อน
และถ้ารัน web นอก compose ต้องแก้ Caddyfile ให้ `jongyoung.localhost` ชี้ `host.docker.internal:3000`
