package dsp

import (
	"math"
)

// ReplayMetrics tracks physical loudspeaker non-linearities and acoustic cavity coloration.
type ReplayMetrics struct {
	THDPercent          float64 `json:"thd_percent"`
	SubBassCutoffEnergy float64 `json:"sub_bass_cutoff_energy"`
	LoudspeakerAnomaly  bool    `json:"loudspeaker_anomaly"`
}

// ComputeReplayFeatures evaluates if incoming audio exhibits physical loudspeaker non-linearities
// characteristic of a fraudster holding a mobile phone speaker near the telephone receiver.
// Small speakers introduce elevated THD (> 18%) and an abrupt cutoff below 140 Hz due to physical driver size.
func ComputeReplayFeatures(mags []float64, f0Hz float64, sampleRate, fftSize int) ReplayMetrics {
	bins := len(mags)
	if f0Hz < 80.0 || f0Hz > 350.0 || bins < 10 {
		return ReplayMetrics{}
	}

	hzPerBin := float64(sampleRate) / float64(fftSize)
	f0Bin := int(math.Round(f0Hz / hzPerBin))
	if f0Bin < 1 || f0Bin >= bins-1 {
		return ReplayMetrics{}
	}

	// Fundamental energy (sum around f0)
	var fundamentalEnergy float64
	for offset := -1; offset <= 1; offset++ {
		b := f0Bin + offset
		if b >= 0 && b < bins {
			fundamentalEnergy += mags[b] * mags[b]
		}
	}

	if fundamentalEnergy <= 1e-9 {
		return ReplayMetrics{}
	}

	// Harmonics 2 through 6
	var harmonicEnergy float64
	for h := 2; h <= 6; h++ {
		hFreq := float64(h) * f0Hz
		if hFreq > 3600.0 {
			break
		}
		hBin := int(math.Round(hFreq / hzPerBin))
		for offset := -1; offset <= 1; offset++ {
			b := hBin + offset
			if b >= 0 && b < bins {
				harmonicEnergy += mags[b] * mags[b]
			}
		}
	}

	// Total Harmonic Distortion: sqrt(sum(H2..H6^2)) / sqrt(H1^2)
	thd := math.Sqrt(harmonicEnergy / fundamentalEnergy)
	thdPercent := thd * 100.0

	// Sub-bass energy (< 140 Hz)
	cutoffBin := int(140.0 / hzPerBin)
	if cutoffBin >= bins {
		cutoffBin = bins - 1
	}
	var subBassEnergy float64
	for k := 1; k <= cutoffBin; k++ {
		subBassEnergy += mags[k] * mags[k]
	}

	// Loudspeaker anomaly flag: elevated THD combined with physical transducer coloration
	isAnomaly := (thdPercent > 22.0)

	return ReplayMetrics{
		THDPercent:          thdPercent,
		SubBassCutoffEnergy: subBassEnergy,
		LoudspeakerAnomaly:  isAnomaly,
	}
}
