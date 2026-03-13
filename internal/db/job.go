package db

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/srad/mediasink/internal/ws"
	"time"

	"gorm.io/gorm/clause"

	log "github.com/sirupsen/logrus"
	"github.com/srad/mediasink/internal/util"
	"gorm.io/gorm"
)

const (
	TaskConvert        JobTask   = "convert"
	TaskPreviewFrames  JobTask   = "preview-frames"
	TaskAnalyzeFrames  JobTask   = "analyze-frames"
	TaskCut            JobTask   = "cut"
	TaskMerge          JobTask   = "merge"
	TaskEnhanceVideo   JobTask   = "enhance-video"
	StatusJobCompleted JobStatus = "completed"
	StatusJobOpen      JobStatus = "open"
	StatusJobError     JobStatus = "error"
	StatusJobCanceled  JobStatus = "canceled"
	JobOrderASC        JobOrder  = "ASC"
	JobOrderDESC       JobOrder  = "DESC"

	// Job priorities (lower value = higher priority)
	PriorityHigh   JobPriority = 1 // Fast jobs: preview frames
	PriorityNormal JobPriority = 3 // Medium jobs: cut, merge, analyze
	PriorityLow    JobPriority = 5 // Slow jobs: enhance, convert
)

type JobTask string
type JobStatus string
type JobOrder string
type JobPriority int

type Job struct {
	Channel   Channel   `json:"-" gorm:"foreignKey:channel_id;references:channel_id;"`
	Recording Recording `json:"-" gorm:"foreignKey:recording_id;references:recording_id"`

	JobID uint `json:"jobId" gorm:"autoIncrement;primaryKey" extensions:"!x-nullable"`

	ChannelID   ChannelID   `json:"channelId" gorm:"column:channel_id;not null;default:null" extensions:"!x-nullable"`
	RecordingID RecordingID `json:"recordingId" gorm:"column:recording_id;not null;default:null" extensions:"!x-nullable"`

	// Unique entry, this is the actual primary key
	ChannelName ChannelName       `json:"channelName" gorm:"not null;default:null" extensions:"!x-nullable"`
	Filename    RecordingFileName `json:"filename" gorm:"not null;default:null" extensions:"!x-nullable"`

	// Default values only not to break migrations.
	Task   JobTask   `json:"task" gorm:"not null;default:preview" extensions:"!x-nullable"`
	Status JobStatus `json:"status" gorm:"not null;default:completed" extensions:"!x-nullable"`

	Priority    JobPriority `json:"priority" gorm:"not null;default:5;index:idx_priority" extensions:"!x-nullable"`
	Filepath    string      `json:"filepath" gorm:"not null;default:null;" extensions:"!x-nullable"`
	Active      bool        `json:"active" gorm:"not null;default:false" extensions:"!x-nullable"`
	CreatedAt   time.Time   `json:"createdAt" gorm:"not null;default:current_timestamp;index:idx_create_at" extensions:"!x-nullable"`
	StartedAt   *time.Time  `json:"startedAt" gorm:"default:null"`
	CompletedAt *time.Time  `json:"completedAt" gorm:"default:null"`
	DurationMs  *uint64     `json:"durationMs" gorm:"default:null"` // Duration in milliseconds

	// Additional information
	Pid      *int    `json:"pid" gorm:"default:null"`
	Command  *string `json:"command" gorm:"default:null"`
	Progress *string `json:"progress" gorm:"default:null"`
	Info     *string `json:"info" gorm:"default:null"`
	Args     *string `json:"args" gorm:"default:null"`
}

func (job *Job) CreateJob() error {
	return DB.Create(job).Error
}

func JobList(skip, take int, status []JobStatus, order JobOrder) ([]*Job, int64, error) {
	var count int64 = 0
	if err := DB.Model(&Job{}).
		Where("status IN (?)", status).
		Count(&count).Error; err != nil {
		return nil, 0, err
	}

	var jobs []*Job
	if err := DB.
		Model(&Job{}).
		Where("status IN (?)", status).
		Order(clause.OrderByColumn{Column: clause.Column{Name: "created_at"}, Desc: order == JobOrderDESC}).
		Offset(skip).
		Limit(take).
		Find(&jobs).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, count, err
	}

	return jobs, count, nil
}

func (channel *Channel) Jobs() ([]*Job, error) {
	var jobs []*Job
	if err := DB.Model(&Job{}).
		Where("channel_id = ?", channel.ChannelID).
		Find(&jobs).Error; err != nil {
		return nil, err
	}

	return jobs, nil
}

func (job *Job) Cancel(reason string) error {
	return job.updateStatus(StatusJobCanceled, &reason)
}

func (job *Job) Completed() error {
	now := time.Now()
	err1 := job.updateStatus(StatusJobCompleted, nil)

	updates := map[string]interface{}{
		"completed_at": now,
	}

	// Calculate duration if job was started - fetch StartedAt from database
	var dbJob Job
	if err := DB.Model(&Job{}).Where("job_id = ?", job.JobID).Select("started_at").First(&dbJob).Error; err == nil {
		if dbJob.StartedAt != nil {
			durationMs := uint64(now.Sub(*dbJob.StartedAt).Milliseconds())
			updates["duration_ms"] = durationMs
		}
	}

	err2 := DB.Model(&Job{}).Where("job_id = ?", job.JobID).
		Updates(updates).Error

	return errors.Join(err1, err2)
}

