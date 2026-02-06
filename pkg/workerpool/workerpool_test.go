//nolint:dupl
package workerpool

import (
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

const (
	managementVanilla    string = "management_vanilla"
	managementTerraform  string = "management_terraform"
	managementVanilla1   string = "management_vanilla_1"
	managementTerraform1 string = "management_terraform_1"
	managementBundle1    string = "management_terraform_bundle_1"
	capacityVanilla0     string = "capacity_vanilla_0"
	capacityTerraform0   string = "capacity_terraform_0"
	capacityVanilla1     string = "capacity_vanilla_1"
	capacityTerraform1   string = "capacity_terraform_1"
	capacityVanilla2     string = "capacity_vanilla_2"
	capacityVanilla3     string = "capacity_vanilla_3"
	capacityTerraform2   string = "capacity_terraform_2"
	capacityTerraform3   string = "capacity_terraform_3"
)

var (
	ready800ms = func(ctx context.Context) (bool, error) { time.Sleep(800 * time.Millisecond); return true, nil }
	ready400ms = func(ctx context.Context) (bool, error) { time.Sleep(400 * time.Millisecond); return true, nil }
	ready200ms = func(ctx context.Context) (bool, error) { time.Sleep(200 * time.Millisecond); return true, nil }
	ready100ms = func(ctx context.Context) (bool, error) { time.Sleep(100 * time.Millisecond); return true, nil }
	ready30ms  = func(ctx context.Context) (bool, error) { time.Sleep(30 * time.Millisecond); return true, nil }
)

var voidLogger = zerolog.New(io.Discard).With().Timestamp().Logger()

func TestWorkerPoolWithManyJobs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dag := DAG{
		capacityVanilla0:     {},
		capacityVanilla1:     {},
		capacityVanilla2:     {},
		capacityVanilla3:     {},
		managementVanilla:    {},
		managementTerraform:  {managementVanilla},
		capacityTerraform0:   {managementTerraform, capacityVanilla0},
		managementVanilla1:   {},
		managementBundle1:    {managementVanilla1},
		managementTerraform1: {managementBundle1},
		capacityTerraform1:   {capacityVanilla1, managementTerraform1},
		capacityTerraform2:   {capacityVanilla2, managementTerraform1},
		capacityTerraform3:   {managementTerraform1, capacityVanilla3, managementTerraform},
	}
	jobs := func() []Job {
		return []Job{
			job(ctrl, managementVanilla, ready100ms, true, nil),
			job(ctrl, managementVanilla1, ready100ms, true, nil),
			job(ctrl, capacityVanilla0, ready200ms, true, nil),
			job(ctrl, capacityVanilla1, ready200ms, true, nil),
			job(ctrl, capacityVanilla2, ready200ms, true, nil),
			job(ctrl, capacityVanilla3, ready200ms, true, nil),
			job(ctrl, managementTerraform, ready400ms, true, nil),
			job(ctrl, managementTerraform1, ready400ms, true, nil),
			job(ctrl, capacityTerraform0, ready100ms, true, nil),
			job(ctrl, capacityTerraform1, ready100ms, true, nil),
			job(ctrl, capacityTerraform2, ready100ms, true, nil),
			job(ctrl, capacityTerraform3, ready100ms, true, nil),
			job(ctrl, managementBundle1, ready100ms, true, nil),
		}
	}

	indexOf := func(key string) int {
		for idx, v := range jobs() {
			if v.String() == key {
				return idx
			}
		}
		panic(key + " not found")
	}

	tests := []struct {
		name              string
		assert            func(DAGResult)
		dag               DAG
		jobs              []Job
		init              func(DAG, []Job)
		timeout           time.Duration
		maxConcurrentJobs int
	}{
		{
			name: "Large graph succeed",
			assert: func(result DAGResult) {
				assert.Len(t, result, 13)
				assert.True(t, result.IsReady())
			},
			dag:               dag,
			jobs:              jobs(),
			init:              func(dag DAG, jobs []Job) {},
			timeout:           3 * time.Second,
			maxConcurrentJobs: 10,
		},
		{
			name: "Long running Bundle failed",
			assert: func(result DAGResult) {
				assert.Len(t, result, 13)
				assert.True(t, result.IsNotReady())
				assert.True(t, result.IsFailed())
				assert.True(t, result[managementVanilla].IsSuccessfull())
				assert.True(t, result[managementVanilla1].IsSuccessfull())
				assert.True(t, result[capacityVanilla0].IsSuccessfull())
				assert.True(t, result[capacityVanilla1].IsSuccessfull())
				assert.True(t, result[capacityVanilla3].IsSuccessfull())
				assert.True(t, result[managementTerraform].IsSuccessfull())
				assert.True(t, result[capacityTerraform0].IsSuccessfull())
				assert.True(t, result[managementBundle1].IsFailed())
				assert.True(t, result[managementTerraform1].Status == JobStatusUnknown)
				assert.True(t, result[capacityTerraform1].Status == JobStatusUnknown)
				assert.True(t, result[capacityTerraform2].Status == JobStatusUnknown)
				assert.True(t, result[capacityTerraform3].Status == JobStatusUnknown)
			},
			dag:  dag,
			jobs: jobs(),
			init: func(_ DAG, jobs []Job) {
				jobs[indexOf(managementBundle1)] = job(ctrl, managementBundle1, ready800ms, false, fmt.Errorf("FAIL"))
			},
			timeout:           3 * time.Second,
			maxConcurrentJobs: 10,
		},
		{
			name: "Short running Bundle failed",
			assert: func(result DAGResult) {
				assert.Len(t, result, 13)
				assert.True(t, result.IsNotReady())
				assert.True(t, result.IsFailed())
				assert.True(t, result[managementVanilla].IsSuccessfull())
				assert.True(t, result[managementVanilla1].IsSuccessfull())
				// TODO: race detected - fix
				// assert.True(t, result[capacityVanilla0].IsSuccessfull())
				// assert.True(t, result[capacityVanilla1].IsSuccessfull())
				// assert.True(t, result[capacityVanilla3].IsSuccessfull())
				assert.True(t, result[managementTerraform].Status == JobStatusUnknown)
				assert.True(t, result[capacityTerraform0].Status == JobStatusUnknown)
				assert.True(t, result[managementBundle1].IsFailed())
				assert.True(t, result[managementTerraform1].Status == JobStatusUnknown)
				assert.True(t, result[capacityTerraform1].Status == JobStatusUnknown)
				assert.True(t, result[capacityTerraform2].Status == JobStatusUnknown)
				assert.True(t, result[capacityTerraform3].Status == JobStatusUnknown)
			},
			dag:  dag,
			jobs: jobs(),
			init: func(_ DAG, jobs []Job) {
				jobs[indexOf(managementBundle1)] = job(ctrl, managementBundle1, ready100ms, false, fmt.Errorf("FAIL"))
			},
			timeout:           5 * time.Second,
			maxConcurrentJobs: 10,
		},
		{
			name: "Short running Bundle InProgress",
			assert: func(result DAGResult) {
				assert.True(t, result.IsNotReady())
				assert.True(t, result[managementVanilla].IsSuccessfull())
				assert.True(t, result[managementVanilla1].IsSuccessfull())
				assert.True(t, result[capacityVanilla0].IsSuccessfull())
				assert.True(t, result[capacityVanilla1].IsSuccessfull())
				assert.True(t, result[capacityVanilla3].IsSuccessfull())
				assert.True(t, result[managementTerraform].IsSuccessfull())
				assert.True(t, result[capacityTerraform0].IsSuccessfull())
				assert.True(t, result[managementBundle1].IsInProgress())
				assert.True(t, result[managementTerraform1].IsSkipped())
				assert.True(t, result[capacityTerraform1].IsSkipped())
				assert.True(t, result[capacityTerraform2].IsSkipped())
				assert.True(t, result[capacityTerraform3].IsSkipped())
				assert.Len(t, result, 13)
			},
			dag:  dag,
			jobs: jobs(),
			init: func(_ DAG, jobs []Job) {
				jobs[indexOf(managementBundle1)] = job(ctrl, managementBundle1, ready100ms, false, nil)
			},
			timeout:           3 * time.Second,
			maxConcurrentJobs: 10,
		},
		{
			name: "Long running Bundle InProgress",
			assert: func(result DAGResult) {
				assert.True(t, result.IsNotReady())
				assert.True(t, result[managementVanilla].IsSuccessfull())
				assert.True(t, result[managementVanilla1].IsSuccessfull())
				assert.True(t, result[capacityVanilla0].IsSuccessfull())
				assert.True(t, result[capacityVanilla1].IsSuccessfull())
				assert.True(t, result[capacityVanilla3].IsSuccessfull())
				assert.True(t, result[managementTerraform].IsSuccessfull())
				assert.True(t, result[capacityTerraform0].IsSuccessfull())
				assert.True(t, result[managementBundle1].IsInProgress())
				assert.True(t, result[managementTerraform1].IsSkipped())
				assert.True(t, result[capacityTerraform1].IsSkipped())
				assert.True(t, result[capacityTerraform2].IsSkipped())
				assert.True(t, result[capacityTerraform3].IsSkipped())
				assert.Len(t, result, 13)
			},
			dag:  dag,
			jobs: jobs(),
			init: func(_ DAG, jobs []Job) {
				jobs[indexOf(managementBundle1)] = job(ctrl, managementBundle1, ready800ms, false, nil)
			},
			timeout:           3 * time.Second,
			maxConcurrentJobs: 10,
		},
		{
			name: "Short running two jobs InProgress",
			assert: func(result DAGResult) {
				assert.True(t, result.IsNotReady())
				assert.True(t, result[managementVanilla].IsSuccessfull())
				assert.True(t, result[managementVanilla1].IsSuccessfull())
				assert.True(t, result[capacityVanilla0].IsSuccessfull())
				assert.True(t, result[capacityVanilla1].IsSuccessfull())
				assert.True(t, result[capacityVanilla3].IsSuccessfull())
				assert.True(t, result[managementTerraform].IsInProgress())
				assert.True(t, result[capacityTerraform0].IsSkipped())
				assert.True(t, result[managementBundle1].IsInProgress())
				assert.True(t, result[managementTerraform1].IsSkipped())
				assert.True(t, result[capacityTerraform1].IsSkipped())
				assert.True(t, result[capacityTerraform2].IsSkipped())
				assert.True(t, result[capacityTerraform3].IsSkipped())
				assert.Len(t, result, 13)
			},
			dag:  dag,
			jobs: jobs(),
			init: func(_ DAG, jobs []Job) {
				jobs[indexOf(managementBundle1)] = job(ctrl, managementBundle1, ready100ms, false, nil)
				jobs[indexOf(managementTerraform)] = job(ctrl, managementTerraform, ready100ms, false, nil)
			},
			timeout:           3 * time.Second,
			maxConcurrentJobs: 10,
		},
		{
			name: "Root nodes InProgress on two jobs",
			assert: func(result DAGResult) {
				assert.True(t, result.IsNotReady())
				assert.True(t, result[managementVanilla].IsInProgress())
				assert.True(t, result[managementVanilla1].IsInProgress())
				assert.True(t, result[capacityVanilla0].IsInProgress())
				assert.True(t, result[capacityVanilla1].IsInProgress())
				assert.True(t, result[capacityVanilla3].IsInProgress())
				assert.True(t, result[managementTerraform].IsSkipped())
				assert.True(t, result[capacityTerraform0].IsSkipped())
				assert.True(t, result[managementBundle1].IsSkipped())
				assert.True(t, result[managementTerraform1].IsSkipped())
				assert.True(t, result[capacityTerraform1].IsSkipped())
				assert.True(t, result[capacityTerraform2].IsSkipped())
				assert.True(t, result[capacityTerraform3].IsSkipped())
				assert.Len(t, result, 13)
			},
			dag:  dag,
			jobs: jobs(),
			init: func(_ DAG, jobs []Job) {
				jobs[indexOf(capacityVanilla0)] = job(ctrl, capacityVanilla0, ready100ms, false, nil)
				jobs[indexOf(capacityVanilla1)] = job(ctrl, capacityVanilla1, ready100ms, false, nil)
				jobs[indexOf(capacityVanilla2)] = job(ctrl, capacityVanilla2, ready100ms, false, nil)
				jobs[indexOf(capacityVanilla3)] = job(ctrl, capacityVanilla3, ready100ms, false, nil)
				jobs[indexOf(managementVanilla)] = job(ctrl, managementVanilla, ready100ms, false, nil)
				jobs[indexOf(managementVanilla1)] = job(ctrl, managementVanilla1, ready100ms, false, nil)
			},
			timeout:           10 * time.Second,
			maxConcurrentJobs: 2,
		},
		{
			name: "Root nodes InProgress on single job",
			assert: func(result DAGResult) {
				assert.True(t, result.IsNotReady())
				assert.True(t, result[managementVanilla].IsInProgress())
				assert.True(t, result[managementVanilla1].IsInProgress())
				assert.True(t, result[capacityVanilla0].IsInProgress())
				assert.True(t, result[capacityVanilla1].IsInProgress())
				assert.True(t, result[capacityVanilla3].IsInProgress())
				assert.True(t, result[managementTerraform].IsSkipped())
				assert.True(t, result[capacityTerraform0].IsSkipped())
				assert.True(t, result[managementBundle1].IsSkipped())
				assert.True(t, result[managementTerraform1].IsSkipped())
				assert.True(t, result[capacityTerraform1].IsSkipped())
				assert.True(t, result[capacityTerraform2].IsSkipped())
				assert.True(t, result[capacityTerraform3].IsSkipped())
				assert.Len(t, result, 13)
			},
			dag:  dag,
			jobs: jobs(),
			init: func(_ DAG, jobs []Job) {
				jobs[indexOf(capacityVanilla0)] = job(ctrl, capacityVanilla0, ready100ms, false, nil)
				jobs[indexOf(capacityVanilla1)] = job(ctrl, capacityVanilla1, ready100ms, false, nil)
				jobs[indexOf(capacityVanilla2)] = job(ctrl, capacityVanilla2, ready100ms, false, nil)
				jobs[indexOf(capacityVanilla3)] = job(ctrl, capacityVanilla3, ready100ms, false, nil)
				jobs[indexOf(managementVanilla)] = job(ctrl, managementVanilla, ready100ms, false, nil)
				jobs[indexOf(managementVanilla1)] = job(ctrl, managementVanilla1, ready100ms, false, nil)
			},
			timeout:           10 * time.Second,
			maxConcurrentJobs: 1,
		},
		{
			name: "Very Short running two jobs InProgress",
			assert: func(result DAGResult) {
				assert.True(t, result.IsNotReady())
				assert.True(t, result[managementVanilla].IsSuccessfull())
				assert.True(t, result[managementVanilla1].IsSuccessfull())
				assert.True(t, result[capacityVanilla0].IsSuccessfull())
				assert.True(t, result[capacityVanilla1].IsSuccessfull())
				assert.True(t, result[capacityVanilla3].IsSuccessfull())
				assert.True(t, result[managementTerraform].IsInProgress())
				assert.True(t, result[capacityTerraform0].IsSkipped())
				assert.True(t, result[managementBundle1].IsInProgress())
				assert.True(t, result[managementTerraform1].IsSkipped())
				assert.True(t, result[capacityTerraform1].IsSkipped())
				assert.True(t, result[capacityTerraform2].IsSkipped())
				assert.True(t, result[capacityTerraform3].IsSkipped())
				assert.Len(t, result, 13)
			},
			dag:  dag,
			jobs: jobs(),
			init: func(_ DAG, jobs []Job) {
				jobs[indexOf(managementVanilla)] = job(ctrl, managementVanilla, ready30ms, true, nil)
				jobs[indexOf(managementVanilla1)] = job(ctrl, managementVanilla1, ready30ms, true, nil)
				jobs[indexOf(capacityVanilla0)] = job(ctrl, capacityVanilla0, ready30ms, true, nil)
				jobs[indexOf(capacityVanilla1)] = job(ctrl, capacityVanilla1, ready30ms, true, nil)
				jobs[indexOf(capacityVanilla2)] = job(ctrl, capacityVanilla2, ready30ms, true, nil)
				jobs[indexOf(capacityVanilla3)] = job(ctrl, capacityVanilla3, ready30ms, true, nil)
				jobs[indexOf(managementTerraform)] = job(ctrl, managementTerraform, ready30ms, false, nil)
				jobs[indexOf(capacityTerraform0)] = job(ctrl, capacityTerraform0, ready30ms, true, nil)
				jobs[indexOf(managementBundle1)] = job(ctrl, managementBundle1, ready30ms, false, nil)
				jobs[indexOf(managementTerraform1)] = job(ctrl, managementTerraform1, ready30ms, true, nil)
				jobs[indexOf(capacityTerraform1)] = job(ctrl, capacityTerraform1, ready30ms, true, nil)
				jobs[indexOf(capacityTerraform2)] = job(ctrl, capacityTerraform2, ready30ms, true, nil)
				jobs[indexOf(capacityTerraform3)] = job(ctrl, capacityTerraform3, ready30ms, true, nil)
			},
			timeout:           3 * time.Second,
			maxConcurrentJobs: 10,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.init(test.dag, test.jobs)
			w, err := NewWorkerPool(test.maxConcurrentJobs, test.jobs, test.dag, &voidLogger, test.timeout)
			assert.NoError(t, err)
			r := w.Start()
			test.assert(r)
		})
	}

}

