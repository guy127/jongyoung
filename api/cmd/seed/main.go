// cmd/seed ใส่ข้อมูลตัวอย่างให้เปิดเว็บมาแล้วเห็นร้าน การจอง และรีวิวทันที (CLAUDE.md ข้อ 4.1)
//
//	go run ./cmd/seed          # ใส่เฉพาะเมื่อยังไม่มีร้านเลย (รันซ้ำได้ ไม่ทับข้อมูลที่มีอยู่)
//	go run ./cmd/seed --reset  # ล้างร้าน/การจอง/รีวิวทั้งหมดแล้วใส่ใหม่
//
// เวลาการจองคิดจาก "วันนี้" ตอนรัน เพื่อให้มีทั้งการจองที่กำลังจะถึงและที่ผ่านไปแล้วเสมอ
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"jongyoung/internal/booking"
	"jongyoung/internal/config"
	"jongyoung/internal/platform/database"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// sub ของบัญชีทดสอบใน keycloak/import/jongyoung-realm.json (กำหนดตายตัว)
const (
	owner1Sub    = "0b8e6f0e-2d4a-4a57-9a61-1f0f3c1a0001"
	owner2Sub    = "0b8e6f0e-2d4a-4a57-9a61-1f0f3c1a0002"
	customer1Sub = "0b8e6f0e-2d4a-4a57-9a61-1f0f3c1a0003"
)

func main() {
	reset := flag.Bool("reset", false, "ล้างข้อมูลร้าน/การจอง/รีวิวเดิมก่อนใส่ใหม่")
	flag.Parse()
	if err := run(*reset); err != nil {
		slog.Error("seed ล้มเหลว", "error", err)
		os.Exit(1)
	}
}

func run(reset bool) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx := context.Background()
	db, err := database.Open(ctx, cfg.Database.DSN, false)
	if err != nil {
		return err
	}

	var count int64
	if err := db.Table("restaurants").Count(&count).Error; err != nil {
		return err
	}
	if count > 0 && !reset {
		slog.Info("มีข้อมูลร้านอยู่แล้ว ข้ามการ seed (ใช้ --reset เพื่อล้างแล้วใส่ใหม่)", "restaurants", count)
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if reset {
			if err := tx.Exec("TRUNCATE reviews, bookings, restaurant_images, restaurants").Error; err != nil {
				return err
			}
		}
		return seed(tx, time.Now().In(booking.Bangkok))
	})
}

type seeder struct {
	tx    *gorm.DB
	today time.Time // เที่ยงคืนวันนี้ เวลาไทย
	err   error     // เก็บ error แรกไว้ ทำให้โค้ดด้านล่างอ่านเป็นลำดับขั้นได้โดยไม่ต้องเช็คทุกบรรทัด
}

