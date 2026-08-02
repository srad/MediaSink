package v1

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/srad/mediasink/server/internal/app"
	"github.com/srad/mediasink/server/internal/db"
	"github.com/srad/mediasink/server/internal/models/responses"
)

// GetUserProfile godoc
// @Summary     Get user profile
// @Description Get the current authenticated user's profile information
// @Tags        user
// @Accept      json
// @Produce     json
// @Success     200 {object} responses.UserProfileResponse "User profile"
// @Failure     400 {} http.StatusBadRequest
// @Failure     500 {} http.StatusInternalServerError
// @Router      /user/profile [get]
func GetUserProfile(c *gin.Context) {
	appG := app.Gin{C: c}
	value, exists := c.Get("currentUser")

	if !exists {
		appG.Error(http.StatusBadRequest, errors.New("user does not exist"))
		return
	}

	// The auth middleware stores a *db.User; assert rather than pass the raw
	// model through, so the password hash can never reach the response.
	user, ok := value.(*db.User)
	if !ok {
		appG.Error(http.StatusInternalServerError, errors.New("invalid current user"))
		return
	}

	appG.Response(http.StatusOK, responses.UserProfileResponse{
		UserID:    user.UserID,
		Username:  user.Username,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	})
}
