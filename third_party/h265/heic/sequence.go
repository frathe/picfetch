package heic

import (
	"errors"
	"math"

	"github.com/gen2brain/h265/hevc"
)

type track struct {
	id        uint32
	handler   string
	timescale uint32
	samples   []extent
	deltas    []uint32
	auxl      []uint32
	auxType   string
	hvcC      *hevcConfig
	width     int
	height    int

	duration   uint64
	hasEdits   bool
	repeating  bool
	segmentDur uint64
}

const indefiniteDuration = ^uint64(0)

// loopCount is zero for forever, -1 to show each frame once, otherwise the
// number of extra plays. ISO/IEC 23008-12 section 9.6.1.
func (t *track) loopCount() int {
	if !t.hasEdits {
		return 0
	}
	if !t.repeating {
		return -1
	}
	if t.duration == indefiniteDuration || t.duration == 0 || t.segmentDur == 0 {
		return 0
	}

	n := t.duration / t.segmentDur
	if t.duration%t.segmentDur != 0 {
		n++
	}
	if n <= 1 {
		return -1
	}
	if n-1 > math.MaxInt32 {
		return 0
	}

	return int(n - 1)
}

type movie struct {
	tracks []track
}

// A movie's expanded sample tables share this budget. HEIC sequences longer
// than this need a streaming API rather than eager table/frame allocation.
const maxSamples = 65536

func parseMoov(b []byte) (*movie, error) {
	m := &movie{}
	samples := 0

	err := eachBox(b, func(typ string, b []byte) error {
		if typ != "trak" {
			return nil
		}
		t, err := parseTrak(b)
		if err != nil {
			return err
		}
		samples += len(t.samples)
		if samples > maxSamples || len(m.tracks) >= 1024 {
			return ErrUnsupported
		}
		m.tracks = append(m.tracks, t)

		return nil
	})
	if err != nil {
		return nil, err
	}

	return m, nil
}

func parseTrak(b []byte) (track, error) {
	var t track

	err := eachBox(b, func(typ string, b []byte) error {
		switch typ {
		case "tkhd":
			r := &reader{b: b}
			v, _ := r.fullBox()
			if v == 1 {
				r.skip(16)
			} else {
				r.skip(8)
			}
			t.id = r.u32()
			r.skip(4)
			if v == 1 {
				t.duration = r.u64()
			} else {
				t.duration = uint64(r.u32())
				if t.duration == 1<<32-1 {
					t.duration = indefiniteDuration
				}
			}
			if r.err {
				return ErrInvalid
			}

		case "edts":
			t.hasEdits = true

			return eachBox(b, func(typ string, b []byte) error {
				if typ != "elst" {
					return nil
				}
				r := &reader{b: b}
				v, flags := r.fullBox()
				if flags&1 == 0 {
					return nil
				}
				t.repeating = true
				if r.u32() != 1 {
					return nil
				}
				if v == 1 {
					t.segmentDur = r.u64()
				} else {
					t.segmentDur = uint64(r.u32())
				}
				if r.err {
					return ErrInvalid
				}

				return nil
			})

		case "tref":
			return eachBox(b, func(typ string, b []byte) error {
				if typ != "auxl" {
					return nil
				}
				r := &reader{b: b}
				for r.remaining() >= 4 {
					t.auxl = append(t.auxl, r.u32())
				}

				return nil
			})

		case "mdia":
			return eachBox(b, func(typ string, b []byte) error {
				switch typ {
				case "mdhd":
					r := &reader{b: b}
					v, _ := r.fullBox()
					if v == 1 {
						r.skip(16)
					} else {
						r.skip(8)
					}
					t.timescale = r.u32()
					if r.err {
						return ErrInvalid
					}

				case "hdlr":
					r := &reader{b: b}
					r.fullBox()
					r.u32()
					t.handler = r.str4()

				case "minf":
					return eachBox(b, func(typ string, b []byte) error {
						if typ != "stbl" {
							return nil
						}

						return t.parseStbl(b)
					})
				}

				return nil
			})
		}

		return nil
	})

	return t, err
}

type chunkRun struct {
	firstChunk, perChunk uint32
}

