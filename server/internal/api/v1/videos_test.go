package v1

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// ConvertVideo validates the media type before any database access, so an
// unsupported type must be rejected with 400 without a configured database.
func TestConvertVideoRejectsUnsupportedMediaType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for _, mediaType := range []string{"mp3", "999", "720p", "../../etc/passwd"} {
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
		c.Params = gin.Params{
			{Key: "id", Value: "1"},
			{Key: "mediaType", Value: mediaType},
		}

		ConvertVideo(c)

		if recorder.Code != http.StatusBadRequest {
			t.Errorf("ConvertVideo with mediaType %q returned %d, want %d",
				mediaType, recorder.Code, http.StatusBadRequest)
		}
	}
}

func TestConvertVideoRejectsInvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	c.Params = gin.Params{
		{Key: "id", Value: "not-a-number"},
		{Key: "mediaType", Value: "720"},
	}

	ConvertVideo(c)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("ConvertVideo with a bad id returned %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}
