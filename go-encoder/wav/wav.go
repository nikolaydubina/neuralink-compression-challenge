package wav

import (
	"encoding/binary"
	"fmt"
	"io"
)

type WAVHeader struct {
	ChunkID       [4]byte // RIFF chunk descriptor
	ChunkSize     uint32
	Format        [4]byte
	Subchunk1ID   [4]byte // "fmt" sub-chunk
	Subchunk1Size uint32
	AudioFormat   uint16
	NumChannels   uint16
	SampleRate    uint32
	ByteRate      uint32
	BlockAlign    uint16
	BitsPerSample uint16
	Subchunk2ID   [4]byte // "data" sub-chunk
	Subchunk2Size uint32
}

func (s WAVHeader) IsPCM() bool { return s.AudioFormat == 1 }

func (s *WAVHeader) MarshalBinary(w io.Writer) error { return binary.Write(w, binary.LittleEndian, s) }

func (s *WAVHeader) UnmarshalBinary(r io.Reader) error {
	if err := binary.Read(r, binary.LittleEndian, s); err != nil {
		return err
	}

	if s.ChunkID != [4]byte{'R', 'I', 'F', 'F'} {
		return fmt.Errorf("invalid chunk ID: (%s) != RIFF", s.ChunkID)
	}

	if s.Format != [4]byte{'W', 'A', 'V', 'E'} {
		return fmt.Errorf("invalid format: (%s) != WAVE", s.Format)
	}

	if s.Subchunk1ID != [4]byte{'f', 'm', 't', ' '} {
		return fmt.Errorf("invalid subchunk1 ID: (%s) != fmt", s.Subchunk1ID)
	}

	if s.Subchunk2ID != [4]byte{'d', 'a', 't', 'a'} {
		return fmt.Errorf("invalid subchunk2 ID: (%s) != data", s.Subchunk2ID)
	}

	return nil
}

type WAVReader struct {
	Header WAVHeader
	r      io.Reader
}

func NewWAVReader(r io.Reader) *WAVReader { return &WAVReader{r: r} }

func (s *WAVReader) ReadHeader() error { return s.Header.UnmarshalBinary(s.r) }

func (s *WAVReader) Next() (uint16, error) {
	var sample uint16
	if err := binary.Read(s.r, binary.LittleEndian, &sample); err != nil {
		return 0, err
	}
	return sample, nil
}

// TODO: correct wrapping bytes into samples
func (s *WAVReader) Read(p []byte) (int, error) { return s.r.Read(p) }

type WAVWriter struct {
	header WAVHeader
	w      io.Writer
}

func NewWAVWriter(h WAVHeader, w io.Writer) *WAVWriter { return &WAVWriter{header: h, w: w} }

func (s *WAVWriter) WriteHeader() error { return s.header.MarshalBinary(s.w) }

func (s *WAVWriter) WriteSample(sample uint16) error {
	return binary.Write(s.w, binary.LittleEndian, sample)
}

// TODO: correct wrapping bytes into samples
func (s *WAVWriter) Write(b []byte) (int, error) { return s.w.Write(b) }