func TestWorkerPoolWithManagementClusterOnly(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dag := DAG{
		managementVanilla:   {},
		managementTerraform: {managementVanilla},
	}
	tests := []struct {
		name    string
		assert  func(DAGResult)
		dag     DAG
		jobs    []Job
		timeout time.Duration
	}{
		{
			name: "MGMT Jobs ready",
			assert: func(result DAGResult) {
				assert.Len(t, result, 2)
				assert.True(t, result.IsSuccessful())
				assert.False(t, result.IsTimeouted())
				assert.True(t, result.IsReady())
			},
			jobs: []Job{
				job(ctrl, managementVanilla, ready100ms, true, nil),
				job(ctrl, managementTerraform, ready100ms, true, nil),
			},
			dag:     dag,
			timeout: time.Second * 3,
		},
		{
			name: "MGMT Terraform Job Failed",
			assert: func(result DAGResult) {
				assert.Len(t, result, 2)
				assert.True(t, result.IsFailed())
				assert.Equal(t, "FAIL", result.FirstError().Error())
				assert.True(t, result.IsNotSuccessful())
				assert.False(t, result.IsTimeouted())
				assert.False(t, result.IsReady())
				assert.True(t, result[managementVanilla].IsSuccessfull())
				assert.False(t, result[managementTerraform].IsSkipped())
				assert.True(t, result[managementTerraform].IsFailed())
			},
			dag: dag,
			jobs: []Job{
				job(ctrl, managementVanilla, ready100ms, true, nil),
				job(ctrl, managementTerraform, ready100ms, true, fmt.Errorf("FAIL")),
			},
			timeout: time.Second * 3,
		},
		{
			name: "MGMT Vanilla Job Failed",
			assert: func(result DAGResult) {
				assert.Len(t, result, 2)
				assert.True(t, result.IsFailed())
				assert.Equal(t, "FAIL", result.FirstError().Error())
				assert.True(t, result.IsNotSuccessful())
				assert.False(t, result.IsTimeouted())
				assert.True(t, result.IsNotReady())
				assert.True(t, result[managementVanilla].IsFailed())
				assert.Equal(t, result[managementTerraform].Status, JobStatusUnknown)
			},
			dag: dag,
			jobs: []Job{
				job(ctrl, managementVanilla, ready100ms, false, fmt.Errorf("FAIL")),
				job(ctrl, managementTerraform, ready100ms, true, nil),
			},
			timeout: time.Second * 3,
		},
		{
			name: "MGMT Vanilla Job is in progress",
			assert: func(result DAGResult) {
				assert.Len(t, result, 2)
				assert.False(t, result.IsFailed())
				assert.Nil(t, result.FirstError())
				assert.True(t, result.IsNotSuccessful())
				assert.False(t, result.IsTimeouted())
				assert.True(t, result.IsNotReady())
				assert.True(t, result[managementVanilla].IsInProgress())
				assert.True(t, result[managementTerraform].IsSkipped())
			},
			dag: dag,
			jobs: []Job{
				job(ctrl, managementVanilla, ready100ms, false, nil),
				job(ctrl, managementTerraform, ready100ms, true, nil),
			},
			timeout: time.Second * 3,
		},
		{
			name: "MGMT Terraform Job is in progress",
			assert: func(result DAGResult) {
				assert.Len(t, result, 2)
				assert.False(t, result.IsFailed())
				assert.Nil(t, result.FirstError())
				assert.True(t, result.IsNotSuccessful())
				assert.False(t, result.IsTimeouted())
				assert.True(t, result.IsNotReady())
				assert.True(t, result[managementVanilla].IsSuccessfull())
				assert.True(t, result[managementTerraform].IsInProgress())
			},
			dag: dag,
			jobs: []Job{
				job(ctrl, managementVanilla, ready100ms, true, nil),
				job(ctrl, managementTerraform, ready100ms, false, nil),
			},
			timeout: time.Second * 3,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w, _ := NewWorkerPool(10, test.jobs, test.dag, &voidLogger, test.timeout)
			r := w.Start()
			test.assert(r)
		})
	}
}

