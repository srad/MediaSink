package middleware

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// Sentinel errors for every outcome of parseToken.
//
// The strings are deliberately identical to the ones the middleware emitted when
// this logic was inline: they are the HTTP response body (see internal/app.Gin.Error,
// which renders err.Error() directly), so changing one is a client-visible API change.
// The API golden tests in internal/api/testdata pin them.
//
// Callers must match with errors.Is, never on the message text.
var (
	errMalformedToken        = errors.New("malformed token")
	errTokenExpired          = errors.New("token expired or not yet valid")
	errTokenUnhandled        = errors.New("couldn't handle this token")
	errInvalidToken          = errors.New("invalid token")
	errTokenExpiredOrInvalid = errors.New("token expired or invalid")
	errInvalidTokenPayload   = errors.New("invalid token payload")
)

// parseToken validates a JWT and returns the user ID from its "id" claim.
//
// The secret is a parameter rather than an os.Getenv call so this stays testable
// without mutating process environment; the caller supplies it.
func parseToken(tokenString, secret string) (uint, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		var ve *jwt.ValidationError
		if errors.As(err, &ve) {
			// Both verbs wrap (Go 1.20+ allows multiple %w), so callers can match the
			// sentinel with errors.Is AND still inspect the underlying jwt error.
			switch {
			case ve.Errors&jwt.ValidationErrorMalformed != 0:
				return 0, fmt.Errorf("%w: %w", errMalformedToken, err)
			case ve.Errors&(jwt.ValidationErrorExpired|jwt.ValidationErrorNotValidYet) != 0:
				return 0, fmt.Errorf("%w: %w", errTokenExpired, err)
			default:
				return 0, fmt.Errorf("%w: %w", errTokenUnhandled, err)
			}
		}
		// Unreachable with golang-jwt/v4: every failure path in parser.go returns
		// *ValidationError. Kept as a guard in case a future version changes that.
		return 0, fmt.Errorf("%w: %w", errInvalidToken, err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		// Unreachable with golang-jwt/v4: jwt.Parse delegates to ParseWithClaims with
		// MapClaims{}, so Claims is always MapClaims. Kept as a guard, as above.
		return 0, errInvalidToken
	}

	// Not redundant with the library's own expiry check. MapClaims.Valid calls
	// VerifyExpiresAt(now, false) — with req=false a *missing* exp verifies as valid,
	// so jwt.Parse happily accepts a token that never expires. This rejects it.
	exp, ok := claims["exp"].(float64)
	if !ok || float64(time.Now().Unix()) > exp {
		return 0, errTokenExpiredOrInvalid
	}

	idFloat, ok := claims["id"].(float64)
	if !ok {
		return 0, errInvalidTokenPayload
	}

	return uint(idFloat), nil
}
