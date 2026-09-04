//go:build !darwin && !windows && !linux

package displays

func platformInspect(_ any) ([]Display, ID, error) {
	return nil, "", &UnsupportedError{Reason: "native adapter is unavailable"}
}
