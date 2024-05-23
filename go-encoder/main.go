package main

import (
	"bufio"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"sort"

	"github.com/nikolaydubina/neuralink-compression-challenge/go-encoder/bits"
	"github.com/nikolaydubina/neuralink-compression-challenge/go-encoder/wav"
)

type CacheSampleEncoderStats struct {
	NumEncodedSamples     int
	NumTotalSamples       int
	RatioEncodedSamples   float32
	NumBytesAdditional    int
	MaxLenHitsAdvanced    int
	MaxLenNotHitsAdvanced int
	NumHitsAdvanced       int
	NumNotHitsAdvanced    int
}

func (s *CacheSampleEncoderStats) AddEncodedAdvanced(advanced int) {
	if advanced > s.MaxLenHitsAdvanced {
		s.MaxLenHitsAdvanced = advanced
	}
	s.NumHitsAdvanced++
}

func (s *CacheSampleEncoderStats) AddNotEncodedAdvanced(advanced int) {
	if advanced > s.MaxLenNotHitsAdvanced {
		s.MaxLenNotHitsAdvanced = advanced
	}
	s.NumNotHitsAdvanced++
}

type CacheConfig struct {
	Size int
}

type cacheEntry struct {
	key   uint16
	count int
}

// Cache is not as efficient, but is ok for prototype.
type Cache struct {
	config CacheConfig
	order  []cacheEntry
}

func NewCache(config CacheConfig) *Cache {
	return &Cache{
		config: config,
		order:  make([]cacheEntry, 0, config.Size),
	}
}

func (s *Cache) Pop() {
	if len(s.order) == 0 {
		return
	}
	s.order = s.order[:len(s.order)-1]
}

func (s *Cache) Add(v uint16) {
	if i := s.Index(v); i >= 0 {
		s.order[i].count++
	} else {
		if s.IsFull() {
			s.Pop()
		}
		s.order = append(s.order, cacheEntry{key: v, count: 1})
	}
	sort.SliceStable(s.order, func(i, j int) bool { return s.order[i].count > s.order[j].count })
}

func (s *Cache) Index(v uint16) int {
	for i, q := range s.order {
		if q.key == v {
			return i
		}
	}
	return -1
}

func (s *Cache) IsFull() bool { return len(s.order) >= s.config.Size }

type CacheSampleEncoderConfig struct {
	EncodingSize        int
	EncodedSeqMinLen    int
	EncodedSeqMaxLen    int
	NotEncodedSeqMaxLen int
	ByteOrder           binary.ByteOrder
}

type CacheSampleEncoder struct {
	config CacheSampleEncoderConfig
	stats  CacheSampleEncoderStats

	maxKeyIndex     int
	encodedSeqAlign int

	cache  *Cache
	buffer []uint16
	w      interface {
		io.ByteWriter
		io.Writer
	}
}

func NewCacheSampleEncoder(
	config CacheSampleEncoderConfig,
	cache *Cache,
	w interface {
		io.ByteWriter
		io.Writer
	},
) *CacheSampleEncoder {
	return &CacheSampleEncoder{
		config:          config,
		cache:           cache,
		maxKeyIndex:     (1 << config.EncodingSize) - 1,
		encodedSeqAlign: 8, // calculate based on encoding size for most compact byte encoding
		w:               w,
		buffer:          make([]uint16, 0, config.EncodedSeqMaxLen),
	}
}

func (s *CacheSampleEncoder) Stats() CacheSampleEncoderStats {
	if s.stats.NumTotalSamples > 0 {
		s.stats.RatioEncodedSamples = float32(s.stats.NumEncodedSamples) / float32(s.stats.NumTotalSamples)
	}
	return s.stats
}

func (s *CacheSampleEncoder) Write(v uint16) error {
	s.stats.NumTotalSamples++
	if len(s.buffer) >= s.config.EncodedSeqMaxLen {
		if err := s.FlushBuffer(); err != nil {
			return err
		}
	}
	s.buffer = append(s.buffer, v)
	return nil
}

