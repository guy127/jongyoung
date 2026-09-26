package booking

import (
	"context"
	"errors"
	"net/http"
	"time"

	"jongyoung/internal/httputil"
	"jongyoung/internal/reqctx"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Service interface {
	Create(ctx context.Context, userID, restaurantID uuid.UUID, c Choice) (Booking, error)
	Update(ctx context.Context, userID, bookingID uuid.UUID, c Choice) (Booking, error)
	Cancel(ctx context.Context, userID, bookingID uuid.UUID) error
	Get(ctx context.Context, userID, bookingID uuid.UUID) (View, error)
	ListMine(ctx context.Context, userID uuid.UUID, status string) ([]View, error)
	Board(ctx context.Context, userID, restaurantID uuid.UUID, date time.Time) (Board, error)
}

type handler struct {
	service Service
	now     func() time.Time
}

func NewHandler(service Service, now func() time.Time) *handler {
	return &handler{service: service, now: now}
}

// Create godoc
//
//	@Summary		จองโต๊ะ
//	@ID				createBooking
//	@Description	409 NOT_ENOUGH_SEATS = ที่นั่งไม่พอ (details.available, details.at); 409 DUPLICATE_BOOKING = มีการจองที่ทับช่วงนี้อยู่แล้ว (details.booking_id)
//	@Tags			bookings
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		BookingRequest	true	"วันทำการ + เวลา + จำนวนคน"
//	@Success		201		{object}	BookingResponse
//	@Failure		400		{object}	httputil.ErrorResponse
//	@Failure		404		{object}	httputil.ErrorResponse
//	@Failure		409		{object}	httputil.ErrorResponse
//	@Router			/bookings [post]
func (h *handler) Create(c *gin.Context) {
	var req BookingRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.RestaurantID == uuid.Nil {
		httputil.BadRequest(c, "ข้อมูลการจองไม่ครบ")
		return
	}
	choice, err := req.ToChoice()
	if err != nil {
		httputil.BadRequest(c, err.Error())
		return
	}
	userID, _ := reqctx.UserID(c.Request.Context())
	b, err := h.service.Create(c.Request.Context(), userID, req.RestaurantID, choice)
	if err != nil {
		h.fail(c, err)
		return
	}
	h.respondView(c, http.StatusCreated, userID, b.ID)
}

// Update godoc
//
//	@Summary	แก้ไขการจอง (จำนวนคน/วัน/เวลา) — ต้องยังไม่เลยเส้นตายของเวลาเดิม
//	@ID			updateBooking
//	@Tags		bookings
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string			true	"booking id"
//	@Param		request	body		BookingRequest	true	"ไม่ต้องส่ง restaurant_id"
//	@Success	200		{object}	BookingResponse
//	@Failure	403		{object}	httputil.ErrorResponse
//	@Failure	409		{object}	httputil.ErrorResponse
//	@Router		/bookings/{id} [put]
func (h *handler) Update(c *gin.Context) {
	id, ok := httputil.PathID(c, "id")
	if !ok {
		return
	}
	var req BookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.BadRequest(c, "ข้อมูลการจองไม่ครบ")
		return
	}
	choice, err := req.ToChoice()
	if err != nil {
		httputil.BadRequest(c, err.Error())
		return
	}
	userID, _ := reqctx.UserID(c.Request.Context())
	if _, err := h.service.Update(c.Request.Context(), userID, id, choice); err != nil {
		h.fail(c, err)
		return
	}
	h.respondView(c, http.StatusOK, userID, id)
}

