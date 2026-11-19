package dsp

import (
	"math"
)

// FFTEngine provides Fast Fourier Transforms with windowing matched to sample length.
type FFTEngine struct {
	windowLen   int
	fftSize     int
	hanningWin  []float64
	bitReversal []int
	cosTable    []float64
	sinTable    []float64
	realBuf     []float64
	imagBuf     []float64
	magsBuf     []float64
	phasesBuf   []float64
}

// NewFFTEngine initializes twiddle tables, window vectors, and scratch buffers.
// WindowLen is the actual audio frame sample count (e.g., 2400 for 300ms at 8000Hz).
func NewFFTEngine(windowLen int) *FFTEngine {
	fftSize := 1
	for fftSize < windowLen {
		fftSize <<= 1
	}

	bins := fftSize/2 + 1

	engine := &FFTEngine{
		windowLen:   windowLen,
		fftSize:     fftSize,
		hanningWin:  make([]float64, windowLen),
		bitReversal: make([]int, fftSize),
		cosTable:    make([]float64, fftSize/2),
		sinTable:    make([]float64, fftSize/2),
		realBuf:     make([]float64, fftSize),
		imagBuf:     make([]float64, fftSize),
		magsBuf:     make([]float64, bins),
		phasesBuf:   make([]float64, bins),
	}

	// Symmetric Hanning window precisely matching the windowLen
	for i := 0; i < windowLen; i++ {
		engine.hanningWin[i] = 0.5 * (1.0 - math.Cos(2.0*math.Pi*float64(i)/float64(windowLen-1)))
	}

	// Bit reversal permutation for fftSize
	bits := int(math.Log2(float64(fftSize)))
	for i := 0; i < fftSize; i++ {
		rev := 0
		for j := 0; j < bits; j++ {
			if (i & (1 << j)) != 0 {
				rev |= 1 << (bits - 1 - j)
			}
		}
		engine.bitReversal[i] = rev
	}

	// Twiddle factors for fftSize
	for i := 0; i < fftSize/2; i++ {
		angle := -2.0 * math.Pi * float64(i) / float64(fftSize)
		engine.cosTable[i] = math.Cos(angle)
		engine.sinTable[i] = math.Sin(angle)
	}

	return engine
}

// Size returns the FFT size (power of 2).
func (e *FFTEngine) Size() int {
	return e.fftSize
}

// WindowLen returns the active window sample count.
func (e *FFTEngine) WindowLen() int {
	return e.windowLen
}

// Process computes magnitude and phase spectra into internal scratch buffers.
func (e *FFTEngine) Process(samples []float32) ([]float64, []float64) {
	n := e.fftSize
	wLen := e.windowLen
	real := e.realBuf
	imag := e.imagBuf

	// Clear imag buffer
	for i := 0; i < n; i++ {
		imag[i] = 0.0
	}

	// Window with true tapering down to 0 at wLen-1
	copyLen := len(samples)
	if copyLen > wLen {
		copyLen = wLen
	}
	for i := 0; i < copyLen; i++ {
		real[i] = float64(samples[i]) * e.hanningWin[i]
	}
	// Zero-pad remainder of FFT buffer
	for i := copyLen; i < n; i++ {
		real[i] = 0.0
	}

	// Bit-reversal reordering
	for i := 0; i < n; i++ {
		j := e.bitReversal[i]
		if j > i {
			real[i], real[j] = real[j], real[i]
			imag[i], imag[j] = imag[j], imag[i]
		}
	}

	// Cooley-Tukey butterfly passes
	for step := 2; step <= n; step <<= 1 {
		halfStep := step >> 1
		twiddleStep := n / step

		for group := 0; group < n; group += step {
			for k := 0; k < halfStep; k++ {
				idx := group + k
				pairIdx := idx + halfStep
				twiddleIdx := k * twiddleStep

				cosW := e.cosTable[twiddleIdx]
				sinW := e.sinTable[twiddleIdx]

				rT := real[pairIdx]*cosW - imag[pairIdx]*sinW
				iT := real[pairIdx]*sinW + imag[pairIdx]*cosW

				real[pairIdx] = real[idx] - rT
				imag[pairIdx] = imag[idx] - iT
				real[idx] += rT
				imag[idx] += iT
			}
		}
	}

	bins := n/2 + 1
	for k := 0; k < bins; k++ {
		r := real[k]
		im := imag[k]
		e.magsBuf[k] = math.Sqrt(r*r + im*im)
		e.phasesBuf[k] = math.Atan2(im, r)
	}

	return e.magsBuf, e.phasesBuf
}
