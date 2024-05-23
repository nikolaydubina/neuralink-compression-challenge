package encoding

import (
	"encoding/binary"
	"fmt"
	"io"
)

type Marker struct {
	Count        int
	EncodingSize int
	IsEncoded    bool
}

func (s *Marker) SizeBytes() int { return 2 }

func (s *Marker) MarshalBinaryToWriter(w io.Writer, endian binary.ByteOrder) error {
	var v uint16

	if s.Count > ((1 << 14) - 1) {
		return fmt.Errorf("count %016b is out of bound", s.Count)
	}

	// count
	count := s.Count
	if !s.IsEncoded {
		count = -count
	}
	v = uint16(count) & ((1 << 14) - 1)

	// encoding size marker
	var encodingSizeMarker uint16
	if s.IsEncoded {
		switch s.EncodingSize {
		case 4:
			encodingSizeMarker = 0
		case 6:
			encodingSizeMarker = 1
		case 7:
			encodingSizeMarker = 2
		default:
			return fmt.Errorf("unsupported encoding size %d", s.EncodingSize)
		}
	}

	v = (v << 2) | encodingSizeMarker
	if err := binary.Write(w, endian, v); err != nil {
		return err
	}

	return nil
}

func (s *Marker) UnmarshalBinaryFromReader(r io.Reader, endian binary.ByteOrder) error {
	var v uint16

	if err := binary.Read(r, endian, &v); err != nil {
		return err
	}

	if v == 0 {
		return io.EOF
	}

	switch encodingSizeMarker := v & 0x3; encodingSizeMarker {
	case 0:
		s.EncodingSize = 4
	case 1:
		s.EncodingSize = 6
	case 2:
		s.EncodingSize = 7
	}

	// restore two-s complement
	// remove encoding bits
	isNegative := (v & 0x8000) != 0
	v >>= 2
	if isNegative {
		v |= (1 << 15)
		v |= (1 << 14)
	}

	count := int16(v)

	s.IsEncoded = count >= 0
	if !s.IsEncoded {
		s.EncodingSize = 0
	}

	if count < 0 {
		count = -count
	}
	s.Count = int(count)
	return nil
}
