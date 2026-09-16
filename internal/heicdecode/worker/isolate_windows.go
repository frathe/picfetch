//go:build windows && (amd64 || arm64)

package worker

import (
	"github.com/frathe/picfetch/internal/heicdecode"
	"github.com/frathe/picfetch/internal/heicdecode/winisolation"
)

func isolate(limits heicdecode.Limits) (int64, error) {
	if err := winisolation.Verify(limits); err != nil {
		return 0, err
	}
	return limits.OSProcessBytes, nil
}
