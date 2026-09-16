package similarity

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// A snapshot is published atomically to the UI, but transported as metadata,
// individual items/merges and an end marker. Neither peer serializes the whole
// map into one allocation. The memory budget covers all frames in a snapshot.
type workerEventFrame struct {
	Snapshot    *Event       `json:",omitempty"`
	HasItems    bool         `json:",omitempty"`
	Item        *Item        `json:",omitempty"`
	Merge       *CohortMerge `json:",omitempty"`
	End         bool         `json:",omitempty"`
	MemoryLimit bool         `json:",omitempty"`
}

type workerEventEncoder struct {
	output io.Writer
	limit  int
}

func newWorkerEventEncoder(output io.Writer, limit int) *workerEventEncoder {
	return &workerEventEncoder{output: output, limit: limit}
}

func (e *workerEventEncoder) Encode(event Event) error {
	remaining := e.limit
	write := func(frame workerEventFrame) error {
		payload, err := json.Marshal(frame)
		if err != nil {
			return err
		}
		payload = append(payload, '\n')
		if len(payload) > remaining {
			// Send the cause even if a partially written snapshot must be discarded.
			_, _ = io.WriteString(e.output, "{\"MemoryLimit\":true}\n")
			return ErrAnalysisMemoryLimit
		}
		remaining -= len(payload)
		n, err := e.output.Write(payload)
		if err == nil && n != len(payload) {
			return io.ErrShortWrite
		}
		return err
	}
	metadata := event
	metadata.Items, metadata.Merges = nil, nil
	if err := write(workerEventFrame{Snapshot: &metadata, HasItems: event.Items != nil}); err != nil {
		return err
	}
	for i := range event.Items {
		if err := write(workerEventFrame{Item: &event.Items[i]}); err != nil {
			return err
		}
	}
	for i := range event.Merges {
		if err := write(workerEventFrame{Merge: &event.Merges[i]}); err != nil {
			return err
		}
	}
	return write(workerEventFrame{End: true})
}

type workerEventDecoder struct {
	scanner          *bufio.Scanner
	limit, itemLimit int
}

func newWorkerEventDecoder(input io.Reader, limit, itemLimit int) *workerEventDecoder {
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, min(64*1024, limit)), limit)
	return &workerEventDecoder{scanner: scanner, limit: limit, itemLimit: itemLimit}
}

func (d *workerEventDecoder) Decode(event *Event) error {
	remaining := d.limit
	var snapshot Event
	started, hasItems := false, false
	for d.scanner.Scan() {
		payload := d.scanner.Bytes()
		if len(payload)+1 > remaining {
			return ErrAnalysisMemoryLimit
		}
		remaining -= len(payload) + 1
		var frame workerEventFrame
		if err := json.Unmarshal(payload, &frame); err != nil {
			return err
		}
		parts := 0
		for _, present := range []bool{frame.Snapshot != nil, frame.Item != nil, frame.Merge != nil, frame.End, frame.MemoryLimit} {
			if present {
				parts++
			}
		}
		if parts != 1 || frame.HasItems && frame.Snapshot == nil {
			return errors.New("invalid similarity worker frame")
		}
		switch {
		case frame.MemoryLimit:
			return ErrAnalysisMemoryLimit
		case frame.Snapshot != nil:
			if started || frame.Snapshot.Items != nil || frame.Snapshot.Merges != nil {
				return errors.New("invalid similarity snapshot header")
			}
			snapshot = *frame.Snapshot
			started, hasItems = true, frame.HasItems
			if hasItems {
				snapshot.Items = []Item{}
			}
		case !started:
			return errors.New("similarity worker frame precedes snapshot header")
		case frame.Item != nil:
			if !hasItems {
				return errors.New("similarity item outside a map")
			}
			if len(snapshot.Items) >= d.itemLimit {
				return ErrAnalysisItemLimit
			}
			snapshot.Items = append(snapshot.Items, *frame.Item)
		case frame.Merge != nil:
			if !hasItems || len(snapshot.Merges) >= len(snapshot.Items) {
				return errors.New("invalid similarity hierarchy size")
			}
			snapshot.Merges = append(snapshot.Merges, *frame.Merge)
		case frame.End:
			*event = snapshot
			return nil
		}
	}
	if err := d.scanner.Err(); err != nil {
		if errors.Is(err, bufio.ErrTooLong) {
			return fmt.Errorf("%w (%d bytes): %w", ErrAnalysisMemoryLimit, d.limit, err)
		}
		return fmt.Errorf("decode similarity worker frame: %w", err)
	}
	if started {
		return io.ErrUnexpectedEOF
	}
	return io.EOF
}
