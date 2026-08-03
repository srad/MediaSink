package middleware

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/srad/mediasink/server/config"
	"github.com/srad/mediasink/server/internal/db"
)

const routerSecret = "router-supplied-secret"

// gateFor builds a gin engine with RequireAuth in front of a trivial handler, so the
// tests exercise the middleware exactly as the router mounts it.
func gateFor(secret string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.GET("/guarded", RequireAuth(secret), func(c *gin.Context) {
		c.String(http.StatusOK, "reached the handler")
	})
	return engine
}

func callGuarded(engine *gin.Engine, authHeader string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/guarded", nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	engine.ServeHTTP(recorder, req)
	return recorder
}

// The five rejection paths all fail before the user lookup, so none of them needs a
// database. The bodies are the wire format: app.Gin.Error JSON-encodes err.Error(),
// which is why the expectations below are quoted — internal/api/testdata pins the
// identical bytes.
func TestRequireAuthRejects(t *testing.T) {
	engine := gateFor(routerSecret)

	tests := []struct {
		name       string
		authHeader string
		wantErr    error
	}{
		{
			name:       "missing authorization header",
			authHeader: "",
			wantErr:    errMissingAuthHeader,
		},
		{
			name:       "header without the Bearer prefix",
			authHeader: "Basic dXNlcjpwYXNz",
			wantErr:    errInvalidTokenFmt,
		},
		{
			name:       "malformed token",
			authHeader: "Bearer not-a-jwt",
			wantErr:    errMalformedToken,
		},
		{
			name: "expired token",
			authHeader: "Bearer " + signed(t, routerSecret, jwt.MapClaims{
				"id":  float64(1),
				"exp": float64(time.Now().Add(-time.Hour).Unix()),
			}),
			wantErr: errTokenExpired,
		},
		{
			// The point of the phase: the gate validates against the secret the
			// router handed it, so a token signed with anything else fails the
			// signature check.
			name:       "token signed with a different secret",
			authHeader: "Bearer " + signed(t, "some-other-secret", validClaims()),
			wantErr:    errTokenUnhandled,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := callGuarded(engine, test.authHeader)

			if recorder.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
			}
			want := `"` + test.wantErr.Error() + `"`
			if got := recorder.Body.String(); got != want {
				t.Errorf("body = %q, want %q", got, want)
			}
		})
	}
}

// The accept path reaches services.GetUserByID, so it needs a real handle. Note what
// this test does NOT do: it sets no environment variable to get one. That is only
// possible because db.Init takes a config value now.
func TestRequireAuthAcceptsTokenSignedWithTheRouterSecret(t *testing.T) {
	userID := seedUser(t)

	recorder := callGuarded(gateFor(routerSecret), "Bearer "+signed(t, routerSecret, jwt.MapClaims{
		"id":  float64(userID),
		"exp": float64(time.Now().Add(time.Hour).Unix()),
	}))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d (body %q), want %d", recorder.Code, recorder.Body.String(), http.StatusOK)
	}
	if got := recorder.Body.String(); got != "reached the handler" {
		t.Errorf("body = %q, want the guarded handler to have run", got)
	}
}

// The regression guard for this whole phase. The middleware used to call
// os.Getenv("SECRET") on every request; if it ever does again, the garbage value
// below wins over the secret the router supplied and this test fails.
func TestRequireAuthIgnoresTheSecretEnvironmentVariable(t *testing.T) {
	t.Setenv("SECRET", "garbage-that-must-not-be-used")

	userID := seedUser(t)

	recorder := callGuarded(gateFor(routerSecret), "Bearer "+signed(t, routerSecret, jwt.MapClaims{
		"id":  float64(userID),
		"exp": float64(time.Now().Add(time.Hour).Unix()),
	}))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d (body %q), want %d: the secret is being read from the environment again",
			recorder.Code, recorder.Body.String(), http.StatusOK)
	}
}

// seedUser initialises a throwaway database and returns the ID of one user. A temp
// file rather than :memory: because that is what db.Init's migrations are proven
// against in the API golden suite.
func seedUser(t *testing.T) uint {
	t.Helper()

	db.Init(config.Cfg{DbFileName: filepath.Join(t.TempDir(), "middleware.db")})

	user := &db.User{Username: "gate@example.com", Password: "irrelevant-for-this-test"}
	if err := db.CreateUser(user); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return user.UserID
}
