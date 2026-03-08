package jobs

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/srad/mediasink/config"
	"github.com/srad/mediasink/internal/db"
	"github.com/srad/mediasink/internal/ws"
)

// processJobs is the main worker loop that processes jobs with specified priorities
func processJobs(ctx context.Context, workerName string, threadCount int, priorities ...db.JobPriority) {
	for {
		select {
		case <-ctx.Done():
			log.Infof("[%s] Worker stopped", workerName)
			processingMutex.Lock()
			processing = false
			processingMutex.Unlock()
			return
		case <-time.After(sleepBetweenRounds):
			processingMutex.Lock()
			processing = true
			processingMutex.Unlock()
			job, errNextJob := db.GetNextJob(priorities...)
			if errNextJob != nil {
				if job != nil {
					_ = job.Error(errNextJob)
				} else {
					log.Errorf("[%s] Failed to get next job: %s", workerName, errNextJob)
				}
				continue
			}
			if job == nil {
				continue
			}

			if err := job.Activate(); err != nil {
				// Activation failed - mark job as error and skip to next iteration
				log.Errorf("[%s] Error activating job %d: %v", workerName, job.JobID, err)
				if errMark := job.Error(fmt.Errorf("failed to activate job: %w", err)); errMark != nil {
					log.Errorf("[%s] Error marking job %d as failed: %v", workerName, job.JobID, errMark)
				}
				ws.BroadCastClients(ws.JobErrorEvent, JobMessage[string]{
					Job:  job,
					Data: fmt.Sprintf("Failed to activate job: %v", err),
				})
				continue
			}
			ws.BroadCastClients(ws.JobActivate, JobMessage[any]{Job: job})

			// Execute the job (this will update job state: Completed or Error)
			if err := executeJob(job, threadCount); err != nil {
				log.Errorf("[%s] Job execution error: %v", workerName, err)
				// Job state already updated by executeJob -> handleJob -> job.Error()
			}

			// Deactivate the job - it should already be inactive after job.Complete() or job.Error()
			// but we ensure it's deactivated in case of state inconsistencies
			if err := job.Deactivate(); err != nil {
				log.Warnf("[%s] Error deactivating job %d after execution: %v - job may remain in active state", workerName, job.JobID, err)
			}

			// Broadcast final state after deactivation
			ws.BroadCastClients(ws.JobDeactivate, JobMessage[any]{Job: job})
		}
	}
}

// processFastJobs worker processes fast tasks (preview frames)
func processFastJobs(ctx context.Context) {
	processJobs(ctx, "FastJobWorker", config.FastJobThreadCount, db.PriorityHigh)
}

// processSlowJobs worker processes slower tasks (cut, merge, enhance, convert)
func processSlowJobs(ctx context.Context) {
	processJobs(ctx, "SlowJobWorker", config.SlowJobThreadCount, db.PriorityNormal, db.PriorityLow)
}
