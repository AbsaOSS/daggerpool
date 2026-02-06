package readiness

import (
	"fmt"
)

type Checker interface {
	// Wait waits for the given resource to be ready for the given amount of time.
	// (false, nil) if the resource is not ready yet,
	// (true, nil) if the resource is ready
	// (false, error) if there is an error or timeouts
	Wait(id fmt.Stringer, minutes int32, readyFunc func() (bool, error)) error

	// IsReady waits for the given resource to be ready for the given amount of time.
	// (InProgress, nil) if the resource is not ready yet,
	// (Ready, nil) if the resource is ready
	// (Error, error) if there is an error
	IsReady(id fmt.Stringer, readyFunc func() (bool, error)) (ReadyStatus, error)
}

type ReadyStatus int

const (
	InProgress ReadyStatus = iota
	Ready
	Error
)

func (r ReadyStatus) IsReady() bool {
	return r == Ready
}

func (r ReadyStatus) IsNotReady() bool {
	return !r.IsReady()
}

func (r ReadyStatus) Failed() bool {
	return r == Error
}

func (r ReadyStatus) AsFuncResult(err error) (bool, error) {
	switch r {
	case InProgress:
		return false, nil
	case Ready:
		return true, nil
	case Error:
		return false, err
	}
	return false, fmt.Errorf("unexpected status: %v", r)
}

type Ref struct {
	Namespace string
	Name      string
}

func (r Ref) String() string {
	if r.Namespace == "" {
		return r.Name
	}
	return r.Namespace + "." + r.Name
}
