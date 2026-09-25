package booking

import "time"

// bkk สร้างเวลาไทยแบบย่อ ใช้ในเทสต์เท่านั้น
func bkk(y int, m time.Month, d, h, min int) time.Time {
	return time.Date(y, m, d, h, min, 0, 0, Bangkok)
}
