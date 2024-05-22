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
	NumEncodedSamples   int
	NumTotalSamples     int
	RatioEncodedSamples float32
	MaxLenHitsAdvanced  int
}

func (s *CacheSampleEncoderStats) AddEncodedAdvanced(advanced int) {
	if s.MaxLenHitsAdvanced < advanced {
		s.MaxLenHitsAdvanced = advanced
	}
}

type CacheConfig struct {
	Size int
}

// Cache is not as efficient, but is ok for prototype.
type Cache struct {
	config CacheConfig
	count  map[uint16]int
}

func NewCache(config CacheConfig) *Cache {
	return &Cache{
		config: config,
		count:  make(map[uint16]int, config.Size),
	}
}

func (s *Cache) Pop() {
	if len(s.count) == 0 {
		return
	}
	var minK uint16
	minCount := -1
	for k, v := range s.count {
		if v <= minCount || minCount < 0 {
			minK, minCount = k, v
		}
	}
	delete(s.count, minK)
}

func (s *Cache) Add(v uint16) {
	if s.IsFull() {
		s.Pop()
	}
	s.count[v]++
}

func (s *Cache) Index(v uint16) int {
	var orderedCache []uint16
	for k := range s.count {
		orderedCache = append(orderedCache, k)
	}
	// TODO: make new key always last, so index should be maximum. stable sort?
	sort.Slice(orderedCache, func(i, j int) bool { return s.count[orderedCache[i]] > s.count[orderedCache[j]] })
	for i, k := range orderedCache {
		if k == v {
			return i
		}
	}
	return -1
}

func (s *Cache) IsFull() bool { return len(s.count) >= s.config.Size }

type CacheSampleEncoderConfig struct {
	EncodingSize     int
	EncodedSeqMaxLen int
	EncodedSeqMinLen int
	ByteOrder        binary.ByteOrder
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

	// to flush more frequently until cache is full
	if s.cache.Index(v) < 0 && !s.cache.IsFull() {
		if err := s.FlushBuffer(); err != nil {
			return err
		}
	}

	return nil
}

func (s *CacheSampleEncoder) encodeOne(v uint16) byte {
	i := s.cache.Index(v)
	if i < 0 || i > s.maxKeyIndex {
		err := fmt.Errorf("value(%v) got index(%v) is out of bound for encoded key, expected [0, %d]", v, i, s.maxKeyIndex)
		panic(err)
	}
	return byte(i)
}

func (s *CacheSampleEncoder) FlushBuffer() error {
	if len(s.buffer) == 0 {
		return nil
	}

	for offset := 0; offset < len(s.buffer); {
		advanced, err := s.flushBufferHits(offset)
		offset += advanced
		if err != nil {
			return err
		}

		// try to flush rest non-hits
		if count := s.flushBufferNotHitsCount(offset); count > 0 {
			advanced, err = s.flushBufferNotHits(offset, count)
			offset += advanced
			if err != nil {
				return err
			}
		}

		// there samples to flush, but they are not hits,
		// and if they are hits they can not be encoded.
		// flush them not-encoded
		if advanced == 0 {
			count := s.encodedSeqAlign
			if (offset + count) > len(s.buffer) {
				count = len(s.buffer) - offset
			}
			advanced, err = s.flushBufferNotHits(offset, count)
			offset += advanced
			if err != nil {
				return err
			}
		}
	}

	s.buffer = s.buffer[:0]
	return nil
}

func (s *CacheSampleEncoder) flushBufferHits(offset int) (advanced int, err error) {
	defer func() {
		s.stats.AddEncodedAdvanced(advanced)
	}()

	count := 0
	for i := offset; i < len(s.buffer) && s.cache.Index(s.buffer[i]) >= 0; i++ {
		count++
	}

	if count < s.config.EncodedSeqMinLen {
		count = 0
	}

	count = count - (count % s.encodedSeqAlign)

	if count == 0 {
		return 0, nil
	}

	marker := uint16(count)
	slog.Debug(fmt.Sprintf("%016b", marker), "marker", "next samples are encoded", "n", count)
	if err := binary.Write(s.w, s.config.ByteOrder, marker); err != nil {
		return 0, err
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

	return count, nil
}

func (s *CacheSampleEncoder) flushBufferNotHitsCount(offset int) int {
	count := 0
	for i := offset; i < len(s.buffer) && s.cache.Index(s.buffer[i]) < 0; i++ {
		count++
	}
	return count
}

func (s *CacheSampleEncoder) flushBufferNotHits(offset int, count int) (advanced int, err error) {
	marker := uint16(-int8(count))
	slog.Debug(fmt.Sprintf("%016b: next %d samples are not encoded", marker, count))
	if err := binary.Write(s.w, s.config.ByteOrder, marker); err != nil {
		return 0, err
	}

	for _, q := range s.buffer[offset : offset+count] {
		slog.Debug(fmt.Sprintf("%016b -> %016b", q, q))
		if err := binary.Write(s.w, s.config.ByteOrder, q); err != nil {
			return 0, err
		}
		s.cache.Add(q)
	}

	return count, nil
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
			EncodingSize:     7,
			EncodedSeqMaxLen: (1 << 15) - 1,
			EncodedSeqMinLen: 8,
			ByteOrder:        binary.LittleEndian,
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
