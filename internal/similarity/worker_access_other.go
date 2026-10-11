//go:build !darwin || !appleappstore

package similarity

import "context"

func prepareWorkerAccess(_ context.Context, _ *request) (func(), error) { return func() {}, nil }
