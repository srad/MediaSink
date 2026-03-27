package services

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/srad/mediasink/server/internal/db"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupChapterRegenerationDB(t *testing.T) {
	t.Helper()

	tempRoot := t.TempDir()
	t.Setenv("DB_FILENAME", filepath.Join(tempRoot, "test.db"))
	t.Setenv("REC_PATH", tempRoot)
	t.Setenv("DATA_DIR", ".previews")
	t.Setenv("DATA_DISK", tempRoot)
	t.Setenv("NET_ADAPTER", "lo")

	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}

	db.DB = database
	if err := db.DB.AutoMigrate(&db.Channel{}, &db.Recording{}, &db.VideoPreview{}, &db.Job{}); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
}

func seedChapterRegenerationData(t *testing.T) (*db.Channel, []*db.Recording) {
	t.Helper()

	channel := &db.Channel{
		ChannelName: "chapter_test",
		DisplayName: "Chapter Test",
		URL:         "https://example.com/channel",
		CreatedAt:   time.Now(),
	}
	if err := db.DB.Create(channel).Error; err != nil {
		t.Fatalf("create channel: %v", err)
	}

	recordings := []*db.Recording{
		{
			ChannelID:    channel.ChannelID,
			ChannelName:  channel.ChannelName,
			Filename:     "one.mp4",
			CreatedAt:    time.Now(),
			VideoType:    "video/mp4",
			PathRelative: filepath.Join("chapter_test", "one.mp4"),
		},
		{
			ChannelID:    channel.ChannelID,
			ChannelName:  channel.ChannelName,
			Filename:     "two.mp4",
			CreatedAt:    time.Now(),
			VideoType:    "video/mp4",
			PathRelative: filepath.Join("chapter_test", "two.mp4"),
		},
	}

	for _, recording := range recordings {
		if err := db.DB.Create(recording).Error; err != nil {
			t.Fatalf("create recording %s: %v", recording.Filename, err)
		}
	}

	return channel, recordings
}

func seedValidPreview(t *testing.T, recording *db.Recording) {
	t.Helper()

	previewPath := recording.RecordingID.GetPreviewFramesPath(recording.ChannelName)
	if err := os.MkdirAll(previewPath, 0o777); err != nil {
		t.Fatalf("mkdir preview path: %v", err)
	}

	for _, name := range []string{"0.jpg", "2.jpg"} {
		if err := os.WriteFile(filepath.Join(previewPath, name), []byte("test"), 0o666); err != nil {
			t.Fatalf("write preview frame %s: %v", name, err)
		}
	}

	preview := &db.VideoPreview{
		RecordingID:   recording.RecordingID,
		FrameCount:    2,
		FrameInterval: 2,
		PreviewPath:   recording.RecordingID.GetRelativePreviewFramesPath(recording.ChannelName),
	}
	if err := db.DB.Create(preview).Error; err != nil {
		t.Fatalf("create preview row: %v", err)
	}

	recording.VideoPreviews = preview
}

func TestRegenerateAllChapters_ReplacesAnalyzeJobsForAllRecordings(t *testing.T) {
	setupChapterRegenerationDB(t)
	channel, recordings := seedChapterRegenerationData(t)
	for _, recording := range recordings {
		seedValidPreview(t, recording)
	}

	oldAnalyze := &db.Job{
		ChannelID:   channel.ChannelID,
		RecordingID: recordings[0].RecordingID,
		ChannelName: channel.ChannelName,
		Filename:    recordings[0].Filename,
		Task:        db.TaskAnalyzeFrames,
		Status:      db.StatusJobCompleted,
		Priority:    db.PriorityNormal,
		Filepath:    filepath.Join("/tmp", recordings[0].Filename.String()),
		CreatedAt:   time.Now(),
	}
	if err := db.DB.Create(oldAnalyze).Error; err != nil {
		t.Fatalf("create old analyze job: %v", err)
	}

	previewJob := &db.Job{
		ChannelID:   channel.ChannelID,
		RecordingID: recordings[1].RecordingID,
		ChannelName: channel.ChannelName,
		Filename:    recordings[1].Filename,
		Task:        db.TaskPreviewFrames,
		Status:      db.StatusJobOpen,
		Priority:    db.PriorityHigh,
		Filepath:    filepath.Join("/tmp", recordings[1].Filename.String()),
		CreatedAt:   time.Now(),
	}
	if err := db.DB.Create(previewJob).Error; err != nil {
		t.Fatalf("create preview job: %v", err)
	}

	result, err := RegenerateAllChapters()
	if err != nil {
		t.Fatalf("RegenerateAllChapters: %v", err)
	}

	if result.RemovedJobs != 1 {
		t.Fatalf("expected 1 removed analyze job, got %d", result.RemovedJobs)
	}
	if result.Enqueued != len(recordings) {
		t.Fatalf("expected %d enqueued jobs, got %d", len(recordings), result.Enqueued)
	}
	if result.Recordings != len(recordings) {
		t.Fatalf("expected %d recordings in result, got %d", len(recordings), result.Recordings)
	}

	var analyzeCount int64
	if err := db.DB.Model(&db.Job{}).Where("task = ?", db.TaskAnalyzeFrames).Count(&analyzeCount).Error; err != nil {
		t.Fatalf("count analyze jobs: %v", err)
	}
	if analyzeCount != int64(len(recordings)) {
		t.Fatalf("expected %d analyze jobs after regeneration, got %d", len(recordings), analyzeCount)
	}

	var previewCount int64
	if err := db.DB.Model(&db.Job{}).Where("task = ?", db.TaskPreviewFrames).Count(&previewCount).Error; err != nil {
		t.Fatalf("count preview jobs: %v", err)
	}
	if previewCount != 1 {
		t.Fatalf("expected preview jobs to remain untouched, got %d", previewCount)
	}
}

func TestRegenerateAllChapters_QueuesPreviewsWhenMissing(t *testing.T) {
	setupChapterRegenerationDB(t)
	_, recordings := seedChapterRegenerationData(t)
	seedValidPreview(t, recordings[0])

	result, err := RegenerateAllChapters()
	if err != nil {
		t.Fatalf("RegenerateAllChapters: %v", err)
	}

	if result.Enqueued != len(recordings) {
		t.Fatalf("expected %d total queued jobs, got %d", len(recordings), result.Enqueued)
	}

	var analyzeCount int64
	if err := db.DB.Model(&db.Job{}).Where("task = ?", db.TaskAnalyzeFrames).Count(&analyzeCount).Error; err != nil {
		t.Fatalf("count analyze jobs: %v", err)
	}
	if analyzeCount != 1 {
		t.Fatalf("expected 1 analyze job for recording with previews, got %d", analyzeCount)
	}

	var previewCount int64
	if err := db.DB.Model(&db.Job{}).Where("task = ?", db.TaskPreviewFrames).Count(&previewCount).Error; err != nil {
		t.Fatalf("count preview jobs: %v", err)
	}
	if previewCount != 1 {
		t.Fatalf("expected 1 preview job for recording missing previews, got %d", previewCount)
	}
}