func TestWorkerPoolCornerCases(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	jobs := func() []Job {
		return []Job{
			job(ctrl, managementVanilla, ready100ms, true, nil),
			job(ctrl, managementTerraform, ready100ms, true, nil),
			job(ctrl, capacityVanilla0, ready200ms, true, nil),
			job(ctrl, capacityTerraform0, ready200ms, true, nil),
		}
	}

	tests := []struct {
		name          string
		expectedError bool
		assert        func(DAGResult)
		init          func(DAG, []Job)
		dag           DAG
		jobs          []Job
		timeout       time.Duration
	}{
		{
			name: "empty jobs, empty graph",
			dag:  DAG{},
			jobs: []Job{},
			assert: func(result DAGResult) {
				assert.Len(t, result, 0)
				assert.True(t, result.IsSuccessful())
				assert.False(t, result.IsTimeouted())
				assert.True(t, result.IsReady())
			},
			init:    func(DAG, []Job) {},
			timeout: time.Second * 3,
		},
		{
			name: "empty graph",
			dag:  DAG{},
			jobs: jobs(),
			assert: func(result DAGResult) {
				assert.Len(t, result, 0)
				assert.True(t, result.IsSuccessful())
				assert.False(t, result.IsTimeouted())
				assert.True(t, result.IsReady())
			},
			init:    func(DAG, []Job) {},
			timeout: time.Second * 3,
		},
		{
			name: "one job graph is READY",
			dag:  DAG{managementVanilla: {}},
			jobs: []Job{job(ctrl, managementVanilla, ready100ms, true, nil)},
			assert: func(result DAGResult) {
				assert.Len(t, result, 1)
				assert.True(t, result.IsSuccessful())
				assert.True(t, result.IsReady())
				assert.False(t, result.IsTimeouted())
			},
			init:    func(DAG, []Job) {},
			timeout: time.Second,
		},
		{
			name: "one job graph IS NOT READY",
			dag:  DAG{managementVanilla: {}},
			jobs: []Job{job(ctrl, managementVanilla, ready100ms, false, nil)},
			assert: func(result DAGResult) {
				assert.Len(t, result, 1)
				assert.False(t, result.IsFailed())
				assert.True(t, result.IsNotSuccessful())
				assert.True(t, result.IsNotReady())
				assert.False(t, result[managementVanilla].IsSkipped())
				assert.True(t, result[managementVanilla].IsInProgress())
				assert.False(t, result.IsTimeouted())
			},
			init:    func(DAG, []Job) {},
			timeout: time.Second,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.init(test.dag, test.jobs)
			w, err := NewWorkerPool(10, test.jobs, test.dag, &voidLogger, test.timeout)
			if test.expectedError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			r := w.Start()
			test.assert(r)
		})
	}
}

