package jobs

import (
	"context"
	"sync"
	"time"

	"github.com/srad/mediasink/jobs/handlers"
)

// JobMessage is an alias for handlers.JobMessage for backwards compatibility
type JobMessage[T any] = handlers.JobMessage[T]

// Module-level variables for job processing coordination
var (
	sleepBetweenRounds  = 1 * time.Second
	ctxJobs, cancelJobs = context.WithCancel(context.Background())
	processing          = false
	processingMutex     sync.Mutex
)
