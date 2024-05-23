# [Neuralink Compression Challenge](https://content.neuralink.com/compression-challenge/README.html)

by nikolay.dubina.pub@gmail.com on 2024-05-23

Compression Ratio 1.78

Algorithm
- read `int16`, if N > 0, then next N samples are encoded, if N < 0 then next abs(N) samples are not encoded
- cache `1024` most frequently observed samples so far, update cache on every encoding/decoding
- use index in that cache to encode value, fixed 6 bits, use top `64` values for encoding, else write unencoded
- when encoding, then N % 4 == 0
- keep original WAV header, overwrite only data segments

Example

```
<encoded bytes>: description
```

```
1111111111111110: marker next 2 samples encoded=false
0000001010100000 -> 0000001010100000
0000001011100000 -> 0000001011100000
0000000101111100: marker next 380 samples encoded=true
0000001110100000 -> N/A (most significant bits in next 3 bytes)
0000001101100000 -> 00001010: only least-significant 6 bits
0000001011100000 -> 11000001: only least-significant 6 bits
0000000110011111 -> 00000110: only least-significant 6 bits
0000000001011111 -> N/A (most significant bits in next 3 bytes)
0000000101011111 -> 01001000: only least-significant 6 bits
0000000111011111 -> 01000000: only least-significant 6 bits
0000000111011111 -> 01000000: only least-significant 6 bits
0000001000100000 -> N/A (most significant bits in next 3 bytes)
0000001011100000 -> 00000001: only least-significant 6 bits
0000001100100000 -> 01010011: only least-significant 6 bits
0000000100011111 -> 11001001: only least-significant 6 bits
0000000111011111 -> N/A (most significant bits in next 3 bytes)
0000000011011111 -> 00000010: only least-significant 6 bits
0000001000100000 -> 00000111: only least-significant 6 bits
0000001001100000 -> 00001110: only least-significant 6 bits
0000001110100000 -> N/A (most significant bits in next 3 bytes)
0000010000100000 -> 00010001: only least-significant 6 bits
0000010000100000 -> 11010001: only least-significant 6 bits
...
```

Properties of Algorithm
- Does not use information within single sample, only sample equality among other samples and their chronology is used
- Adaptive Dictionary
- Fixed-Length Coding
- Byte-Aligned Coding
- Turn Off/On Switch with raw data

Other Materials
- `/go-encoder` - Go version
- `/research` - research code and Python version

## References

* http://tiny.systems/software/soundProgrammer/WavFormatDocs.pdf
* https://iopscience.iop.org/article/10.1088/1741-2552/acf5a4
* https://docs.scipy.org/doc/scipy/reference/generated/scipy.io.wavfile.read.html
* https://docs.python.org/3/library/wave.html
* https://en.wikipedia.org/wiki/Variable-length_code
* https://en.wikipedia.org/wiki/Prefix_code
* https://rosettacode.org/wiki/Huffman_coding#Python
* https://golang.google.cn/src/compress/bzip2/huffman.go
* https://github.com/go-audio/wav
* https://github.com/bearmini/bitstream-go
* https://github.com/icza/bitio
* https://github.com/icza/huffman
* https://github.com/studiawan/data-compression
* https://en.wikipedia.org/wiki/Adaptive_Huffman_coding
* https://github.com/kei-g/vitter
* https://en.wikipedia.org/wiki/Deflate
