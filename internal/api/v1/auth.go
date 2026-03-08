package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/srad/mediasink/internal/app"
	"github.com/srad/mediasink/internal/models/requests"
	"github.com/srad/mediasink/internal/services"
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
func CreateUser(c *gin.Context) {
	appG := app.Gin{C: c}
	var auth requests.AuthenticationRequest

	if err := c.BindJSON(&auth); err != nil {
		appG.Error(http.StatusBadRequest, err)
		return
	}

	if err := services.CreateUser(auth); err != nil {
		appG.Error(http.StatusInternalServerError, err)
		return
	} else {
		appG.Response(http.StatusOK, nil)
	}
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
func Login(c *gin.Context) {
	appG := app.Gin{C: c}

	var auth requests.AuthenticationRequest
	if err := c.BindJSON(&auth); err != nil {
		appG.Error(http.StatusBadRequest, err)
		return
	}

	jwt, err := services.AuthenticateUser(auth)
	if err != nil {
		appG.Error(http.StatusUnauthorized, err)
		return
	}

	/*
		http.SetCookie(c.Writer, &http.Cookie{
			Name:     "jwt",
			Value:    jwt,
			Path:     "/",
			Domain:   "",                    //".example.com", // Or leave blank for same-origin; use if client is on subdomain
			HttpOnly: true,                  // Prevent access from JS (safer)
			Secure:   false,                 // Must be true for HTTPS
			SameSite: http.SameSiteNoneMode, // Required for cross-domain cookie sharing
			MaxAge:   86400,                 // 1 day
		})
	*/

	//appG.Response(http.StatusOK, gin.H{"message": "Login successful"})
	appG.Response(http.StatusOK, gin.H{"token": jwt})
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
func Logout(c *gin.Context) {
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
