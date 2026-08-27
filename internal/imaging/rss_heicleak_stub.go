//go:build heicleak && !linux

package imaging

func readRSS() (uint64, bool) {
	return 0, false
}