func seed(tx *gorm.DB, now time.Time) error {
	y, m, d := now.Date()
	s := &seeder{tx: tx, today: time.Date(y, m, d, 0, 0, 0, 0, booking.Bangkok)}

	owner1 := s.user(owner1Sub, "owner1@jongyoung.test", "สมศรี ใจงาม")
	owner2 := s.user(owner2Sub, "owner2@jongyoung.test", "วีระ ทองดี")
	customer1 := s.user(customer1Sub, "customer1@jongyoung.test", "มะลิ วงศ์ดี")
	// ผู้รีวิวตัวอย่าง (ไม่ใช่บัญชี Keycloak จริง — 1 คนรีวิวได้ร้านละ 1 ครั้ง จึงต้องมีหลายคน)
	reviewers := make([]uuid.UUID, 50)
	for i := range reviewers {
		reviewers[i] = s.user(fmt.Sprintf("seed-reviewer-%02d", i+1), fmt.Sprintf("reviewer%02d@example.test", i+1), reviewerNames[i%len(reviewerNames)])
	}

	// 1) เคสปกติ + มีช่วงเกือบเต็มคืนนี้
	tamsang := s.restaurant(owner1, "ครัวป้าแดง ตามสั่ง", "อาหารไทย", "ซอยอารีย์ 3 แขวงพญาไท เขตพญาไท กรุงเทพฯ",
		"ข้าวกะเพราหมูสับไข่ดาว ผัดซีอิ๊ว ต้มยำน้ำข้น รสจัดจ้านแบบบ้าน ๆ เปิดทุกวัน", 10, 10*60, 21*60, 30, 1)
	// 2) เคสปกติ + รีวิวเยอะ ★4.8 จาก 46 รีวิว + มีช่วงพักบ่าย
	sushi := s.restaurant(owner2, "ซูชิ ทาคุมิ", "ญี่ปุ่น", "ชั้น 5 ศูนย์การค้าสยามพารากอน กรุงเทพฯ",
		"โอมากาเสะและซูชิหน้าปลาสดส่งตรงจากตลาดโทโยสุ นั่งเคาน์เตอร์ดูเชฟทำสด", 12, 11*60, 22*60, 60, 2)
	// ร้านนี้มีช่วงพัก 15:00–17:00 (booking ใน seed ของร้านนี้คือ 19:00–20:30 จึงไม่ทับ)
	s.exec(`UPDATE restaurants SET break_start_minute = ?, break_end_minute = ? WHERE id = ?`, 15*60, 17*60, sushi)
	// 3) เปิดเฉพาะเย็น + ปิดทุกวันจันทร์ + รีวิวเดียว ★5.0 (โชว์ว่า Bayesian ไม่ให้แซงร้านรีวิวเยอะ และ UI แสดง "รีวิวน้อย")
	//    ร้านนี้ไม่มีการจองที่ active จึงไม่มีทางมีการจองตกวันปิด ไม่ว่า seed จะรันวันไหน
	buffet := s.restaurant(owner1, "บ้านชาบู บุฟเฟ่ต์", "บุฟเฟ่ต์", "ถนนบรรทัดทอง แขวงวังใหม่ เขตปทุมวัน กรุงเทพฯ",
		"ชาบูน้ำซุปกระดูกหมูเคี่ยว 12 ชั่วโมง เนื้อสไลซ์และผักสดไม่อั้น 90 นาที ปิดทุกวันจันทร์", 30, 17*60, 23*60, 30, 3)
	s.exec(`UPDATE restaurants SET closed_weekdays = ? WHERE id = ?`, booking.WeekdayMask(time.Monday), buffet)
	// 4) ข้ามเที่ยงคืน 18:00–02:00
	seafood := s.restaurant(owner2, "ท่าเรือซีฟู้ดบาร์", "ซีฟู้ด", "ริมแม่น้ำเจ้าพระยา ท่าเรือราชวงศ์ เขตสัมพันธวงศ์ กรุงเทพฯ",
		"กุ้งแม่น้ำเผา หอยแมลงภู่อบ และค็อกเทลริมน้ำ เปิดยันตีสอง", 10, 18*60, 2*60, 30, 4)
	// 5) เปิด 24 ชม. + ยังไม่มีรีวิว (โชว์ว่าร้านไม่มีรีวิวอยู่ท้ายสุดของ "คะแนนสูงสุด")
	jok := s.restaurant(owner1, "โจ๊กสามย่าน 24 ชม.", "อาหารไทย", "ตลาดสามย่าน ถนนพญาไท เขตปทุมวัน กรุงเทพฯ",
		"โจ๊กหมูสับใส่ไข่ ปาท่องโก๋ร้อน ๆ เปิดตลอด 24 ชั่วโมง", 8, 0, 0, 30, 5)
	// 6) เต็มทั้งรอบคืนนี้ (โชว์ empty state "ว่างวันถัดไป")
	baansuan := s.restaurant(owner2, "ครัวบ้านสวน", "อาหารไทย", "ซอยสุขุมวิท 49 แขวงคลองตันเหนือ เขตวัฒนา กรุงเทพฯ",
		"อาหารไทยรสมือแม่ในบ้านไม้ริมสวน แกงเขียวหวานไก่บ้าน ปลากะพงทอดน้ำปลา", 8, 11*60, 22*60, 30, 6)

	// รีวิว
	s.reviews(sushi, reviewers[:46], []int{5, 5, 5, 5, 4}, sushiReviews)       // sum 221 / 46 ≈ 4.8
	s.reviews(buffet, reviewers[:1], []int{5}, buffetReviews)                  // ★5.0 จาก 1
	s.reviews(tamsang, reviewers[10:22], []int{5, 4, 4, 5, 4, 3}, thaiReviews) // ≈ 4.2
	s.reviews(seafood, reviewers[20:27], []int{5, 4, 5, 4, 4}, seaReviews)     // ≈ 4.4
	s.reviews(baansuan, reviewers[28:40], []int{5, 5, 5, 4}, thaiReviews)      // ≈ 4.8 จาก 12
	// jok: ไม่มีรีวิว

	tomorrow := s.today.AddDate(0, 0, 1)

	// การจองของ customer1: กำลังจะถึง 2 รายการ (หนึ่งรายการคร่อมเที่ยงคืน) + ผ่านไปแล้ว 1 + ยกเลิกแล้ว 1
	s.book(seafood, customer1, 3, s.at(tomorrow, 23, 30), s.at(tomorrow.AddDate(0, 0, 1), 0, 30), booking.StatusActive)
	s.book(sushi, customer1, 2, s.at(tomorrow.AddDate(0, 0, 2), 19, 0), s.at(tomorrow.AddDate(0, 0, 2), 20, 30), booking.StatusActive)
	s.book(tamsang, customer1, 4, s.at(s.today.AddDate(0, 0, -3), 12, 0), s.at(s.today.AddDate(0, 0, -3), 13, 0), booking.StatusActive)
	s.book(buffet, customer1, 2, s.at(s.today.AddDate(0, 0, 5), 18, 0), s.at(s.today.AddDate(0, 0, 5), 19, 30), booking.StatusCancelled)

	// ร้านตามสั่ง: พรุ่งนี้ 18:00–19:30 เกือบเต็ม (เหลือ 2 จาก 10) → ปุ่มเวลา "เหลือน้อย"
	s.book(tamsang, reviewers[40], 4, s.at(tomorrow, 18, 0), s.at(tomorrow, 19, 30), booking.StatusActive)
	s.book(tamsang, reviewers[41], 4, s.at(tomorrow, 18, 30), s.at(tomorrow, 19, 30), booking.StatusActive)
	// ครัวบ้านสวน: เต็มทั้งรอบพรุ่งนี้ 11:00–22:00
	s.book(baansuan, reviewers[42], 8, s.at(tomorrow, 11, 0), s.at(tomorrow, 15, 0), booking.StatusActive)
	s.book(baansuan, reviewers[43], 8, s.at(tomorrow, 15, 0), s.at(tomorrow, 19, 0), booking.StatusActive)
	s.book(baansuan, reviewers[44], 8, s.at(tomorrow, 19, 0), s.at(tomorrow, 22, 0), booking.StatusActive)
	// ซีฟู้ด: คืนพรุ่งนี้มีคนหลังเที่ยงคืนด้วย (บอร์ดเจ้าของร้านต้องอยู่ในวันพรุ่งนี้ ไม่ใช่วันถัดไป)
	s.book(seafood, reviewers[45], 4, s.at(tomorrow, 20, 0), s.at(tomorrow, 22, 0), booking.StatusActive)
	s.book(seafood, reviewers[46], 2, s.at(tomorrow.AddDate(0, 0, 1), 1, 0), s.at(tomorrow.AddDate(0, 0, 1), 2, 0), booking.StatusActive)
	// โจ๊ก 24 ชม.: มีคนคร่อมเที่ยงคืน
	s.book(jok, reviewers[47], 3, s.at(tomorrow, 23, 30), s.at(tomorrow.AddDate(0, 0, 1), 0, 30), booking.StatusActive)

	// คะแนนรวมคำนวณจากรีวิวจริง จึงตรงกันเสมอ
	s.exec(`UPDATE restaurants r SET
		rating_sum = COALESCE((SELECT SUM(score) FROM reviews WHERE restaurant_id = r.id), 0),
		rating_count = (SELECT COUNT(*) FROM reviews WHERE restaurant_id = r.id)`)

	if s.err == nil {
		slog.Info("seed เสร็จ", "restaurants", 6)
	}
	return s.err
}

