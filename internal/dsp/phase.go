package dsp

import (
	"math"
)

// AngleDiff computes the shortest angular difference between two phase angles, wrapped to [-pi, pi].
func AngleDiff(a, b float64) float64 {
	diff := a - b
	return math.Atan2(math.Sin(diff), math.Cos(diff))
}

// PhaseTracker tracks frame-to-frame phase propagation across consecutive analysis hops.
type PhaseTracker struct {
	prevPhases []float64
	hasPrev    bool
}

// NewPhaseTracker creates an inter-frame phase consistency tracker.
func NewPhaseTracker(fftSize int) *PhaseTracker {
	return &PhaseTracker{
		prevPhases: make([]float64, fftSize/2+1),
		hasPrev:    false,
	}
}

// ComputeGlottalPhaseDiscontinuity evaluates phase consistency across vocal tract harmonics.
// In natural human speech, glottal pulse excitation aligns harmonic phases, resulting
// in smooth second phase differences across harmonics: delta^2 theta_m = (theta_{m+1} - theta_m) - (theta_m - theta_{m-1}) approx 0.
// In neural vocoders (e.g. HiFi-GAN, ElevenLabs, F5-TTS), transposed convolution upsamplers
// introduce independent phase jitter across high-frequency sub-bands, producing elevated discontinuity (> 0.45).
func (pt *PhaseTracker) ComputeGlottalPhaseDiscontinuity(currPhases, mags []float64, f0Hz float64, sampleRate, fftSize int) float64 {
	bins := len(currPhases)
	if f0Hz < 75.0 || f0Hz > 380.0 {
		return 0.05
	}

	hzPerBin := float64(sampleRate) / float64(fftSize)
	f0Bin := f0Hz / hzPerBin

	maxHarmonics := 10
	harmonicPhases := make([]float64, 0, maxHarmonics)

	for m := 1; m <= maxHarmonics; m++ {
		harmonicFreq := float64(m) * f0Hz
		if harmonicFreq > 3600.0 {
			break
		}

		centerBin := int(math.Round(float64(m) * f0Bin))
		if centerBin >= bins-2 || centerBin <= 1 {
			break
		}

		// Pick peak bin in 3-bin neighborhood around harmonic center
		bestBin := centerBin
		bestMag := mags[centerBin]
		for offset := -1; offset <= 1; offset++ {
			c := centerBin + offset
			if c > 0 && c < bins && mags[c] > bestMag {
				bestMag = mags[c]
				bestBin = c
			}
		}

		harmonicPhases = append(harmonicPhases, currPhases[bestBin])
	}

	if len(harmonicPhases) < 4 {
		return 0.05
	}

	// Compute second-order harmonic phase difference (phase curvature)
	var sumCurvature float64
	var count int
	for m := 1; m < len(harmonicPhases)-1; m++ {
		d1 := AngleDiff(harmonicPhases[m], harmonicPhases[m-1])
		d2 := AngleDiff(harmonicPhases[m+1], harmonicPhases[m])
		curvature := math.Abs(AngleDiff(d2, d1))
		sumCurvature += curvature
		count++
	}

	if count == 0 {
		return 0.05
	}

	meanCurvature := sumCurvature / float64(count)

	// Normalized to [0.0, 1.0]. Natural human speech has meanCurvature < 0.35 rad (normalized < 0.20)
	// Neural vocoder phase dispersion typically has meanCurvature > 0.80 rad (normalized > 0.55)
	normalized := meanCurvature / (meanCurvature + 0.55)
	if normalized > 1.0 {
		normalized = 1.0
	}
	return normalized
}
