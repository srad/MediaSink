package database

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/srad/mediasink/conf"
	"github.com/srad/mediasink/helpers"
)

var (
	validChannelName, _ = regexp.Compile("(?i)^[a-z_0-9]+$")
	SnapshotFilename    = "live.jpg"
)

type ChannelName string

type RecordingPaths struct {
	AbsolutePreviewStripePath string
	AbsoluteRecordingsPath    string
	AbsolutePreviewVideosPath string
	AbsolutePreviewCoverPath  string
	Filepath                  string
	RelativeVideosPath        string
	RelativeStripePath        string
	RelativeCoverPath         string
	JPG                       string
	MP4                       string
	//ScreensPath            string
}

// Scan Restores the channel type from the database
func (channelName *ChannelName) Scan(src any) error {
	channelNameString, ok := src.(string)
	if !ok {
		return errors.New("src value cannot cast to string")
	}
	*channelName = ChannelName(channelNameString)
	return nil
}

// Value Stores the channel name in the database.
func (channelName *ChannelName) Value() (driver.Value, error) {
	if channelName == nil {
		return nil, nil
	}

	if err := channelName.IsValid(); err != nil {
		return nil, err
	}

	normalizedName := channelName.normalize()

	if !validChannelName.MatchString(normalizedName.String()) {
		return nil, fmt.Errorf("invalid channel name %s", channelName)
	}

	return normalizedName, nil
}

func (channelName *ChannelName) IsValid() error {
	if channelName == nil {
		return errors.New("channel name is nil")
	}

	str := channelName.normalize()
	if !validChannelName.MatchString(str.String()) {
		return fmt.Errorf("invalid normalized channel name %s", str)
	}

	return nil
}

func (channelName ChannelName) normalize() ChannelName {
	return ChannelName(strings.ToLower(strings.TrimSpace(string(channelName))))
}

func (channelName ChannelName) String() string {
	return string(channelName)
}

func (channelName ChannelName) AbsoluteChannelDataPath() string {
	cfg := conf.Read()
	return filepath.Join(cfg.RecordingsAbsolutePath, channelName.String(), cfg.DataPath)
}

func (channelName ChannelName) AbsoluteChannelPath() string {
	cfg := conf.Read()
	return filepath.Join(cfg.RecordingsAbsolutePath, channelName.String())
}

// safeJoinPath safely joins a base path with a relative path, preventing path traversal attacks
func safeJoinPath(basePath, relativePath string) (string, error) {
	if strings.Contains(relativePath, "..") {
		return "", fmt.Errorf("path traversal detected: relative path contains '..'")
	}

	fullPath := filepath.Join(basePath, relativePath)
	absBasePath, err := filepath.Abs(basePath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute base path: %w", err)
	}

	absFullPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute full path: %w", err)
	}

	// Ensure the full path is within the base path
	if !strings.HasPrefix(absFullPath, absBasePath) {
		return "", fmt.Errorf("path traversal detected: resolved path is outside base directory")
	}

	return fullPath, nil
}

func (channelName ChannelName) MkDir() error {
	dir := channelName.AbsoluteChannelPath()
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		log.Infoln("Creating folder: " + dir)
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return fmt.Errorf("error creating folder: '%s': %s", dir, err)
		}
	}
	dataPath := channelName.AbsoluteChannelDataPath()
	if _, err := os.Stat(dataPath); os.IsNotExist(err) {
		log.Infoln("Creating folder: " + dataPath)
		if err := os.MkdirAll(dataPath, os.ModePerm); err != nil {
			return fmt.Errorf("error creating data path '%s': %s", dataPath, err)
		}
		if err := copyDefaultSnapshotTo(dataPath); err != nil {
			log.Errorln(err)
		}
	}

	return nil
}

func (channelName ChannelName) PreviewPath() string {
	return filepath.Join(channelName.RelativeDataPath(), SnapshotFilename)
	//return filepath.Join(channelName.AbsoluteChannelPath(), cfg.DataPath, SnapshotFilename)
}

func copyDefaultSnapshotTo(dataPath string) error {
	pwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	filePath := filepath.Join(pwd, "assets", "live.jpg")
	srcFile, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open default snapshot file '%s': %w", filePath, err)
	}
	defer func(srcFile *os.File) {
		if err := srcFile.Close(); err != nil {
			log.Errorf("Error closing default live.jpg image file: %s", err)
		}
	}(srcFile)

	destFilePath := filepath.Join(dataPath, SnapshotFilename)
	destFile, err := os.Create(destFilePath) // creates if file doesn't exist
	if err != nil {
		return fmt.Errorf("failed to create snapshot file at '%s': %w", destFilePath, err)
	}
	defer func(destFile *os.File) {
		if err := destFile.Close(); err != nil {
			log.Errorf("Error closing snapshot file: %s", err)
		}
	}(destFile)

	_, err = io.Copy(destFile, srcFile) // check first var for number of bytes copied
	if err != nil {
		return fmt.Errorf("failed to copy default snapshot file to '%s': %w", destFilePath, err)
	}

	if err := destFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync snapshot file '%s': %w", destFilePath, err)
	}

	return nil
}

