package v1

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/srad/mediasink/server/config"
)

// util.Info sleeps twice for the requested number of seconds, so an unbounded
// path parameter let a single request occupy a goroutine indefinitely.
// Rejection happens before any measurement, so these cases return immediately.
func TestGetInfoRejectsOutOfRangeSeconds(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// The handler now closes over its configuration; a zero Cfg is enough, because
	// every case here is rejected before the config is ever consulted.
	handler := GetInfo(config.Cfg{})

	for _, seconds := range []string{"0", "61", "99999999999", "18446744073709551616", "abc", "-1"} {
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Request = httptest.NewRequest(http.MethodGet, "/info/"+seconds, nil)
		c.Params = gin.Params{{Key: "seconds", Value: seconds}}

		handler(c)

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
