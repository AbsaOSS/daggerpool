package readiness

import (
	"context"
	"fmt"
	"time"
)

type Poller struct{}

func NewPoller() *Poller {
	return &Poller{}
}

func (w *Poller) Wait(id fmt.Stringer, minutes int32, readyFunc func() (bool, error)) error {
	ch := make(chan string)
	errch := make(chan error)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(minutes)*time.Minute)
	defer cancel()
	go func() {
		for range time.NewTicker(5 * time.Second).C {
			ready, err := readyFunc()
			if err != nil {
				errch <- err
			}
			if ready {
				break
			}
		}
		ch <- "Done"
	}()
	select {
	case <-ctx.Done():
		return fmt.Errorf("timeout! %v minutes expired %s", minutes, id)
	case <-ch:
		return nil
	case <-errch:
		return fmt.Errorf("error %s", id)
	}
}
