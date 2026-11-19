package dsp

import (
	"math"
)

// VADResult holds voice activity metrics for an audio frame.
type VADResult struct {
	IsVoiced float64 // 1.0 if voiced speech, 0.0 if silence or background noise
	Energy   float64 // Mean squared amplitude
	ZCR      float64 // Zero-crossing rate normalized to [0, 1]
}

// ComputeVAD assesses whether a PCM audio frame contains active human speech.
func ComputeVAD(samples []float32, energyThreshold float64) VADResult {
	n := len(samples)
	if n == 0 {
		return VADResult{}
	}

	var sumEnergy float64
	var zeroCrossings int

	for i := 0; i < n; i++ {
		s := float64(samples[i])
		sumEnergy += s * s

		if i > 0 {
			prev := float64(samples[i-1])
			if (s >= 0 && prev < 0) || (s < 0 && prev >= 0) {
				zeroCrossings++
			}
		}
	}

	meanEnergy := sumEnergy / float64(n)
	zcr := float64(zeroCrossings) / float64(n-1)

	// In telephony speech, active phonation requires minimum RMS energy and reasonable ZCR.
	// Pure silence or DC drift fails this gate.
	isVoiced := 0.0
	if meanEnergy >= energyThreshold && !math.IsNaN(meanEnergy) && !math.IsInf(meanEnergy, 0) {
		isVoiced = 1.0
	}

	return VADResult{
		IsVoiced: isVoiced,
		Energy:   meanEnergy,
		ZCR:      zcr,
	}
}
