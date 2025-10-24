package services

import (
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/srad/mediasink/database"
)

var (
	isUpdating      = false
	isUpdatingMutex sync.Mutex
)

func UpdateVideoInfo() error {
	log.Infoln("[Recorder] Updating all recordings info")
	recordings, err := database.RecordingsList()
	if err != nil {
		log.Errorln(err)
		return err
	}
	isUpdatingMutex.Lock()
	isUpdating = true
	isUpdatingMutex.Unlock()
	count := len(recordings)

	i := 1
	for _, rec := range recordings {
		info, err := database.GetVideoInfo(rec.ChannelName, rec.Filename)
		if err != nil {
			log.Errorf("[UpdateVideoInfo] Error updating video info: %s", err)
			continue
		}

		if err := rec.UpdateInfo(info); err != nil {
			log.Errorf("[Recorder] Error updating video info: %s", err)
			continue
		}
		log.Infof("[Recorder] Updated %s (%d/%d)", rec.Filename, i, count)
		i++
	}

	isUpdatingMutex.Lock()
	isUpdating = false
	isUpdatingMutex.Unlock()

	return nil
}

func IsUpdatingRecordings() bool {
	isUpdatingMutex.Lock()
	defer isUpdatingMutex.Unlock()
	return isUpdating
}