func (job *Job) Error(reason error) error {
	err := reason.Error()
	return job.updateStatus(StatusJobError, &err)
}

func (job *Job) updateStatus(status JobStatus, reason *string) error {
	if job.JobID == 0 {
		return errors.New("invalid job id")
	}

	if job.Pid != nil {
		if err := util.Interrupt(*job.Pid); err != nil {
			log.Errorf("[Destroy] Error interrupting process: %s", err)
			return err
		}
	}

	return DB.Model(&Job{}).Where("job_id = ?", job.JobID).
		Updates(map[string]interface{}{"status": status, "info": reason, "active": false}).Error
}

func JobExists(recordingID RecordingID, task JobTask) (*Job, bool, error) {
	if recordingID == 0 {
		return nil, false, errors.New("recording id is 0")
	}

	var job *Job
	result := DB.Model(&Job{}).Where("recording_id = ? AND task = ? AND status = ?", recordingID, task, StatusJobOpen).First(&job)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	if result.Error != nil {
		return nil, false, result.Error
	}

	return job, result.RowsAffected > 0, nil
}

// JobHasStatus checks if a job with the given ID has a specific status
func JobHasStatus(jobID uint, status JobStatus) (bool, error) {
	if jobID == 0 {
		return false, errors.New("invalid job id")
	}

	var count int64
	result := DB.Model(&Job{}).Where("job_id = ? AND status = ?", jobID, status).Count(&count)

	if result.Error != nil {
		return false, result.Error
	}

	return count > 0, nil
}

// DeleteJob cancels a job by interrupting its process and marking status as canceled.
// This ensures proper cleanup and state tracking. Jobs are permanently deleted only during
// the automated 30-day cleanup process.
func DeleteJob(id uint) error {
	if id == 0 {
		return fmt.Errorf("invalid job id: %d", id)
	}

	var job *Job
	if err := DB.Where("job_id = ?", id).First(&job).Error; err != nil {
		return err
	}

	// Interrupt the running process if it exists
	if job.Pid != nil {
		if err := util.Interrupt(*job.Pid); err != nil {
			log.Errorf("[DeleteJob] Error interrupting process: %s", err)
			return err
		}
	}

	// Mark job as canceled instead of deleting
	// This allows downstream processes to detect the cancellation
	reason := "canceled by user"
	if err := DB.Model(&Job{}).
		Where("job_id = ?", id).
		Updates(map[string]interface{}{
			"status": StatusJobCanceled,
			"info":   reason,
			"active": false,
		}).Error; err != nil {
		return err
	}

	return nil
}

// GetNextJob Any job is attached to a recording which it will process.
// The caller must know which type the JSON serialized argument originally had.
// Jobs are ordered by priority (lower value = higher priority) then by creation time.
// If priorities is nil or empty, all jobs are considered. Otherwise, only jobs with matching priorities.
func GetNextJob(priorities ...JobPriority) (*Job, error) {
	var job *Job
	query := DB.Where("status = ? AND active = ?", StatusJobOpen, false)

	if len(priorities) > 0 {
		query = query.Where("priority IN (?)", priorities)
	}

	err := query.
		Preload("Channel").
		Preload("Recording").
		Order("jobs.priority ASC, jobs.created_at ASC").
		First(&job).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	return job, err
}

func UnmarshalJobArg[T any](job *Job) (*T, error) {
	// Deserialize the arguments, if existent.
	if job.Args != nil && *job.Args != "" {
		var data *T
		if err := json.Unmarshal([]byte(*job.Args), &data); err != nil {
			log.Errorf("[Job] Error parsing cutting job arguments: %s", err)
			if errDestroy := job.Error(err); errDestroy != nil {
				log.Errorf("[Job] Error destroying job: %s", errDestroy)
			}
			return nil, err
		}
		return data, nil
	}

	return nil, errors.New("job arg nil or empty")
}

// GetNextJobTask Any job is attached to a recording which it will process.
// The caller must know which type the JSON serialized argument originally had.
func GetNextJobTask[T any](task JobTask) (*Job, *T, error) {
	var job *Job
	err := DB.Where("task = ? AND status = ? AND active = ?", task, StatusJobOpen, false).
		Order("jobs.created_at asc").
		First(&job).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, nil
	}

	// Deserialize the arguments, if existent.
	if job.Args != nil && *job.Args != "" {
		var data *T
		if err := json.Unmarshal([]byte(*job.Args), &data); err != nil {
			log.Errorf("[Job] Error parsing cutting job arguments: %s", err)
			if errDestroy := job.Error(err); errDestroy != nil {
				log.Errorf("[Job] Error destroying job: %s", errDestroy)
			}
			return job, nil, err
		}
		return job, data, err
	}

	return job, nil, err
}

