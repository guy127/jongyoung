package restaurant

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"jongyoung/internal/booking"
	"jongyoung/internal/httputil"
	"jongyoung/internal/reqctx"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Service interface {
	Create(ctx context.Context, ownerID uuid.UUID, in Input, imageURLs []string) (Restaurant, error)
	Get(ctx context.Context, id uuid.UUID) (Restaurant, error)
	Update(ctx context.Context, userID, id uuid.UUID, in Input) (Restaurant, error)
	Delete(ctx context.Context, userID, id uuid.UUID) error
	AddImage(ctx context.Context, userID, id uuid.UUID, url string) (Image, error)
	DeleteImage(ctx context.Context, userID, id, imageID uuid.UUID) error
	List(ctx context.Context, q ListQuery) ([]ListItem, int64, error)
	Availability(ctx context.Context, id uuid.UUID, date time.Time) (Restaurant, []booking.Slot, error)
	NextAvailable(ctx context.Context, id uuid.UUID, date time.Time, minute, party int) (*NextAvailable, error)
}

type handler struct {
	service Service
}

func NewHandler(service Service) *handler {
	return &handler{service: service}
}

// List godoc
//
//	@Summary	ค้นหาร้าน (+ ปุ่มเวลาว่าง 5 ช่วงเมื่อส่ง date, time, party_size มาครบ)
//	@Tags		restaurants
//	@Produce	json
//	@Param		q			query		string	false	"ชื่อร้าน/เมนู"
//	@Param		cuisine		query		string	false	"ประเภทอาหาร"
//	@Param		sort		query		string	false	"rating | reviews | newest"
//	@Param		page		query		int		false	"หน้า"
//	@Param		limit		query		int		false	"จำนวนต่อหน้า (สูงสุด 50)"
//	@Param		date		query		string	false	"วันทำการ YYYY-MM-DD"
//	@Param		time		query		string	false	"เวลาประมาณ HH:MM"
//	@Param		party_size	query		int		false	"จำนวนคน"
//	@Success	200			{object}	ListResponse
//	@Failure	400			{object}	httputil.ErrorResponse
//	@Router		/restaurants [get]
func (h *handler) List(c *gin.Context) {
	page := httputil.ParsePage(c)
	sort := c.Query("sort")
	if sort != "" && sort != "rating" && sort != "reviews" && sort != "newest" {
		httputil.BadRequest(c, "sort ต้องเป็น rating, reviews หรือ newest")
		return
	}
	q := ListQuery{Q: c.Query("q"), Cuisine: c.Query("cuisine"), Sort: sort, Limit: page.Limit, Offset: page.Offset()}

	if c.Query("date") != "" || c.Query("time") != "" {
		date, err := booking.ParseDate(c.Query("date"))
		if err != nil {
			httputil.BadRequest(c, "date ต้องเป็นรูปแบบ YYYY-MM-DD")
			return
		}
		minute, err := ParseClock(c.Query("time"))
		if err != nil {
			httputil.BadRequest(c, "time ต้องเป็นรูปแบบ HH:MM และลง :00 หรือ :30")
			return
		}
		q.Date, q.Minute, q.Party = &date, minute, queryInt(c, "party_size", 2)
	}

	items, total, err := h.service.List(c.Request.Context(), q)
	if err != nil {
		httputil.Internal(c, err)
		return
	}
	resp := ListResponse{Items: make([]ListItemResponse, len(items)), Page: page.Page, Limit: page.Limit, Total: total}
	for i, it := range items {
		resp.Items[i] = ListItemResponse{RestaurantResponse: NewRestaurantResponse(it.Restaurant)}
		if it.Slots != nil {
			resp.Items[i].Slots = newCardSlots(it.Slots)
		}
	}
	c.JSON(http.StatusOK, resp)
}

// Get godoc
//
//	@Summary	รายละเอียดร้าน
//	@Tags		restaurants
//	@Produce	json
//	@Param		id	path		string	true	"restaurant id"
//	@Success	200	{object}	RestaurantResponse
//	@Failure	404	{object}	httputil.ErrorResponse
//	@Router		/restaurants/{id} [get]
func (h *handler) Get(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	rest, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, NewRestaurantResponse(rest))
}

