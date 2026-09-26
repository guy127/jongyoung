package review

import (
	"context"
	"errors"
	"net/http"

	"jongyoung/internal/httputil"
	"jongyoung/internal/reqctx"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Service interface {
	Create(ctx context.Context, userID, restaurantID uuid.UUID, score int, body string) (Review, error)
	Update(ctx context.Context, userID, restaurantID uuid.UUID, score int, body string) (Review, error)
	Delete(ctx context.Context, userID, restaurantID uuid.UUID) error
	Mine(ctx context.Context, userID, restaurantID uuid.UUID) (Review, error)
	List(ctx context.Context, restaurantID uuid.UUID, limit, offset int) ([]View, int64, error)
}

type handler struct {
	service Service
}

func NewHandler(service Service) *handler {
	return &handler{service: service}
}

// List godoc
//
//	@Summary	รีวิวของร้าน (ใหม่สุดก่อน)
//	@ID			listReviews
//	@Tags		reviews
//	@Produce	json
//	@Param		id		path		string	true	"restaurant id"
//	@Param		page	query		int		false	"หน้า"
//	@Param		limit	query		int		false	"จำนวนต่อหน้า"
//	@Success	200		{object}	ListResponse
//	@Router		/restaurants/{id}/reviews [get]
func (h *handler) List(c *gin.Context) {
	rid, ok := httputil.PathID(c, "id")
	if !ok {
		return
	}
	page := httputil.ParsePage(c)
	list, total, err := h.service.List(c.Request.Context(), rid, page.Limit, page.Offset())
	if err != nil {
		httputil.Internal(c, err)
		return
	}
	resp := ListResponse{Items: make([]ReviewResponse, len(list)), Page: page.Page, Limit: page.Limit, Total: total}
	for i, v := range list {
		resp.Items[i] = newResponse(v.Review, v.AuthorName)
	}
	c.JSON(http.StatusOK, resp)
}

// Mine godoc
//
//	@Summary	รีวิวของฉันในร้านนี้ (ไม่มี → 404)
//	@ID			getMyReview
//	@Tags		reviews
//	@Security	BearerAuth
//	@Produce	json
//	@Param		id	path		string	true	"restaurant id"
//	@Success	200	{object}	ReviewResponse
//	@Failure	404	{object}	httputil.ErrorResponse
//	@Router		/restaurants/{id}/reviews/mine [get]
func (h *handler) Mine(c *gin.Context) {
	rid, ok := httputil.PathID(c, "id")
	if !ok {
		return
	}
	userID, _ := reqctx.UserID(c.Request.Context())
	rv, err := h.service.Mine(c.Request.Context(), userID, rid)
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, newResponse(rv, ""))
}

// Create godoc
//
//	@Summary	เขียนรีวิว (มีอยู่แล้ว → 409 REVIEW_EXISTS, ร้านตัวเอง → 403 OWN_RESTAURANT)
//	@ID			createReview
//	@Tags		reviews
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string			true	"restaurant id"
//	@Param		request	body		ReviewRequest	true	"คะแนน 1–5 + ข้อความ"
//	@Success	201		{object}	ReviewResponse
//	@Failure	403		{object}	httputil.ErrorResponse
//	@Failure	409		{object}	httputil.ErrorResponse
//	@Router		/restaurants/{id}/reviews [post]
func (h *handler) Create(c *gin.Context) {
	h.write(c, http.StatusCreated, h.service.Create)
}

// Update godoc
//
//	@Summary	แก้รีวิวของตัวเอง (ยังไม่เคยรีวิว → 404)
//	@ID			updateReview
//	@Tags		reviews
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string			true	"restaurant id"
//	@Param		request	body		ReviewRequest	true	"คะแนน 1–5 + ข้อความ"
//	@Success	200		{object}	ReviewResponse
//	@Failure	404		{object}	httputil.ErrorResponse
//	@Router		/restaurants/{id}/reviews [put]
func (h *handler) Update(c *gin.Context) {
	h.write(c, http.StatusOK, h.service.Update)
}

// Delete godoc
//
//	@Summary	ลบรีวิวของตัวเอง
//	@ID			deleteReview
//	@Tags		reviews
//	@Security	BearerAuth
//	@Param		id	path	string	true	"restaurant id"
//	@Success	204
//	@Failure	404	{object}	httputil.ErrorResponse
//	@Router		/restaurants/{id}/reviews [delete]
func (h *handler) Delete(c *gin.Context) {
	rid, ok := httputil.PathID(c, "id")
	if !ok {
		return
	}
	userID, _ := reqctx.UserID(c.Request.Context())
	if err := h.service.Delete(c.Request.Context(), userID, rid); err != nil {
		fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

type writeFunc func(ctx context.Context, userID, restaurantID uuid.UUID, score int, body string) (Review, error)

func (h *handler) write(c *gin.Context, status int, fn writeFunc) {
	rid, ok := httputil.PathID(c, "id")
	if !ok {
		return
	}
	var req ReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.Abort(c, http.StatusBadRequest, "INVALID_REVIEW", "คะแนนต้องเป็นจำนวนเต็ม 1–5", nil)
		return
	}
	userID, _ := reqctx.UserID(c.Request.Context())
	rv, err := fn(c.Request.Context(), userID, rid, req.Score, req.Body)
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(status, newResponse(rv, ""))
}

func fail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrRestaurantNotFound):
		httputil.NotFound(c, "ไม่พบร้าน")
	case errors.Is(err, ErrReviewNotFound):
		httputil.NotFound(c, "ยังไม่ได้รีวิวร้านนี้")
	case errors.Is(err, ErrOwnRestaurant):
		httputil.Forbidden(c, "OWN_RESTAURANT", "เจ้าของร้านรีวิวร้านตัวเองไม่ได้")
	case errors.Is(err, ErrReviewExists):
		httputil.Abort(c, http.StatusConflict, "REVIEW_EXISTS", "คุณรีวิวร้านนี้แล้ว แก้ไขรีวิวเดิมแทนได้", nil)
	default:
		httputil.Internal(c, err)
	}
}
