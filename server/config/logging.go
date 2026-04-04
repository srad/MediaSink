package config

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
)

type StreamDebugLevel string

type StreamDebugProfile struct {
	level StreamDebugLevel
}

const (
	StreamDebugQuiet   StreamDebugLevel = "quiet"
	StreamDebugError   StreamDebugLevel = "error"
	StreamDebugWarning StreamDebugLevel = "warning"
	StreamDebugInfo    StreamDebugLevel = "info"
	StreamDebugDebug   StreamDebugLevel = "debug"
	StreamDebugTrace   StreamDebugLevel = "trace"
)

func ParseLogLevel(raw string) (log.Level, error) {
	normalized := strings.TrimSpace(strings.ToLower(raw))
	if normalized == "" {
		return log.InfoLevel, nil
	}

	level, err := log.ParseLevel(normalized)
	if err != nil {
		return log.InfoLevel, fmt.Errorf("supported values are panic, fatal, error, warn, info, debug, trace")
	}

	return level, nil
}

func ParseStreamDebugLevel(raw string) (StreamDebugLevel, error) {
	normalized := strings.TrimSpace(strings.ToLower(raw))
	switch normalized {
	case "", string(StreamDebugError):
		return StreamDebugError, nil
	case string(StreamDebugQuiet):
		return StreamDebugQuiet, nil
	case string(StreamDebugWarning):
		return StreamDebugWarning, nil
	case string(StreamDebugInfo):
		return StreamDebugInfo, nil
	case string(StreamDebugDebug):
		return StreamDebugDebug, nil
	case string(StreamDebugTrace):
		return StreamDebugTrace, nil
	default:
		return StreamDebugError, fmt.Errorf("supported values are quiet, error, warning, info, debug, trace")
	}
}

func NewStreamDebugProfile(level StreamDebugLevel) *StreamDebugProfile {
	return &StreamDebugProfile{level: level}
}

func (profile *StreamDebugProfile) Level() StreamDebugLevel {
	if profile == nil {
		return StreamDebugError
	}
	return profile.level
}

func (profile *StreamDebugProfile) FFmpegLogLevel() string {
	switch profile.Level() {
	case StreamDebugQuiet:
		return "quiet"
	case StreamDebugWarning:
		return "warning"
	case StreamDebugInfo:
		return "info"
	case StreamDebugDebug:
		return "debug"
	case StreamDebugTrace:
		return "trace"
	default:
		return "error"
	}
}

func (profile *StreamDebugProfile) YTDLPArgs() []string {
	switch profile.Level() {
	case StreamDebugQuiet:
		return []string{"--quiet", "--no-warnings"}
	case StreamDebugWarning, StreamDebugInfo:
		return nil
	case StreamDebugDebug:
		return []string{"--verbose"}
	case StreamDebugTrace:
		return []string{"--verbose", "--print-traffic"}
	default:
		return []string{"--no-warnings"}
	}
}

func (profile *StreamDebugProfile) LogCommandDetails() bool {
	level := profile.Level()
	return level == StreamDebugDebug || level == StreamDebugTrace
}

func (profile *StreamDebugProfile) LogCommandOutputOnSuccess() bool {
	switch profile.Level() {
	case StreamDebugInfo, StreamDebugDebug, StreamDebugTrace:
		return true
	default:
		return false
	}
}
