//go:build !darwin || !cgo

package worker

import (
	"errors"

	"github.com/frathe/picfetch/internal/heicdecode"
)

func isolate(_ heicdecode.Limits) (int64, error) {
	return 0, errors.New("HEIC native sandbox is unavailable on this build")
}
