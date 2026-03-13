package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	authsvc "github.com/srad/mediasink/internal/service/auth"
	userssvc "github.com/srad/mediasink/internal/service/users"
)

const (
	CurrentUserKey   = "currentUser"
	CurrentUserIDKey = "currentUserID"
)

type AuthMiddleware struct {
	auth  *authsvc.Service
	users *userssvc.Service
}

func NewAuthMiddleware(auth *authsvc.Service, users *userssvc.Service) *AuthMiddleware {
	return &AuthMiddleware{
		auth:  auth,
		users: users,
	}
}

func (m *AuthMiddleware) RequireAuth(c *gin.Context) {
	tokenString, err := extractBearerToken(c)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	userID, err := m.auth.ParseToken(tokenString)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	user, err := m.users.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user not found or invalid"})
		return
	}

	c.Set(CurrentUserKey, user)
	c.Set(CurrentUserIDKey, user.UserID)
	c.Next()
}

func extractBearerToken(c *gin.Context) (string, error) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		if token, exists := c.GetQuery("Authorization"); exists && token != "" {
			return token, nil
		}
		return "", errors.New("authorization header is missing")
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", errors.New("invalid token format")
	}
	return parts[1], nil
}