func (s *seeder) exec(sql string, args ...any) {
	if s.err == nil {
		s.err = s.tx.Exec(sql, args...).Error
	}
}

func (s *seeder) scanID(sql string, args ...any) uuid.UUID {
	var id uuid.UUID
	if s.err == nil {
		s.err = s.tx.Raw(sql, args...).Row().Scan(&id)
	}
	return id
}

func (s *seeder) user(sub, email, name string) uuid.UUID {
	return s.scanID(`INSERT INTO users (keycloak_uid, email, display_name) VALUES (?, ?, ?)
		ON CONFLICT (keycloak_uid) WHERE deleted_at IS NULL
		DO UPDATE SET email = EXCLUDED.email, display_name = EXCLUDED.display_name
		RETURNING id`, sub, email, name)
}

// restaurant สร้างร้าน + รูป 3 รูป; ageDays = สร้างไว้กี่วันก่อน (ให้ "ใหม่ล่าสุด" เรียงแตกต่างกัน)
func (s *seeder) restaurant(owner uuid.UUID, name, cuisine, address, desc string, seats, open, shut, cancelBefore, ageDays int) uuid.UUID {
	id := s.scanID(`INSERT INTO restaurants (owner_id, name, cuisine, address, description, seats, open_minute, close_minute, cancel_before_minutes, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, now() - make_interval(days => ?)) RETURNING id`,
		owner, name, cuisine, address, desc, seats, open, shut, cancelBefore, ageDays)
	for i := range 3 {
		s.exec(`INSERT INTO restaurant_images (restaurant_id, url, sort_order) VALUES (?, ?, ?)`,
			id, fmt.Sprintf("https://picsum.photos/seed/jongyoung-%d-%d/960/540", ageDays, i), i)
	}
	return id
}

