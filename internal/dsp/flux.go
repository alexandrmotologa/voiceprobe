package dsp

// SpectralMetrics aggregates frequency domain flux, centroid, and rolloff.
type SpectralMetrics struct {
	Flux     float64 `json:"spectral_flux"`
	Centroid float64 `json:"spectral_centroid"`
	Rolloff  float64 `json:"spectral_rolloff"`
}

// ComputeSpectralFlux measures the normalized rate of spectral variation between successive frames.
// Normalized by total current frame magnitude to produce a scale-invariant metric in [0.0, 1.0].
func ComputeSpectralFlux(currMags, prevMags []float64) float64 {
	if len(prevMags) == 0 {
		return 0.0
	}

	bins := len(currMags)
	if len(prevMags) < bins {
		bins = len(prevMags)
	}

	var flux, totalMag float64
	for k := 0; k < bins; k++ {
		totalMag += currMags[k]
		diff := currMags[k] - prevMags[k]
		if diff > 0 {
			flux += diff
		}
	}

	if totalMag <= 1e-6 {
		return 0.0
	}

	normFlux := flux / totalMag
	if normFlux > 1.0 {
		normFlux = 1.0
	}
	return normFlux
}

// ComputeSpectralCentroid calculates the center of mass of the frequency spectrum.
func ComputeSpectralCentroid(mags []float64, sampleRate int, fftSize int) float64 {
	bins := len(mags)
	if bins == 0 {
		return 0.0
	}

	var weightedSum, totalMag float64
	hzPerBin := float64(sampleRate) / float64(fftSize)

	for k := 0; k < bins; k++ {
		freq := float64(k) * hzPerBin
		weightedSum += freq * mags[k]
		totalMag += mags[k]
	}

	if totalMag <= 1e-9 {
		return 0.0
	}

	return weightedSum / totalMag
}

// ComputeSpectralRolloff finds the frequency below which 85% of spectral energy is concentrated.
func ComputeSpectralRolloff(mags []float64, sampleRate int, fftSize int, percentile float64) float64 {
	bins := len(mags)
	if bins == 0 {
		return 0.0
	}

	var totalEnergy float64
	energies := make([]float64, bins)
	for k := 0; k < bins; k++ {
		e := mags[k] * mags[k]
		energies[k] = e
		totalEnergy += e
	}

	if totalEnergy <= 1e-9 {
		return 0.0
	}

	target := totalEnergy * percentile
	var running float64
	hzPerBin := float64(sampleRate) / float64(fftSize)

	for k := 0; k < bins; k++ {
		running += energies[k]
		if running >= target {
			return float64(k) * hzPerBin
		}
	}

	return float64(bins-1) * hzPerBin
}
