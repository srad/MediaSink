package v1

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/srad/mediasink/server/internal/db"
	"golang.org/x/crypto/bcrypt"
)

// The handler used to return the raw *db.User, whose Password field was tagged
// `json:"password"`, so every profile request handed the stored bcrypt hash to
// the client.
func TestGetUserProfileDoesNotLeakPasswordHash(t *testing.T) {
	gin.SetMode(gin.TestMode)

	hash, err := bcrypt.GenerateFromPassword([]byte("hunter2"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/user/profile", nil)
	c.Set("currentUser", &db.User{UserID: 1, Username: "saman", Password: string(hash)})

	GetUserProfile(c)

	if recorder.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d", recorder.Code, http.StatusOK)
	}

	body := recorder.Body.String()
	if strings.Contains(body, string(hash)) {
		t.Errorf("response leaked the password hash: %s", body)
	}
	if strings.Contains(body, "password") {
		t.Errorf("response contains a password field: %s", body)
	}
	if !strings.Contains(body, "saman") {
		t.Errorf("response is missing the username: %s", body)
	}
}

func TestGetUserProfileWithoutUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/user/profile", nil)

	GetUserProfile(c)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("got status %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

// The middleware is expected to store a *db.User; anything else must be
// reported rather than panicking on the type assertion.
func TestGetUserProfileWithWrongType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/user/profile", nil)
	c.Set("currentUser", "not-a-user")

	GetUserProfile(c)

	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("got status %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
}
