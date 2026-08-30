//go:build windows

package update

import (
	"math"
	"syscall"
	"testing"
	"time"
)

func TestWaitMilliseconds_NeverConvertsToAnUnboundedWait(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
		want    uint32
	}{
		{"negative", -time.Second, 0},
		{"zero", 0, 0},
		{"below a millisecond", 999 * time.Microsecond, 0},
		{"one millisecond", time.Millisecond, 1},
		{"the predecessor timeout", awaitPredecessorTimeout, 15000},
		{"one below infinite", (syscall.INFINITE - 1) * time.Millisecond, syscall.INFINITE - 1},
		{"exactly infinite", syscall.INFINITE * time.Millisecond, syscall.INFINITE - 1},
		{"beyond infinite", math.MaxInt64, syscall.INFINITE - 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := waitMilliseconds(tt.timeout); got != tt.want {
				t.Errorf("waitMilliseconds(%v) = %d, want %d", tt.timeout, got, tt.want)
			}
		})
	}
}
