package booking

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBangkokOffsetIsPlus7(t *testing.T) {
	_, offset := bkk(2026, 10, 10, 12, 0).Zone()
	assert.Equal(t, 7*60*60, offset)
}
