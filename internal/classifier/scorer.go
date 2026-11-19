package classifier

import (
	"fmt"
	"math"

	"github.com/alexandrmotologa/voiceprobe/internal/dsp"
)

// RiskLevel represents the categorical threat evaluation of a call.
type RiskLevel string

const (
	RiskClean      RiskLevel = "CLEAN"
	RiskSuspicious RiskLevel = "SUSPICIOUS"
	RiskCritical   RiskLevel = "CRITICAL"
)

// FrameDiagnosis provides the per-window security evaluation.
type FrameDiagnosis struct {
	InstantScore   float64     `json:"instant_score"`
	SmoothedScore  float64     `json:"smoothed_score"`
	RiskLevel      RiskLevel   `json:"risk_level"`
	Confidence     float64     `json:"confidence"`
	IsVoiced       bool        `json:"is_voiced"`
	ConsecutiveHits int        `json:"consecutive_hits"`
	KeyIndicators  []string    `json:"key_indicators"`
	Features       *dsp.FrameFeatures `json:"features,omitempty"`
}

// SessionState tracks the rolling classification telemetry for an ongoing phone call.
type SessionState struct {
	CallID            string
	TotalFrames       int
	VoicedFrames      int
	SmoothedScore     float64
	MaxScore          float64
	ConsecutiveAlerts int
	LastRiskLevel     RiskLevel
}

// Scorer orchestrates feature classification and temporal smoothing.
type Scorer struct {
	classifier          *NeuralClassifier
	suspiciousThreshold float64
	criticalThreshold   float64
	smoothingAlpha      float64
	minVoicedFrames     int
}

// NewScorer creates a temporal scoring engine.
func NewScorer(suspiciousThreshold, criticalThreshold, smoothingAlpha float64, minVoicedFrames int) *Scorer {
	return &Scorer{
		classifier:          NewDefaultClassifier(),
		suspiciousThreshold: suspiciousThreshold,
		criticalThreshold:   criticalThreshold,
		smoothingAlpha:      smoothingAlpha,
		minVoicedFrames:     minVoicedFrames,
	}
}

// NewSession initializes tracking state for a new inbound call.
func (s *Scorer) NewSession(callID string) *SessionState {
	return &SessionState{
		CallID:        callID,
		SmoothedScore: 0.05,
		LastRiskLevel: RiskClean,
	}
}

// EvaluateFrame assesses one 300ms window within an active session.
func (s *Scorer) EvaluateFrame(session *SessionState, feat *dsp.FrameFeatures) FrameDiagnosis {
	session.TotalFrames++

	// If unvoiced (silence / pause), decay score slightly and skip neural classification
	if feat.VAD.IsVoiced == 0.0 {
		session.SmoothedScore = math.Max(0.02, session.SmoothedScore*0.96)
		return FrameDiagnosis{
			InstantScore:    0.0,
			SmoothedScore:   session.SmoothedScore,
			RiskLevel:       RiskClean,
			Confidence:      0.95,
			IsVoiced:        false,
			ConsecutiveHits: 0,
			KeyIndicators:   []string{"Unvoiced frame (comfort noise or silence)"},
			Features:        feat,
		}
	}

	session.VoicedFrames++

	// 1. Raw neural network prediction
	rawProb := s.classifier.Predict(feat.FeatureVector)

	// 2. Exponential Moving Average (EMA) smoothing
	if session.VoicedFrames == 1 {
		session.SmoothedScore = rawProb
	} else {
		session.SmoothedScore = s.smoothingAlpha*rawProb + (1.0-s.smoothingAlpha)*session.SmoothedScore
	}

	if session.SmoothedScore > session.MaxScore {
		session.MaxScore = session.SmoothedScore
	}

	// 3. Determine risk category
	var level RiskLevel
	if session.SmoothedScore >= s.criticalThreshold {
		level = RiskCritical
		session.ConsecutiveAlerts++
	} else if session.SmoothedScore >= s.suspiciousThreshold {
		level = RiskSuspicious
		session.ConsecutiveAlerts = 0
	} else {
		level = RiskClean
		session.ConsecutiveAlerts = 0
	}
	session.LastRiskLevel = level

	// 4. Formulate diagnostic indicators
	var indicators []string
	if feat.PhaseDiscontinuity >= 0.050 {
		indicators = append(indicators, fmt.Sprintf("High phase discontinuity (%.3f): vocoder upsampling artifact", feat.PhaseDiscontinuity))
	}
	if feat.F0Hz > 0 && feat.Jitter < 0.005 && feat.VAD.IsVoiced > 0 {
		indicators = append(indicators, fmt.Sprintf("Suppressed pitch jitter (%.4f): absent laryngeal micro-tremor", feat.Jitter))
	}
	if feat.F0Hz > 0 && feat.Shimmer < 0.015 && feat.VAD.IsVoiced > 0 {
		indicators = append(indicators, fmt.Sprintf("Suppressed shimmer (%.4f): synthetic amplitude symmetry", feat.Shimmer))
	}
	if feat.SpectralTilt > -3.5 && feat.SpectralTilt < 0.0 {
		indicators = append(indicators, fmt.Sprintf("Shallow spectral tilt (%.1f dB/oct): uniform vocoder sub-band loss", feat.SpectralTilt))
	}
	if feat.LoudspeakerAnomaly || feat.THDPercent > 20.0 {
		indicators = append(indicators, fmt.Sprintf("Acoustic loudspeaker replay anomaly (%.1f%% THD)", feat.THDPercent))
	}
	if feat.SpectralFlux > 0.150 {
		indicators = append(indicators, fmt.Sprintf("Abrupt spectral flux transition (%.3f)", feat.SpectralFlux))
	}
	if len(indicators) == 0 {
		indicators = append(indicators, "Organic acoustic harmonics verified")
	}

	// Confidence estimator based on cumulative voiced frames
	confidence := math.Min(0.99, 0.60+float64(session.VoicedFrames)*0.08)

	return FrameDiagnosis{
		InstantScore:    rawProb,
		SmoothedScore:   session.SmoothedScore,
		RiskLevel:       level,
		Confidence:      confidence,
		IsVoiced:        true,
		ConsecutiveHits: session.ConsecutiveAlerts,
		KeyIndicators:   indicators,
		Features:        feat,
	}
}
