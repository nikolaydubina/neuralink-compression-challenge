package bits

type Packer interface {
	Pack(vs []byte) []byte
	Unpack(vs []byte) []byte
	MaxKeyIndex() int
	PackedLen() int
	UnpackedLen() int
	EncodingSize() int
}

var Packers = map[int]Packer{
	4: FourBitPacker{},
	6: SixBitPacker{},
	7: SevenBitPacker{},
}

type FourBitPacker struct{}

func (s FourBitPacker) Pack(vs []byte) []byte { return SlicePack2x4bit(vs) }

func (s FourBitPacker) Unpack(vs []byte) []byte { return SliceUnpack2x4bit(vs) }

func (s FourBitPacker) MaxKeyIndex() int { return (1 << 4) - 1 }

func (s FourBitPacker) PackedLen() int { return 1 }

func (s FourBitPacker) UnpackedLen() int { return 2 }

func (s FourBitPacker) EncodingSize() int { return 4 }

type SixBitPacker struct{}

func (s SixBitPacker) Pack(vs []byte) []byte { return SlicePack4x6bit(vs) }

func (s SixBitPacker) Unpack(vs []byte) []byte { return SliceUnpack4x6bit(vs) }

func (s SixBitPacker) MaxKeyIndex() int { return (1 << 6) - 1 }

func (s SixBitPacker) PackedLen() int { return 3 }

func (s SixBitPacker) UnpackedLen() int { return 4 }

func (s SixBitPacker) EncodingSize() int { return 6 }

type SevenBitPacker struct{}

func (s SevenBitPacker) Pack(vs []byte) []byte { return SlicePack8x7bit(vs) }

func (s SevenBitPacker) Unpack(vs []byte) []byte { return SliceUnpack8x7bit(vs) }

func (s SevenBitPacker) MaxKeyIndex() int { return (1 << 7) - 1 }

func (s SevenBitPacker) PackedLen() int { return 7 }

func (s SevenBitPacker) UnpackedLen() int { return 8 }

func (s SevenBitPacker) EncodingSize() int { return 7 }
