package workerpool

import (
	"fmt"
	"sort"
	"time"
)

type JobStatus int

const (
	JobStatusUnknown JobStatus = iota
	JobStatusSuccess
	JobStatusInProgress
	JobStatusSkipped
	JobStatusFailed
)

const timeoutError = "timeout_error"

// DAGResult contains results from orchestrator.
// collection is not touched from particular jobs so as resource can't be thread safe
// (only orchestrator can it)
type DAGResult map[string]*JobResult

type JobResult struct {
	Name   string
	Status JobStatus
	Error  error
}

func newDAGResult(dag DAG) DAGResult {
	result := make(map[string]*JobResult)
	for jobName := range dag {
		result[jobName] = &JobResult{
			Status: JobStatusUnknown,
			Name:   jobName,
		}
	}
	return result
}

func (j *JobResult) IsFailed() bool {
	return j.Status == JobStatusFailed
}

func (j *JobResult) IsSkipped() bool {
	return j.Status == JobStatusSkipped
}

func (j *JobResult) IsNotSkipped() bool {
	return !j.IsSkipped()
}

func (j *JobResult) IsInProgress() bool {
	return j.Status == JobStatusInProgress
}

func (j *JobResult) IsSuccessfull() bool {
	return j.Status == JobStatusSuccess
}

func (j *JobResult) Skip() {
	j.Status = JobStatusSkipped
}

func (j *JobResult) Success() {
	j.Status = JobStatusSuccess
}

func (j *JobResult) Fail(err error) {
	j.Status = JobStatusFailed
	j.Error = err
}

func (j *JobResult) NotReady() {
	j.Status = JobStatusInProgress
}

func (r DAGResult) TimeoutError(deadline time.Duration) {
	r[timeoutError] = &JobResult{
		Status: JobStatusFailed,
		Error:  fmt.Errorf("timeout after %v", deadline.Seconds()),
	}
}

func (r DAGResult) FirstError() error {
	for _, jobResult := range r {
		if jobResult.Error != nil {
			return jobResult.Error
		}
	}
	return nil
}

// IsFailed says result has Error(s)
func (r DAGResult) IsFailed() bool {
	return r.FirstError() != nil
}

func (r DAGResult) IsTimeouted() bool {
	_, found := r[timeoutError]
	return found
}

func (r DAGResult) IsReady() bool {
	for _, key := range r.orderedJobs() {
		if r[key].IsSkipped() || r[key].IsInProgress() || r[key].IsFailed() {
			return false
		}
	}
	return true
}

func (r DAGResult) IsNotReady() bool {
	return !r.IsReady()
}

func (r DAGResult) FirstInProgress() *JobResult {
	for _, key := range r.orderedJobs() {
		if r[key].IsInProgress() {
			return r[key]
		}
	}
	return nil
}

func (r DAGResult) IsSuccessful() bool {
	for _, result := range r {
		if result.Status != JobStatusSuccess {
			return false
		}
	}
	return true
}

func (r DAGResult) IsNotSuccessful() bool {
	return !r.IsSuccessful()
}

func (r DAGResult) orderedJobs() []string {
	keys := make([]string, 0)
	for job := range r {
		keys = append(keys, job)
	}
	sort.Strings(keys)
	return keys
}
