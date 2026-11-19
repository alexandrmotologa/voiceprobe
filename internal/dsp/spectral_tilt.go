package dsp

import (
	"math"
)

// SpectralTiltMetrics holds acoustic resonance decay metrics.
type SpectralTiltMetrics struct {
	SlopeDbPerOctave float64   `json:"slope_db_per_octave"`
	FormantFrequencies []float64 `json:"formant_frequencies"`
	FormantDispersion  float64   `json:"formant_dispersion"`
}

// ComputeSpectralTilt measures spectral energy decay across octaves and vocal tract formant dispersion.
// Human phonation exhibits a natural downward slope of -4 to -8 dB/octave.
// Neural vocoders frequently exhibit unnaturally shallow or flat slopes (-1 to -3 dB/octave)
// due to uniform sub-band loss weights during training.
func ComputeSpectralTilt(mags []float64, sampleRate, fftSize int) SpectralTiltMetrics {
	bins := len(mags)
	hzPerBin := float64(sampleRate) / float64(fftSize)

	// Frequency range of interest: 250 Hz to 3600 Hz (telephony vocal band)
	startBin := int(math.Round(250.0 / hzPerBin))
	endBin := int(math.Round(3600.0 / hzPerBin))
	if endBin >= bins {
		endBin = bins - 1
	}
	if startBin < 1 {
		startBin = 1
	}

	var sumX, sumY, sumXY, sumX2 float64
	var count float64

	for k := startBin; k <= endBin; k++ {
		freqHz := float64(k) * hzPerBin
		mag := mags[k]
		if mag < 1e-6 {
			mag = 1e-6
		}

		// X: octaves relative to 250 Hz (log2(f / 250))
		x := math.Log2(freqHz / 250.0)
		// Y: power in decibels (20 * log10(mag))
		y := 20.0 * math.Log10(mag)

		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
		count++
	}

	slope := -6.0 // Natural default
	if count > 2 {
		denom := count*sumX2 - sumX*sumX
		if math.Abs(denom) > 1e-9 {
			slope = (count*sumXY - sumX*sumY) / denom
		}
	}

	// Peak picking for Formants F1, F2, F3 in regions:
	// F1: 300 - 900 Hz, F2: 900 - 2200 Hz, F3: 2200 - 3600 Hz
	f1 := findPeakFreq(mags, 300.0, 900.0, hzPerBin, bins)
	f2 := findPeakFreq(mags, 900.0, 2200.0, hzPerBin, bins)
	f3 := findPeakFreq(mags, 2200.0, 3600.0, hzPerBin, bins)

	formants := []float64{f1, f2, f3}

	// Formant dispersion: average spacing between consecutive resonances: (F3 - F1) / 2
	dispersion := (f3 - f1) / 2.0
	if dispersion <= 0 {
		dispersion = 1000.0 // Standard human vocal tract average (~17.5cm tract)
	}

	return SpectralTiltMetrics{
		SlopeDbPerOctave:   slope,
		FormantFrequencies: formants,
		FormantDispersion:  dispersion,
	}
}

func findPeakFreq(mags []float64, minHz, maxHz, hzPerBin float64, maxBin int) float64 {
	start := int(minHz / hzPerBin)
	end := int(maxHz / hzPerBin)
	if start < 1 {
		start = 1
	}
	if end >= maxBin {
		end = maxBin - 1
	}

	maxMag := -1.0
	bestBin := start
	for k := start; k <= end; k++ {
		if mags[k] > maxMag {
			maxMag = mags[k]
			bestBin = k
		}
	}

	return float64(bestBin) * hzPerBin
}
