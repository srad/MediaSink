package database

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-playground/validator/v10"

	log "github.com/sirupsen/logrus"
	"github.com/srad/mediasink/conf"
	"github.com/srad/mediasink/helpers"
	"gorm.io/gorm"
)

const (
	PreviewFrames PreviewType = "preview-frames"
)

type PreviewType string

type RecordingID uint

// GetPreviewFramesPath returns the absolute path to the preview frames directory for this recording
func (recordingID RecordingID) GetPreviewFramesPath(channelName ChannelName) string {
	cfg := conf.Read()
	return filepath.Join(
		cfg.RecordingsAbsolutePath,
		channelName.String(),
		cfg.DataPath,
		helpers.PreviewFramesFolder,
		fmt.Sprintf("%d", recordingID),
	)
}

// GetRelativePreviewFramesPath returns the relative path to the preview frames directory for this recording
func (recordingID RecordingID) GetRelativePreviewFramesPath(channelName ChannelName) string {
	cfg := conf.Read()
	return filepath.Join(
		channelName.String(),
		cfg.DataPath,
		helpers.PreviewFramesFolder,
		fmt.Sprintf("%d", recordingID),
	)
}

type Recording struct {
	RecordingID RecordingID `json:"recordingId" gorm:"autoIncrement;primaryKey;column:recording_id" extensions:"!x-nullable" validate:"gte=0"`

	Channel     Channel           `json:"-" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;foreignKey:channel_id;references:channel_id"`
	ChannelID   ChannelID         `json:"channelId" gorm:"not null;default:null" extensions:"!x-nullable" validate:"gte=0"`
	ChannelName ChannelName       `json:"channelName" gorm:"not null;default:null;index:idx_file,unique" extensions:"!x-nullable" validate:"required"`
	Filename    RecordingFileName `json:"filename" gorm:"not null;default:null;index:idx_file,unique" extensions:"!x-nullable" validate:"required"`
	Bookmark    bool              `json:"bookmark" gorm:"index:idx_bookmark;not null" extensions:"!x-nullable"`
	CreatedAt   time.Time         `json:"createdAt" gorm:"not null;default:null;index" extensions:"!x-nullable"`
	VideoType   string            `json:"videoType" gorm:"default:null;not null" extensions:"!x-nullable" validate:"required"`

	Packets  uint64  `json:"packets" gorm:"default:0;not null" extensions:"!x-nullable"` // Total number of video packets/frames.
	Duration float64 `json:"duration" gorm:"default:0;not null" extensions:"!x-nullable"`
	Size     uint64  `json:"size" gorm:"default:0;not null" extensions:"!x-nullable"`
	BitRate  uint64  `json:"bitRate" gorm:"default:0;not null" extensions:"!x-nullable"`
	Width    uint    `json:"width" gorm:"default:0" extensions:"!x-nullable"`
	Height   uint    `json:"height" gorm:"default:0" extensions:"!x-nullable"`

	PathRelative string `json:"pathRelative" gorm:"default:null;not null" validate:"required,filepath"`

	VideoPreviews *VideoPreview `json:"videoPreview" gorm:"foreignKey:recording_id;references:recording_id"`
}

func FindRecordingByID(recordingID RecordingID) (*Recording, error) {
	if recordingID == 0 {
		return nil, fmt.Errorf("invalid recording recordingID %d", recordingID)
	}

	var recording *Recording
	if err := DB.Model(Recording{}).
		Preload("VideoPreviews").
		Where("recordings.recording_id = ?", recordingID).
		First(&recording).Error; err != nil {
		return nil, err
	}

	return recording, nil
}

func FavRecording(id uint, fav bool) error {
	return DB.Model(Recording{}).
		Where("recording_id = ?", id).
		Update("bookmark", fav).Error
}

func SortBy(column string, order string, skip, take int) ([]*Recording, int64, error) {
	var count int64 = 0

	// Create a base query - helps if you add filters later
	query := DB.Model(&Recording{})

	// 1. Get the total count (without offset/limit/order)
	if err := query.Count(&count).Error; err != nil {
		// Return error if counting fails
		return nil, 0, err
	}

	// 2. If count is 0, return early with an empty slice.
	if count == 0 {
		return make([]*Recording, 0), 0, nil
	}

	// 3. Initialize 'recordings' as an empty, non-nil slice.
	recordings := make([]*Recording, 0)

	// 4. Perform the Find query
	err := query.
		Preload("VideoPreviews").
		Order(fmt.Sprintf("%s %s", column, order)).
		Offset(skip).
		Limit(take).
		Find(&recordings).Error

	// 5. Check for errors, but specifically ignore 'ErrRecordNotFound'.
	if err != nil {
		// Check if the error is specifically 'Record Not Found'.
		// As mentioned, this is unlikely with Find + slice, but it's safe to check.
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// We found nothing, which is NOT an error in this context.
			// Return the initialized empty slice and the count we already got.
			return recordings, count, nil
		}
		// It's some other, real database error, so return it.
		return nil, 0, err
	}

	// 6. If no error occurred, return the (potentially populated) slice and count.
	return recordings, count, nil
}