// Cancel godoc
//
//	@Summary	ยกเลิกการจอง (เปลี่ยนสถานะเป็น cancelled ไม่ลบจริง)
//	@ID			cancelBooking
//	@Tags		bookings
//	@Security	BearerAuth
//	@Param		id	path	string	true	"booking id"
//	@Success	204
//	@Failure	403	{object}	httputil.ErrorResponse	"CANCEL_WINDOW_PASSED (details.cancel_until) หรือไม่ใช่การจองของตัวเอง"
//	@Failure	409	{object}	httputil.ErrorResponse
//	@Router		/bookings/{id} [delete]
func (h *handler) Cancel(c *gin.Context) {
	id, ok := httputil.PathID(c, "id")
	if !ok {
		return
	}
	userID, _ := reqctx.UserID(c.Request.Context())
	if err := h.service.Cancel(c.Request.Context(), userID, id); err != nil {
		h.fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Get godoc
//
//	@Summary	รายละเอียดการจอง (หน้ายืนยัน) — เจ้าของการจองหรือเจ้าของร้านเท่านั้น
//	@ID			getBooking
//	@Tags		bookings
//	@Security	BearerAuth
//	@Produce	json
//	@Param		id	path		string	true	"booking id"
//	@Success	200	{object}	BookingResponse
//	@Failure	403	{object}	httputil.ErrorResponse
//	@Failure	404	{object}	httputil.ErrorResponse
//	@Router		/bookings/{id} [get]
func (h *handler) Get(c *gin.Context) {
	id, ok := httputil.PathID(c, "id")
	if !ok {
		return
	}
	userID, _ := reqctx.UserID(c.Request.Context())
	h.respondView(c, http.StatusOK, userID, id)
}

// ListMine godoc
//
//	@Summary	การจองของฉัน
//	@ID			listMyBookings
//	@Tags		me
//	@Security	BearerAuth
//	@Produce	json
//	@Param		status	query	string	false	"upcoming (ค่าเริ่มต้น) | past | cancelled"
//	@Success	200		{array}	BookingResponse
//	@Router		/me/bookings [get]
func (h *handler) ListMine(c *gin.Context) {
	status := c.DefaultQuery("status", "upcoming")
	if status != "upcoming" && status != "past" && status != "cancelled" {
		httputil.BadRequest(c, "status ต้องเป็น upcoming, past หรือ cancelled")
		return
	}
	userID, _ := reqctx.UserID(c.Request.Context())
	views, err := h.service.ListMine(c.Request.Context(), userID, status)
	if err != nil {
		httputil.Internal(c, err)
		return
	}
	now := h.now()
	out := make([]BookingResponse, len(views))
	for i, v := range views {
		out[i] = NewBookingResponse(v, now)
		out[i].CustomerName = ""
	}
	c.JSON(http.StatusOK, out)
}

// Board godoc
//
//	@Summary	บอร์ดการจองรายวันทำการของร้าน (เจ้าของร้านเท่านั้น) + คนในร้านต่อช่วง 30 นาที
//	@ID			getBookingBoard
//	@Tags		restaurants
//	@Security	BearerAuth
//	@Produce	json
//	@Param		id		path		string	true	"restaurant id"
//	@Param		date	query		string	true	"วันทำการ YYYY-MM-DD — แขกตี 1 คืนวันเสาร์อยู่ในวันเสาร์"
//	@Success	200		{object}	BoardResponse
//	@Failure	403		{object}	httputil.ErrorResponse
//	@Router		/restaurants/{id}/bookings [get]
func (h *handler) Board(c *gin.Context) {
	id, ok := httputil.PathID(c, "id")
	if !ok {
		return
	}
	date, err := ParseDate(c.Query("date"))
	if err != nil {
		httputil.BadRequest(c, "date ต้องเป็นรูปแบบ YYYY-MM-DD")
		return
	}
	userID, _ := reqctx.UserID(c.Request.Context())
	board, err := h.service.Board(c.Request.Context(), userID, id, date)
	if err != nil {
		h.fail(c, err)
		return
	}
	now := h.now()
	resp := BoardResponse{
		BusinessDate: date.Format("2006-01-02"), Closed: board.Closed, OpensAt: board.OpensAt, ClosesAt: board.ClosesAt,
		Seats: board.Restaurant.Seats, Slots: make([]BoardSlot, len(board.Slots)), Bookings: make([]BookingResponse, len(board.Bookings)),
	}
	for i, s := range board.Slots {
		resp.Slots[i] = BoardSlot{StartAt: s.StartAt, EndAt: s.EndAt, Booked: board.Restaurant.Seats - s.Available}
	}
	for i, v := range board.Bookings {
		resp.Bookings[i] = NewBookingResponse(v, now)
	}
	c.JSON(http.StatusOK, resp)
}

func (h *handler) respondView(c *gin.Context, status int, userID, id uuid.UUID) {
	v, err := h.service.Get(c.Request.Context(), userID, id)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(status, NewBookingResponse(v, h.now()))
}

// fail แปลง error เป็น HTTP status + code — code เป็นสิ่งที่หน้าเว็บใช้ตัดสินใจ (ตาราง CLAUDE.md ข้อ 6)
func (h *handler) fail(c *gin.Context, err error) {
	var rule *RuleError
	var seats *NotEnoughSeatsError
	var dup *DuplicateBookingError
	var window *CancelWindowError
	var closed *ClosedError
	switch {
	case errors.As(err, &rule):
		details := any(nil)
		if rule == ErrTooLateToBook {
			details = gin.H{"earliest_start_at": h.now().Add(LeadTime).In(Bangkok)}
		}
		httputil.Abort(c, http.StatusBadRequest, rule.Code, rule.Message, details)
	case errors.As(err, &seats):
		httputil.Abort(c, http.StatusConflict, "NOT_ENOUGH_SEATS", "ที่นั่งไม่พอในช่วงที่เลือก",
			gin.H{"available": seats.Available, "at": seats.At.In(Bangkok)})
	case errors.As(err, &dup):
		httputil.Abort(c, http.StatusConflict, "DUPLICATE_BOOKING", "คุณมีการจองที่ทับช่วงนี้อยู่แล้ว",
			gin.H{"booking_id": dup.BookingID})
	case errors.As(err, &window):
		httputil.Abort(c, http.StatusForbidden, "CANCEL_WINDOW_PASSED", "เลยเวลาที่แก้ไข/ยกเลิกได้แล้ว",
			gin.H{"cancel_until": window.Until.In(Bangkok)})
	case errors.As(err, &closed):
		httputil.Abort(c, http.StatusBadRequest, "RESTAURANT_CLOSED", "ร้านปิดในช่วงที่เลือก",
			gin.H{"reason": closed.Closure.Reason, "start_at": closed.Closure.StartAt.In(Bangkok), "end_at": closed.Closure.EndAt.In(Bangkok)})
	case errors.Is(err, ErrRestaurantNotFound):
		httputil.NotFound(c, "ไม่พบร้าน")
	case errors.Is(err, ErrBookingNotFound):
		httputil.NotFound(c, "ไม่พบการจอง")
	case errors.Is(err, ErrForbidden):
		httputil.Forbidden(c, "NOT_BOOKING_OWNER", "ไม่ใช่การจองของคุณ")
	case errors.Is(err, ErrNotRestaurantOwner):
		httputil.Forbidden(c, "NOT_OWNER", "เฉพาะเจ้าของร้านเท่านั้น")
	case errors.Is(err, ErrBookingCancelled):
		httputil.Abort(c, http.StatusConflict, "BOOKING_CANCELLED", "การจองนี้ถูกยกเลิกไปแล้ว", nil)
	case errors.Is(err, ErrAlreadyStarted):
		httputil.Abort(c, http.StatusConflict, "BOOKING_ALREADY_STARTED", "การจองนี้เริ่มหรือผ่านไปแล้ว", nil)
	default:
		httputil.Internal(c, err)
	}
}
