package v1

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// util.Info sleeps twice for the requested number of seconds, so an unbounded
// path parameter let a single request occupy a goroutine indefinitely.
// Rejection happens before any measurement, so these cases return immediately.
func TestGetInfoRejectsOutOfRangeSeconds(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for _, seconds := range []string{"0", "61", "99999999999", "18446744073709551616", "abc", "-1"} {
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Request = httptest.NewRequest(http.MethodGet, "/info/"+seconds, nil)
		c.Params = gin.Params{{Key: "seconds", Value: seconds}}

		GetInfo(c)

		if recorder.Code != http.StatusBadRequest {
			t.Errorf("seconds=%q returned %d, want %d", seconds, recorder.Code, http.StatusBadRequest)
		}
	}
}

func TestMaxInfoSecondsIsBounded(t *testing.T) {
	if maxInfoSeconds <= 0 || maxInfoSeconds > 300 {
		t.Fatalf("maxInfoSeconds = %d, expected a small positive bound", maxInfoSeconds)
	}
}
