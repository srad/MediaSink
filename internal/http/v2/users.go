package v2

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	httpmiddleware "github.com/srad/mediasink/internal/http/middleware"
	userssvc "github.com/srad/mediasink/internal/service/users"
)

type UsersHandler struct {
	users *userssvc.Service
}

type UserResponse struct {
	UserID    uint      `json:"userId"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func NewUsersHandler(users *userssvc.Service) *UsersHandler {
	return &UsersHandler{users: users}
}

func (h *UsersHandler) Me(c *gin.Context) {
	userIDValue, exists := c.Get(httpmiddleware.CurrentUserIDKey)
	if !exists {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing current user"})
		return
	}

	userID, ok := userIDValue.(uint)
	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid current user"})
		return
	}

	user, err := h.users.GetByID(c.Request.Context(), userID)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, UserResponse{
		UserID:    user.UserID,
		Username:  user.Username,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	})
}
