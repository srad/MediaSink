package handlers

import (
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/srad/mediasink/server/internal/db"
)

// A conversion job queued by an older build can still carry an unsupported
// media type such as "mp3". The handler must fail it cleanly rather than
// handing the value to ffmpeg. Validation happens before any database or
// ffmpeg access, so no fixtures are required.
func TestConversionHandlerRejectsUnsupportedMediaType(t *testing.T) {
	logger := log.New()
	logger.SetOutput(newDiscard())
	handler := NewConversionHandler(NewHandlerDependencies(logger, nil))

	for _, mediaType := range []string{"mp3", "999", "720p"} {
		args := `"` + mediaType + `"`
		job := &db.Job{
			JobID: 1,
			Task:  db.TaskConvert,
			Args:  &args,
		}

		err := handler.Handle(job, 1)
		if err == nil {
			t.Errorf("conversion job with media type %q returned no error", mediaType)
			continue
		}
		if !strings.Contains(err.Error(), "unsupported media type") {
			t.Errorf("conversion job with media type %q failed with %q, want an unsupported-media-type error",
				mediaType, err)
		}
	}
}

type discard struct{}

func (discard) Write(p []byte) (int, error) { return len(p), nil }

func newDiscard() discard { return discard{} }
