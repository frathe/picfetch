package imaging

import "context"

// jpegEntropyState checks exact Huffman consumption for metadata removal. It
// retains only progressive nonzero positions, never pixels or coefficient
// values. image/jpeg independently reconstructs and validates the image later.
type jpegEntropyState struct {
	width, height        int
	maxH, maxV           int
	horizontal, vertical [3]int
	restart              int
	tables               [2][4]jpegEntropyHuffman
	nonzero              [3][]uint64
}

type jpegEntropyHuffman struct {
	count, first, offset [16]int
	values               [256]byte
}

func (h *jpegEntropyHuffman) set(counts, values []byte) error {
	code, offset := 0, 0
	for i, count := range counts {
		// T.81 Annex C excludes oversubscribed trees and all-one codes,
		// which would make end-of-interval padding ambiguous.
		if code+int(count) >= 1<<(i+1) {
			return ErrJPEGMetadataStructure
		}
		h.count[i], h.first[i], h.offset[i] = int(count), code, offset
		offset += int(count)
		code = (code + int(count)) * 2
	}
	copy(h.values[:], values)
	return nil
}

type jpegEntropyReader struct {
	data []byte
	pos  int
	bits uint8
	left uint8
	err  error
}

func (r *jpegEntropyReader) read(n int) int {
	value := 0
	for range n {
		if r.err != nil {
			return 0
		}
		if r.left == 0 {
			if r.pos >= len(r.data) {
				r.err = ErrJPEGMetadataStructure
				return 0
			}
			r.bits = r.data[r.pos]
			r.pos++
			if r.bits == 0xff {
				if r.pos >= len(r.data) || r.data[r.pos] != 0 {
					r.err = ErrJPEGMetadataStructure
					return 0
				}
				r.pos++
			}
			r.left = 8
		}
		r.left--
		value = value*2 + int(r.bits>>r.left&1)
	}
	return value
}

func (r *jpegEntropyReader) symbol(h *jpegEntropyHuffman) int {
	code := 0
	for i := range h.count {
		code = code*2 + r.read(1)
		if r.err != nil {
			return 0
		}
		index := code - h.first[i]
		if index >= 0 && index < h.count[i] {
			return int(h.values[h.offset[i]+index])
		}
	}
	r.err = ErrJPEGMetadataStructure
	return 0
}

func (r *jpegEntropyReader) boundary(ctx context.Context) (byte, int, error) {
	if r.err != nil {
		return 0, r.pos, r.err
	}
	mask := uint8((1 << r.left) - 1)
	if r.bits&mask != mask {
		return 0, r.pos, ErrJPEGMetadataStructure
	}
	r.left = 0
	return jpegRemovalMarker(ctx, r.data, r.pos)
}

