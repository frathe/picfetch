package heic

import "golang.org/x/sys/unix"

func systemVersion() string {
	version, err := unix.Sysctl("kern.osrelease")
	if err != nil {
		return ""
	}
	return version
}
