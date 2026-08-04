package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/srad/mediasink/server/internal/app"
	"github.com/srad/mediasink/server/internal/models/requests"
	"github.com/srad/mediasink/server/internal/services"
)

// CreateUser godoc
// @Summary     Create new user account
// @Description Create a new user account with username and password
// @Tags        auth
// @Param       AuthenticationRequest body requests.AuthenticationRequest true "Username and password"
// @Accept      json
// @Produce     json
// @Success     200 {} nil "User created successfully"
// @Failure     400 {string} string "Error message"
// @Failure     500 {string} string "Error message"
// @Router      /auth/signup [post]
func (h *AuthHandler) CreateUser(c *gin.Context) {
	appG := app.Gin{C: c}
	var auth requests.AuthenticationRequest

	if err := c.BindJSON(&auth); err != nil {
		appG.Error(http.StatusBadRequest, err)
		return
	}

	// 500 is wrong for a name that is merely taken — 409 is. Changing it is a wire
	// change owned by the API redesign phase; public_auth.golden pins today's answer
	// so that change shows up as a diff.
	if err := h.users.CreateUser(c.Request.Context(), auth); err != nil {
		appG.Error(http.StatusInternalServerError, err)
		return
	}
	appG.Response(http.StatusOK, nil)
}

// Login godoc
// @Summary     User login
// @Description User login
// @Tags        auth
// @Param       AuthenticationRequest body requests.AuthenticationRequest true "Username and password"
// @Accept      json
// @Produce     json
// @Success     200 {object} responses.LoginResponse "JWT token for authentication"
// @Failure     401 {string} string "Error message"
// @Failure     400 {string} string "Error message"
// @Router      /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	appG := app.Gin{C: c}

	var auth requests.AuthenticationRequest
	if err := c.BindJSON(&auth); err != nil {
		appG.Error(http.StatusBadRequest, err)
		return
	}

	jwt, err := h.users.Authenticate(c.Request.Context(), auth)
	if err != nil {
		appG.Error(http.StatusUnauthorized, err)
		return
	}

	appG.Response(http.StatusOK, gin.H{"token": jwt})
}

// AuthHandler serves the /auth routes.
type AuthHandler struct {
	users *services.UserService
}

func NewAuthHandler(users *services.UserService) *AuthHandler {
	return &AuthHandler{users: users}
}

// Logout godoc
// @Summary     User logout
// @Description User logout, clears the authentication session
// @Tags        auth
// @Accept      json
// @Produce     json
// @Success     200 {} object "Logout successful message"
// @Failure     401 {string} string "Error message"
// @Failure     400 {string} string "Error message"
// @Router      /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	appG := app.Gin{C: c}

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "jwt",
		Value:    "",
		Path:     "/",
		Domain:   "", //".example.com",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteNoneMode,
	})

	appG.Response(http.StatusOK, gin.H{"message": "Logged out"})
}
