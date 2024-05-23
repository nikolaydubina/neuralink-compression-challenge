# [Neuralink Compression Challenge](https://content.neuralink.com/compression-challenge/README.html)

by nikolay.dubina.pub@gmail.com on 2024-05-23

Compression Ratio 2.291409476957349

Algorithm
- read `int16`, if N > 0, then next N samples are encoded, if N < 0 then next abs(N) samples are not encoded
- cache `1024` most frequently observed samples so far, update cache on every encoding/decoding
- use index in that cache to encode value, fixed 6 bits, use top `64` values for encoding, else write unencoded 
- when encoding, then N % 4 == 0
- keep original WAV header, overwrite only data segments

Example

```
new bytes: <old bytes> or description what they mean and compression ratio
```

```
00000111111100011: <- 0000111111100011: 1.06 raw sample
10000000001: flush buffer, next n(1) samples are encoded with dictionary
0100111: <- 0000100000100001: 0.44
00001000100100011: <- 0001000100100011: 1.06 raw sample
00001001000100011: <- 0001001000100011: 1.06 raw sample
10000000011: flush buffer, next n(3) samples are encoded with dictionary
0100111: <- 0000100000100001: 0.44
1000000: <- 0000100000100001: 0.44
1000011: <- 0000100000100001: 0.44
00001001011100100: <- 0001001011100100: 1.06 raw sample
10000000111: flush buffer, next n(7) samples are encoded with dictionary
0110100: <- 0000100000100001: 0.44
0010001: <- 0000100000100001: 0.44
0100101: <- 0000100000100001: 0.44
0100111: <- 0000100000100001: 0.44
0101010: <- 0000100000100001: 0.44
0101000: <- 0000100000100001: 0.44
1000001: <- 0000100000100001: 0.44
00001000110100011: <- 0001000110100011: 1.06 raw sample
10100100001: flush buffer, next n(289) samples are encoded with dictionary
1000111: <- 0000100000100001: 0.44
0110000: <- 0000100000100001: 0.44
0110011: <- 0000100000100001: 0.44
0000010: <- 0000100000100001: 0.44
0100100: <- 0000100000100001: 0.44
0100100: <- 0000100000100001: 0.44
0100011: <- 0000100000100001: 0.44
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
