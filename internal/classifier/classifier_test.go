package classifier

import (
	"testing"

	"github.com/alexandrmotologa/voiceprobe/internal/dsp"
)

func TestClassifierOrganicVsSynthetic(t *testing.T) {
	clf := NewDefaultClassifier()

	// Vector 1: Organic Human Voice
	// Low phase discontinuity (0.015), natural jitter (0.020), natural shimmer (0.045)
	organicVec := make([]float32, 48)
	organicVec[40] = 0.015 // Phase disc
	organicVec[41] = 0.010 // Flux
	organicVec[42] = 0.250 // Centroid
	organicVec[43] = 0.400 // Rolloff
	organicVec[44] = 0.350 // F0
	organicVec[45] = 0.800 // Jitter scaled (high/natural)
	organicVec[46] = 0.750 // Shimmer scaled (high/natural)
	organicVec[47] = 0.500 // VAD Energy

	organicScore := clf.Predict(organicVec)

	// Vector 2: Synthetic Vocoder Clone
	// High phase discontinuity (0.085), suppressed jitter (0.001), suppressed shimmer (0.008)
	syntheticVec := make([]float32, 48)
	syntheticVec[40] = 0.850 // Phase disc (high)
	syntheticVec[41] = 0.450 // Flux (high)
	syntheticVec[42] = 0.300
	syntheticVec[43] = 0.450
	syntheticVec[44] = 0.350
	syntheticVec[45] = 0.050 // Jitter scaled (low/suppressed)
	syntheticVec[46] = 0.050 // Shimmer scaled (low/suppressed)
	syntheticVec[47] = 0.500 // VAD Energy

	syntheticScore := clf.Predict(syntheticVec)

	if organicScore >= 0.40 {
		t.Fatalf("expected organic score < 0.40, got %f", organicScore)
	}
	if syntheticScore <= 0.60 {
		t.Fatalf("expected synthetic score > 0.60, got %f", syntheticScore)
	}
	if syntheticScore <= organicScore {
		t.Fatalf("synthetic score (%f) must be significantly higher than organic score (%f)", syntheticScore, organicScore)
	}
}

func TestScorerSession(t *testing.T) {
	scorer := NewScorer(0.40, 0.75, 0.35, 3)
	session := scorer.NewSession("test-call-101")

	// Frame 1: Silence
	silenceFeat := &dsp.FrameFeatures{
		VAD: dsp.VADResult{IsVoiced: 0.0, Energy: 0.0001},
	}
	diag1 := scorer.EvaluateFrame(session, silenceFeat)
	if diag1.RiskLevel != RiskClean {
		t.Fatalf("expected silence to evaluate as CLEAN, got %s", diag1.RiskLevel)
	}

	// Stream 5 synthetic frames
	synthFeat := &dsp.FrameFeatures{
		VAD:                dsp.VADResult{IsVoiced: 1.0, Energy: 0.02},
		PhaseDiscontinuity: 0.082,
		SpectralFlux:       0.065,
		Jitter:             0.0012,
		Shimmer:            0.009,
		FeatureVector:      make([]float32, 48),
	}
	synthFeat.FeatureVector[40] = 0.850
	synthFeat.FeatureVector[41] = 0.450
	synthFeat.FeatureVector[45] = 0.050
	synthFeat.FeatureVector[46] = 0.050
	synthFeat.FeatureVector[47] = 0.500

	var lastDiag FrameDiagnosis
	for i := 0; i < 6; i++ {
		lastDiag = scorer.EvaluateFrame(session, synthFeat)
	}

	if lastDiag.RiskLevel != RiskCritical {
		t.Fatalf("expected series of synthetic frames to reach CRITICAL risk, got %s (score=%f)", lastDiag.RiskLevel, lastDiag.SmoothedScore)
	}
	if lastDiag.ConsecutiveHits == 0 {
		t.Fatalf("expected consecutive hits > 0")
	}
}
