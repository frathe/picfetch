package heicdecode

import (
	"encoding/binary"
	"errors"
	"io"
)

// Ready is sent before any image input. File and network denial must have been
// observed by native probes. NativeMemoryBytes is zero when no hard OS memory
// ceiling was installed; macOS permits that explicitly approved limitation.
type Ready struct {
	WASMMemoryBytes   int64
	NativeMemoryBytes int64
}

// WriteReady publishes a fixed-size startup acknowledgement. The worker calls
// it only after platform policy and actual capability checks have succeeded.
func WriteReady(w io.Writer, ready Ready) error {
	if !ready.valid() {
		return errors.New("invalid HEIC worker readiness")
	}
	var data [24]byte
	copy(data[:8], "PHREADY2")
	binary.LittleEndian.PutUint64(data[8:16], uint64(ready.WASMMemoryBytes))
	binary.LittleEndian.PutUint64(data[16:24], uint64(ready.NativeMemoryBytes))
	return writeAll(w, data[:])
}

// ReadReady reads only the startup frame; the later response follows on the
// same pipe. It never consumes image output or infers native OS memory bounds.
func ReadReady(r io.Reader) (Ready, error) {
	var data [24]byte
	if _, err := io.ReadFull(r, data[:]); err != nil {
		return Ready{}, err
	}
	ready := Ready{WASMMemoryBytes: int64(binary.LittleEndian.Uint64(data[8:16])), NativeMemoryBytes: int64(binary.LittleEndian.Uint64(data[16:24]))}
	if string(data[:8]) != "PHREADY2" || !ready.valid() {
		return Ready{}, errors.New("invalid HEIC worker readiness")
	}
	return ready, nil
}

func (r Ready) valid() bool {
	return r.WASMMemoryBytes > 0 && r.WASMMemoryBytes <= 1024*1024*1024 && r.WASMMemoryBytes%(64*1024) == 0 &&
		(r.NativeMemoryBytes == 0 || r.NativeMemoryBytes >= r.WASMMemoryBytes && r.NativeMemoryBytes <= 2*1024*1024*1024)
}
