package imaging

import "encoding/binary"

// jpegScanPolicy qualifies complete 8-bit Huffman sequential/progressive frames.
// Decoder tolerance is not used to infer missing scans, tables or marker lengths.
type jpegScanPolicy struct {
	progressive  bool
	ids          []byte
	quantizers   []byte
	quant        [4]bool
	huffman      [2][4]bool
	coefficients [3][64]int
	entropy      jpegEntropyState
}

func (s *jpegScanPolicy) frame(marker byte, p []byte) error {
	if len(s.ids) != 0 || len(p) < 6 || p[0] != 8 || (p[5] != 1 && p[5] != 3) || len(p) != 6+3*int(p[5]) {
		return ErrJPEGMetadataProcess
	}
	if binary.BigEndian.Uint16(p[1:3]) == 0 || binary.BigEndian.Uint16(p[3:5]) == 0 {
		return ErrJPEGMetadataStructure
	}
	s.progressive = marker == 0xc2
	s.entropy.width = int(binary.BigEndian.Uint16(p[3:5]))
	s.entropy.height = int(binary.BigEndian.Uint16(p[1:3]))
	for i := 0; i < int(p[5]); i++ {
		c := p[6+3*i : 9+3*i]
		for _, id := range s.ids {
			if id == c[0] {
				return ErrJPEGMetadataStructure
			}
		}
		if c[1]>>4 < 1 || c[1]>>4 > 4 || c[1]&15 < 1 || c[1]&15 > 4 || c[2] > 3 {
			return ErrJPEGMetadataProcess
		}
		s.ids = append(s.ids, c[0])
		s.quantizers = append(s.quantizers, c[2])
		s.entropy.horizontal[i], s.entropy.vertical[i] = int(c[1]>>4), int(c[1]&15)
		if p[5] == 1 {
			s.entropy.horizontal[i], s.entropy.vertical[i] = 1, 1
		}
		s.entropy.maxH = max(s.entropy.maxH, s.entropy.horizontal[i])
		s.entropy.maxV = max(s.entropy.maxV, s.entropy.vertical[i])
		for k := range s.coefficients[i] {
			s.coefficients[i][k] = -1
		}
	}
	return nil
}

func (s *jpegScanPolicy) table(marker byte, p []byte) error {
	if marker == 0xdd {
		if len(p) != 2 {
			return ErrJPEGMetadataStructure
		}
		s.entropy.restart = int(binary.BigEndian.Uint16(p))
		return nil
	}
	if len(p) == 0 {
		return ErrJPEGMetadataStructure
	}
	for len(p) > 0 {
		kind, id := p[0]>>4, p[0]&15
		p = p[1:]
		if id > 3 {
			return ErrJPEGMetadataStructure
		}
		if marker == 0xdb {
			// The qualified 8-bit process uses 8-bit quantization tables.
			if kind != 0 || len(p) < 64 {
				return ErrJPEGMetadataProcess
			}
			for _, value := range p[:64] {
				if value == 0 {
					return ErrJPEGMetadataStructure
				}
			}
			s.quant[id] = true
			p = p[64:]
		} else {
			if kind > 1 || len(p) < 16 {
				return ErrJPEGMetadataStructure
			}
			n := 0
			for _, count := range p[:16] {
				n += int(count)
			}
			if n == 0 || n > 256 || len(p) < 16+n {
				return ErrJPEGMetadataStructure
			}
			if err := s.entropy.tables[kind][id].set(p[:16], p[16:16+n]); err != nil {
				return err
			}
			s.huffman[kind][id] = true
			p = p[16+n:]
		}
	}
	return nil
}

func (s *jpegScanPolicy) scan(p []byte) error {
	if len(p) < 6 || int(p[0]) > len(s.ids) || p[0] == 0 || len(p) != 4+2*int(p[0]) {
		return ErrJPEGMetadataStructure
	}
	start, end, high, low := int(p[len(p)-3]), int(p[len(p)-2]), int(p[len(p)-1]>>4), int(p[len(p)-1]&15)
	if start > end || end > 63 || high > 13 || low > 13 {
		return ErrJPEGMetadataStructure
	}
	if s.progressive {
		if start == 0 && end != 0 || start != 0 && p[0] != 1 || high != 0 && low != high-1 {
			return ErrJPEGMetadataStructure
		}
	} else if start != 0 || end != 63 || high != 0 || low != 0 {
		return ErrJPEGMetadataStructure
	}
	var used [3]bool
	for i := 0; i < int(p[0]); i++ {
		id, tables := p[1+2*i], p[2+2*i]
		index := -1
		for j, component := range s.ids {
			if component == id {
				index = j
				break
			}
		}
		if index < 0 || used[index] || tables>>4 > 3 || tables&15 > 3 {
			return ErrJPEGMetadataStructure
		}
		used[index] = true
		if !s.quant[s.quantizers[index]] || start == 0 && high == 0 && !s.huffman[0][tables>>4] || end > 0 && !s.huffman[1][tables&15] {
			return ErrJPEGMetadataStructure
		}
		for k := start; k <= end; k++ {
			previous := s.coefficients[index][k]
			if high == 0 && previous != -1 || high != 0 && previous != high {
				return ErrJPEGMetadataStructure
			}
			s.coefficients[index][k] = low
		}
	}
	return nil
}

func (s *jpegScanPolicy) complete() bool {
	if len(s.ids) == 0 {
		return false
	}
	for i := range s.ids {
		for _, value := range s.coefficients[i] {
			if value != 0 {
				return false
			}
		}
	}
	return true
}
