package bits_test

import (
	"fmt"
	"testing"

	"github.com/nikolaydubina/neuralink-compression-challenge/go-encoder/bits"
)

func ExamplePack2x4bit() {
	vs := [2]byte{
		0b0000_0111,
		0b0000_1110,
	}
	packed := bits.Pack2x4bit(vs)
	for _, p := range packed {
		fmt.Printf("%08b\n", p)
	}
	// Output:
	// 11100111
}

func FuzzExamplePack2x4bit(f *testing.F) {
	f.Fuzz(func(t *testing.T, v0, v1 byte) {
		vs := [2]byte{v0, v1}

		for i := range vs {
			vs[i] = vs[i] & 0x0F
		}

		packed := bits.Pack2x4bit(vs)
		unpacked := bits.Unpack2x4bit(packed)

		if unpacked != vs {
			t.Errorf("exp(%v) != got (%v)", vs, unpacked)
		}
	})
}

func FuzzExamplePack4x6bit(f *testing.F) {
	f.Fuzz(func(t *testing.T, v0, v1, v2, v3 byte) {
		vs := [4]byte{v0, v1, v2, v3}

		// clear most significant bits
		for i := range vs {
			vs[i] = vs[i] & 0x3F
		}

		packed := bits.Pack4x6bit(vs)
		unpacked := bits.Unpack4x6bit(packed)

		if unpacked != vs {
			t.Errorf("exp(%v) != got (%v)", vs, unpacked)
		}
	})
}

func ExamplePack8x7bit() {
	vs := [8]byte{
		0b0001_0010,
		0b0010_0101,
		0b0011_0110,
		0b0100_1001,
		0b0101_1010,
		0b0110_1100,
		0b0111_1110,
		0b0001_1111,
	}
	packed := bits.Pack8x7bit(vs)
	for _, p := range packed {
		fmt.Printf("%08b\n", p)
	}
	// Output:
	// 00100101
	// 00110110
	// 11001001
	// 01011010
	// 01101100
	// 11111110
	// 00011111
}

func ExampleUnpack8x7bit() {
	vs := [7]byte{
		0b00100101,
		0b00110110,
		0b11001001,
		0b01011010,
		0b01101100,
		0b11111110,
		0b00011111,
	}
	packed := bits.Unpack8x7bit(vs)
	for _, p := range packed {
		fmt.Printf("%08b\n", p)
	}
	// Output:
	// 00010010
	// 00100101
	// 00110110
	// 01001001
	// 01011010
	// 01101100
	// 01111110
	// 00011111
}

func FuzzExamplePack8x7bit(f *testing.F) {
	f.Fuzz(func(t *testing.T, v0, v1, v2, v3, v4, v5, v6, v7 byte) {
		vs := [8]byte{v0, v1, v2, v3, v4, v5, v6, v7}

		// clear most significant bit
		for i := range vs {
			vs[i] = vs[i] & 0x7F
		}

		packed := bits.Pack8x7bit(vs)
		unpacked := bits.Unpack8x7bit(packed)

		if unpacked != vs {
			t.Errorf("exp(%v) != got (%v)", vs, unpacked)
		}
	})
}
