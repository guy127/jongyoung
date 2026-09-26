package review

import (
	"time"

	"github.com/google/uuid"
)

type ReviewRequest struct {
	Score int    `json:"score" binding:"required,min=1,max=5" example:"5"`
	Body  string `json:"body" binding:"max=2000" example:"อร่อยมาก"`
}

type ReviewResponse struct {
	ID         uuid.UUID `json:"id"`
	AuthorName string    `json:"author_name,omitempty"`
	Score      int       `json:"score"`
	Body       string    `json:"body"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type ListResponse struct {
	Items []ReviewResponse `json:"items"`
	Page  int              `json:"page"`
	Limit int              `json:"limit"`
	Total int64            `json:"total"`
}

func newResponse(rv Review, author string) ReviewResponse {
	return ReviewResponse{ID: rv.ID, AuthorName: author, Score: rv.Score, Body: rv.Body, CreatedAt: rv.CreatedAt, UpdatedAt: rv.UpdatedAt}
}
