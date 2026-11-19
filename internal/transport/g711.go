package transport

// MuLawToLinearTable provides instantaneous O(1) decompression from 8-bit G.711 mu-law to 16-bit linear PCM.
var MuLawToLinearTable [256]int16

// ALawToLinearTable provides instantaneous O(1) decompression from 8-bit G.711 A-law to 16-bit linear PCM.
var ALawToLinearTable [256]int16

func init() {
	// Initialize G.711 mu-law lookup table
	for i := 0; i < 256; i++ {
		MuLawToLinearTable[i] = decodeMuLawByte(byte(i))
	}

	// Initialize G.711 A-law lookup table
	for i := 0; i < 256; i++ {
		ALawToLinearTable[i] = decodeALawByte(byte(i))
	}
}

func decodeMuLawByte(uVal byte) int16 {
	uVal = ^uVal
	t := int16((uVal&0x0F)<<3) + 0x84
	t <<= (uVal & 0x70) >> 4
	if (uVal & 0x80) != 0 {
		return 0x84 - t
	}
	return t - 0x84
}

func decodeALawByte(aVal byte) int16 {
	aVal ^= 0x55
	t := int16(aVal & 0x7F)
	seg := int16((aVal & 0x70) >> 4)
	if seg > 0 {
		t = ((t & 0x0F) | 0x10) << (seg + 3)
	} else {
		t = (t << 4) + 8
	}
	if (aVal & 0x80) != 0 {
		return t
	}
	return -t
}

// DecodeMuLaw converts a slice of G.711 mu-law bytes into normalized float32 samples in [-1.0, 1.0].
func DecodeMuLaw(src []byte) []float32 {
	out := make([]float32, len(src))
	for i, b := range src {
		pcm := MuLawToLinearTable[b]
		out[i] = float32(pcm) / 32768.0
	}
	return out
}

// DecodeALaw converts a slice of G.711 A-law bytes into normalized float32 samples in [-1.0, 1.0].
func DecodeALaw(src []byte) []float32 {
	out := make([]float32, len(src))
	for i, b := range src {
		pcm := ALawToLinearTable[b]
		out[i] = float32(pcm) / 32768.0
	}
	return out
}

// DecodePCM16LittleEndian converts 16-bit signed integer bytes into float32 samples.
func DecodePCM16LittleEndian(src []byte) []float32 {
	n := len(src) / 2
	out := make([]float32, n)
	for i := 0; i < n; i++ {
		val := int16(src[i*2]) | (int16(src[i*2+1]) << 8)
		out[i] = float32(val) / 32768.0
	}
	return out
}
