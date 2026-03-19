package services

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	log "github.com/sirupsen/logrus"
)

type PreviewFrame struct {
	Path      string
	Timestamp uint64
}

func LoadPreviewFrames(previewPath string) ([]PreviewFrame, error) {
	files, err := os.ReadDir(previewPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	frames := make([]PreviewFrame, 0, len(files))
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		var timestamp uint64
		if _, err := fmt.Sscanf(file.Name(), "%d.jpg", &timestamp); err != nil {
			log.Warnf("[LoadPreviewFrames] Skipping file with invalid name: %s", file.Name())
			continue
		}

		frames = append(frames, PreviewFrame{
			Path:      filepath.Join(previewPath, file.Name()),
			Timestamp: timestamp,
		})
	}

	if len(frames) == 0 {
		if len(files) > 0 {
			return nil, fmt.Errorf("invalid preview frame format in %s: expected files named <timestamp>.jpg", previewPath)
		}
		return []PreviewFrame{}, nil
	}

	sort.Slice(frames, func(i, j int) bool {
		return frames[i].Timestamp < frames[j].Timestamp
	})

	return frames, nil
}
