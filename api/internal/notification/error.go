package notification

import "errors"

// ErrNotFound: ไม่มี หรือเป็นของคนอื่น (ตอบ 404 เหมือนกัน ไม่บอกว่ามีอยู่จริง)
var ErrNotFound = errors.New("notification not found")
