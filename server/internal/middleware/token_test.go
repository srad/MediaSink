package middleware

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

const testSecret = "test-secret-for-parse-token"

// signed builds a token with the given claims, signed with secret using HS256.
func signed(t *testing.T, secret string, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return s
}

// forgedAlg hand-builds a token with an arbitrary "alg" header. Used to exercise the
// signing-method check without needing an RSA key: the keyfunc rejects anything that
// is not *jwt.SigningMethodHMAC, and an unknown alg is rejected by the parser itself.
func forgedAlg(t *testing.T, alg string, claims jwt.MapClaims) string {
	t.Helper()
	enc := func(v any) string {
		raw, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		return base64.RawURLEncoding.EncodeToString(raw)
	}
	header := enc(map[string]string{"alg": alg, "typ": "JWT"})
	return header + "." + enc(claims) + ".c2ln"
}

func validClaims() jwt.MapClaims {
	return jwt.MapClaims{
		"id":  float64(42),
		"exp": float64(time.Now().Add(time.Hour).Unix()),
	}
}

func TestParseToken_Valid(t *testing.T) {
	id, err := parseToken(signed(t, testSecret, validClaims()), testSecret)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 42 {
		t.Errorf("user id = %d, want 42", id)
	}
}

func TestParseToken_Errors(t *testing.T) {
	tests := []struct {
		name  string
		token string
		want  error
	}{
		{
			name:  "garbage string",
			token: "not-a-jwt",
			want:  errMalformedToken,
		},
		{
			name:  "empty string",
			token: "",
			want:  errMalformedToken,
		},
		{
			name: "expired exp",
			token: signed(t, testSecret, jwt.MapClaims{
				"id":  float64(42),
				"exp": float64(time.Now().Add(-time.Hour).Unix()),
			}),
			// Rejected by jwt.Parse itself, so it lands on errTokenExpired rather than
			// on the manual exp check below.
			want: errTokenExpired,
		},
		{
			name:  "signed with the wrong secret",
			token: signed(t, "some-other-secret", validClaims()),
			want:  errTokenUnhandled,
		},
		{
			name:  "alg none (algorithm confusion)",
			token: forgedAlg(t, "none", validClaims()),
			want:  errTokenUnhandled,
		},
		{
			name:  "unknown alg",
			token: forgedAlg(t, "XX999", validClaims()),
			want:  errTokenUnhandled,
		},
		{
			name: "exp claim absent",
			// jwt.Parse ACCEPTS this: MapClaims.Valid calls VerifyExpiresAt(now, false),
			// and with req=false a missing exp verifies as valid. Only parseToken's
			// manual check rejects a token that never expires.
			token: signed(t, testSecret, jwt.MapClaims{"id": float64(42)}),
			want:  errTokenExpiredOrInvalid,
		},
		{
			name:  "exp is zero",
			token: signed(t, testSecret, jwt.MapClaims{"id": float64(42), "exp": float64(0)}),
			want:  errTokenExpiredOrInvalid,
		},
		{
			name:  "exp is a string",
			token: signed(t, testSecret, jwt.MapClaims{"id": float64(42), "exp": "tomorrow"}),
			// Not errTokenExpiredOrInvalid, which is the intuitive guess. A non-numeric
			// exp falls past both type cases in MapClaims.VerifyExpiresAt to `return
			// false`, so the library reports it as EXPIRED and jwt.Parse rejects it
			// before parseToken's own check ever runs. Safe either way; pinned here so
			// the behaviour is recorded rather than assumed.
			want: errTokenExpired,
		},
		{
			name: "id claim absent",
			token: signed(t, testSecret, jwt.MapClaims{
				"exp": float64(time.Now().Add(time.Hour).Unix()),
			}),
			want: errInvalidTokenPayload,
		},
		{
			name: "id claim is a string",
			token: signed(t, testSecret, jwt.MapClaims{
				"id":  "42",
				"exp": float64(time.Now().Add(time.Hour).Unix()),
			}),
			want: errInvalidTokenPayload,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := parseToken(tt.token, testSecret)
			if err == nil {
				t.Fatalf("expected error %v, got nil (id=%d)", tt.want, id)
			}
			if !errors.Is(err, tt.want) {
				t.Errorf("error = %v, want errors.Is(..., %v)", err, tt.want)
			}
			if id != 0 {
				t.Errorf("id = %d on error, want 0", id)
			}
		})
	}
}

// TestTokenErrorSentinel_RendersBareString guards the wire format. app.Gin.Error writes
// err.Error() straight into the response body, so if the middleware ever rendered the
// wrapped error instead of the sentinel, every client would see a changed message.
func TestTokenErrorSentinel_RendersBareString(t *testing.T) {
	tests := []struct {
		name  string
		token string
		want  string
	}{
		{"malformed", "not-a-jwt", "malformed token"},
		{
			"expired",
			signed(t, testSecret, jwt.MapClaims{
				"id": float64(1), "exp": float64(time.Now().Add(-time.Hour).Unix()),
			}),
			"token expired or not yet valid",
		},
		{"wrong secret", signed(t, "other", validClaims()), "couldn't handle this token"},
		{
			"no exp",
			signed(t, testSecret, jwt.MapClaims{"id": float64(1)}),
			"token expired or invalid",
		},
		{
			"no id",
			signed(t, testSecret, jwt.MapClaims{
				"exp": float64(time.Now().Add(time.Hour).Unix()),
			}),
			"invalid token payload",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseToken(tt.token, testSecret)
			if err == nil {
				t.Fatal("expected an error")
			}
			got := tokenErrorSentinel(err).Error()
			if got != tt.want {
				t.Errorf("rendered body = %q, want %q", got, tt.want)
			}
			if strings.Contains(got, ":") {
				t.Errorf("rendered body %q leaks wrapped detail; clients see this verbatim", got)
			}
		})
	}
}

// TestParseToken_SecretIsUsed proves the secret parameter is honoured rather than
// ignored in favour of an environment variable.
func TestParseToken_SecretIsUsed(t *testing.T) {
	token := signed(t, "secret-A", validClaims())

	if _, err := parseToken(token, "secret-A"); err != nil {
		t.Fatalf("correct secret rejected: %v", err)
	}
	if _, err := parseToken(token, "secret-B"); err == nil {
		t.Fatal("wrong secret accepted")
	}
}
