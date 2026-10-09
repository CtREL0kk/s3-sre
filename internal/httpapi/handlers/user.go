package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"myS3/internal/dto"
	"myS3/internal/httpapi/middleware"
	"myS3/internal/repository/user"
)

type User struct {
	users *user.UserRepo
}

func NewUser(users *user.UserRepo) *User {
	return &User{users: users}
}

func (h *User) Me(c *gin.Context) {
	userID := c.MustGet(middleware.ContextUserID).(uuid.UUID)

	u, err := h.users.GetByID(c.Request.Context(), userID)
	if err != nil {
		respondDomainError(c, err)
		return
	}

	respondJSON(c, http.StatusOK, dto.User{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		CreatedAt: u.CreatedAt,
	})
}
