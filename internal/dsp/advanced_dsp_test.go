package dsp

import (
	"math"
	"testing"
)

func TestWienerDenoiser(t *testing.T) {
	denoiser := NewWienerDenoiser(1024)
	bins := 513

	// Frame 1: Pure background noise (unvoiced)
	noise := make([]float64, bins)
	for i := range noise {
		noise[i] = 2.0
	}
	denoiser.Process(noise, false)

	// Frame 2: Tone signal + noise (voiced) - allow EMA filter convergence
	signalAndNoise := make([]float64, bins)
	for iter := 0; iter < 4; iter++ {
		for i := range signalAndNoise {
			signalAndNoise[i] = 2.0 // Noise component
		}
		signalAndNoise[100] = 20.0 // Strong voice tone
		denoiser.Process(signalAndNoise, true)
	}

	// Tone bin should preserve high energy while background is attenuated
	if signalAndNoise[100] < 16.0 {
		t.Fatalf("expected voice tone to be preserved, got %f", signalAndNoise[100])
	}
	if signalAndNoise[10] > 1.5 {
		t.Fatalf("expected noise floor attenuation at non-tone bins, got %f", signalAndNoise[10])
	}
}

func TestSpectralTilt(t *testing.T) {
	bins := 513
	sampleRate := 8000
	fftSize := 1024
	hzPerBin := float64(sampleRate) / float64(fftSize)

	// 1. Natural steep downward slope: mag(f) ~ 1/f^1.5 (-9 dB/octave)
	steepMags := make([]float64, bins)
	for k := 1; k < bins; k++ {
		freq := float64(k) * hzPerBin
		steepMags[k] = 1000.0 / math.Pow(freq/100.0, 1.5)
	}
	tiltSteep := ComputeSpectralTilt(steepMags, sampleRate, fftSize)

	// 2. Unnaturally flat slope (typical neural vocoder): mag(f) ~ constant
	flatMags := make([]float64, bins)
	for k := 1; k < bins; k++ {
		flatMags[k] = 50.0
	}
	tiltFlat := ComputeSpectralTilt(flatMags, sampleRate, fftSize)

	if tiltFlat.SlopeDbPerOctave <= tiltSteep.SlopeDbPerOctave {
		t.Fatalf("expected flat vocoder slope (%f) > steep natural slope (%f)",
			tiltFlat.SlopeDbPerOctave, tiltSteep.SlopeDbPerOctave)
	}
}

func TestReplayLoudspeakerDistortion(t *testing.T) {
	bins := 513
	sampleRate := 8000
	fftSize := 1024
	hzPerBin := float64(sampleRate) / float64(fftSize)
	f0 := 160.0
	f0Bin := int(math.Round(f0 / hzPerBin))

	// 1. Clean direct phonation: minimal harmonic distortion (THD < 6%)
	cleanMags := make([]float64, bins)
	cleanMags[f0Bin] = 100.0
	cleanMags[int(math.Round(2.0*f0/hzPerBin))] = 3.0
	cleanMags[int(math.Round(3.0*f0/hzPerBin))] = 2.0
	cleanMags[int(math.Round(4.0*f0/hzPerBin))] = 1.0

	cleanReplay := ComputeReplayFeatures(cleanMags, f0, sampleRate, fftSize)
	if cleanReplay.LoudspeakerAnomaly {
		t.Fatalf("clean voice should not trigger loudspeaker anomaly: THD=%f%%", cleanReplay.THDPercent)
	}

	// 2. Loudspeaker replay attack: strong non-linear distortion (THD > 30%)
	distortedMags := make([]float64, bins)
	distortedMags[f0Bin] = 100.0
	distortedMags[int(math.Round(2.0*f0/hzPerBin))] = 35.0
	distortedMags[int(math.Round(3.0*f0/hzPerBin))] = 25.0
	distortedMags[int(math.Round(4.0*f0/hzPerBin))] = 20.0

	distortedReplay := ComputeReplayFeatures(distortedMags, f0, sampleRate, fftSize)
	if !distortedReplay.LoudspeakerAnomaly {
		t.Fatalf("distorted replay should trigger loudspeaker anomaly: THD=%f%%", distortedReplay.THDPercent)
	}
}