func (s *CacheSampleEncoder) encodeOne(v uint16) byte {
	i := s.cache.Index(v)
	if i < 0 || i > s.maxKeyIndex {
		err := fmt.Errorf("value(%v) got index(%v) is out of bound for encoded key, expected [0, %d]", v, i, s.maxKeyIndex)
		panic(err)
	}
	s.cache.Add(v)
	return byte(i)
}

func (s *CacheSampleEncoder) FlushBuffer() error {
	if len(s.buffer) == 0 {
		return nil
	}

	for offset := 0; offset < len(s.buffer); {
		slog.Debug(fmt.Sprintf("cache: %v", s.cache.order))

		countHits := s.flushBufferHitsCount(offset)
		countNotHits := s.flushBufferNotHitsCount(offset + countHits)

		// there samples to flush, but they are not hits,
		// and if they are hits they can not be encoded.
		// this number is within un-aligned encoded sequence.
		// flush them not-encoded.
		if countHits == 0 && countNotHits == 0 {
			countNotHits = s.encodedSeqAlign
			if (offset + countNotHits) > len(s.buffer) {
				countNotHits = len(s.buffer) - offset
			}
		}

		if countHits > 0 {
			if err := s.flushBufferHits(offset, countHits); err != nil {
				return err
			}
		}

		if countNotHits > 0 {
			if err := s.flushBufferNotHits(offset+countHits, countNotHits); err != nil {
				return err
			}
		}

		offset += countHits + countNotHits
	}

	s.buffer = s.buffer[:0]
	return nil
}

func (s *CacheSampleEncoder) flushBufferHitsCount(offset int) int {
	count := 0
	for i := offset; i < len(s.buffer) && s.cache.Index(s.buffer[i]) >= 0; i++ {
		count++
	}
	if count < s.config.EncodedSeqMinLen {
		count = 0
	}
	return count - (count % s.encodedSeqAlign)
}

func (s *CacheSampleEncoder) flushBufferNotHitsCount(offset int) int {
	count := 0
	for i := offset; i < len(s.buffer) && s.cache.Index(s.buffer[i]) < 0; i++ {
		count++
	}
	if count > s.config.NotEncodedSeqMaxLen {
		count = s.config.NotEncodedSeqMaxLen
	}
	return count
}

func (s *CacheSampleEncoder) flushMarkerHits(count int) error {
	marker := uint16(count)
	slog.Debug(fmt.Sprintf("%016b: marker next %d samples are encoded", marker, count))
	if err := binary.Write(s.w, s.config.ByteOrder, marker); err != nil {
		return err
	}
	s.stats.NumBytesAdditional += 2
	return nil
}

func (s *CacheSampleEncoder) flushBufferHits(offset, count int) error {
	defer func() { s.stats.AddEncodedAdvanced(count) }()

	if err := s.flushMarkerHits(count); err != nil {
		return err
	}

	for i := 0; i < count; i += 8 {
		encoded := bits.Pack8x7bit([8]byte{
			s.encodeOne(s.buffer[(offset + i + 0)]),
			s.encodeOne(s.buffer[(offset + i + 1)]),
			s.encodeOne(s.buffer[(offset + i + 2)]),
			s.encodeOne(s.buffer[(offset + i + 3)]),
			s.encodeOne(s.buffer[(offset + i + 4)]),
			s.encodeOne(s.buffer[(offset + i + 5)]),
			s.encodeOne(s.buffer[(offset + i + 6)]),
			s.encodeOne(s.buffer[(offset + i + 7)]),
		})

		for j := range 8 {
			s.stats.NumEncodedSamples++
			if j == 0 {
				slog.Debug(fmt.Sprintf("%016b -> N/A (most significant bits in next %d bytes)", s.buffer[(offset+i+j)], s.encodedSeqAlign-1))
				continue
			}
			slog.Debug(fmt.Sprintf("%016b -> %08b: only least-significant %d bits", s.buffer[(offset+i+j)], encoded[j-1], s.config.EncodingSize))
			s.w.WriteByte(encoded[j-1])
		}
	}

	return nil
}

