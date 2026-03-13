package v2

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/srad/mediasink/internal/models/requests"
	authsvc "github.com/srad/mediasink/internal/service/auth"
)

type AuthHandler struct {
	auth *authsvc.Service
}

func NewAuthHandler(auth *authsvc.Service) *AuthHandler {
	return &AuthHandler{auth: auth}
}

func (h *AuthHandler) SignUp(c *gin.Context) {
	var req requests.AuthenticationRequest
	if err := c.BindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.auth.CreateUser(c.Request.Context(), req.Username, req.Password); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "user created"})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req requests.AuthenticationRequest
	if err := c.BindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := h.auth.Authenticate(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}