func TestWorkerPoolValidations(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	jobs := func() []Job {
		return []Job{
			job(ctrl, managementVanilla, ready100ms, true, nil),
			job(ctrl, managementTerraform, ready100ms, true, nil),
			job(ctrl, capacityVanilla0, ready200ms, true, nil),
			job(ctrl, capacityTerraform0, ready200ms, true, nil),
		}
	}

	tests := []struct {
		name              string
		expectedError     bool
		dag               DAG
		jobs              []Job
		timeout           time.Duration
		maxConcurrentJobs int
	}{
		{
			name:              "empty jobs, one graph",
			dag:               DAG{capacityVanilla1: {}},
			jobs:              []Job{},
			timeout:           time.Second * 3,
			expectedError:     true,
			maxConcurrentJobs: 10,
		},
		{
			name:              "jobs and dag doesnt match no.1",
			dag:               DAG{capacityVanilla1: {"non_existing_job"}},
			jobs:              jobs(),
			timeout:           time.Second * 3,
			expectedError:     true,
			maxConcurrentJobs: 10,
		},
		{
			name:              "jobs and dag doesnt match no.2",
			dag:               DAG{"non_existing_job": {capacityVanilla1}},
			jobs:              jobs(),
			timeout:           time.Second * 3,
			expectedError:     true,
			maxConcurrentJobs: 10,
		},
		{
			name: "identify cycle",
			dag: DAG{
				capacityVanilla0:    {},
				managementVanilla:   {},
				managementTerraform: {managementVanilla},
				capacityTerraform0:  {capacityVanilla0, managementTerraform, capacityTerraform0},
			},
			jobs:              jobs(),
			timeout:           time.Second * 3,
			expectedError:     true,
			maxConcurrentJobs: 10,
		},
		{
			name: "0 concurrent jobs",
			dag: DAG{
				capacityVanilla0:    {},
				managementVanilla:   {},
				managementTerraform: {managementVanilla},
				capacityTerraform0:  {capacityVanilla0, managementTerraform},
			},
			jobs:              jobs(),
			timeout:           time.Second * 3,
			expectedError:     true,
			maxConcurrentJobs: 0,
		},
		{
			name: "1 concurrent jobs",
			dag: DAG{
				capacityVanilla0:    {},
				managementVanilla:   {},
				managementTerraform: {managementVanilla},
				capacityTerraform0:  {capacityVanilla0, managementTerraform},
			},
			jobs:              jobs(),
			timeout:           time.Second * 3,
			expectedError:     false,
			maxConcurrentJobs: 1,
		},
	}
	{
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				_, err := NewWorkerPool(test.maxConcurrentJobs, test.jobs, test.dag, &voidLogger, test.timeout)
				if test.expectedError {
					assert.Error(t, err)
					return
				}
				assert.NoError(t, err)
			})
		}
	}
}