// Create godoc
//
//	@Summary	สร้างร้าน (ผู้สร้างเป็นเจ้าของ)
//	@Tags		restaurants
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		request	body		RestaurantRequest	true	"ข้อมูลร้าน (ต้องมี image_urls อย่างน้อย 1)"
//	@Success	201		{object}	RestaurantResponse
//	@Failure	400		{object}	httputil.ErrorResponse
//	@Failure	401		{object}	httputil.ErrorResponse
//	@Router		/restaurants [post]
func (h *handler) Create(c *gin.Context) {
	userID, in, req, ok := h.bindInput(c)
	if !ok {
		return
	}
	rest, err := h.service.Create(c.Request.Context(), userID, in, req.ImageURLs)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, NewRestaurantResponse(rest))
}

// Update godoc
//
//	@Summary		แก้ไขร้าน (เฉพาะเจ้าของ)
//	@Description	ลดที่นั่งต่ำกว่าคนสูงสุดของการจองที่จะถึง หรือย่นเวลาจนการจองตกนอกเวลา → 409
//	@Tags			restaurants
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string				true	"restaurant id"
//	@Param			request	body		RestaurantRequest	true	"ข้อมูลร้าน (ไม่ใช้ image_urls)"
//	@Success		200		{object}	RestaurantResponse
//	@Failure		403		{object}	httputil.ErrorResponse
//	@Failure		409		{object}	httputil.ErrorResponse
//	@Router			/restaurants/{id} [put]
func (h *handler) Update(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	userID, in, _, ok := h.bindInput(c)
	if !ok {
		return
	}
	rest, err := h.service.Update(c.Request.Context(), userID, id, in)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, NewRestaurantResponse(rest))
}

// Delete godoc
//
//	@Summary	ลบร้าน (soft delete) และยกเลิกการจองที่ยังไม่เริ่มทั้งหมด
//	@Tags		restaurants
//	@Security	BearerAuth
//	@Param		id	path	string	true	"restaurant id"
//	@Success	204
//	@Failure	403	{object}	httputil.ErrorResponse
//	@Router		/restaurants/{id} [delete]
func (h *handler) Delete(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	userID, _ := reqctx.UserID(c.Request.Context())
	if err := h.service.Delete(c.Request.Context(), userID, id); err != nil {
		h.fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// AddImage godoc
//
//	@Summary	เพิ่มรูปร้าน (URL)
//	@Tags		restaurants
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string			true	"restaurant id"
//	@Param		request	body		ImageRequest	true	"URL รูป"
//	@Success	201		{object}	ImageResponse
//	@Router		/restaurants/{id}/images [post]
func (h *handler) AddImage(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var req ImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.BadRequest(c, "url ไม่ถูกต้อง")
		return
	}
	userID, _ := reqctx.UserID(c.Request.Context())
	img, err := h.service.AddImage(c.Request.Context(), userID, id, req.URL)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, ImageResponse{ID: img.ID, URL: img.URL, SortOrder: img.SortOrder})
}