func FindRandom(limit int) ([]*Recording, error) {
	var recordings []*Recording

	err := DB.Model(Recording{}).
		Preload("VideoPreviews").
		Order("RANDOM()").
		Limit(limit).
		Find(&recordings).Error

	if err != nil {
		return nil, err
	}

	return recordings, nil
}

func RecordingsList() ([]*Recording, error) {
	var recordings []*Recording

	err := DB.Model(Recording{}).
		Preload("VideoPreviews").
		Select("recordings.*").
		Find(&recordings).Error

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	return recordings, nil
}

// FindRecordingsByIDs returns recordings for the given IDs with video previews preloaded.
func FindRecordingsByIDs(ids []RecordingID) ([]*Recording, error) {
	if len(ids) == 0 {
		return []*Recording{}, nil
	}
	uids := make([]uint, 0, len(ids))
	for _, id := range ids {
		uids = append(uids, uint(id))
	}

	var recordings []*Recording
	err := DB.Model(Recording{}).
		Preload("VideoPreviews").
		Where("recording_id IN ?", uids).
		Find(&recordings).Error

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	return recordings, nil
}

func BookmarkList() ([]*Recording, error) {
	var recordings []*Recording
	err := DB.Model(Recording{}).
		Preload("VideoPreviews").
		Where("bookmark = ?", true).
		Select("recordings.*").Order("recordings.channel_name asc").
		Find(&recordings).Error

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	return recordings, nil
}

func GetPaths(channelName ChannelName, filename RecordingFileName) RecordingPaths {
	return channelName.GetRecordingsPaths(filename)
}

func CreateRecording(channelId ChannelID, filename RecordingFileName, videoType string) (*Recording, error) {
	channel, errChannel := GetChannelByID(channelId)
	if errChannel != nil {
		return nil, errChannel
	}

	info, err := GetVideoInfo(channel.ChannelName, filename)
	if err != nil {
		return nil, err
	}

	recording := &Recording{
		RecordingID:  0,
		Channel:      Channel{},
		ChannelID:    channelId,
		ChannelName:  channel.ChannelName,
		Filename:     filename,
		Bookmark:     false,
		CreatedAt:    time.Now(),
		VideoType:    videoType,
		Packets:      info.PacketCount,
		Duration:     info.Duration,
		Size:         info.Size,
		BitRate:      info.BitRate,
		Width:        info.Width,
		Height:       info.Height,
		PathRelative: channel.ChannelName.ChannelPath(filename),
	}

	// Check for existing recording first, then create if not found
	existing := &Recording{}
	existsResult := DB.Where("channel_id = ? AND filename = ?", channelId, filename).First(existing)

	if existsResult.Error == nil {
		// Recording already exists, return it
		return existing, nil
	}

	if !errors.Is(existsResult.Error, gorm.ErrRecordNotFound) {
		// Database error
		return nil, fmt.Errorf("error checking existing recording: %w", existsResult.Error)
	}

	// Create new recording using transaction to ensure atomic operation
	tx := BeginTx()
	if err := tx.Create(&recording).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("error creating recording: %w", err)
	}
	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("error committing recording creation: %w", err)
	}

	return recording, nil
}

func DestroyJobs(id RecordingID) error {
	// Delete all jobs for this recording
	// With foreign key constraints enabled, this will cascade automatically
	// But we also do it manually for safety
	result := DB.Where("recording_id = ?", id).Delete(&Job{})
	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return fmt.Errorf("error deleting jobs for recording %d: %w", id, result.Error)
	}
	return nil
}

// DestroyRecording Deletes all recording related files, jobs, and database item.
// Deletes from database first (atomic transaction), then cleans up files.
func (recording *Recording) DestroyRecording() error {
	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(recording); err != nil {
		return fmt.Errorf("invalid recording values: %w", err)
	}

	// Delete from database first (wrapped in transaction for atomicity)
	// This prevents orphaned records if subsequent file operations fail
	tx := BeginTx()
	if err := tx.Delete(&Recording{}, "recording_id = ?", recording.RecordingID).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("error deleting recording %d from database: %w", recording.RecordingID, err)
	}
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("error committing transaction for recording %d: %w", recording.RecordingID, err)
	}

	// Now clean up associated files and jobs
	// Collect all cleanup errors but continue with cleanup
	var cleanupErrors []error

	// Delete associated jobs
	if err := DestroyJobs(recording.RecordingID); err != nil {
		cleanupErrors = append(cleanupErrors, fmt.Errorf("error deleting jobs: %w", err))
	}

	// Delete recording file
	if err := DeleteFile(recording.ChannelName, recording.Filename); err != nil {
		cleanupErrors = append(cleanupErrors, fmt.Errorf("error deleting recording file: %w", err))
	}

	// Delete preview files
	if err := recording.DestroyPreviews(); err != nil {
		cleanupErrors = append(cleanupErrors, fmt.Errorf("error deleting preview files: %w", err))
	}

	if len(cleanupErrors) > 0 {
		return errors.Join(cleanupErrors...)
	}

	return nil
}