func (s *CacheSampleEncoder) flushMarkerNotHits(count int) error {
	marker := uint8(-int8(count))
	slog.Debug(fmt.Sprintf("%08b: marker next %d samples are not encoded", marker, count))
	if err := binary.Write(s.w, s.config.ByteOrder, marker); err != nil {
		return err
	}
	s.stats.NumBytesAdditional += 1
	return nil
}

func (s *CacheSampleEncoder) flushBufferNotHits(offset int, count int) error {
	defer func() { s.stats.AddNotEncodedAdvanced(count) }()

	if err := s.flushMarkerNotHits(count); err != nil {
		return err
	}

	for _, q := range s.buffer[offset : offset+count] {
		slog.Debug(fmt.Sprintf("%016b -> %016b", q, q))
		if err := binary.Write(s.w, s.config.ByteOrder, q); err != nil {
			return err
		}
		s.cache.Add(q)
	}

	return nil
}

func ValidateWAVHeader(header wav.WAVHeader) error {
	if !header.IsPCM() {
		return errors.New("PCM required")
	}
	if header.NumChannels != 1 {
		return errors.New("single channel required")
	}
	if header.BitsPerSample != 16 {
		return errors.New("16 bits per sample required")
	}
	if header.BlockAlign != 2 {
		return errors.New("block align is wrong, we need 2, for 16 bits per sample")
	}
	return nil
}

func main() {
	logLevel := slog.LevelInfo
	if s := os.Getenv("LOG_LEVEL"); s != "" {
		if err := (&logLevel).UnmarshalText([]byte(s)); err != nil {
			log.Fatal(err)
		}
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})))

	var (
		mode        string
		inFilename  string
		outFilename string
	)
	flag.StringVar(&mode, "mode", "encode", "encode or decode")
	flag.StringVar(&inFilename, "in", "", "filepath for input")
	flag.StringVar(&outFilename, "out", "", "filepath for output")
	flag.Parse()

	var in io.Reader = os.Stdin
	var out io.Writer = os.Stdout

	if inFilename != "" {
		f, err := os.Open(inFilename)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		in = f
	}

	if outFilename != "" {
		f, err := os.Create(outFilename)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		out = f
	}

	wavReader := wav.NewWAVReader(in)
	if err := wavReader.ReadHeader(); err != nil {
		log.Fatal(err)
	}

	slog.Info("wav info", "header", wavReader.Header, "PCM", wavReader.Header.IsPCM())

	if err := ValidateWAVHeader(wavReader.Header); err != nil {
		log.Fatal(err)
	}

	wavWriter := wav.NewWAVWriter(wavReader.Header, out)
	wavWriter.WriteHeader()

	byteWAVWriter := bufio.NewWriter(wavWriter)
	defer byteWAVWriter.Flush()

	encoder := NewCacheSampleEncoder(
		CacheSampleEncoderConfig{
			EncodingSize:        7,
			EncodedSeqMaxLen:    (1 << 15) - 1,
			EncodedSeqMinLen:    8,
			NotEncodedSeqMaxLen: (1 << 7) - 1,
			ByteOrder:           binary.LittleEndian,
		},
		NewCache(CacheConfig{Size: 128}),
		byteWAVWriter,
	)
	defer func() { slog.Info("done", "stats", encoder.Stats()) }()
	defer encoder.FlushBuffer()

	for sample, err := wavReader.Next(); err != io.EOF; sample, err = wavReader.Next() {
		if err := encoder.Write(sample); err != nil {
			log.Fatal(err)
		}
	}
}
