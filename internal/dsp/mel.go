package dsp

import (
	"math"
)

// MelFilterbank projects linear frequency FFT bins into Mel-spaced energy bands.
type MelFilterbank struct {
	numFilters int
	fftSize    int
	sampleRate int
	weights    [][]float64 // [numFilters][fftBins]
}

// HzToMel converts a frequency in Hertz to the Mel scale.
func HzToMel(hz float64) float64 {
	return 2595.0 * math.Log10(1.0+hz/700.0)
}

// MelToHz converts a Mel value back to Hertz.
func MelToHz(mel float64) float64 {
	return 700.0 * (math.Pow(10.0, mel/2595.0) - 1.0)
}

// NewMelFilterbank creates a triangular filterbank matrix.
func NewMelFilterbank(numFilters, fftSize, sampleRate int) *MelFilterbank {
	bins := fftSize/2 + 1
	fb := &MelFilterbank{
		numFilters: numFilters,
		fftSize:    fftSize,
		sampleRate: sampleRate,
		weights:    make([][]float64, numFilters),
	}

	lowHz := 60.0
	highHz := float64(sampleRate) / 2.0
	lowMel := HzToMel(lowHz)
	highMel := HzToMel(highHz)

	melPoints := make([]float64, numFilters+2)
	for i := 0; i < len(melPoints); i++ {
		melPoints[i] = lowMel + float64(i)*(highMel-lowMel)/float64(numFilters+1)
	}

	hzPoints := make([]float64, len(melPoints))
	binPoints := make([]int, len(melPoints))
	for i := 0; i < len(melPoints); i++ {
		hzPoints[i] = MelToHz(melPoints[i])
		bin := int(math.Floor((float64(fftSize)+1.0)*hzPoints[i]/float64(sampleRate) + 0.5))
		if bin >= bins {
			bin = bins - 1
		}
		binPoints[i] = bin
	}

	for m := 0; m < numFilters; m++ {
		fb.weights[m] = make([]float64, bins)
		left := binPoints[m]
		center := binPoints[m+1]
		right := binPoints[m+2]

		// Upward slope
		if center > left {
			for k := left; k < center; k++ {
				fb.weights[m][k] = float64(k-left) / float64(center-left)
			}
		}

		// Downward slope
		if right > center {
			for k := center; k <= right; k++ {
				fb.weights[m][k] = float64(right-k) / float64(right-center)
			}
		}
	}

	return fb
}

// ComputeInto computes log Mel energy bands into a pre-allocated dst slice without memory allocations.
func (fb *MelFilterbank) ComputeInto(mags []float64, dst []float64) {
	bins := len(mags)

	for m := 0; m < fb.numFilters && m < len(dst); m++ {
		var sum float64
		weights := fb.weights[m]
		limit := len(weights)
		if bins < limit {
			limit = bins
		}

		for k := 0; k < limit; k++ {
			sum += mags[k] * weights[k]
		}

		dst[m] = math.Log(1.0 + sum)
	}
}

// Compute allocates and returns log Mel energy bands.
func (fb *MelFilterbank) Compute(mags []float64) []float64 {
	energies := make([]float64, fb.numFilters)
	fb.ComputeInto(mags, energies)
	return energies
}
