package wincom

import "testing"

func TestMonitorAttached_DistinguishesDetachedStatus(t *testing.T) {
	// GetMonitorRECT documents S_FALSE for a detached display, even though
	// that HRESULT has no failure bit. It is not an attached empty rectangle.
	tests := []struct {
		name     string
		result   uintptr
		attached bool
		wantErr  bool
	}{
		{name: "attached S_OK", result: 0, attached: true},
		{name: "detached S_FALSE", result: 1},
		{name: "unknown ID E_INVALIDARG", result: 0x80070057, wantErr: true},
		{name: "invalid output E_POINTER", result: 0x80004003, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attached, err := MonitorAttached(tt.result)
			if attached != tt.attached || (err != nil) != tt.wantErr {
				t.Fatalf("MonitorAttached(0x%x) = (%v, %v), want attached=%v error=%v", tt.result, attached, err, tt.attached, tt.wantErr)
			}
		})
	}
}
