//go:build !windows

package client

import "context"

func prepareInstalled(ctx context.Context, helper string, _ [32]byte, _ string) (string, func(), error) {
	if err := ctx.Err(); err != nil {
		return "", nil, err
	}
	return helper, func() {}, nil
}
