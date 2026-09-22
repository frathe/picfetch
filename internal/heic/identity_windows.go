package heic

import (
	"fmt"

	"golang.org/x/sys/windows"
)

func systemVersion() string {
	version := windows.RtlGetVersion()
	return fmt.Sprintf("%d.%d.%d", version.MajorVersion, version.MinorVersion, version.BuildNumber)
}
