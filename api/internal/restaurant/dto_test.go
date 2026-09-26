package restaurant

import (
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
