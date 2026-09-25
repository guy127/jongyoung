package booking

import (
	"time"
	_ "time/tzdata" // ฝัง tzdata ในไบนารี — container distroless หา Asia/Bangkok เจอเสมอ
)

// Bangkok คือ timezone ของทุกร้าน (ไม่มี DST)
var Bangkok = mustLoadLocation("Asia/Bangkok")

func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}
