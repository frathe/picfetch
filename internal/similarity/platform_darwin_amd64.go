package similarity

import "syscall"

func runtimeVersionSupported() bool {
	version, err := syscall.Sysctl("kern.osproductversion")
	return err == nil && supportsIntelMacOS(version)
}