func (job *Job) UpdateInfo(pid int, command string) error {
	if job.JobID == 0 {
		return errors.New("invalid job id")
	}

	return DB.Model(&Job{}).Where("job_id = ?", job.JobID).
		Update("pid", pid).
		Update("command", command).Error
}

func (job *Job) UpdateProgress(progress string) error {
	if job.JobID == 0 {
		return errors.New("invalid job id")
	}

	return DB.Model(&Job{}).Where("job_id = ?", job.JobID).
		Update("progress", progress).Error
}

func (job *Job) Activate() error {
	if job.JobID == 0 {
		return errors.New("invalid job id")
	}

	return DB.Model(&Job{}).Where("job_id = ?", job.JobID).Updates(map[string]interface{}{"started_at": time.Now(), "active": true}).Error
}

func (job *Job) Deactivate() error {
	if job.JobID == 0 {
		return errors.New("invalid job id")
	}

	return DB.Model(&Job{}).Where("job_id = ?", job.JobID).Update("active", false).Error
}

func CreateJob[T any](recording *Recording, task JobTask, args *T) (*Job, error) {
	data := ""
	if args != nil {
		bytes, err := json.Marshal(args)
		if err != nil {
			return nil, err
		}
		data = string(bytes)
	}

	// Set priority based on task type
	priority := PriorityLow // Default
	switch task {
	case TaskPreviewFrames:
		priority = PriorityHigh
	case TaskCut, TaskMerge, TaskAnalyzeFrames:
		priority = PriorityNormal
	case TaskEnhanceVideo, TaskConvert:
		priority = PriorityLow
	}

	job := &Job{
		ChannelID:   recording.ChannelID,
		ChannelName: recording.ChannelName,
		RecordingID: recording.RecordingID,
		Filename:    recording.Filename,
		Filepath:    recording.ChannelName.AbsoluteChannelFilePath(recording.Filename),
		Status:      StatusJobOpen,
		Task:        task,
		Priority:    priority,
		Args:        &data,
		Active:      false,
		CreatedAt:   time.Now(),
	}

	err := job.CreateJob()

	return job, err
}

func (recording *Recording) EnqueueConversionJob(mediaType string) (*Job, error) {
	return enqueueJob[string](recording, TaskConvert, &mediaType)
}

func (recording *Recording) EnqueuePreviewFramesJob() (*Job, error) {
	job, exists, err := JobExists(recording.RecordingID, TaskPreviewFrames)
	if err != nil {
		return job, err
	}
	if exists {
		return job, nil
	}
	return enqueueJob[*any](recording, TaskPreviewFrames, nil)
}

func (recording *Recording) EnqueueCuttingJob(args *util.CutArgs) (*Job, error) {
	return enqueueJob(recording, TaskCut, args)
}

func (recording *Recording) EnqueueAnalysisJob() (*Job, error) {
	job, exists, err := JobExists(recording.RecordingID, TaskAnalyzeFrames)
	if err != nil {
		return job, err
	}
	if exists {
		return job, nil
	}
	return enqueueJob[*any](recording, TaskAnalyzeFrames, nil)
}

func enqueueJob[T any](recording *Recording, task JobTask, args *T) (*Job, error) {
	if job, err := CreateJob(recording, task, args); err != nil {
		return nil, err
	} else {
		ws.BroadCastClients(ws.JobCreateEvent, job)
		return job, nil
	}
}

func EnqueueCuttingJob(id uint, args *util.CutArgs) (*Job, error) {
	if rec, err := RecordingID(id).FindRecordingByID(); err != nil {
		return nil, err
	} else {
		if job, err := rec.EnqueueCuttingJob(args); err != nil {
			return nil, err
		} else {
			return job, nil
		}
	}
}

func EnqueueMergeJob(channelID ChannelID, recordingIDs []uint, reEncode bool) (*Job, error) {
	if len(recordingIDs) == 0 {
		return nil, errors.New("no recording IDs provided for merge")
	}

	// Get first recording's channel to validate they all belong to same channel
	firstRec, err := RecordingID(recordingIDs[0]).FindRecordingByID()
	if err != nil {
		return nil, fmt.Errorf("failed to find recording %d: %w", recordingIDs[0], err)
	}

	if firstRec.ChannelID != channelID {
		return nil, errors.New("not all recordings belong to the specified channel")
	}

	// Create merge args with recording IDs
	mergeArgs := &util.MergeJobArgs{
		RecordingIDs: recordingIDs,
		ReEncode:     reEncode,
	}

	// Create the merge job attached to the first recording
	if job, err := CreateJob(firstRec, TaskMerge, mergeArgs); err != nil {
		return nil, err
	} else {
		ws.BroadCastClients(ws.JobCreateEvent, job)
		return job, nil
	}
}

func EnqueueEnhanceVideoJob(recordingID uint, args *util.EnhanceArgs) (*Job, error) {
	rec, err := RecordingID(recordingID).FindRecordingByID()
	if err != nil {
		return nil, fmt.Errorf("failed to find recording %d: %w", recordingID, err)
	}

	if job, err := CreateJob(rec, TaskEnhanceVideo, args); err != nil {
		return nil, err
	} else {
		ws.BroadCastClients(ws.JobCreateEvent, job)
		return job, nil
	}
}
