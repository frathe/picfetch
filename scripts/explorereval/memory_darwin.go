package main

import "syscall"

func peakRSS() (int64, error) {
	var usage syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &usage); err != nil {
		return 0, err
	}
	return usage.Maxrss, nil // Darwin reports bytes; this is an OS high-water mark, not a sample.
}
