package dsp

import (
	"math"
)

// WienerDenoiser implements an adaptive spectral subtraction and Wiener filtering stage.
// It tracks background acoustic noise during unvoiced intervals and suppresses additive noise
// without musical noise artifacts, stabilizing feature extraction on degraded telephony lines.
type WienerDenoiser struct {
	bins         int
	noiseFloor   []float64
	smoothedPsd  []float64
	gainBuf      []float64
	alphaNoise   float64 // Smoothing factor for noise tracking
	alphaPsd     float64 // Smoothing factor for signal PSD
	overSubFact  float64 // Over-subtraction factor
	minGain      float64 // Spectral floor minimum gain
	hasNoiseEst  bool
}

// NewWienerDenoiser creates an adaptive noise filter with pre-allocated scratch buffers.
func NewWienerDenoiser(fftSize int) *WienerDenoiser {
	bins := fftSize/2 + 1
	return &WienerDenoiser{
		bins:        bins,
		noiseFloor:  make([]float64, bins),
		smoothedPsd: make([]float64, bins),
		gainBuf:     make([]float64, bins),
		alphaNoise:  0.92,
		alphaPsd:    0.70,
		overSubFact: 1.25,
		minGain:     0.15, // Preserve natural acoustic breath floor (-16dB)
		hasNoiseEst: false,
	}
}

// Process adapts the noise floor estimate and filters input magnitudes in-place.
// Returns spectral attenuation ratio (0.0 to 1.0).
func (wd *WienerDenoiser) Process(mags []float64, isVoiced bool) float64 {
	n := len(mags)
	if n > wd.bins {
		n = wd.bins
	}

	// 1. Initial noise floor calibration on startup
	if !wd.hasNoiseEst {
		for k := 0; k < n; k++ {
			wd.noiseFloor[k] = mags[k] * 0.8
			wd.smoothedPsd[k] = mags[k]
		}
		wd.hasNoiseEst = true
		return 1.0
	}

	var sumEnergyBefore, sumEnergyAfter float64

	// 2. Track background stationary noise floor during unvoiced speech segments
	if !isVoiced {
		for k := 0; k < n; k++ {
			wd.noiseFloor[k] = wd.alphaNoise*wd.noiseFloor[k] + (1.0-wd.alphaNoise)*mags[k]
		}
	}

	// 3. Compute smoothed PSD and Wiener attenuation gain
	for k := 0; k < n; k++ {
		curMag := mags[k]
		sumEnergyBefore += curMag * curMag

		wd.smoothedPsd[k] = wd.alphaPsd*wd.smoothedPsd[k] + (1.0-wd.alphaPsd)*curMag
		noiseVal := wd.noiseFloor[k] * wd.overSubFact

		// Wiener gain: G = max((P - N) / P, minGain)
		gain := 1.0
		if wd.smoothedPsd[k] > 1e-9 {
			sub := wd.smoothedPsd[k] - noiseVal
			if sub < 0 {
				sub = 0
			}
			gain = sub / wd.smoothedPsd[k]
		}

		if gain < wd.minGain {
			gain = wd.minGain
		}
		if gain > 1.0 {
			gain = 1.0
		}

		wd.gainBuf[k] = gain
		mags[k] = curMag * gain
		sumEnergyAfter += mags[k] * mags[k]
	}

	if sumEnergyBefore > 1e-9 {
		return math.Sqrt(sumEnergyAfter / sumEnergyBefore)
	}
	return 1.0
}
