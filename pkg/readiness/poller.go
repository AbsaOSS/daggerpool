package readiness

import (
	"context"
	"fmt"
	"time"
)

// Poller repeatedly calls the underlying Checker at the configured period until Ready,
// Error, or context cancellation/timeout.
// prober := readiness.NewProber(1 * time.Minute)   // single-check timeout
// poller := readiness.NewPoller(5*time.Second, prober) // polling loop
type Poller struct {
	period  time.Duration
	checker Checker
}

func NewPoller(period time.Duration, checker Checker) *Poller {
	return &Poller{
		period:  period,
		checker: checker,
	}
}

func (p *Poller) Check(ctx context.Context, id fmt.Stringer, waitFunc func(context.Context) (bool, error)) (ReadyStatus, error) {
	ticker := time.NewTicker(p.period)
	defer ticker.Stop()

	// Immediate first check.
	status, err := p.checker.Check(ctx, id, waitFunc)
	if status == Ready || status == Error {
		return status, err
	}

	for {
		select {
		case <-ctx.Done():
			return Error, fmt.Errorf("poller context done for %s: %w", id, ctx.Err())
		case <-ticker.C:
			status, err := p.checker.Check(ctx, id, waitFunc)
			if err != nil {
				return Error, err
			}
			if status == Ready {
				return Ready, nil
			}
			// InProgress -> continue polling
		}
	}
}