// GetRecordingsPaths generates the file paths for various recording assets such as video, poster, and stripe images
// based on the provided recording file name. It constructs both absolute and relative paths for the files, including
// video (MP4), stripe image (JPG), and poster image (JPG). These paths are returned in a `RecordingPaths` struct.
//
// Parameters:
//   - name (RecordingFileName): The name of the recording file, which will be used to derive the paths for related assets.
//
// Returns:
//   - RecordingPaths: A struct containing the absolute and relative paths for the recording's video, stripe image,
//     poster image, and their respective preview paths. These paths are derived from the provided channel name and
//     configuration settings.
//
// The function makes use of several helpers and configuration settings:
//   - `conf.Read()`: Reads the configuration to obtain the absolute path for recordings.
//   - `channelName.AbsoluteChannelFilePath(name)`: Computes the absolute file path for the channel's recording file.
//   - `channelName.RelativeDataPath()`: Computes the relative data path for the channel's recordings.
//   - The generated paths for MP4 and JPG files are based on the provided `RecordingFileName`.
//
// Example usage:
//
//	channelName := ChannelName("example_channel")
//	name := RecordingFileName("example_video.mp4")
//	paths := channelName.GetRecordingsPaths(name)
//	fmt.Println(paths.AbsoluteRecordingsPath) // Will print the absolute path for recordings directory.
func (channelName ChannelName) GetRecordingsPaths(name RecordingFileName) RecordingPaths {
	filename := name.String()
	posterJpg := strings.TrimSuffix(filename, filepath.Ext(filename)) + ".jpg"
	stripeJpg := strings.TrimSuffix(filename, filepath.Ext(filename)) + ".jpg"
	mp4 := strings.TrimSuffix(filename, filepath.Ext(filename)) + ".mp4"

	cfg := conf.Read()

	return RecordingPaths{
		AbsoluteRecordingsPath: cfg.RecordingsAbsolutePath,

		Filepath:                  channelName.AbsoluteChannelFilePath(name),
		RelativeVideosPath:        filepath.Join(channelName.RelativeDataPath(), helpers.VideosFolder, mp4),
		RelativeStripePath:        filepath.Join(channelName.RelativeDataPath(), helpers.StripesFolder, stripeJpg),
		RelativeCoverPath:         filepath.Join(channelName.RelativeDataPath(), helpers.CoverFolder, posterJpg),
		AbsolutePreviewVideosPath: filepath.Join(channelName.AbsoluteChannelDataPath(), helpers.VideosFolder, mp4),
		AbsolutePreviewStripePath: filepath.Join(channelName.AbsoluteChannelDataPath(), helpers.StripesFolder, stripeJpg),
		AbsolutePreviewCoverPath:  filepath.Join(channelName.AbsoluteChannelDataPath(), helpers.CoverFolder, posterJpg),
		JPG:                       stripeJpg,
		MP4:                       mp4,
	}
}

func (channelName ChannelName) RelativeDataPath() string {
	cfg := conf.Read()
	return filepath.Join(channelName.String(), cfg.DataPath)
}

func (channelName ChannelName) ChannelPath(filename RecordingFileName) string {
	return filepath.Join(channelName.String(), filename.String())
}

func (channelName ChannelName) AbsoluteChannelFilePath(filename RecordingFileName) string {
	cfg := conf.Read()
	return filepath.Join(cfg.RecordingsAbsolutePath, channelName.String(), filename.String())
}

func (channelName ChannelName) MakeRecordingFilename() (RecordingFileName, time.Time) {
	now := time.Now()
	stamp := now.Format("2006_01_02_15_04_05")
	return RecordingFileName(fmt.Sprintf("%s_%s.mp4", channelName.String(), stamp)), now
}

func (channelName ChannelName) MakeMp3Filename() (RecordingFileName, time.Time) {
	now := time.Now()
	stamp := now.Format("2006_01_02_15_04_05")
	return RecordingFileName(fmt.Sprintf("%s_%s.mp3", channelName.String(), stamp)), now
}