func TestDAGTimeout(t *testing.T) {
	// arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	dag := DAG{
		capacityVanilla0:    {},
		managementVanilla:   {},
		managementTerraform: {managementVanilla},
		capacityTerraform0:  {capacityVanilla0, managementTerraform},
	}

	jobs := func() []Job {
		return []Job{
			job(ctrl, managementVanilla, ready100ms, true, nil),
			job(ctrl, capacityVanilla0, ready200ms, true, nil),
			job(ctrl, managementTerraform, ready400ms, true, nil),
			job(ctrl, capacityTerraform0, ready100ms, true, nil),
		}
	}

	// act
	w, err := NewWorkerPool(10, jobs(), dag, &voidLogger, time.Millisecond*400)
	assert.NoError(t, err)
	result := w.Start()

	// assert
	assert.Len(t, result, 5)
	assert.True(t, result.IsNotSuccessful())
	assert.True(t, result.IsTimeouted())
	assert.Error(t, result.FirstError())
	assert.NoError(t, result[managementTerraform].Error)
	assert.NoError(t, result[managementVanilla].Error)
	assert.NoError(t, result[capacityVanilla0].Error)
	assert.NoError(t, result[capacityTerraform0].Error)

}

// starts workerpool multiple times and checks if it returns stable results
func TestDAGIsStable(t *testing.T) {
	// arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dag := DAG{
		capacityVanilla0:    {},
		managementVanilla:   {},
		managementTerraform: {managementVanilla},
		capacityTerraform0:  {capacityVanilla0, managementTerraform},
	}

	jobs := func() []Job {
		return []Job{
			job(ctrl, managementVanilla, ready100ms, true, nil),
			job(ctrl, capacityVanilla0, ready200ms, true, nil),
			job(ctrl, managementTerraform, ready400ms, true, nil),
			job(ctrl, capacityTerraform0, ready100ms, true, nil),
		}
	}

	// jobs, dag := cpoWithOneCapacity(ctrl)

	// act
	w, err := NewWorkerPool(10, jobs(), dag, &voidLogger, time.Second*200)
	assert.NoError(t, err)

	for i := 0; i < 2; i++ {
		result := w.Start()
		// assert
		assert.True(t, result.IsSuccessful())
		assert.False(t, result.IsTimeouted())
		assert.True(t, result[capacityVanilla0].IsSuccessfull())
		assert.True(t, result[capacityTerraform0].IsSuccessfull())
		assert.True(t, result[managementVanilla].IsSuccessfull())
		assert.True(t, result[managementTerraform].IsSuccessfull())
		assert.Len(t, result, 4)
	}
}

func job(ctrl *gomock.Controller, name string, work func(ctx context.Context) (bool, error), isReady bool, err error) Job {
	j := NewMockJob(ctrl)
	j.EXPECT().Do(gomock.Any()).DoAndReturn(work).Return(isReady, err).AnyTimes()
	j.EXPECT().String().Return(name).AnyTimes()
	return j
}
