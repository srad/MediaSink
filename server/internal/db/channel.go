package db

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/srad/mediasink/server/config"
	"github.com/srad/mediasink/server/internal/util"
	"gorm.io/gorm"
)

// Channel Represent a single stream, that shall be recorded. It can also serve as a folder for videos.
type Channel struct {
	ChannelID   ChannelID   `json:"channelId" gorm:"autoIncrement;primaryKey;column:channel_id" extensions:"!x-nullable"`
	ChannelName ChannelName `json:"channelName" gorm:"unique;not null;" extensions:"!x-nullable"`
	DisplayName string      `json:"displayName" gorm:"not null;default:''" extensions:"!x-nullable"`
	SkipStart   uint        `json:"skipStart" gorm:"not null;default:0" extensions:"!x-nullable"`
	MinDuration uint        `json:"minDuration" gorm:"not null;default:0" extensions:"!x-nullable"`
	URL         string      `json:"url" gorm:"not null;default:''" extensions:"!x-nullable"`
	Tags        *Tags       `json:"tags" gorm:"type:text;default:null"`
	Fav         bool        `json:"fav" gorm:"index:idx_fav,not null" extensions:"!x-nullable"`
	IsPaused    bool        `json:"isPaused" gorm:"not null,default:false" extensions:"!x-nullable"`
	Deleted     bool        `json:"deleted" gorm:"not null,default:false" extensions:"!x-nullable"`
	CreatedAt   time.Time   `json:"createdAt" gorm:"not null;default:current_timestamp" extensions:"!x-nullable"`

	// Only for query result.
	RecordingsCount uint `json:"recordingsCount" gorm:"<-:false;-:migration" extensions:"!x-nullable"`
	RecordingsSize  uint `json:"recordingsSize" gorm:"<-:false;-:migration" extensions:"!x-nullable"`

	// 1:n
	Recordings []Recording `json:"recordings" gorm:"foreignKey:channel_id;constraint:OnDelete:CASCADE"`
}

func CreateChannel(channelName ChannelName, displayName, url string) (*Channel, error) {
	var channel *Channel
	if err := DB.Model(Channel{}).Where("channel_name = ?", channelName).First(&channel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			newChannel := newChannel(channelName, displayName, url)
			if err := DB.Create(&newChannel).Error; err != nil {
				return nil, err
			}
			return &newChannel, nil
		}
	}

	return channel, nil
}

func DestroyChannelRecordings(channelID ChannelID) error {
	if channelID == 0 {
		return errors.New("invalid channel id")
	}

	channel, errChannel := GetChannelByID(channelID)
	if errChannel != nil {
		return errChannel
	}

	var deleteErrors []error

	// 1. Terminate and delete all jobs.
	if jobs, err := channel.Jobs(); err != nil {
		log.Errorln(err)
		deleteErrors = append(deleteErrors, fmt.Errorf("error fetching jobs for channel %d: %w", channelID, err))
	} else {
		for _, job := range jobs {
			if err := DeleteJob(job.JobID); err != nil {
				log.Errorf("Error destroying job: %s", err)
				deleteErrors = append(deleteErrors, fmt.Errorf("error deleting job %d: %w", job.JobID, err))
			}
		}
	}

	// 2. Delete all associated recordings.
	var recordings []*Recording
	if err := DB.Model(&Recording{}).
		Where("channel_id = ?", channelID).
		Find(&recordings).Error; err != nil {
		log.Errorf("Error finding recordings to destroy for channel %s: %s", channel.ChannelName, err)
		deleteErrors = append(deleteErrors, fmt.Errorf("error finding recordings for channel %d: %w", channelID, err))
	} else {
		for _, recording := range recordings {
			if err := recording.DestroyRecording(); err != nil {
				log.Errorf("Error deleting recording %s: %s", recording.Filename, err)
				deleteErrors = append(deleteErrors, fmt.Errorf("error destroying recording %d: %w", recording.RecordingID, err))
			}
		}
	}

	// Return aggregated errors if any occurred
	if len(deleteErrors) > 0 {
		return errors.Join(deleteErrors...)
	}

	return nil
}

func CreateChannelDetail(channel Channel) (*Channel, error) {
	if err := DB.Create(&channel).Error; err != nil {
		return nil, err
	}

	if err := channel.ChannelName.MkDir(); err != nil {
		return nil, err
	}
	//channel.WriteJson()

	return &channel, nil
}

