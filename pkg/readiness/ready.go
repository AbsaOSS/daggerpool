package readiness

import (
	"context"
	"fmt"
	"time"
)

// IsReady async function release thread when readyFunc waits for result from k8s api
// if readyfunc blocks it timeouts after 1 minute
func (w *Poller) IsReady(id fmt.Stringer, readyFunc func() (bool, error)) (ReadyStatus, error) {
	ch := make(chan bool)
	errch := make(chan error)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	go func() {
		ready, err := readyFunc()
		if err != nil {
			errch <- err
		}
		ch <- ready
	}()

	select {
	// just because readyFunc will block forever we do not want to block forever here
	case <-ctx.Done():
		return Error, fmt.Errorf("timeout! %v minutes expired %s", 1, id)
	case ready := <-ch:
		if !ready {
			return InProgress, nil
		}
		return Ready, nil
	case <-errch:
		return Error, fmt.Errorf("error %s", id)
	}
}
