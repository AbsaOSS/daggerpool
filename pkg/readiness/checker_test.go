package readiness

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWait(t *testing.T) {
	// arrange
	ref := Ref{Name: "unit", Namespace: "cp"}
	i := 0
	tests := []struct {
		name          string
		f             func(context.Context) (bool, error)
		minutes       int32
		expectedError string
	}{
		{
			name:          "Test Timeout",
			f:             func(_ context.Context) (bool, error) { return true, nil },
			minutes:       0,
			expectedError: fmt.Sprintf("timeout! 0 minutes expired %s.%s", ref.Namespace, ref.Name),
		},
		{
			name:          "Test Wait Pass",
			f:             func(_ context.Context) (bool, error) { return true, nil },
			minutes:       1,
			expectedError: "",
		},
		{
			name:          "Test Wait dindt Pass",
			f:             func(_ context.Context) (bool, error) { return true, fmt.Errorf("blank error") },
			minutes:       1,
			expectedError: fmt.Sprintf("readiness check error for %s: blank error", ref),
		},
		{
			name: "Test Wait Pass after while",
			f: func(_ context.Context) (bool, error) {
				i++
				return i == 2, nil
			},
			minutes:       1,
			expectedError: "",
		},
	}
	// act
	// assert
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			status, err := NewPoller(time.Second*1, NewProber(time.Second*10)).Check(context.TODO(), ref, test.f)
			if err != nil {
				assert.True(t, err.Error() == test.expectedError)
				assert.Equal(t, Error, status)
				return
			}
			assert.Equal(t, Ready, status)
		})
	}
}

func TestIsReady(t *testing.T) {
	// arrange
	ref := Ref{Name: "unit", Namespace: "cp"}
	i := 0
	tests := []struct {
		name           string
		f              func(context.Context) (bool, error)
		expectedError  string
		expectedStatus ReadyStatus
		counter        int
	}{
		{
			name:           "Test IsReady dindt Pass on error",
			f:              func(_ context.Context) (bool, error) { return true, fmt.Errorf("blank error") },
			expectedError:  fmt.Sprintf("readiness check error for %s: blank error", ref),
			expectedStatus: Error,
			counter:        1,
		},
		{
			name: "Test IsReady didnt Pass after first  hit",
			f: func(_ context.Context) (bool, error) {
				i++
				return i >= 2, nil
			},
			expectedError:  "",
			expectedStatus: InProgress,
			counter:        1,
		},
		{
			name: "Test IsReady Pass after second hit",
			f: func(_ context.Context) (bool, error) {
				i++
				return i >= 2, nil
			},
			expectedError:  "",
			expectedStatus: Ready,
			counter:        2,
		},
		{
			name: "Test IsReady Pass after third hit",
			f: func(_ context.Context) (bool, error) {
				i++
				return i >= 2, nil
			},
			expectedError:  "",
			expectedStatus: Ready,
			counter:        2,
		},
	}
	// act
	// assert
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			cw := NewProber(time.Minute) // NewPoller()
			var s ReadyStatus
			var err error
			for x := 0; x < test.counter; x++ {
				s, err = cw.Check(context.TODO(), ref, test.f)
			}
			if err != nil {
				assert.True(t, err.Error() == test.expectedError)
			}
			assert.Equal(t, test.expectedStatus, s)
		})
	}
}