// entropyEnd implements scan/MCU ordering and interval boundaries from T.81.
// Each read is demanded by coefficient syntax; no byte search or restart
// resynchronization can silently retain unused bytes inside an entropy interval.
func (s *jpegScanPolicy) entropyEnd(ctx context.Context, data []byte, pos int, scan []byte) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	e := &s.entropy
	columns := (e.width + 8*e.maxH - 1) / (8 * e.maxH)
	rows := (e.height + 8*e.maxV - 1) / (8 * e.maxV)
	count := int(scan[0])
	var indices [3]int
	for i := range count {
		for j, id := range s.ids {
			if id == scan[1+2*i] {
				indices[i] = j
			}
		}
		j := indices[i]
		if s.progressive && e.nonzero[j] == nil {
			e.nonzero[j] = make([]uint64, columns*rows*e.horizontal[j]*e.vertical[j])
		}
	}
	stride := columns
	if count == 1 {
		j := indices[0]
		columns = (e.width*e.horizontal[j] + 8*e.maxH - 1) / (8 * e.maxH)
		rows = (e.height*e.vertical[j] + 8*e.maxV - 1) / (8 * e.maxV)
	}
	start, end, high := int(scan[len(scan)-3]), int(scan[len(scan)-2]), int(scan[len(scan)-1]>>4)
	r := jpegEntropyReader{data: data, pos: pos}
	eob, restart := 0, byte(0xd0)
	for mcu := 0; mcu < columns*rows; mcu++ {
		if mcu%64 == 0 {
			if err := ctx.Err(); err != nil {
				return 0, err
			}
		}
		for i := range count {
			j := indices[i]
			h, v := e.horizontal[j], e.vertical[j]
			blocks := h * v
			if count == 1 {
				blocks = 1
			}
			for block := range blocks {
				x, y := mcu%columns, mcu/columns
				if count != 1 {
					x, y = x*h+block%h, y*v+block/h
				}
				var scratch uint64
				mask := &scratch
				if s.progressive {
					mask = &e.nonzero[j][y*stride*h+x]
				}
				tables := scan[2+2*i]
				if high != 0 && start != 0 {
					r.refine(&e.tables[1][tables&15], mask, start, end, &eob)
				} else if high != 0 {
					r.read(1)
				} else {
					r.initial(&e.tables[0][tables>>4], &e.tables[1][tables&15], mask, start, end, s.progressive, &eob)
				}
				if r.err != nil {
					return 0, r.err
				}
			}
		}
		if e.restart > 0 && (mcu+1)%e.restart == 0 && mcu+1 < columns*rows {
			marker, after, err := r.boundary(ctx)
			if err != nil {
				return 0, err
			}
			if marker != restart || eob != 0 {
				return 0, ErrJPEGMetadataStructure
			}
			restart = 0xd0 + (restart-0xd0+1)%8
			r.pos = after
		}
	}
	marker, _, err := r.boundary(ctx)
	if err != nil {
		return 0, err
	}
	if marker == 0 || marker >= 0xd0 && marker <= 0xd7 || eob != 0 {
		return 0, ErrJPEGMetadataStructure
	}
	return r.pos, nil
}

func (r *jpegEntropyReader) initial(dc, ac *jpegEntropyHuffman, mask *uint64, start, end int, progressive bool, eob *int) {
	if start == 0 {
		size := r.symbol(dc)
		if size > 11 {
			r.err = ErrJPEGMetadataStructure
			return
		}
		r.read(size)
		start = 1
	}
	if start > end {
		return
	}
	if *eob > 0 {
		*eob--
		return
	}
	for k := start; k <= end && r.err == nil; {
		symbol := r.symbol(ac)
		run, size := symbol>>4, symbol&15
		if size == 0 && run != 15 {
			if !progressive && run != 0 {
				r.err = ErrJPEGMetadataStructure
				return
			}
			*eob = (1 << run) + r.read(run) - 1
			return
		}
		if size == 0 {
			k += 16
			if k > end+1 {
				r.err = ErrJPEGMetadataStructure
			}
			continue
		}
		k += run
		if k > end || size > 10 {
			r.err = ErrJPEGMetadataStructure
			return
		}
		r.read(size)
		*mask |= uint64(1) << k
		k++
	}
}

func (r *jpegEntropyReader) refine(ac *jpegEntropyHuffman, mask *uint64, start, end int, eob *int) {
	k := start
	for *eob == 0 && k <= end && r.err == nil {
		symbol := r.symbol(ac)
		run, size := symbol>>4, symbol&15
		if size == 0 && run != 15 {
			*eob = (1 << run) + r.read(run)
			break
		}
		if size > 1 {
			r.err = ErrJPEGMetadataStructure
			return
		}
		if size == 1 {
			r.read(1) // New coefficient sign precedes existing-value refinements.
		} else {
			run = 16
		}
		for k <= end {
			if *mask&(uint64(1)<<k) != 0 {
				r.read(1)
			} else {
				if run == 0 {
					break
				}
				run--
			}
			k++
			if size == 0 && run == 0 {
				break
			}
		}
		if run != 0 || size == 1 && k > end {
			r.err = ErrJPEGMetadataStructure
			return
		}
		if size == 1 {
			*mask |= uint64(1) << k
			k++
		}
	}
	if *eob > 0 {
		for ; k <= end; k++ {
			if *mask&(uint64(1)<<k) != 0 {
				r.read(1)
			}
		}
		*eob--
	}
}
