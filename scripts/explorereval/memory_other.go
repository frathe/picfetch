//go:build !darwin

package main

import "fmt"

func peakRSS() (int64, error) {
	return 0, fmt.Errorf("native memory measurement requires the Mac trial runtime")
}
