package wincom

import "fmt"

// FailedHRESULT reports whether an HRESULT has its failure bit set.
func FailedHRESULT(value uintptr) bool {
	return int32(uint32(value)) < 0
}

// MonitorAttached interprets an IDesktopWallpaper::GetMonitorRECT result.
// Native calls remain in the platform adapters; this protocol check is shared
// by enumeration and wallpaper preflight and can be exercised on every host.
func MonitorAttached(result uintptr) (bool, error) {
	switch uint32(result) {
	case 0: // S_OK: the returned rectangle describes an attached display.
		return true, nil
	case 1: // S_FALSE: Windows retains this detached display's wallpaper.
		return false, nil
	default:
		return false, fmt.Errorf("HRESULT 0x%08x", uint32(result))
	}
}
