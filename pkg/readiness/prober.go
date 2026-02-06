package readiness

import (
	"context"
	"fmt"
	"time"
)

// Prober performs a single readiness check and returns Ready/InProgress/Error.
// Each check is bounded by the timeout configured in the constructor.
//
// Usage:
//
//	status, err := readiness.NewProber(time.Minute).Check(ctx, id, fn)
type Prober struct {
	timeout time.Duration
}

func NewProber(proberTimeout time.Duration) *Prober {
	return &Prober{
		timeout: proberTimeout,
	}
}

func (p *Prober) Check(ctx context.Context, id fmt.Stringer, checkFunc func(context.Context) (bool, error)) (ReadyStatus, error) {
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()
	type result struct {
		ready bool
		err   error
	}
	resCh := make(chan result, 1)
	go func() {
		ready, err := checkFunc(ctx)
		if ctx.Err() != nil {
			return // drop result
		}
		resCh <- result{ready: ready, err: err}
	}()

	select {
	// Avoid blocking forever if checkFunc gets stuck.
	case <-ctx.Done():
		return Error, fmt.Errorf("readiness check canceled for %s after %s: %w", id, p.timeout, ctx.Err())
	case res := <-resCh:
		if res.err != nil {
			return Error, fmt.Errorf("readiness check error for %s: %w", id, res.err)
		}
		if res.ready {
			return Ready, nil
		}
		return InProgress, nil
	}
}
