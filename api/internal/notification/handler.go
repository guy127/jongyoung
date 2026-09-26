package notification

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
	List(ctx context.Context, userID uuid.UUID) ([]Notification, int64, error)
	MarkRead(ctx context.Context, userID, id uuid.UUID) error
	MarkAllRead(ctx context.Context, userID uuid.UUID) error
}

type handler struct {
	service Service
}

func NewHandler(service Service) *handler {
	return &handler{service: service}
}

// ListMine godoc
//
//	@Summary	แจ้งเตือนล่าสุดของฉัน + จำนวนที่ยังไม่อ่าน
//	@ID			listMyNotifications
//	@Tags		me
//	@Security	BearerAuth
//	@Produce	json
//	@Success	200	{object}	ListResponse
//	@Failure	401	{object}	httputil.ErrorResponse
//	@Router		/me/notifications [get]
func (h *handler) ListMine(c *gin.Context) {
	userID, ok := reqctx.UserID(c.Request.Context())
	if !ok {
		httputil.Unauthorized(c, "ต้องเข้าสู่ระบบ")
		return
	}
	list, unread, err := h.service.List(c.Request.Context(), userID)
	if err != nil {
		httputil.Internal(c, err)
		return
	}
	resp := ListResponse{Items: make([]NotificationResponse, len(list)), UnreadCount: unread}
	for i, n := range list {
		resp.Items[i] = NewNotificationResponse(n)
	}
	c.JSON(http.StatusOK, resp)
}

// MarkRead godoc
//
//	@Summary	กดอ่านแจ้งเตือนหนึ่งรายการ (ของคนอื่น → 404)
//	@ID			markNotificationRead
//	@Tags		me
//	@Security	BearerAuth
//	@Param		id	path	string	true	"notification id"
//	@Success	204
//	@Failure	404	{object}	httputil.ErrorResponse
//	@Router		/me/notifications/{id}/read [put]
func (h *handler) MarkRead(c *gin.Context) {
	id, ok := httputil.PathID(c, "id")
	if !ok {
		return
	}
	userID, ok := reqctx.UserID(c.Request.Context())
	if !ok {
		httputil.Unauthorized(c, "ต้องเข้าสู่ระบบ")
		return
	}
	err := h.service.MarkRead(c.Request.Context(), userID, id)
	if errors.Is(err, ErrNotFound) {
		httputil.NotFound(c, "ไม่พบการแจ้งเตือน")
		return
	}
	if err != nil {
		httputil.Internal(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// MarkAllRead godoc
//
//	@Summary	กดอ่านแจ้งเตือนทั้งหมดของฉัน
//	@ID			markAllNotificationsRead
//	@Tags		me
//	@Security	BearerAuth
//	@Success	204
//	@Router		/me/notifications/read-all [put]
func (h *handler) MarkAllRead(c *gin.Context) {
	userID, ok := reqctx.UserID(c.Request.Context())
	if !ok {
		httputil.Unauthorized(c, "ต้องเข้าสู่ระบบ")
		return
	}
	if err := h.service.MarkAllRead(c.Request.Context(), userID); err != nil {
		httputil.Internal(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
