package encoding_test

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/nikolaydubina/neuralink-compression-challenge/go-encoder/encoding"
)

func ExampleMarker() {
	marker := encoding.Marker{
		Count:        2,
		IsEncoded:    true,
		EncodingSize: 7,
	}
	var b bytes.Buffer
	marker.MarshalBinaryToWriter(&b, binary.LittleEndian)
	fmt.Printf("%08b\n", b.Bytes())
	// Output: [00001010 00000000]
}

func ExampleMarker_otherEncodingSize() {
	marker := encoding.Marker{
		Count:        2,
		IsEncoded:    true,
		EncodingSize: 6,
	}
	var b bytes.Buffer
	marker.MarshalBinaryToWriter(&b, binary.LittleEndian)
	fmt.Printf("%08b\n", b.Bytes())
	// Output: [00001001 00000000]
}

func ExampleMarker_notEncoded() {
	marker := encoding.Marker{
		Count:     3,
		IsEncoded: false,
	}
	var b bytes.Buffer
	marker.MarshalBinaryToWriter(&b, binary.LittleEndian)
	var x uint16
	fmt.Printf("%08b and -3=(%016b)\n", b.Bytes(), x-3)
	// Output: [11110100 11111111] and -3=(1111111111111101)
}

func FuzzMarker(f *testing.F) {
	f.Fuzz(func(t *testing.T, count int, enc uint8) {
		marker := encoding.Marker{
			Count: count,
		}
		if marker.Count < 0 {
			marker.IsEncoded = false
			marker.EncodingSize = 0
			marker.Count = -marker.Count
		} else {
			marker.IsEncoded = true
			switch enc % 3 {
			case 0:
				marker.EncodingSize = 7
			case 1:
				marker.EncodingSize = 6
			case 2:
				marker.EncodingSize = 4
			}
		}

		var b bytes.Buffer
		if err := marker.MarshalBinaryToWriter(&b, binary.LittleEndian); err != nil {
			t.Error(err)
		}

		var unpacked encoding.Marker
		if err := (&unpacked).UnmarshalBinaryFromReader(&b, binary.LittleEndian); err != nil {
			t.Error(err)
		}

		if marker != unpacked {
			t.Errorf("exp(%#v) != got(%#v)", marker, unpacked)
		}
	})
}
