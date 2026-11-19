package dsp

import (
	"math"
	"testing"
)

func TestRingBuffer(t *testing.T) {
	rb := NewRingBuffer(100)
	if rb.Available() != 0 {
		t.Fatalf("expected 0 available, got %d", rb.Available())
	}

	samples := make([]float32, 60)
	for i := range samples {
		samples[i] = float32(i)
	}
	rb.Write(samples)

	if rb.Available() != 60 {
		t.Fatalf("expected 60 available, got %d", rb.Available())
	}

	dst := make([]float32, 10)
	read := rb.ReadLatest(dst)
	if read != 10 {
		t.Fatalf("expected 10 read, got %d", read)
	}
	// Last sample was 59, so dst should be 50..59
	if dst[9] != 59.0 || dst[0] != 50.0 {
		t.Fatalf("unexpected read values: dst[0]=%f, dst[9]=%f", dst[0], dst[9])
	}

	// Write more to cause wrap around
	more := make([]float32, 80)
	for i := range more {
		more[i] = float32(100 + i)
	}
	rb.Write(more)

	if rb.Available() != 100 {
		t.Fatalf("expected capacity 100, got %d", rb.Available())
	}

	rb.Reset()
	if rb.Available() != 0 {
		t.Fatalf("expected 0 after reset, got %d", rb.Available())
	}
}

func TestVAD(t *testing.T) {
	silence := make([]float32, 2400)
	resSilence := ComputeVAD(silence, 0.001)
	if resSilence.IsVoiced != 0.0 {
		t.Errorf("expected silence to be unvoiced, got %f", resSilence.IsVoiced)
	}

	// Sine wave 200Hz at 8000Hz sample rate
	speech := make([]float32, 2400)
	for i := range speech {
		speech[i] = float32(0.5 * math.Sin(2.0*math.Pi*200.0*float64(i)/8000.0))
	}
	resSpeech := ComputeVAD(speech, 0.001)
	if resSpeech.IsVoiced != 1.0 {
		t.Errorf("expected sine wave to be voiced, got %f", resSpeech.IsVoiced)
	}
	if resSpeech.Energy <= 0.001 {
		t.Errorf("expected energy > 0.001, got %f", resSpeech.Energy)
	}
}

func TestSTFT(t *testing.T) {
	windowLen := 1024
	engine := NewFFTEngine(windowLen)
	fftSize := engine.Size()
	sampleRate := 8000

	// 500 Hz tone
	targetFreq := 500.0
	samples := make([]float32, fftSize)
	for i := range samples {
		samples[i] = float32(math.Sin(2.0 * math.Pi * targetFreq * float64(i) / float64(sampleRate)))
	}

	mags, phases := engine.Process(samples)
	if len(mags) != fftSize/2+1 || len(phases) != fftSize/2+1 {
		t.Fatalf("unexpected spectrum length: mags=%d, phases=%d", len(mags), len(phases))
	}

	// Find peak frequency bin
	maxMag := 0.0
	peakBin := 0
	for k, m := range mags {
		if m > maxMag {
			maxMag = m
			peakBin = k
		}
	}

	hzPerBin := float64(sampleRate) / float64(fftSize)
	detectedHz := float64(peakBin) * hzPerBin

	// Peak should be within one bin of 500 Hz (bin resolution is ~7.81 Hz)
	if math.Abs(detectedHz-targetFreq) > hzPerBin*1.5 {
		t.Fatalf("expected peak near 500Hz, detected %fHz (bin %d)", detectedHz, peakBin)
	}
}

func TestMelFilterbank(t *testing.T) {
	fb := NewMelFilterbank(40, 1024, 8000)
	mags := make([]float64, 513)
	for i := range mags {
		mags[i] = 1.0
	}

	energies := fb.Compute(mags)
	if len(energies) != 40 {
		t.Fatalf("expected 40 mel energies, got %d", len(energies))
	}
	for i, e := range energies {
		if math.IsNaN(e) || math.IsInf(e, 0) || e <= 0.0 {
			t.Errorf("invalid mel energy at index %d: %f", i, e)
		}
	}
}

func TestPhaseDiscontinuity(t *testing.T) {
	fftSize := 1024
	sampleRate := 8000
	f0 := 160.0
	hzPerBin := float64(sampleRate) / float64(fftSize)
	f0Bin := f0 / hzPerBin

	mags := make([]float64, 513)
	for m := 1; m <= 6; m++ {
		b := int(math.Round(float64(m) * f0Bin))
		if b < 513 {
			mags[b] = 10.0
		}
	}

	tracker := NewPhaseTracker(fftSize)

	// Clean linear harmonic phase progression (glottal pulse alignment)
	phasesClean := make([]float64, 513)
	for m := 1; m <= 6; m++ {
		b := int(math.Round(float64(m) * f0Bin))
		if b < 513 {
			phasesClean[b] = float64(m) * 0.42 // Constant first-derivative phase
		}
	}
	cleanDisc := tracker.ComputeGlottalPhaseDiscontinuity(phasesClean, mags, f0, sampleRate, fftSize)

	// Chaotic vocoder phase jump across harmonics
	phasesVocoder := make([]float64, 513)
	for m := 1; m <= 6; m++ {
		b := int(math.Round(float64(m) * f0Bin))
		if b < 513 {
			phasesVocoder[b] = math.Sin(float64(m)*2.7) * math.Pi
		}
	}
	noisyDisc := tracker.ComputeGlottalPhaseDiscontinuity(phasesVocoder, mags, f0, sampleRate, fftSize)

	if noisyDisc <= cleanDisc {
		t.Fatalf("expected vocoder phase disc (%f) > clean phase disc (%f)", noisyDisc, cleanDisc)
	}
}

func TestMicroTremor(t *testing.T) {
	sampleRate := 8000
	samples := make([]float32, 2400)
	targetF0 := 160.0 // Male/female voice fundamental

	for i := range samples {
		samples[i] = float32(0.6 * math.Sin(2.0*math.Pi*targetF0*float64(i)/float64(sampleRate)))
	}

	metrics := ComputeMicroTremor(samples, sampleRate)
	if math.Abs(metrics.F0Hz-targetF0) > 15.0 {
		t.Fatalf("expected F0 near 160Hz, detected %fHz", metrics.F0Hz)
	}
}

func TestPipeline(t *testing.T) {
	pipe := NewPipeline(8000, 300, 150, 40, 0.0008)
	samples := make([]float32, 2400)
	for i := range samples {
		samples[i] = float32(0.4 * math.Sin(2.0*math.Pi*220.0*float64(i)/8000.0))
	}

	features := pipe.ExtractFeatures(samples)
	if features == nil {
		t.Fatal("features should not be nil")
	}
	if features.VAD.IsVoiced != 1.0 {
		t.Errorf("expected voiced signal, got %f", features.VAD.IsVoiced)
	}
	if len(features.MelEnergies) != 40 {
		t.Errorf("expected 40 mel energies, got %d", len(features.MelEnergies))
	}
	if len(features.FeatureVector) != 48 {
		t.Errorf("expected 48 feature vector length, got %d", len(features.FeatureVector))
	}
}
