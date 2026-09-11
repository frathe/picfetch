//go:build !darwin

package main

import (
	"context"
	"fmt"
	"io"
)

func nativeTrial(_ context.Context, _ configuration, _ string, _ io.Writer) error {
	return fmt.Errorf("native library trials require macOS")
}
