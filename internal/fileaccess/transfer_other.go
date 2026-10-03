//go:build !darwin || !appleappstore

package fileaccess

import "errors"

func createTransfer(_ string) ([]byte, error) {
	return nil, errors.New("native worker grants unavailable")
}
func resolveTransfer(_ Transfer) (func(), error) {
	return nil, errors.New("native worker grants unavailable")
}
