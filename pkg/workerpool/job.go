package workerpool

import (
	"context"
	"fmt"
)

type Job interface {
	fmt.Stringer
	Do(ctx context.Context) (bool, error)
}

type Result struct {
	JobName string
	Err     error
	IsReady bool
}
