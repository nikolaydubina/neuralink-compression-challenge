package bits

func Pack8x7bit(vs [8]byte) [7]byte {
	return [7]byte{
		(vs[1] | ((vs[0] << 1) & 0x80)),
		(vs[2] | ((vs[0] << 2) & 0x80)),
		(vs[3] | ((vs[0] << 3) & 0x80)),
		(vs[4] | ((vs[0] << 4) & 0x80)),
		(vs[5] | ((vs[0] << 5) & 0x80)),
		(vs[6] | ((vs[0] << 6) & 0x80)),
		(vs[7] | ((vs[0] << 7) & 0x80)),
	}
}

func Unpack8x7bit(vs [7]byte) [8]byte {
	var v0 byte
	for i := range 7 {
		v0 |= (vs[i] & 0x80) >> (i + 1)
	}
	return [8]byte{
		v0,
		(vs[0] & 0x7F),
		(vs[1] & 0x7F),
		(vs[2] & 0x7F),
		(vs[3] & 0x7F),
		(vs[4] & 0x7F),
		(vs[5] & 0x7F),
		(vs[6] & 0x7F),
	}
}

func Pack4x6bit(vs [4]byte) [3]byte {
	return [3]byte{
		(vs[1] | ((vs[0] << 2) & 0xC0)),
		(vs[2] | ((vs[0] << 4) & 0xC0)),
		(vs[3] | ((vs[0] << 6) & 0xC0)),
	}
}

func Unpack4x6bit(vs [3]byte) [4]byte {
	var v0 byte
	for i := range 3 {
		v0 |= (vs[i] & 0xC0) >> ((i + 1) * 2)
	}
	return [4]byte{
		v0,
		(vs[0] & 0x3F),
		(vs[1] & 0x3F),
		(vs[2] & 0x3F),
	}
}

func SlicePack8x7bit(vs []byte) []byte {
	e := Pack8x7bit([8]byte{vs[0], vs[1], vs[2], vs[3], vs[4], vs[5], vs[6], vs[7]})
	return e[:]
}

func SliceUnpack8x7bit(vs []byte) []byte {
	e := Unpack8x7bit([7]byte{vs[0], vs[1], vs[2], vs[3], vs[4], vs[5], vs[6]})
	return e[:]
}

func SlicePack4x6bit(vs []byte) []byte {
	e := Pack4x6bit([4]byte{vs[0], vs[1], vs[2], vs[3]})
	return e[:]
}

func SliceUnpack4x6bit(vs []byte) []byte {
	e := Unpack4x6bit([3]byte{vs[0], vs[1], vs[2]})
	return e[:]
}