func (channel *Channel) ExistsJSON() bool {
	return util.FileExists(channel.jsonPath())
}

func (channel *Channel) FolderExists() bool {
	return util.FileExists(channel.ChannelName.AbsoluteChannelPath())
}

func (channel *Channel) jsonPath() string {
	return path.Join(channel.ChannelName.AbsoluteChannelPath(), "channel.json")
}

func (channel *Channel) Update() error {
	// Validation
	url := strings.TrimSpace(channel.URL)
	displayName := strings.TrimSpace(channel.DisplayName)

	if len(url) == 0 || len(displayName) == 0 {
		return fmt.Errorf("invalid parameters: %v", channel)
	}

	err := DB.Save(&channel).Error

	return err
}

func (channel *Channel) QueryStreamURLs() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second) // 30-second timeout
	defer cancel()

	debugProfile := config.Read().StreamDebugProfile
	cmdArgs := buildQueryStreamURLArgs(channel.URL, debugProfile)
	cmd := exec.CommandContext(ctx, "yt-dlp", cmdArgs...)

	if debugProfile.LogCommandDetails() {
		if executablePath, err := exec.LookPath("yt-dlp"); err == nil {
			log.Debugf("[QueryStreamURL] using yt-dlp executable %q", executablePath)
		}
		log.Debugf("[QueryStreamURL] executing: yt-dlp %s", strings.Join(cmdArgs, " "))
	}

	outputBytes, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(outputBytes))

	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return nil, fmt.Errorf("yt-dlp command timed out for URL %s", channel.URL)
	}

	if err != nil {
		// err from exec.CommandContext will be non-nil if youtube-dl exits with a non-zero status
		// output will contain stderr from youtube-dl, which is useful context
		return nil, fmt.Errorf("yt-dlp failed for URL %s: %v\nOutput: %s", channel.URL, err, output)
	}

	urls := extractStreamURLs(output)
	if len(urls) == 0 {
		return nil, fmt.Errorf("yt-dlp returned empty or invalid output for URL %s: %s", channel.URL, output)
	}

	return urls, nil
}

func buildQueryStreamURLArgs(channelURL string, debugProfile *config.StreamDebugProfile) []string {
	args := []string{
		"--force-ipv4",
	}
	args = append(args, debugProfile.YTDLPArgs()...)
	args = append(args,
		"-f", "bv*+ba/b",
		"--get-url",
		channelURL,
	)
	return args
}

func extractStreamURLs(output string) []string {
	lines := strings.Split(output, "\n")
	urls := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "http://") || strings.HasPrefix(line, "https://") || strings.HasPrefix(line, "rtmp://") {
			urls = append(urls, line)
		}
	}
	return urls
}

func ChannelList() ([]*Channel, error) {
	var channels []*Channel

	err := DB.Model(&Channel{}).
		Select("channels.*", "(SELECT COUNT(*) FROM recordings WHERE recordings.channel_id = channels.channel_id) recordings_count", "(SELECT SUM(size) FROM recordings WHERE recordings.channel_name = channels.channel_name) recordings_size").
		Find(&channels).Error

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	return channels, nil
}

func ChannelListNotDeleted() ([]*Channel, error) {
	var result []*Channel

	err := DB.Model(&Channel{}).
		Where("channels.deleted = ?", false).
		Select("channels.*", "(SELECT COUNT(*) FROM recordings WHERE recordings.channel_id = channels.channel_id) recordings_count", "(SELECT SUM(size) FROM recordings WHERE recordings.channel_id = channels.channel_id) recordings_size").
		Find(&result).Error

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	return result, nil
}

func EnabledChannelList() ([]*Channel, error) {
	var channels []*Channel

	// Query favourites first
	err := DB.Model(&Channel{}).
		Where("deleted = ?", false).
		Where("is_paused = ?", false).
		Select("channels.*", "(SELECT COUNT(*) FROM recordings WHERE recordings.channel_id = channels.channel_id) recordings_count").
		Order("fav desc").
		Find(&channels).Error

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	return channels, nil
}

func newChannel(channelName ChannelName, displayName, url string) Channel {
	return Channel{
		ChannelName: channelName,
		DisplayName: displayName,
		SkipStart:   0,
		MinDuration: 10,
		URL:         strings.TrimSpace(url),
		Tags:        nil,
		Fav:         false,
		IsPaused:    false,
		Deleted:     false,
		CreatedAt:   time.Now(),
	}
}
