package encoding

import (
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
)

type Marker struct {
	Count        int
	EncodingSize int
	IsEncoded    bool
}

func (s *Marker) SizeBytes() int { return 2 }

func (s *Marker) MarshalBinaryToWriter(w io.Writer, endian binary.ByteOrder) error {
	var v uint16

	if s.Count > 0x7FFF {
		return fmt.Errorf("count %016b is out of bound, 15 bits expected", s.Count)
	}

	// count
	count := s.Count
	if !s.IsEncoded {
		count = -count
	}
	v = uint16(count) & 0x7FFF

	// encoding size marker
	var encodingSizeMarker uint16
	if s.IsEncoded {
		switch s.EncodingSize {
		case 6:
			encodingSizeMarker = 0
		case 7:
			encodingSizeMarker = 1
		default:
			return fmt.Errorf("unsupported encoding size %d", s.EncodingSize)
		}
	}

	v = (v << 1) | encodingSizeMarker
	if err := binary.Write(w, endian, v); err != nil {
		return err
	}

	slog.Debug(fmt.Sprintf("%016b:", uint16(v)), "marker", *s)
	return nil
}

func (s *Marker) UnmarshalBinaryFromReader(r io.Reader, endian binary.ByteOrder) error {
	var v uint16
	if err := binary.Read(r, endian, &v); err != nil {
		return err
	}

	switch encodingSizeMarker := v & 1; encodingSizeMarker {
	case 0:
		s.EncodingSize = 6
	case 1:
		s.EncodingSize = 7
	}

	// remove encoding bit
	v >>= 1

	// restore two-s complement of int15
	if (v & 0x4000) == 0x4000 {
		v |= 0x8000
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

	slog.Debug(fmt.Sprintf("%016b:", uint16(v)), "marker", *s)
	return nil
}
