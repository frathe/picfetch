package hevc

import "sync/atomic"

// DecodeBudget bounds the total coded pixel area processed by decoders sharing
// it, including pictures that are cropped, discarded or never output. A budget
// may be shared across concurrent grid/alpha decoders. A nil budget is unlimited.
type DecodeBudget struct {
	remaining atomic.Int64
}

// NewDecodeBudget creates a budget of n coded pixels. Negative n permits none.
func NewDecodeBudget(n int64) *DecodeBudget {
	b := &DecodeBudget{}
	b.remaining.Store(max(n, 0))
	return b
}

func (b *DecodeBudget) take(n int64) bool {
	if b == nil {
		return true
	}
	for {
		left := b.remaining.Load()
		if n > left || n <= 0 {
			return false
		}
		if b.remaining.CompareAndSwap(left, left-n) {
			return true
		}
	}
}

// Budget attaches an aggregate work budget without resetting its consumption.
// Configure the decoder before calling DecodeNAL.
func (d *Decoder) Budget(b *DecodeBudget) { d.budget = b }
