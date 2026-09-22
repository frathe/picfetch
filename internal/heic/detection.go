package heic

import (
	"encoding/binary"
	"strings"
)

// IsExtension recognizes the still-image names, independently of availability.
func IsExtension(extension string) bool {
	switch strings.ToLower(extension) {
	case ".heic", ".heif":
		return true
	}
	return false
}

// IsData recognizes HEVC brands for dispatch only. The native worker still
// validates the container, codec, primary image and supported still profile.
func IsData(data []byte) bool {
	if len(data) < 16 || string(data[4:8]) != "ftyp" {
		return false
	}
	end := uint64(binary.BigEndian.Uint32(data[:4]))
	if end < 16 || end > uint64(len(data)) || end > 4096 || end%4 != 0 {
		return false
	}
	recognized := false
	for offset := 8; offset < int(end); offset += 4 {
		if offset == 12 {
			continue // Minor version is not a brand.
		}
		switch string(data[offset : offset+4]) {
		case "avif", "avis":
			return false
		case "mif1", "msf1", "heic", "heix", "hevc", "hevx", "heim", "heis", "hevm", "hevs":
			recognized = true
		}
	}
	return recognized
}
