package dsp

import (
	"math"
)

// FrameFeatures holds all extracted acoustic diagnostics for one 300ms window.
type FrameFeatures struct {
	VAD                VADResult `json:"vad"`
	SpectralFlux       float64   `json:"spectral_flux"`
	SpectralCentroid   float64   `json:"spectral_centroid"`
	SpectralRolloff    float64   `json:"spectral_rolloff"`
	PhaseDiscontinuity float64   `json:"phase_discontinuity"`
	F0Hz               float64   `json:"f0_hz"`
	Jitter             float64   `json:"jitter"`
	Shimmer            float64   `json:"shimmer"`
	MelEnergies        []float64 `json:"mel_energies"`
	RawMagnitudes      []float64 `json:"raw_magnitudes,omitempty"`
	FeatureVector      []float32 `json:"-"`
}

// Pipeline orchestrates acoustic feature extraction for real-time telephony frames.
type Pipeline struct {
	sampleRate         int
	windowSize         int
	hopSize            int
	vadEnergyThreshold float64
	fftEngine          *FFTEngine
	melFilterbank      *MelFilterbank
	phaseTracker       *PhaseTracker
	prevMagnitudes     []float64
	hasPrevFrame       bool
	melEnergiesBuf     []float64
	featVecBuf         []float32
}

// NewPipeline initializes all DSP processing blocks.
func NewPipeline(sampleRate, windowMs, hopMs, melBins int, vadEnergyThreshold float64) *Pipeline {
	windowSamples := sampleRate * windowMs / 1000
	hopSamples := sampleRate * hopMs / 1000

	fftEngine := NewFFTEngine(windowSamples)
	fftSize := fftEngine.Size()
	bins := fftSize/2 + 1

	return &Pipeline{
		sampleRate:         sampleRate,
		windowSize:         windowSamples,
		hopSize:            hopSamples,
		vadEnergyThreshold: vadEnergyThreshold,
		fftEngine:          fftEngine,
		melFilterbank:      NewMelFilterbank(melBins, fftSize, sampleRate),
		phaseTracker:       NewPhaseTracker(fftSize),
		prevMagnitudes:     make([]float64, bins),
		hasPrevFrame:       false,
		melEnergiesBuf:     make([]float64, melBins),
		featVecBuf:         make([]float32, 48),
	}
}

// WindowSize returns the expected sample count per analysis frame.
func (p *Pipeline) WindowSize() int {
	return p.windowSize
}

// HopSize returns the step sample count between successive frames.
func (p *Pipeline) HopSize() int {
	return p.hopSize
}

// ExtractFeatures computes acoustic and spectral features on a time-domain frame without allocations.
func (p *Pipeline) ExtractFeatures(samples []float32) *FrameFeatures {
	// 1. Voice Activity Detection
	vad := ComputeVAD(samples, p.vadEnergyThreshold)

	// 2. STFT: Magnitudes and Phase Spectrum
	mags, phases := p.fftEngine.Process(samples)

	// 3. Spectral Metrics
	var flux float64
	if p.hasPrevFrame {
		flux = ComputeSpectralFlux(mags, p.prevMagnitudes)
	} else {
		flux = 0.0
		p.hasPrevFrame = true
	}
	centroid := ComputeSpectralCentroid(mags, p.sampleRate, p.fftEngine.Size())
	rolloff := ComputeSpectralRolloff(mags, p.sampleRate, p.fftEngine.Size(), 0.85)

	copy(p.prevMagnitudes, mags)

	// 4. Neuromuscular Micro-Tremor (Jitter, Shimmer, F0)
	microTremor := ComputeMicroTremor(samples, p.sampleRate)

	// 5. Glottal Harmonic Phase Discontinuity (Physical excitation coherence vs vocoder dispersion)
	phaseDisc := p.phaseTracker.ComputeGlottalPhaseDiscontinuity(phases, mags, microTremor.F0Hz, p.sampleRate, p.fftEngine.Size())

	// 6. Mel Filterbank Log-Energies
	p.melFilterbank.ComputeInto(mags, p.melEnergiesBuf)

	// 7. Compute Cepstral Mean Subtraction (CMN) on Mel bands
	var melMean float64
	melCount := len(p.melEnergiesBuf)
	for i := 0; i < melCount && i < 40; i++ {
		melMean += p.melEnergiesBuf[i]
	}
	if melCount > 0 {
		melMean /= float64(melCount)
	}

	// 8. Assemble 48-dimensional normalized feature vector
	for i := 0; i < melCount && i < 40; i++ {
		p.featVecBuf[i] = float32((p.melEnergiesBuf[i] - melMean) / 4.0)
	}
	p.featVecBuf[40] = float32(phaseDisc)
	p.featVecBuf[41] = float32(flux)
	p.featVecBuf[42] = float32(centroid / float64(p.sampleRate/2))
	p.featVecBuf[43] = float32(rolloff / float64(p.sampleRate/2))
	p.featVecBuf[44] = float32(microTremor.F0Hz / 400.0)
	p.featVecBuf[45] = float32(math.Min(microTremor.Jitter*50.0, 1.0))
	p.featVecBuf[46] = float32(math.Min(microTremor.Shimmer*20.0, 1.0))
	p.featVecBuf[47] = float32(math.Min(vad.Energy*100.0, 1.0))

	return &FrameFeatures{
		VAD:                vad,
		SpectralFlux:       flux,
		SpectralCentroid:   centroid,
		SpectralRolloff:    rolloff,
		PhaseDiscontinuity: phaseDisc,
		F0Hz:               microTremor.F0Hz,
		Jitter:             microTremor.Jitter,
		Shimmer:            microTremor.Shimmer,
		MelEnergies:        p.melEnergiesBuf,
		FeatureVector:      p.featVecBuf,
	}
}
