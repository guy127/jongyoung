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

// checkSeats: existing = booking ที่ทับกับ [start,end) โดยไม่รวมตัวที่กำลังแก้ (excludeID)
func checkSeats(existing []Booking, seats, partySize int, start, end time.Time) error {
	peak, at := maxConcurrent(existing, start, end)
	if peak+partySize > seats {
		return &NotEnoughSeatsError{At: at, Available: seats - peak}
	}
	return nil
}
