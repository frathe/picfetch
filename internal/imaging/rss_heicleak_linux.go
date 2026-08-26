//go:build heicleak && linux

package imaging

import (
	"os"
	"strconv"
	"strings"
)

func readRSS() (uint64, bool) {
	data, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return 0, false
	}
	for line := range strings.SplitSeq(string(data), "\n") {
		if after, ok := strings.CutPrefix(line, "VmRSS:"); ok {
			fields := strings.Fields(after)
			if len(fields) == 0 {
				return 0, false
			}
			kb, err := strconv.ParseUint(fields[0], 10, 64)
			if err != nil {
				return 0, false
			}
			return kb * 1024, true
		}
	}
	return 0, false
}
