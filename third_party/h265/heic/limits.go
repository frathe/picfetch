package heic

import "github.com/gen2brain/h265/hevc"

// eachNAL validates framing and visits one unit at a time. Splitting the whole
// item first retains every unescaped payload and a slice entry per tiny NAL.
func eachNAL(data []byte, lengthSize int, visit func(hevc.NALUnit) error) error {
	if lengthSize < 1 || lengthSize > 4 {
		return ErrInvalid
	}
	for count := 0; len(data) > 0; count++ {
		if count >= maxSamples || len(data) < lengthSize {
			return ErrInvalid
		}
		var n uint64
		for _, b := range data[:lengthSize] {
			n = n*256 + uint64(b)
		}
		data = data[lengthSize:]
		if n < 2 || n > uint64(len(data)) {
			return ErrInvalid
		}
		u, ok := hevc.ParseNAL(data[:int(n)])
		if !ok {
			return ErrInvalid
		}
		if err := visit(u); err != nil {
			return err
		}
		data = data[int(n):]
	}
	return nil
}

func releasePictures(pics []*hevc.Picture) {
	for _, pic := range pics {
		pic.Release()
	}
}