func DeleteRecordingData(channelName ChannelName, filename RecordingFileName) error {
	// Get recording ID for preview frames cleanup
	var recording Recording
	if err := DB.Where("channel_name = ? AND filename = ?", channelName, filename).First(&recording).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("error retrieving recording: %w", err)
	}

	// Delete from database (wrapped in transaction) to prevent orphaned records
	tx := BeginTx()
	if err := tx.Delete(&Recording{}, "channel_name = ? AND filename = ?", channelName, filename).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		tx.Rollback()
		return fmt.Errorf("error deleting recordings of file '%s' from channel '%s': %w", filename, channelName, err)
	}
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("error committing transaction for recording '%s' in channel '%s': %w", filename, channelName, err)
	}

	// Now clean up files and previews
	var cleanupErrors []error

	if err := DeleteFile(channelName, filename); err != nil {
		cleanupErrors = append(cleanupErrors, fmt.Errorf("error deleting file: %w", err))
	}

	if err := DeletePreviewFiles(recording.RecordingID, channelName); err != nil {
		cleanupErrors = append(cleanupErrors, fmt.Errorf("error deleting preview files: %w", err))
	}

	if len(cleanupErrors) > 0 {
		return errors.Join(cleanupErrors...)
	}

	return nil
}

func DeleteFile(channelName ChannelName, filename RecordingFileName) error {
	paths := channelName.GetRecordingsPaths(filename)

	if err := os.Remove(paths.Filepath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("error deleting recording: %s", err)
	}

	return nil
}

func (recordingID RecordingID) FindRecordingByID() (*Recording, error) {
	var recording *Recording
	err := DB.Table("recordings").
		Preload("VideoPreviews").
		Where("recording_id = ?", recordingID).
		First(&recording).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	return recording, nil
}

func (channelId ChannelID) FindJobs() (*[]Job, error) {
	var jobs *[]Job
	err := DB.Model(&Job{}).
		Where("channel_id = ?", channelId).
		Find(&jobs).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	return jobs, nil
}

func AddIfNotExists(channelId ChannelID, channelName ChannelName, filename RecordingFileName) (*Recording, error) {
	var recording *Recording

	err := DB.Model(Recording{}).
		Where("channel_name = ? AND filename = ?", channelName, filename).
		First(&recording).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		log.Infof("No recording found, creating: Recording(channel-name: %s, filename: '%s')", channelName, filename)

		created, errCreate := CreateRecording(channelId, filename, "recording")
		if errCreate != nil {
			return nil, fmt.Errorf("error creating recording '%s'", errCreate)
		}
		log.Infof("Created recording %s/%s", channelName, filename)
		return created, nil
	}

	return recording, err
}

func GetVideoInfo(channelName ChannelName, filename RecordingFileName) (*helpers.FFProbeInfo, error) {
	video := helpers.Video{FilePath: channelName.AbsoluteChannelFilePath(filename)}
	return video.GetVideoInfo()
}

func (recording *Recording) UpdateInfo(info *helpers.FFProbeInfo) error {
	return DB.Model(recording).Where("recording_id = ?", recording.RecordingID).Updates(&Recording{ChannelName: recording.ChannelName, Filename: recording.Filename, Duration: info.Duration, BitRate: info.BitRate, Size: info.Size, Width: info.Width, Height: info.Height, Packets: info.PacketCount}).Error
}

func (recording *Recording) AbsoluteChannelFilepath() string {
	return recording.ChannelName.AbsoluteChannelFilePath(recording.Filename)
}

func (recording *Recording) DataFolder() string {
	return recording.ChannelName.AbsoluteChannelDataPath()
}

func (recording *Recording) DestroyPreviews() error {
	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(recording); err != nil {
		return nil
	}

	return DeletePreviewFiles(recording.RecordingID, recording.ChannelName)
}

func (recording *Recording) DestroyPreview(previewType PreviewType) error {
	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(recording); err != nil {
		return err
	}

	recording, errFind := recording.RecordingID.FindRecordingByID()
	if errFind != nil {
		return errFind
	}

	switch previewType {
	case PreviewFrames:
		previewFramesPath := recording.RecordingID.GetPreviewFramesPath(recording.ChannelName)
		if err := os.RemoveAll(previewFramesPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("error deleting preview frames directory: %w", err)
		}
		return DeleteVideoPreviewByRecordingID(recording.RecordingID)
	}

	return fmt.Errorf("invalid preview type %s", previewType)
}

func DeletePreviewFiles(recordingID RecordingID, channelName ChannelName) error {
	// Delete preview frames directory
	previewFramesPath := recordingID.GetPreviewFramesPath(channelName)
	if err := os.RemoveAll(previewFramesPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("error deleting preview frames directory: %w", err)
	}

	// Delete preview metadata from database
	return DeleteVideoPreviewByRecordingID(recordingID)
}

func (recording *Recording) Save() error {
	return DB.Model(&recording).Save(&recording).Error
}
