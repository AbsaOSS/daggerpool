//nolint:lll
package workerpool

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// DAG - Direct Acyclic Graph
type DAG map[string][]string

type WorkerPool interface {
	Start() DAGResult
}

type WorkerPoolImpl struct {
	maxConcurrentJobs int
	dag               DAG
	jobs              map[string]Job
	totalJobs         int
	logger            *zerolog.Logger
	timeout           time.Duration
}

func NewWorkerPool(maxConcurrentJobs int, jobs []Job, dag DAG, logger *zerolog.Logger, timeout time.Duration) (WorkerPool, error) {
	mJob := make(map[string]Job)
	for _, job := range jobs {
		mJob[job.String()] = job
	}

	if err := validate(maxConcurrentJobs, dag, mJob); err != nil {
		return nil, err
	}
	return &WorkerPoolImpl{
		maxConcurrentJobs: maxConcurrentJobs,
		dag:               dag,
		jobs:              mJob,
		totalJobs:         len(dag),
		logger:            logger,
		timeout:           timeout,
	}, nil
}

func (w *WorkerPoolImpl) Start() DAGResult {
	ctx, cancel := context.WithTimeout(context.Background(), w.timeout)
	defer cancel()
	w.logger.Info().Msg("Starting workerpool...")
	jobsCh := make(chan Job, len(w.jobs))
	resultsCh := make(chan Result, len(w.jobs))
	orchestratorDone := make(chan struct{})

	results := newDAGResult(w.dag)
	if len(w.dag) == 0 {
		return results
	}

	var wg sync.WaitGroup

	wg.Add(w.maxConcurrentJobs)

	for i := 0; i < w.maxConcurrentJobs; i++ {
		go w.worker(ctx, i, jobsCh, resultsCh, &wg)
	}

	// Close results channel when all workers are done.
	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	remainingDeps := w.remainingDeps()

	reverseDeps := w.reversedDeps()

	schedule := func(name string) {
		select {
		case <-ctx.Done():
			return
		case jobsCh <- w.jobs[name]:
			w.logger.Debug().
				Str("name", name).
				Msg("Scheduled job")
		}
	}

	// Orchestrator.
	go func() {
		defer close(orchestratorDone)
		defer close(jobsCh)

		for name := range w.jobs {
			if value, ok := remainingDeps[name]; ok && value == 0 {
				//	results[name].Success()
				schedule(name)
			}
		}

		completed := 0

		for res := range resultsCh {
			completed++

			if res.Err != nil {
				w.logger.Err(res.Err).
					Str("name", res.JobName).
					Msg("job FAILED")
				// On first failure we cancel the whole pipeline, we don't care about completed jobs.
				results[res.JobName].Fail(res.Err)
				cancel()
				continue
			}

			if !res.IsReady {
				w.logger.Info().
					Str("name", res.JobName).
					Msg("job NOT READY")
				// On first NOT READY we paint subtree to SKIP and jobs will not be scheduled
				// The root Job we set to NOT_READY.
				// Worker now returs Result with Skip flag, so we continue in graph processing,
				// so completed Jobs will fit at the end but Do() func is not called
				// -1 because I already have one job InProcess
				completed += w.skipAllSuccessors(results, res.JobName, reverseDeps) - 1
				results[res.JobName].NotReady()
			} else {
				results[res.JobName].Success()
			}

			// Reduce dependency counters for all successors.
			for _, succ := range reverseDeps[res.JobName] {
				remainingDeps[succ]--
				if remainingDeps[succ] == 0 && results[succ].Status != JobStatusSkipped {
					schedule(succ)
				}
			}

			if completed == w.totalJobs {
				w.logger.Info().Msg("All jobs completed successfully")
				cancel()
			}
		}

	}()
	<-ctx.Done()
	<-orchestratorDone
	err := context.Cause(ctx)
	if errors.Is(err, context.DeadlineExceeded) {
		results.TimeoutError(w.timeout)
	}
	return results

}

func (w *WorkerPoolImpl) remainingDeps() map[string]int {
	remainingDeps := make(map[string]int, len(w.jobs))
	for jobName := range w.dag {
		remainingDeps[jobName] = len(w.dag[jobName])
	}
	return remainingDeps
}

func (w *WorkerPoolImpl) reversedDeps() DAG {
	reversedDeps := make(DAG, len(w.jobs))
	for jobName, deps := range w.dag {
		for _, dep := range deps {
			reversedDeps[dep] = append(reversedDeps[dep], jobName)
		}
	}
	return reversedDeps
}

// Worker reads jobs from the jobs channel, executes them and sends results.
func (w *WorkerPoolImpl) worker(ctx context.Context, id int, jobs <-chan Job, results chan<- Result, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			w.logger.Debug().
				Int("woker", id).
				Msg("exiting (context cancelled)")
			return
		case job, ok := <-jobs:
			if !ok {
				w.logger.Debug().
					Int("woker", id).
					Msg("exiting (context cancelled)")
				return
			}
			w.logger.Info().
				Str("job", job.String()).
				Msg("running job")
			isready, err := job.Do(ctx)
			// If Job was cancelled during execution I dont want to send anything into results Channel,
			// so job can remain in JobStatusUnknown state
			if ctx.Err() != nil {
				w.logger.Info().
					Str("job", job.String()).
					Msg("context canceled")
				return
			}
			w.logger.Info().
				Str("job", job.String()).
				Bool("isready", isready).
				Err(err).
				Msg("completed")

			results <- Result{JobName: job.String(), Err: err, IsReady: isready}
		}
	}
}

func validate(maxConcurrentJobs int, dag DAG, jobs map[string]Job) error {
	if maxConcurrentJobs < 1 {
		return fmt.Errorf("maxConcurrentJobs must be greater than zero")
	}

	for job, deps := range dag {
		if _, found := jobs[job]; !found {
			return fmt.Errorf("DAG job %s not found in jobs", job)
		}
		for _, dep := range deps {
			if _, found := jobs[dep]; !found {
				return fmt.Errorf("DAG job %s not found in jobs", dep)
			}
		}

		for _, dep := range deps {
			if dep == job {
				return fmt.Errorf("DAG job %s contains %s dependency", dep, job)
			}
		}
	}

	return nil
}

func (w *WorkerPoolImpl) skipAllSuccessors(results DAGResult, name string, succs DAG) int {
	cnt := 0
	if results[name].Status == JobStatusUnknown {
		results[name].Skip()
		cnt++
	}

	for _, succ := range succs[name] {
		cnt += w.skipAllSuccessors(results, succ, succs)
	}
	return cnt
}
