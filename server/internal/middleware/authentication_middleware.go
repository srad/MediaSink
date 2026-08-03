package middleware

import (
	"errors"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/gin-gonic/gin"
	"github.com/srad/mediasink/server/internal/app"
	"github.com/srad/mediasink/server/internal/db"
	"github.com/srad/mediasink/server/internal/services"
)

var (
	errMissingAuthHeader = errors.New("authorization header is missing")
	errInvalidTokenFmt   = errors.New("invalid token format")
	errUserNotFound      = errors.New("user not found or invalid")
)

// RequireAuth returns the bearer-token gate, closing over the store and the JWT secret.
// Both are captured once at router construction. The secret used to be read from the
// process environment on every single authenticated request, and the user lookup used
// to reach a package-level database handle; neither is true now, which is what makes
// this package testable without mutating global state.
func RequireAuth(store *db.Store, secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		appG := app.Gin{C: c}

		tokenString, err := extractBearerToken(c)
		if err != nil {
			log.Errorln(err)
			appG.Error(http.StatusUnauthorized, err)
			return
		}

		userID, err := parseToken(tokenString, secret)
		if err != nil {
			// Render the bare sentinel, never the wrapped error: app.Gin.Error writes
			// err.Error() as the response body, so wrapping would change the wire format.
			// The wrapped detail goes to the log instead.
			appG.Error(http.StatusUnauthorized, tokenErrorSentinel(err))
			return
		}

		user, err := services.GetUserByID(store, userID)
		if err != nil {
			appG.Error(http.StatusUnauthorized, errUserNotFound)
			return
		}

		c.Set("currentUser", user)
		c.Next()
	}
}

// tokenErrorSentinel logs err at the level this path has always used and returns the
// bare sentinel to render. The malformed/expired split matters: expired tokens are
// routine (clients retry), malformed ones are not.
func tokenErrorSentinel(err error) error {
	switch {
	case errors.Is(err, errMalformedToken):
		log.Error("Malformed token")
		return errMalformedToken
	case errors.Is(err, errTokenExpired):
		log.Warn("Token expired or not yet valid")
		return errTokenExpired
	case errors.Is(err, errTokenUnhandled):
		log.Errorf("Couldn't handle this token: %v", err)
		return errTokenUnhandled
	case errors.Is(err, errTokenExpiredOrInvalid):
		return errTokenExpiredOrInvalid
	case errors.Is(err, errInvalidTokenPayload):
		return errInvalidTokenPayload
	default:
		log.Errorf("JWT parsing error: %v", err)
		return errInvalidToken
	}
}

// extractBearerToken pulls the raw JWT out of the request.
func extractBearerToken(c *gin.Context) (string, error) {
	authHeader := c.GetHeader("Authorization")

	if authHeader == "" {
		// Workaround for JWT over websockets. The bearer can also be sent as get parameter.
		getAuth, exists := c.GetQuery("Authorization")
		if !exists || getAuth == "" {
			return "", errMissingAuthHeader
		}
		log.Debugln("Received authentication as get parameter. Likely from a socket.")
		authHeader = "Bearer " + getAuth
	}

	authToken := strings.Split(authHeader, " ")
	if len(authToken) != 2 || authToken[0] != "Bearer" {
		return "", errInvalidTokenFmt
	}

	return authToken[1], nil
}
