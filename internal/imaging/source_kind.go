package imaging

import (
	"encoding/binary"
	"errors"
	"io"
)

const maxSourcePrefix = 64 * 1024

type sourceKind uint8

const (
	sourceOrdinary sourceKind = iota
	sourceAVIF
	sourceCR3
	sourceIsolated
)

// Inspect only the leading file-type box. Unknown/oversized BMFF types and
// containers beginning with another common box go to isolation, never a native
// fallback. This is dispatch, not a walk of image items, properties or metadata.
func routeSource(data []byte) sourceKind {
	if len(data) < 8 {
		return sourceOrdinary
	}
	switch string(data[4:8]) {
	case "free", "skip", "wide", "meta", "moov", "mdat":
		return sourceIsolated
	case "ftyp":
	default:
		return sourceOrdinary
	}
	size, header := sourceTypeSize(data)
	if size < header+8 || size > maxSourcePrefix || size > uint64(len(data)) || (size-header)%4 != 0 {
		return sourceIsolated
	}
	kind := sourceIsolated
	for offset := header; offset < size; offset += 4 {
		if offset == header+4 { // minor version, not a compatible brand
			continue
		}
		switch string(data[offset : offset+4]) {
		case "heic", "heix", "hevc", "hevx", "heim", "heis", "hevm", "hevs":
			return sourceIsolated
		case "avif", "avis":
			kind = sourceAVIF
		case "crx ":
			if kind != sourceAVIF {
				kind = sourceCR3
			}
		}
	}
	return kind
}

func sourceTypeSize(data []byte) (size, header uint64) {
	size, header = uint64(binary.BigEndian.Uint32(data[:4])), 8
	if size == 1 {
		if len(data) < 16 {
			return 0, 16
		}
		return binary.BigEndian.Uint64(data[8:16]), 16
	}
	return size, header
}

func sourcePrefix(r io.Reader, limit int64) ([]byte, error) {
	var prefix []byte
	readTo := func(size uint64) error {
		size = min(size, uint64(maxSourcePrefix), uint64(max(0, limit)))
		if size <= uint64(len(prefix)) {
			return nil
		}
		old := len(prefix)
		prefix = append(prefix, make([]byte, int(size)-old)...)
		n, err := io.ReadFull(r, prefix[old:])
		prefix = prefix[:old+n]
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return nil
		}
		return err
	}
	if err := readTo(8); err != nil {
		return nil, err
	}
	if len(prefix) < 8 || string(prefix[4:8]) != "ftyp" {
		return prefix, nil
	}
	if binary.BigEndian.Uint32(prefix[:4]) == 1 {
		if err := readTo(16); err != nil {
			return nil, err
		}
	}
	size, _ := sourceTypeSize(prefix)
	if size <= maxSourcePrefix {
		if err := readTo(size); err != nil {
			return nil, err
		}
	}
	return prefix, nil
}
