package db

import (
	"fmt"
	"path/filepath"
	"regexp"
	"testing"
)

func TestRelativeDataPath(t *testing.T) {
	channelName := ChannelName("my_channel")
	expected := "my_channel/.previews"
	fact := channelName.RelativeDataPath()

	if fact != expected {
		t.Errorf("RelativeDataPath() is %s but should be %s", fact, expected)
	}
}

func TestChannelPath(t *testing.T) {
	channelName := ChannelName("my_channel")
	filename := RecordingFileName("my_file.mp4")
	expected := fmt.Sprintf("my_channel/%s", filename)
	fact := channelName.ChannelPath(filename)

	if fact != expected {
		t.Errorf("ChannelPath() %s but should be %s", fact, expected)
	}
}

func TestAbsoluteChannelFilePath(t *testing.T) {
	channelName := ChannelName("my_channel")
	filename := RecordingFileName("my_file.mp4")
	expected := filepath.Join("/tmp", "my_channel", filename.String())
	fact := channelName.AbsoluteChannelFilePath(filename)

	if fact != expected {
		t.Errorf("AbsoluteChannelFilePath() is %s but should be %s", fact, expected)
	}
}

func TestMakeRecordingFilename(t *testing.T) {
	channelName := ChannelName("my_channel")
	filePattern, _ := regexp.Compile(`^[a-z0-9_]+_\d\d\d\d_\d\d_\d\d_\d\d_\d\d_\d\d.mp4$`)
	fact, _ := ChannelName(channelName).MakeRecordingFilename()

	if !filePattern.MatchString(fact.String()) {
		t.Errorf("MakeRecordingFilename() is %s but should match pattern %s", fact, filePattern.String())
	}
}

func TestCreateMp3Filename(t *testing.T) {
	channelName := ChannelName("my_channel")
	filePattern, _ := regexp.Compile(`^[a-z0-9_]+_\d\d\d\d_\d\d_\d\d_\d\d_\d\d_\d\d.mp3$`)
	fact, _ := channelName.MakeMp3Filename()

	if !filePattern.MatchString(fact.String()) {
		t.Errorf("MakeMp3Filename() is %s but should match pattern %s", fact, filePattern.String())
	}
}
