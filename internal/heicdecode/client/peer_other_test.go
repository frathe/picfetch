//go:build !windows

package client

import "os"

func installOwnedPeer(source, destination string) error { return os.Link(source, destination) }
