package services

import (
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/srad/mediasink/database"
	"github.com/srad/mediasink/helpers"
	"gorm.io/gorm"
)

type ChannelInfo struct {
	database.Channel
	IsRecording   bool    `json:"isRecording" extensions:"!x-nullable"`
	IsOnline      bool    `json:"isOnline" extensions:"!x-nullable"`
	IsTerminating bool    `json:"isTerminating" extensions:"!x-nullable"`
	Preview       string  `json:"preview" extensions:"!x-nullable"`
	MinRecording  float64 `json:"minRecording" extensions:"!x-nullable"`
}

// CreateChannel Persistent channel generation.
func CreateChannel(name, displayName string, skipStart, minDuration uint, url string, fav bool, tags *database.Tags, isPaused bool) (*ChannelInfo, error) {
	channel := database.Channel{
		ChannelName: database.ChannelName(name),
		DisplayName: displayName,
		SkipStart:   skipStart,
		MinDuration: minDuration,
		CreatedAt:   time.Now(),
		URL:         url,
		Fav:         fav,
		Tags:        tags,
		IsPaused:    isPaused}

	newChannel, err := database.CreateChannelDetail(channel)

	if err != nil {
		log.Errorln(err)

		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, fmt.Errorf("error creating record: %s", err)
		}
		return nil, err
	}

	info := &ChannelInfo{
		Channel:      *newChannel,
		IsRecording:  false,
		IsOnline:     false,
		Preview:      newChannel.ChannelName.PreviewPath(),
		MinRecording: 0,
	}

	return info, nil
}

// GetChannels Adds additional streaming and recording information to the channel data in the database.
func GetChannels() ([]ChannelInfo, error) {
	channels, err := database.ChannelListNotDeleted()
	if err != nil {
		return nil, err
	}

	response := make([]ChannelInfo, len(channels))

	for index, channel := range channels {
		// Add to each channel current system information
		response[index] = ChannelInfo{
			Channel:       *channel,
			Preview:       channel.ChannelName.PreviewPath(),
			IsOnline:      IsOnline(channel.ChannelID),
			IsTerminating: IsTerminating(channel.ChannelID),
			IsRecording:   IsRecordingStream(channel.ChannelID),
			MinRecording:  GetRecordingMinutes(channel.ChannelID),
		}
	}

	return response, nil
}

// GetChannel Single Channel data with streaming and recording information.
func GetChannel(id uint) (*ChannelInfo, error) {
	channelID := database.ChannelID(id)
	channel, err := database.GetChannelByIDWithRecordings(channelID)
	if err != nil {
		return nil, fmt.Errorf("channel not found: %w", err)
	}

	return &ChannelInfo{
		Channel:       *channel,
		IsOnline:      IsOnline(channel.ChannelID),
		IsTerminating: IsTerminating(channel.ChannelID),
		IsRecording:   IsRecordingStream(channel.ChannelID),
		MinRecording:  GetRecordingMinutes(channel.ChannelID),
		Preview:       channel.ChannelName.PreviewPath(),
	}, nil
}

func DeleteChannel(channelID database.ChannelID) error {
	var err1, err2 error
	if err := TerminateProcess(channelID); err != nil {
		err1 = fmt.Errorf("process could not be terminated: %s", err.Error())
	}

	if err := database.TryDeleteChannel(channelID); err != nil {
		err2 = fmt.Errorf("channel could not be deleted: %s", err.Error())
	}

	err := errors.Join(err1, err2)
	if err == nil {
		log.Infof("Deleted channel %d", channelID)
	}

	return err
}

// UploadRecording handles uploading a video file to a channel, validating it, and enqueueing preview jobs
func UploadRecording(channelID database.ChannelID, fileReader io.Reader) (*database.Recording, error) {
	// Create recording entry and get output path
	recording, outputPath, err := database.NewRecording(channelID, "recording")
	if err != nil {
		return nil, fmt.Errorf("error creating recording entry: %w", err)
	}

	// Create output file
	out, err := os.Create(outputPath)
	if err != nil {
		return nil, fmt.Errorf("error creating output file: %w", err)
	}

	// Copy file content to disk
	_, err = io.Copy(out, fileReader)
	if closeErr := out.Close(); closeErr != nil && err == nil {
		err = closeErr
	}

	if err != nil {
		if cleanupErr := os.Remove(outputPath); cleanupErr != nil {
			log.Warnf("Error deleting partial upload file: %v", cleanupErr)
		}
		return nil, fmt.Errorf("error copying file: %w", err)
	}

	// Validate and extract metadata from uploaded file
	videoInfo := &helpers.Video{FilePath: outputPath}
	ffProbeInfo, err := videoInfo.GetVideoInfo()
	if err != nil {
		if cleanupErr := os.Remove(outputPath); cleanupErr != nil {
			log.Warnf("Error deleting invalid video file: %v", cleanupErr)
		}
		return nil, fmt.Errorf("uploaded file is not a valid video: %w", err)
	}

	// Update recording with video metadata
	recording.Duration = ffProbeInfo.Duration
	recording.Size = ffProbeInfo.Size
	recording.BitRate = ffProbeInfo.BitRate
	recording.Width = ffProbeInfo.Width
	recording.Height = ffProbeInfo.Height
	recording.Packets = ffProbeInfo.PacketCount

	// Save recording to database
	if err := recording.Save(); err != nil {
		if cleanupErr := os.Remove(outputPath); cleanupErr != nil {
			log.Warnf("Error deleting upload file after DB save failure: %v", cleanupErr)
		}
		return nil, fmt.Errorf("error saving recording to database: %w", err)
	}

	// Enqueue preview frames generation job
	if _, err := recording.EnqueuePreviewFramesJob(); err != nil {
		log.Errorf("Error enqueueing preview frames job for uploaded recording: %v", err)
		// Cleanup the recording and file if preview job fails
		if cleanupErr := recording.DestroyRecording(); cleanupErr != nil {
			log.Errorf("Error cleaning up orphaned recording: %v", cleanupErr)
		}
		return nil, fmt.Errorf("error enqueueing preview job: %w", err)
	}

	log.Infof("Successfully uploaded recording to channel %d: %s", channelID, recording.Filename)
	return recording, nil
}
