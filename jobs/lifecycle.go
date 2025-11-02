package jobs

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/srad/mediasink/database"
	"github.com/srad/mediasink/network"
)

// StartJobProcessing initializes and starts all job processing workers
func StartJobProcessing() {
	ctxJobs, cancelJobs = context.WithCancel(context.Background())
	go processFastJobs(ctxJobs)
	go processSlowJobs(ctxJobs)
	go cleanupOldJobs(ctxJobs)
}

// StopJobProcessing stops all job processing workers
func StopJobProcessing() {
	cancelJobs()
}

// cleanupOldJobs automatically removes completed and error jobs older than 30 days
func cleanupOldJobs(ctx context.Context) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	// Run cleanup immediately on startup
	performCleanup()

	for {
		select {
		case <-ctx.Done():
			log.Infoln("[cleanupOldJobs] Cleanup worker stopped")
			return
		case <-ticker.C:
			performCleanup()
		}
	}
}

// performCleanup removes old job records from the database
func performCleanup() {
	cutoff := time.Now().AddDate(0, 0, -30)
	result := database.DB.
		Where("status IN (?) AND completed_at < ?",
			[]database.JobStatus{database.StatusJobCompleted, database.StatusJobError},
			cutoff).
		Delete(&database.Job{})

	if result.Error != nil {
		log.Errorf("[cleanupOldJobs] Error cleaning up old jobs: %s", result.Error)
	} else if result.RowsAffected > 0 {
		log.Infof("[cleanupOldJobs] Cleaned up %d old jobs (older than 30 days)", result.RowsAffected)
	}
}

// DeleteJob marks a job as deleted and broadcasts the event
func DeleteJob(id uint) error {
	if err := database.DeleteJob(id); err != nil {
		return err
	}
	network.BroadCastClients(network.JobDeleteEvent, id)
	return nil
}

// IsJobProcessing returns whether the job processing system is active
func IsJobProcessing() bool {
	processingMutex.Lock()
	defer processingMutex.Unlock()
	return processing
}