func (t *track) parseStbl(b []byte) error {
	var sizes []uint32
	var offsets []uint64
	var runs []chunkRun
	type timingRun struct{ count, delta uint32 }
	var timing []timingRun
	timingCount := 0
	seen := make(map[string]bool)

	err := eachBox(b, func(typ string, b []byte) error {
		r := &reader{b: b}
		// Multiple copies could otherwise repeatedly replace or expand a
		// table, and make its final size conceal the parsing cost.
		switch typ {
		case "stts", "stsz", "stsd", "stsc", "stco", "co64":
			key := typ
			if key == "co64" {
				key = "stco"
			}
			if seen[key] {
				return ErrInvalid
			}
			seen[key] = true
		}

		switch typ {
		case "stts":
			r.fullBox()
			n := int(r.u32())
			if r.err || n < 0 || n > maxSamples || n > r.remaining()/8 {
				return ErrInvalid
			}
			for range n {
				count := r.u32()
				delta := r.u32()
				if r.err || count == 0 || uint64(count) > uint64(maxSamples-timingCount) {
					return ErrInvalid
				}
				timingCount += int(count)
				timing = append(timing, timingRun{count, delta})
			}

		case "stsz":
			r.fullBox()
			uniform := r.u32()
			n := int(r.u32())
			if r.err || n < 0 || n > maxSamples || (uniform == 0 && n > r.remaining()/4) {
				return ErrInvalid
			}
			sizes = make([]uint32, n)
			for i := range n {
				if uniform != 0 {
					sizes[i] = uniform
				} else {
					sizes[i] = r.u32()
				}
			}

		case "stsd":
			r.fullBox()
			r.u32()

			return eachBox(b[r.i:], func(typ string, b []byte) error {
				if len(b) < 78 {
					return nil
				}

				rr := &reader{b: b}
				rr.skip(24)
				t.width = int(rr.u16())
				t.height = int(rr.u16())

				return eachBox(b[78:], func(typ string, b []byte) error {
					switch typ {
					case "auxi":
						rr := &reader{b: b}
						rr.fullBox()
						t.auxType = rr.cstr()

					case "hvcC":
						c, err := parseHvcC(&reader{b: b})
						if err != nil {
							return err
						}
						t.hvcC = c
					}

					return nil
				})
			})

		case "stsc":
			r.fullBox()
			n := int(r.u32())
			if r.err || n < 0 || n > maxSamples || n > r.remaining()/12 {
				return ErrInvalid
			}
			for range n {
				first := r.u32()
				per := r.u32()
				r.u32()
				if r.err || first == 0 || per == 0 || per > maxSamples ||
					(len(runs) > 0 && first <= runs[len(runs)-1].firstChunk) {
					return ErrInvalid
				}
				runs = append(runs, chunkRun{first, per})
			}

		case "stco", "co64":
			r.fullBox()
			n := int(r.u32())
			entrySize := 4
			if typ == "co64" {
				entrySize = 8
			}
			if r.err || n < 0 || n > maxSamples || n > r.remaining()/entrySize {
				return ErrInvalid
			}
			offsets = make([]uint64, n)
			for i := range n {
				if typ == "stco" {
					offsets[i] = uint64(r.u32())
				} else {
					offsets[i] = r.u64()
				}
			}
		}

		if r.err {
			return ErrInvalid
		}

		return nil
	})
	if err != nil {
		return err
	}

	if timingCount != len(sizes) {
		return ErrInvalid
	}
	t.samples, err = layoutSamples(sizes, offsets, runs)
	if err != nil {
		return err
	}
	t.deltas = make([]uint32, 0, timingCount)
	for _, run := range timing {
		for range int(run.count) {
			t.deltas = append(t.deltas, run.delta)
		}
	}
	return nil
}

// layoutSamples walks the chunk table to give every sample a file offset.
func layoutSamples(sizes []uint32, offsets []uint64, runs []chunkRun) ([]extent, error) {
	if len(sizes) == 0 || len(offsets) == 0 || len(runs) == 0 {
		if len(sizes) != 0 || len(offsets) != 0 || len(runs) != 0 {
			return nil, ErrInvalid
		}
		return nil, nil
	}
	if runs[0].firstChunk != 1 || uint64(runs[len(runs)-1].firstChunk) > uint64(len(offsets)) {
		return nil, ErrInvalid
	}

	out := make([]extent, 0, len(sizes))
	s := 0
	runIndex := 0

	for c := range offsets {
		for runIndex+1 < len(runs) && uint32(c+1) >= runs[runIndex+1].firstChunk {
			runIndex++
		}
		per := runs[runIndex].perChunk

		off := offsets[c]
		for range int(per) {
			if s >= len(sizes) {
				return nil, ErrInvalid
			}
			if sizes[s] == 0 || uint64(sizes[s]) > math.MaxUint64-off {
				return nil, ErrInvalid
			}
			out = append(out, extent{off: off, len: uint64(sizes[s])})
			off += uint64(sizes[s])
			s++
		}
	}

	if s != len(sizes) {
		return nil, ErrInvalid
	}
	return out, nil
}

func (m *movie) pictTrack() *track {
	for i := range m.tracks {
		if m.tracks[i].handler == "pict" && len(m.tracks[i].samples) > 0 {
			return &m.tracks[i]
		}
	}

	return nil
}