// reviews ให้ผู้รีวิวแต่ละคนรีวิว 1 ครั้ง วนคะแนนและข้อความตามรูปแบบที่ให้
func (s *seeder) reviews(restaurant uuid.UUID, users []uuid.UUID, scores []int, bodies []string) {
	for i, u := range users {
		s.exec(`INSERT INTO reviews (restaurant_id, user_id, score, body, created_at) VALUES (?, ?, ?, ?, now() - make_interval(days => ?))`,
			restaurant, u, scores[i%len(scores)], bodies[i%len(bodies)], i+1)
	}
}

func (s *seeder) book(restaurant, user uuid.UUID, party int, start, end time.Time, status string) {
	var cancelledAt *time.Time
	if status == booking.StatusCancelled {
		t := s.today.AddDate(0, 0, -1)
		cancelledAt = &t
	}
	s.exec(`INSERT INTO bookings (restaurant_id, user_id, party_size, start_at, end_at, status, cancelled_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		restaurant, user, party, start, end, status, cancelledAt)
}

func (s *seeder) at(day time.Time, hour, minute int) time.Time {
	y, m, d := day.Date()
	return time.Date(y, m, d, hour, minute, 0, 0, booking.Bangkok)
}

var reviewerNames = []string{
	"ปวีณา ร.", "ธนกฤต ส.", "กมลวรรณ พ.", "ณัฐพล ก.", "สุนิสา ม.", "อนุชา ท.", "พิมพ์ชนก ด.", "วรเมธ จ.",
	"ศศิธร บ.", "ภาคิน ล.",
}

var sushiReviews = []string{
	"ปลาสดมาก ข้าวซูชิอุณหภูมิกำลังดี เชฟอธิบายทุกคำ",
	"โอมากาเสะคุ้มราคา อูนิหวานละลายในปาก",
	"นั่งเคาน์เตอร์ได้ดูเชฟทำสด บรรยากาศเงียบสงบ",
	"จองผ่านเว็บได้โต๊ะตรงเวลา ไม่ต้องรอ",
	"อร่อยแต่คิวแน่นช่วงเย็น ควรจองล่วงหน้า",
}

var buffetReviews = []string{"น้ำซุปกระดูกหมูหวานกลมกล่อม เนื้อสไลซ์เติมไว"}

var thaiReviews = []string{
	"รสจัดจ้านถึงใจ กะเพราหอมใบกะเพราจริง",
	"ราคาไม่แพง ได้เยอะ พนักงานใจดี",
	"ต้มยำน้ำข้นเข้มข้น เผ็ดกำลังดี",
	"ร้านเล็กแต่สะอาด อาหารออกเร็ว",
	"ช่วงเที่ยงคนเยอะมาก แนะนำให้จองก่อน",
	"รสมือเหมือนกินข้าวบ้าน อบอุ่น",
}

var seaReviews = []string{
	"กุ้งแม่น้ำตัวใหญ่ มันกุ้งเยิ้ม",
	"นั่งริมน้ำลมเย็นสบาย ค็อกเทลดี",
	"หอยแมลงภู่อบหม้อดินหอมมาก",
	"เปิดดึกดี เลิกงานแล้วยังมาทันได้",
}
