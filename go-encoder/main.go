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
	"github.com/nikolaydubina/neuralink-compression-challenge/go-encoder/encoding"
	"github.com/nikolaydubina/neuralink-compression-challenge/go-encoder/wav"
)

type CacheSampleEncoderStats struct {
	NumEncodedSamples               int
	NumTotalSamples                 int
	RatioEncodedSamples             float32
	NumBytesAdditional              int
	MaxLenHitsAdvanced              int
	MaxLenNotHitsAdvanced           int
	NumHitsAdvanced                 int
	NumNotHitsAdvanced              int
	NumForcedUnpacked               int
	NumBytesForcedUnpacked          int
	NumSamplesEncodedByEncodingSize map[int]int
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

func (s *Cache) At(i int) uint16 { return s.order[i].key }

func (s *Cache) IsFull() bool { return len(s.order) >= s.config.Size }

type Packer interface {
	Pack(vs []byte) []byte
	Unpack(vs []byte) []byte
	MaxKeyIndex() int
	PackedLen() int
	UnpackedLen() int
	EncodingSize() int
}

type SixBitPacker struct{}

func (s SixBitPacker) Pack(vs []byte) []byte { return bits.SlicePack4x6bit(vs) }

func (s SixBitPacker) Unpack(vs []byte) []byte { return bits.SliceUnpack4x6bit(vs) }

func (s SixBitPacker) MaxKeyIndex() int { return (1 << 6) - 1 }

func (s SixBitPacker) PackedLen() int { return 3 }

func (s SixBitPacker) UnpackedLen() int { return 4 }

func (s SixBitPacker) EncodingSize() int { return 6 }

type SevenBitPacker struct{}

func (s SevenBitPacker) Pack(vs []byte) []byte { return bits.SlicePack8x7bit(vs) }

func (s SevenBitPacker) Unpack(vs []byte) []byte { return bits.SliceUnpack8x7bit(vs) }

func (s SevenBitPacker) MaxKeyIndex() int { return (1 << 7) - 1 }

func (s SevenBitPacker) PackedLen() int { return 7 }

func (s SevenBitPacker) UnpackedLen() int { return 8 }

func (s SevenBitPacker) EncodingSize() int { return 7 }

var Packers = map[int]Packer{
	6: SixBitPacker{},
	7: SevenBitPacker{},
}

type CacheSampleEncoderConfig struct {
	EncodedSeqMaxLen    int
	NotEncodedSeqMaxLen int
	ByteOrder           binary.ByteOrder
}

type CacheSampleEncoder struct {
	config CacheSampleEncoderConfig
	stats  CacheSampleEncoderStats
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
		config: config,
		stats: CacheSampleEncoderStats{
			NumSamplesEncodedByEncodingSize: make(map[int]int),
		},
		cache:  cache,
		w:      w,
		buffer: make([]uint16, 0, config.EncodedSeqMaxLen),
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

func (s *CacheSampleEncoder) encodeOne(v uint16, encodingSize int) byte {
	i := s.cache.Index(v)
	if i < 0 || i > Packers[encodingSize].MaxKeyIndex() {
		err := fmt.Errorf("value(%v) got index(%v) is out of bound for encoded key, expected [0, %d]", v, i, Packers[encodingSize].MaxKeyIndex())
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

		packer, countHits := s.flushBufferHitsCount(offset)
		countNotHits := s.flushBufferNotHitsCount(offset + countHits)

		// there samples to flush, but they are not hits,
		// and if they are hits they can not be encoded.
		// this number is within un-aligned encoded sequence.
		// flush them not-encoded.
		if countHits == 0 && countNotHits == 0 {
			countNotHits = Packers[6].UnpackedLen()
			if (offset + countNotHits) > len(s.buffer) {
				countNotHits = len(s.buffer) - offset
			}

			s.stats.NumForcedUnpacked++
			s.stats.NumBytesForcedUnpacked += countNotHits
		}

		if countHits > 0 {
			if err := s.flushBufferHits(offset, countHits, packer); err != nil {
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

func (s *CacheSampleEncoder) flushBufferHitsCount(offset int) (p Packer, count int) {
	count6 := 0
	for i := offset; i < len(s.buffer); i++ {
		if idx := s.cache.Index(s.buffer[i]); idx < 0 || idx > Packers[6].MaxKeyIndex() {
			break
		}
		count6++
	}
	count6 = count6 - (count6 % Packers[6].UnpackedLen())

	count7 := 0
	for i := offset; i < len(s.buffer); i++ {
		if idx := s.cache.Index(s.buffer[i]); idx < 0 || idx > Packers[7].MaxKeyIndex() {
			break
		}
		count7++
	}
	count7 = count7 - (count7 % Packers[7].UnpackedLen())

	// pick packing that leads to less bytes
	numBytes6 := float64(count6) * 3 / 4
	numBytes7 := float64(count7) * 7 / 8

	if numBytes6 > 0 && numBytes6 < numBytes7 {
		p, count = Packers[6], count6
	} else if numBytes7 > 0 {
		p, count = Packers[7], count7
	}

	return p, count
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

func (s *CacheSampleEncoder) flushBufferHits(offset, count int, packer Packer) error {
	defer func() { s.stats.AddEncodedAdvanced(count) }()

	marker := encoding.Marker{
		Count:        count,
		EncodingSize: packer.EncodingSize(),
		IsEncoded:    true,
	}
	s.stats.NumBytesAdditional += marker.SizeBytes()
	if err := marker.MarshalBinaryToWriter(s.w, s.config.ByteOrder); err != nil {
		return err
	}

	unpacked := make([]byte, packer.UnpackedLen())
	for i := 0; i < count; i += packer.UnpackedLen() {
		for j := range unpacked {
			unpacked[j] = s.encodeOne(s.buffer[(offset+i+j)], packer.EncodingSize())
		}

		encoded := packer.Pack(unpacked)
		for j := range packer.UnpackedLen() {
			s.stats.NumEncodedSamples++
			s.stats.NumSamplesEncodedByEncodingSize[packer.EncodingSize()] += 1

			if j == 0 {
				slog.Debug(fmt.Sprintf("%016b -> N/A (most significant bits in next %d bytes)", s.buffer[(offset+i+j)], packer.UnpackedLen()-1))
				continue
			}
			slog.Debug(fmt.Sprintf("%016b -> %08b: only least-significant %d bits", s.buffer[(offset+i+j)], encoded[j-1], packer.EncodingSize()))
			s.w.WriteByte(encoded[j-1])
		}
	}

	return nil
}

func (s *CacheSampleEncoder) flushBufferNotHits(offset int, count int) error {
	defer func() { s.stats.AddNotEncodedAdvanced(count) }()

	marker := encoding.Marker{Count: count, IsEncoded: false}
	s.stats.NumBytesAdditional += marker.SizeBytes()
	if err := marker.MarshalBinaryToWriter(s.w, s.config.ByteOrder); err != nil {
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

type CacheSampleDecoder struct {
	config CacheSampleEncoderConfig
	cache  *Cache
	r      io.Reader
	buffer []uint16 // reverse order
}

func NewCacheSampleDecoder(
	config CacheSampleEncoderConfig,
	cache *Cache,
	r io.Reader,
) *CacheSampleDecoder {
	return &CacheSampleDecoder{
		config: config,
		cache:  cache,
		r:      r,
		buffer: make([]uint16, 0, config.EncodedSeqMaxLen),
	}
}

func (s *CacheSampleDecoder) Next() (sample uint16, err error) {
	if len(s.buffer) == 0 {
		if err := s.readIntoBuffer(); err != nil {
			return 0, err
		}
	}

	sample = s.buffer[len(s.buffer)-1]
	s.buffer = s.buffer[:len(s.buffer)-1]
	return sample, nil
}

func (s *CacheSampleDecoder) readIntoBuffer() error {
	var marker encoding.Marker
	if err := marker.UnmarshalBinaryFromReader(s.r, s.config.ByteOrder); err != nil {
		return err
	}

	if marker.IsEncoded {
		return s.readEncoded(marker.Count, Packers[marker.EncodingSize])
	}

	return s.readNotEncoded(marker.Count)
}

func (s *CacheSampleDecoder) readNotEncoded(count int) error {
	for i := 0; i < count; i++ {
		var sample uint16
		if err := binary.Read(s.r, s.config.ByteOrder, &sample); err != nil {
			return err
		}
		s.cache.Add(sample)
		// reverse order into buffer
		s.buffer = append([]uint16{sample}, s.buffer...)
		slog.Debug(fmt.Sprintf("%016b -> %016b", sample, sample))
	}
	return nil
}

func (s *CacheSampleDecoder) readEncoded(count int, packer Packer) error {
	packed := make([]byte, packer.PackedLen())
	for i := 0; i < count; i += packer.UnpackedLen() {
		if _, err := io.ReadFull(s.r, packed); err != nil {
			return err
		}
		unpacked := packer.Unpack(packed)
		for i, q := range unpacked {
			decoded := s.cache.At(int(q))
			s.cache.Add(decoded)
			// reverse order into buffer
			s.buffer = append([]uint16{decoded}, s.buffer...)
			slog.Debug(fmt.Sprintf("%08b -> %016b", q, decoded), "pack_idx", i, "pack_len", len(unpacked))
		}
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
	flag.StringVar(&mode, "mode", "encode", "encode, decode, read (new-line delimited ASCII of binary of WAV samples)")
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

	encoderConfig := CacheSampleEncoderConfig{
		EncodedSeqMaxLen:    (1 << 14) - 1,
		NotEncodedSeqMaxLen: (1 << 7) - 1,
		ByteOrder:           binary.LittleEndian,
	}
	cacheConfig := CacheConfig{
		Size: 1 << 10,
	}

	switch mode {
	case "read":
		for sample, err := wavReader.Next(); err != io.EOF; sample, err = wavReader.Next() {
			fmt.Printf("%016b\n", sample)
		}
	case "encode":
		encoder := NewCacheSampleEncoder(encoderConfig, NewCache(cacheConfig), byteWAVWriter)
		defer func() { slog.Info("done", "stats", encoder.Stats()) }()
		defer encoder.FlushBuffer()

		for sample, err := wavReader.Next(); err != io.EOF; sample, err = wavReader.Next() {
			if err := encoder.Write(sample); err != nil {
				log.Fatal(err)
			}
		}
	case "decode":
		decoder := NewCacheSampleDecoder(encoderConfig, NewCache(cacheConfig), wavReader)

		for sample, err := decoder.Next(); err != io.EOF; sample, err = decoder.Next() {
			if err := wavWriter.WriteSample(sample); err != nil {
				log.Fatal(err)
			}
		}
	}
}
