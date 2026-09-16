//go:build !windows

package client

import "os"

func installOwnedPeer(source, destination string) error { return os.Link(source, destination) }

func prepareNativeHelper(_ string) error { return nil }
