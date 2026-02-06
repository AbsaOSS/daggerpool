package readiness

import (
	"context"
	"fmt"
)

// Checker checks readiness and returns a Status.
// It may be a single probe, watch or a polling wait, depending on implementation.
type Checker interface {
	Check(ctx context.Context, id fmt.Stringer, check func(context.Context) (bool, error)) (ReadyStatus, error)
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
