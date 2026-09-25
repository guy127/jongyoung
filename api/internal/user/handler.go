package user

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
	Me(ctx context.Context, id uuid.UUID) (Me, error)
}

type handler struct {
	service Service
}

func NewHandler(service Service) *handler {
	return &handler{service: service}
}

type MeResponse struct {
	ID          uuid.UUID         `json:"id"`
	Email       string            `json:"email"`
	DisplayName string            `json:"display_name"`
	Restaurants []OwnedRestaurant `json:"restaurants"`
}

// Me godoc
//
//	@Summary	โปรไฟล์ของผู้ใช้ปัจจุบัน + ร้านที่เป็นเจ้าของ
//	@Tags		me
//	@Security	BearerAuth
//	@Produce	json
//	@Success	200	{object}	MeResponse
//	@Failure	401	{object}	httputil.ErrorResponse
//	@Router		/me [get]
func (h *handler) Me(c *gin.Context) {
	userID, ok := reqctx.UserID(c.Request.Context())
	if !ok {
		httputil.Unauthorized(c, "ต้องเข้าสู่ระบบ")
		return
	}
	me, err := h.service.Me(c.Request.Context(), userID)
	if errors.Is(err, ErrUserNotFound) {
		httputil.Unauthorized(c, "ไม่พบผู้ใช้")
		return
	}
	if err != nil {
		httputil.Internal(c, err)
		return
	}
	c.JSON(http.StatusOK, MeResponse{
		ID:          me.User.ID,
		Email:       me.User.Email,
		DisplayName: me.User.DisplayName,
		Restaurants: me.Restaurants,
	})
}
