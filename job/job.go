package job

import (
	"time"

	log "github.com/binodlamsal/zerophish/logger"
	"github.com/google/uuid"
)

// Job is a generic job with its own state
type Job struct {
	id         string
	params     interface{}
	done       bool
	finishedAt time.Time
	progress   int
	errors     []error
	Done       chan bool
	Progress   chan int
	Errors     chan error
}

var jobs map[string]*Job

func init() {
	jobs = make(map[string]*Job)

	go func() {
		for {
			for _, job := range jobs {
				if job.done {
					continue
				}

				delay := time.NewTimer(100 * time.Millisecond)
				select {
				case progress := <-job.Progress:
					job.progress = progress
					log.Infof("[Job %s] Progress: %d\n", job.id, job.progress)
				case err := <-job.Errors:
					job.errors = append(job.errors, err)
					log.Infof("[Job %s] Error: %s\n", job.id, err.Error())
				case <-job.Done:
					job.done = true
					job.finishedAt = time.Now()
					log.Infof("[Job %s] Finished at %v\n", job.id, job.finishedAt)
				case <-delay.C:
				}
			}

			for id, job := range jobs {
				if job.done && job.finishedAt.Add(5*time.Second).Before(time.Now()) {
					delete(jobs, id)
					log.Infof("[Job %s] Deleted\n", job.id)
				}
			}

			time.Sleep(100 * time.Millisecond)
		}
	}()
}

// New creates a new job
func New(params interface{}) *Job {
	id := uuid.New().String()

	job := &Job{
		id:       id,
		params:   params,
		Done:     make(chan bool),
		Progress: make(chan int),
		Errors:   make(chan error),
	}

	jobs[id] = job
	return job
}

// Get finds and returns a job by its id or nil if not found
func Get(id string) *Job {
	if j, ok := jobs[id]; ok {
		return j
	}

	return nil
}

// ID returns ID of this job
func (j *Job) ID() string {
	return j.id
}

// GetProgress returns progress of this job
func (j *Job) GetProgress() int {
	return j.progress
}

// GetErrors returns errors of this job
func (j *Job) GetErrors() []string {
	errs := []string{}

	for _, e := range j.errors {
		errs = append(errs, e.Error())
	}

	return errs
}

// Start starts this job by running the given worker func
func (j *Job) Start(worker func(*Job)) {
	go worker(j)
}

// Params returns params of this job
func (j *Job) Params() interface{} {
	return j.params
}
