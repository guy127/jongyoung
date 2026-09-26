package restaurant

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToInputClosedWeekdays(t *testing.T) {
	req := RestaurantRequest{OpenTime: "11:00", CloseTime: "22:00"}

	req.ClosedWeekdays = []int{1, 6}
	in, err := req.ToInput()
	require.NoError(t, err)
	assert.Equal(t, 0b1000010, in.ClosedWeekdays, "จันทร์ + เสาร์")

	req.ClosedWeekdays = nil
	in, err = req.ToInput()
	require.NoError(t, err)
	assert.Zero(t, in.ClosedWeekdays, "ไม่ส่งมา = เปิดทุกวัน")

	req.ClosedWeekdays = []int{0, 1, 2, 3, 4, 5, 6}
	_, err = req.ToInput()
	assert.ErrorIs(t, err, errClosedAllWeek)
}

func TestToInputMapURL(t *testing.T) {
	req := RestaurantRequest{OpenTime: "11:00", CloseTime: "22:00"}

	for _, ok := range []string{
		"https://maps.app.goo.gl/AbCdEf123", // ลิงก์แชร์จากแอป Google Maps
		"https://www.google.com/maps/place/Siam+Paragon/@13.74,100.53,17z",
		"https://www.google.co.th/maps/search/?api=1&query=สยามพารากอน",
		"https://maps.google.com/?q=13.74,100.53",
		"https://goo.gl/maps/AbCdEf123",
		"  https://maps.app.goo.gl/AbCdEf123  ",
	} {
		req.MapURL = ok
		in, err := req.ToInput()
		require.NoError(t, err, ok)
		assert.Equal(t, strings.TrimSpace(ok), in.MapURL)
	}

	req.MapURL = ""
	in, err := req.ToInput()
	require.NoError(t, err)
	assert.Empty(t, in.MapURL, "ไม่กรอก = ใช้การค้นหาจากที่อยู่แทน")

	for _, bad := range []string{
		"javascript:alert(1)",
		"http://maps.app.goo.gl/AbCdEf123", // ต้องเป็น https
		"https://evil.example/maps",
		"https://www.google.com/search?q=ร้าน",   // google แต่ไม่ใช่ /maps
		"https://goo.gl/AbCdEf123",               // goo.gl ทั่วไปไม่ใช่แผนที่
		"https://maps.app.goo.gl.evil.example/x", // โดเมนหลอก
	} {
		req.MapURL = bad
		_, err := req.ToInput()
		assert.ErrorIs(t, err, errInvalidMapURL, bad)
	}
}