func (m *movie) alphaTrack(id uint32) *track {
	for i := range m.tracks {
		t := &m.tracks[i]
		if t.handler != "auxv" || len(t.samples) == 0 || !isAlphaURN(t.auxType) {
			continue
		}
		for _, to := range t.auxl {
			if to == id {
				return t
			}
		}
	}

	return nil
}

// decodeSequence decodes an image sequence track.
func (f *file) decodeSequence(o Options, maxFrames int) (*HEIC, error) {
	t := f.movie.pictTrack()
	if t == nil || t.timescale == 0 {
		return nil, nil
	}

	colors, err := f.decodeTrack(t, maxFrames)
	if err != nil {
		return nil, err
	}
	// Planar output aliases these pictures; RGB output owns its pixels.
	if !o.ToYCbCr {
		defer releasePictures(colors)
	}

	var alphas []*hevc.Picture

	if at := f.movie.alphaTrack(t.id); at != nil {
		if alphas, err = f.decodeTrack(at, maxFrames); err != nil {
			return nil, err
		}
		if !o.ToYCbCr {
			defer releasePictures(alphas)
		}
	}

	var first *hevc.Picture
	if len(colors) > 0 {
		first = colors[0]
	}

	out := &HEIC{LoopCount: t.loopCount(), Color: f.sequenceColor(first)}

	for i, pic := range colors {
		var alpha *hevc.Picture
		if i < len(alphas) {
			alpha = alphas[i]
		}

		img, err := toImage(pic, alpha, out.Color, o.ToYCbCr)
		if err != nil {
			return nil, err
		}

		if o.AutoRotate && f.meta != nil {
			if it, err := f.primary(); err == nil {
				if img, err = f.transform(it, img); err != nil {
					return nil, err
				}
			}
		}

		out.Image = append(out.Image, img)

		d := 0.0
		if i < len(t.deltas) {
			d = float64(t.deltas[i]) / float64(t.timescale)
		}

		out.Delay = append(out.Delay, d)
	}

	if len(out.Image) == 0 {
		return nil, nil
	}

	return out, nil
}

// sequenceColor takes the color description from the primary item when the
// file carries one, and otherwise from what the sequence itself declares.
func (f *file) sequenceColor(pic *hevc.Picture) ColorInfo {
	var it *item

	if f.meta != nil {
		it, _ = f.primary()
	}

	return f.colorInfo(it, pic)
}

func (f *file) decodeTrack(t *track, maxFrames int) ([]*hevc.Picture, error) {
	if t.hvcC == nil {
		return nil, ErrInvalid
	}

	// A sample needs at least one byte, so a table claiming more samples than
	// the file has bytes is describing data that cannot exist.
	if uint64(len(t.samples)) > f.src.size {
		return nil, ErrInvalid
	}

	var d hevc.Decoder
	d.FrameSizeLimit(f.limit())
	d.Budget(f.budget)
	d.Threads(f.workers(0))

	for _, nal := range t.hvcC.paramSets {
		u, ok := hevc.ParseNAL(nal)
		if !ok || u.Type.IsVCL() {
			return nil, ErrInvalid
		}

		if _, err := d.DecodeNAL(u); err != nil {
			return nil, wrap(err)
		}
	}

	var out []*hevc.Picture
	complete := false
	defer func() {
		if !complete {
			releasePictures(out)
		}
	}()
	var pixels uint64
	take := func(pics []*hevc.Picture) error {
		for i, pic := range pics {
			if maxFrames > 0 && len(out) >= maxFrames {
				releasePictures(pics[i:])
				return errStop
			}
			n := uint64(pic.Width) * uint64(pic.Height)
			if limit := f.limit(); limit > 0 && n > uint64(limit)-pixels {
				releasePictures(pics[i:])
				return ErrUnsupported
			}
			pixels += n
			out = append(out, pic)
		}
		if maxFrames > 0 && len(out) >= maxFrames {
			return errStop
		}
		return nil
	}

	for sampleIndex, s := range t.samples {
		// A first-frame request must not scan a long run of non-output
		// samples. The DPB itself is independently bounded by the SPS parser.
		if maxFrames == 1 && sampleIndex >= 64 {
			return nil, ErrUnsupported
		}
		if s.len == 0 || s.len > maxItemBytes {
			return nil, ErrInvalid
		}

		b, err := f.src.at(s.off, s.len)
		if err != nil {
			return nil, err
		}

		err = eachNAL(b, t.hvcC.lengthSize, func(u hevc.NALUnit) error {
			pics, err := d.DecodeNAL(u)
			if err != nil {
				releasePictures(pics)
				return wrap(err)
			}
			return take(pics)
		})
		if errors.Is(err, errStop) {
			complete = true
			return out, nil
		}
		if err != nil {
			return nil, err
		}
	}
	if err := take(d.Flush()); err != nil && !errors.Is(err, errStop) {
		return nil, err
	}
	complete = true
	return out, nil
}
