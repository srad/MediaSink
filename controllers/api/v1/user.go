package v1

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/srad/mediasink/app"
)

// GetUserProfile godoc
// @Summary     Get user profile
// @Description Get the current authenticated user's profile information
// @Tags        user
// @Accept      json
// @Produce     json
// @Success     200 {object} object "User profile"
// @Failure     400 {} http.StatusBadRequest
// @Router      /user/profile [get]
func GetUserProfile(c *gin.Context) {
	appG := app.Gin{C: c}
	user, exists := c.Get("currentUser")

	if !exists {
		appG.Error(http.StatusBadRequest, errors.New("user does not exist"))
		return
	}

	appG.Response(http.StatusOK, user)
}