// DeleteImage godoc
//
//	@Summary	ลบรูปร้าน (ห้ามลบรูปสุดท้าย)
//	@Tags		restaurants
//	@Security	BearerAuth
//	@Param		id		path	string	true	"restaurant id"
//	@Param		imageId	path	string	true	"image id"
//	@Success	204
//	@Failure	400	{object}	httputil.ErrorResponse
//	@Router		/restaurants/{id}/images/{imageId} [delete]
func (h *handler) DeleteImage(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	imageID, ok := pathID(c, "imageId")
	if !ok {
		return
	}
	userID, _ := reqctx.UserID(c.Request.Context())
	if err := h.service.DeleteImage(c.Request.Context(), userID, id, imageID); err != nil {
		h.fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Availability godoc
//
//	@Summary	ที่ว่างทุกช่วง 30 นาทีของรอบวันทำการ
//	@Tags		restaurants
//	@Produce	json
//	@Param		id		path		string	true	"restaurant id"
//	@Param		date	query		string	true	"วันทำการ YYYY-MM-DD (รอบที่เริ่มเปิดในวันนั้น)"
//	@Success	200		{object}	AvailabilityResponse
//	@Router		/restaurants/{id}/availability [get]
func (h *handler) Availability(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	date, err := booking.ParseDate(c.Query("date"))
	if err != nil {
		httputil.BadRequest(c, "date ต้องเป็นรูปแบบ YYYY-MM-DD")
		return
	}
	rest, slots, err := h.service.Availability(c.Request.Context(), id, date)
	if err != nil {
		h.fail(c, err)
		return
	}
	opensAt, closesAt := rest.Hours().Window(date)
	c.JSON(http.StatusOK, AvailabilityResponse{
		BusinessDate: date.Format("2006-01-02"), OpensAt: opensAt, ClosesAt: closesAt,
		Seats: rest.Seats, Slots: newSlots(slots),
	})
}

// NextAvailable godoc
//
//	@Summary	วันทำการถัดไป (ไม่เกิน 14 วัน) ที่มีช่วงรอบเวลาที่ค้นว่างพอ — ไม่เจอคืน null
//	@Tags		restaurants
//	@Produce	json
//	@Param		id			path		string	true	"restaurant id"
//	@Param		date		query		string	true	"วันทำการที่ค้น YYYY-MM-DD"
//	@Param		time		query		string	true	"HH:MM"
//	@Param		party_size	query		int		true	"จำนวนคน"
//	@Success	200			{object}	NextAvailableResponse
//	@Router		/restaurants/{id}/next-available [get]
func (h *handler) NextAvailable(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	date, err := booking.ParseDate(c.Query("date"))
	if err != nil {
		httputil.BadRequest(c, "date ต้องเป็นรูปแบบ YYYY-MM-DD")
		return
	}
	minute, err := ParseClock(c.Query("time"))
	if err != nil {
		httputil.BadRequest(c, "time ต้องเป็นรูปแบบ HH:MM และลง :00 หรือ :30")
		return
	}
	next, err := h.service.NextAvailable(c.Request.Context(), id, date, minute, queryInt(c, "party_size", 2))
	if err != nil {
		h.fail(c, err)
		return
	}
	if next == nil {
		c.JSON(http.StatusOK, nil)
		return
	}
	c.JSON(http.StatusOK, NextAvailableResponse{BusinessDate: next.Date.Format("2006-01-02"), Slots: newCardSlots(next.Slots)})
}

// fail แปลง error ของ domain เป็น HTTP status + code ตามตารางใน CLAUDE.md ข้อ 6
func (h *handler) fail(c *gin.Context, err error) {
	var seats *booking.SeatsBelowBookingsError
	var hours *booking.HoursConflictError
	switch {
	case errors.Is(err, ErrNotFound):
		httputil.NotFound(c, "ไม่พบร้าน")
	case errors.Is(err, ErrImageNotFound):
		httputil.NotFound(c, "ไม่พบรูป")
	case errors.Is(err, ErrNotOwner):
		httputil.Forbidden(c, "NOT_OWNER", "เฉพาะเจ้าของร้านเท่านั้น")
	case errors.Is(err, ErrImageRequired):
		httputil.Abort(c, http.StatusBadRequest, "IMAGE_REQUIRED", "ร้านต้องมีรูปอย่างน้อย 1 รูป", nil)
	case errors.As(err, &seats):
		httputil.Abort(c, http.StatusConflict, "SEATS_BELOW_EXISTING_BOOKINGS",
			"ลดที่นั่งไม่ได้ มีการจองที่ใช้ที่นั่งมากกว่านี้",
			gin.H{"at": seats.At.In(booking.Bangkok), "peak": seats.Peak})
	case errors.As(err, &hours):
		httputil.Abort(c, http.StatusConflict, "HOURS_CONFLICT_EXISTING_BOOKINGS",
			"เปลี่ยนเวลาไม่ได้ มีการจองที่อยู่นอกเวลาใหม่", gin.H{"booking_ids": hours.BookingIDs})
	default:
		httputil.Internal(c, err)
	}
}

func (h *handler) bindInput(c *gin.Context) (uuid.UUID, Input, RestaurantRequest, bool) {
	var req RestaurantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.BadRequest(c, "ข้อมูลร้านไม่ครบหรือไม่ถูกต้อง")
		return uuid.Nil, Input{}, req, false
	}
	in, err := req.ToInput()
	if err != nil {
		httputil.Abort(c, http.StatusBadRequest, "INVALID_OPENING_HOURS", err.Error(), nil)
		return uuid.Nil, Input{}, req, false
	}
	userID, ok := reqctx.UserID(c.Request.Context())
	if !ok {
		httputil.Unauthorized(c, "ต้องเข้าสู่ระบบ")
		return uuid.Nil, Input{}, req, false
	}
	return userID, in, req, true
}

func pathID(c *gin.Context, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		httputil.NotFound(c, "ไม่พบข้อมูล")
		return uuid.Nil, false
	}
	return id, true
}

func queryInt(c *gin.Context, name string, fallback int) int {
	var v int
	if _, err := fmt.Sscan(c.Query(name), &v); err != nil || v < 1 {
		return fallback
	}
	return v
}
