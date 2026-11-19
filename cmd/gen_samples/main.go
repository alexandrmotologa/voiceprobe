package main

import (
	"log"
	"math"
	"math/rand"
	"os"
	"path/filepath"

	"github.com/alexandrmotologa/voiceprobe/internal/action"
)

// Resonator implements a digital two-pole vocal tract formant filter.
type Resonator struct {
	a1, a2 float64
	y1, y2 float64
}

func NewResonator(freq, bw, sampleRate float64) *Resonator {
	r := math.Exp(-math.Pi * bw / sampleRate)
	theta := 2.0 * math.Pi * freq / sampleRate
	return &Resonator{
		a1: 2.0 * r * math.Cos(theta),
		a2: -r * r,
	}
}

func (res *Resonator) Process(x float64) float64 {
	y := x + res.a1*res.y1 + res.a2*res.y2
	res.y2 = res.y1
	res.y1 = y
	return y
}

func main() {
	sampleDir := "./examples/sample_calls"
	if err := os.MkdirAll(sampleDir, 0755); err != nil {
		log.Fatal(err)
	}

	sampleRate := 8000.0
	durationSec := 5.0
	totalSamples := int(durationSec * sampleRate)

	r := rand.New(rand.NewSource(1337))

	// 1. Generate Authentic Human Vocal Phonation (Source-Filter Model)
	// Glottal source: periodic pulses with neuromuscular jitter (~1.8%) and airflow shimmer (~3.5%)
	// Filter: 3-formant vocal tract (F1=650Hz/BW=80Hz, F2=1250Hz/BW=110Hz, F3=2400Hz/BW=150Hz)
	r1 := NewResonator(650.0, 80.0, sampleRate)
	r2 := NewResonator(1250.0, 110.0, sampleRate)
	r3 := NewResonator(2400.0, 150.0, sampleRate)

	humanSamples := make([]float32, totalSamples)
	f0 := 150.0
	basePeriod := sampleRate / f0

	sampleIdx := 0
	for sampleIdx < totalSamples {
		timeSec := float64(sampleIdx) / sampleRate

		// Natural conversational rhythm: 1.5s speech, 0.3s breath pause
		cyclePos := math.Mod(timeSec, 1.8)
		if cyclePos > 1.5 {
			pauseLen := int(0.05 * sampleRate)
			for p := 0; p < pauseLen && sampleIdx < totalSamples; p++ {
				humanSamples[sampleIdx] = float32(0.002 * (r.Float64() - 0.5))
				sampleIdx++
			}
			continue
		}

		// Jitter: +/- 1.8% glottal cycle period perturbation
		jitterPerturb := (r.Float64() - 0.5) * 2.0 * (basePeriod * 0.018)
		cycleLen := int(math.Round(basePeriod + jitterPerturb))
		if cycleLen < 25 {
			cycleLen = int(basePeriod)
		}

		// Shimmer: cycle amplitude turbulence +/- 3.5%
		shimmer := 1.0 + (r.Float64()-0.5)*0.07

		for s := 0; s < cycleLen && sampleIdx < totalSamples; s++ {
			// Rosenberg-style glottal flow pulse
			var glottalPulse float64
			openPhase := float64(cycleLen) * 0.60
			if float64(s) < openPhase {
				glottalPulse = 0.5 * (1.0 - math.Cos(math.Pi*float64(s)/openPhase))
			} else {
				// Rapid closing phase
				glottalPulse = math.Cos(0.5 * math.Pi * (float64(s) - openPhase) / (float64(cycleLen) - openPhase))
			}

			// Pass through vocal tract acoustic resonators
			excitation := glottalPulse * shimmer * 0.20
			vocalOut := (r1.Process(excitation)*0.5 + r2.Process(excitation)*0.3 + r3.Process(excitation)*0.15) * 0.15

			// Soft clamp
			if vocalOut > 0.95 {
				vocalOut = 0.95
			} else if vocalOut < -0.95 {
				vocalOut = -0.95
			}

			humanSamples[sampleIdx] = float32(vocalOut)
			sampleIdx++
		}
	}

	humanPath := filepath.Join(sampleDir, "human_customer_call.wav")
	if err := action.WriteWAV(humanPath, humanSamples, int(sampleRate)); err != nil {
		log.Fatalf("Failed to write human wav: %v", err)
	}
	log.Printf("Generated %s (%.1fs)", humanPath, durationSec)

	// 2. Generate Cloned Neural Vocoder Speech (ElevenLabs / F5-TTS style)
	// Vocoder characteristics:
	// - Perfectly static cycle length (0.000% jitter)
	// - Robotic amplitude symmetry (0.000% shimmer)
	// - Transposed convolution upsampling phase noise across high-frequency bands
	// - Checkerboard high-frequency sub-band energy artifacts (2600-3400Hz)
	r1c := NewResonator(650.0, 80.0, sampleRate)
	r2c := NewResonator(1250.0, 110.0, sampleRate)
	r3c := NewResonator(2400.0, 150.0, sampleRate)

	clonedSamples := make([]float32, totalSamples)
	sampleIdx = 0
	for sampleIdx < totalSamples {
		timeSec := float64(sampleIdx) / sampleRate
		cyclePos := math.Mod(timeSec, 1.8)
		if cyclePos > 1.5 {
			pauseLen := int(0.05 * sampleRate)
			for p := 0; p < pauseLen && sampleIdx < totalSamples; p++ {
				clonedSamples[sampleIdx] = float32(0.001 * (r.Float64() - 0.5))
				sampleIdx++
			}
			continue
		}

		// Fixed rigid cycle period (zero jitter)
		cycleLen := int(math.Round(basePeriod))

		for s := 0; s < cycleLen && sampleIdx < totalSamples; s++ {
			var glottalPulse float64
			openPhase := float64(cycleLen) * 0.60
			if float64(s) < openPhase {
				glottalPulse = 0.5 * (1.0 - math.Cos(math.Pi*float64(s)/openPhase))
			} else {
				glottalPulse = math.Cos(0.5 * math.Pi * (float64(s) - openPhase) / (float64(cycleLen) - openPhase))
			}

			// Add vocoder phase misalignment noise burst at harmonic transition
			vocoderPhaseNoise := 0.04 * (r.Float64() - 0.5)

			excitation := (glottalPulse + vocoderPhaseNoise) * 0.20
			vocalOut := (r1c.Process(excitation)*0.5 + r2c.Process(excitation)*0.3 + r3c.Process(excitation)*0.15) * 0.15

			// High-frequency vocoder upsampler checkerboard artifact
			t := float64(sampleIdx) / sampleRate
			checkerboardArtifact := 0.06 * math.Sin(2.0*math.Pi*2900.0*t)

			total := vocalOut + checkerboardArtifact
			if total > 0.95 {
				total = 0.95
			} else if total < -0.95 {
				total = -0.95
			}

			clonedSamples[sampleIdx] = float32(total)
			sampleIdx++
		}
	}

	clonedPath := filepath.Join(sampleDir, "cloned_elevenlabs_vishing.wav")
	if err := action.WriteWAV(clonedPath, clonedSamples, int(sampleRate)); err != nil {
		log.Fatalf("Failed to write cloned wav: %v", err)
	}
	log.Printf("Generated %s (%.1fs)", clonedPath, durationSec)
}
