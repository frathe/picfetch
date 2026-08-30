//go:build !windows

package update

// classifyPlatform has no Windows errno to inspect off Windows;
// ClassifyApplyError falls through to its portable fs.ErrPermission check.
func classifyPlatform(_ error) (FailureReason, bool) {
	return "", false
}
