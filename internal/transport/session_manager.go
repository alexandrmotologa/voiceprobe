package transport

import (
	"sync"
	"time"

	"github.com/alexandrmotologa/voiceprobe/internal/action"
	"github.com/alexandrmotologa/voiceprobe/internal/classifier"
	"github.com/alexandrmotologa/voiceprobe/internal/config"
	"github.com/alexandrmotologa/voiceprobe/internal/dsp"
)

// CallSession tracks an active phone call stream with pre-allocated zero-alloc buffers.
type CallSession struct {
	mu                sync.RWMutex
	CallID            string                     `json:"call_id"`
	Transport         string                     `json:"transport"`
	Codec             string                     `json:"codec"`
	StartTime         time.Time                  `json:"start_time"`
	LastActive        time.Time                  `json:"last_active"`
	SampleRate        int                        `json:"sample_rate"`
	TotalSamples      int64                      `json:"total_samples"`
	ringBuffer        *dsp.RingBuffer
	dspPipeline       *dsp.Pipeline
	sessionState      *classifier.SessionState
	samplesSinceHop   int
	recentAudio       []float32                  // Fixed circular buffer for incident capture
	recentWriteIdx    int
	recentTotal       int64
	windowBuf         []float32                  // Pre-allocated analysis window scratch buffer
	LastDiagnosis      classifier.FrameDiagnosis `json:"last_diagnosis"`
	PeakScore          float64                   `json:"peak_score"`
	OverallRisk        classifier.RiskLevel      `json:"overall_risk"`
	AlertTriggered     bool                      `json:"alert_triggered"`
	TerminatedByPolicy bool                      `json:"terminated_by_policy"`
	ActivePolicy       string                    `json:"active_policy"`
}

// SessionManager tracks concurrent telephone calls across all transports.
type SessionManager struct {
	mu           sync.RWMutex
	cfg          *config.Config
	scorer       *classifier.Scorer
	recorder     *action.IncidentRecorder
	dispatcher   *action.WebhookDispatcher
	policyEngine *action.PolicyEngine
	sessions     map[string]*CallSession
	listeners    []func(session *CallSession, diag classifier.FrameDiagnosis)
}

// NewSessionManager initializes session storage and scoring bindings.
func NewSessionManager(cfg *config.Config, recorder *action.IncidentRecorder, dispatcher *action.WebhookDispatcher) *SessionManager {
	scorer := classifier.NewScorer(
		cfg.Detection.SuspiciousThreshold,
		cfg.Detection.CriticalThreshold,
		cfg.Detection.SmoothingAlpha,
		cfg.Detection.MinVoicedFrames,
	)

	return &SessionManager{
		cfg:          cfg,
		scorer:       scorer,
		recorder:     recorder,
		dispatcher:   dispatcher,
		policyEngine: action.NewPolicyEngine(),
		sessions:     make(map[string]*CallSession),
	}
}

// SetPolicyEngine binds a custom policy engine.
func (sm *SessionManager) SetPolicyEngine(pe *action.PolicyEngine) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.policyEngine = pe
}

// GetPolicyEngine returns the active policy engine.
func (sm *SessionManager) GetPolicyEngine() *action.PolicyEngine {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.policyEngine
}

// AddListener registers a subscriber callback for real-time frame telemetry.
func (sm *SessionManager) AddListener(listener func(session *CallSession, diag classifier.FrameDiagnosis)) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.listeners = append(sm.listeners, listener)
}

// GetOrCreateSession retrieves an active session or instantiates a new one.
func (sm *SessionManager) GetOrCreateSession(callID, transportName, codec string, sampleRate int) *CallSession {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sess, exists := sm.sessions[callID]; exists {
		return sess
	}

	if sampleRate <= 0 {
		sampleRate = sm.cfg.DSP.SampleRate
	}

	dspPipe := dsp.NewPipeline(
		sampleRate,
		sm.cfg.DSP.WindowDurationMs,
		sm.cfg.DSP.HopDurationMs,
		sm.cfg.DSP.MelBins,
		sm.cfg.DSP.VADEnergyThreshold,
	)

	// Fixed circular buffer for incident capture: 3 seconds
	historyCapacity := sampleRate * 3
	windowSize := dspPipe.WindowSize()

	sess := &CallSession{
		CallID:          callID,
		Transport:       transportName,
		Codec:           codec,
		StartTime:       time.Now().UTC(),
		LastActive:      time.Now().UTC(),
		SampleRate:      sampleRate,
		ringBuffer:      dsp.NewRingBuffer(sampleRate * 2),
		dspPipeline:     dspPipe,
		sessionState:    sm.scorer.NewSession(callID),
		recentAudio:     make([]float32, historyCapacity),
		windowBuf:       make([]float32, windowSize),
		OverallRisk:     classifier.RiskClean,
	}

	sm.sessions[callID] = sess
	return sess
}

// GetSession looks up an active session by call ID.
func (sm *SessionManager) GetSession(callID string) (*CallSession, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	sess, exists := sm.sessions[callID]
	return sess, exists
}

// ListSessions returns snapshots of all currently monitored calls.
func (sm *SessionManager) ListSessions() []*CallSession {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	list := make([]*CallSession, 0, len(sm.sessions))
	for _, s := range sm.sessions {
		list = append(list, s)
	}
	return list
}

// RemoveSession cleans up a completed call.
func (sm *SessionManager) RemoveSession(callID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.sessions, callID)
}

// FeedAudio ingests incoming PCM samples, advances the DSP ring buffer, and triggers scoring hops.
func (sm *SessionManager) FeedAudio(session *CallSession, samples []float32) {
	session.mu.Lock()
	session.LastActive = time.Now().UTC()
	session.TotalSamples += int64(len(samples))
	session.ringBuffer.Write(samples)
	session.samplesSinceHop += len(samples)

	// In-place circular write to recent audio history
	recCap := len(session.recentAudio)
	for _, s := range samples {
		session.recentAudio[session.recentWriteIdx] = s
		session.recentWriteIdx = (session.recentWriteIdx + 1) % recCap
		session.recentTotal++
	}

	windowSize := session.dspPipeline.WindowSize()
	hopSize := session.dspPipeline.HopSize()

	for session.samplesSinceHop >= hopSize && session.ringBuffer.Available() >= windowSize {
		session.samplesSinceHop -= hopSize

		session.ringBuffer.ReadLatest(session.windowBuf)

		// 1. Run DSP feature extraction (zero alloc)
		features := session.dspPipeline.ExtractFeatures(session.windowBuf)

		// 2. Evaluate classifier score
		diag := sm.scorer.EvaluateFrame(session.sessionState, features)
		session.LastDiagnosis = diag

		if diag.SmoothedScore > session.PeakScore {
			session.PeakScore = diag.SmoothedScore
		}
		if diag.RiskLevel == classifier.RiskCritical {
			session.OverallRisk = classifier.RiskCritical
		} else if diag.RiskLevel == classifier.RiskSuspicious && session.OverallRisk != classifier.RiskCritical {
			session.OverallRisk = classifier.RiskSuspicious
		}

		// 3. Check for threat alerts with active security policy
		minHits := sm.cfg.Detection.MinVoicedFrames
		var activePolicy action.SecurityPolicy
		if sm.policyEngine != nil {
			activePolicy = sm.policyEngine.GetActivePolicy()
			minHits = activePolicy.MinVoicedHits
			session.ActivePolicy = string(activePolicy.Name)
		}

		if diag.RiskLevel == classifier.RiskCritical && !session.AlertTriggered && diag.ConsecutiveHits >= minHits {
			session.AlertTriggered = true
			if activePolicy.Action == action.ActionAutoTerminateCall {
				session.TerminatedByPolicy = true
			}

			// Extract ordered audio slice from circular history buffer
			var incidentSamples []float32
			avail := int(session.recentTotal)
			if avail > recCap {
				avail = recCap
			}
			incidentSamples = make([]float32, avail)
			startIdx := (session.recentWriteIdx - avail + recCap) % recCap
			for i := 0; i < avail; i++ {
				incidentSamples[i] = session.recentAudio[(startIdx+i)%recCap]
			}

			wavPath := ""
			if sm.recorder != nil {
				meta, err := sm.recorder.Record(session.CallID, incidentSamples, session.SampleRate, diag)
				if err == nil && meta != nil {
					wavPath = meta.AudioWAVPath
				}
			}

			if sm.dispatcher != nil {
				sm.dispatcher.Dispatch(action.AlertPayload{
					Event:         "deepfake.detected",
					CallID:        session.CallID,
					Timestamp:     time.Now().UTC(),
					ThreatLevel:   diag.RiskLevel,
					PeakScore:     session.PeakScore,
					Confidence:    diag.Confidence,
					KeyIndicators: diag.KeyIndicators,
					AudioWAVPath:  wavPath,
				})
			}
		}

		session.mu.Unlock()

		// 4. Notify listeners outside lock
		sm.notifyListeners(session, diag)

		session.mu.Lock()
	}

	session.mu.Unlock()
}

func (sm *SessionManager) notifyListeners(session *CallSession, diag classifier.FrameDiagnosis) {
	sm.mu.RLock()
	listeners := make([]func(*CallSession, classifier.FrameDiagnosis), len(sm.listeners))
	copy(listeners, sm.listeners)
	sm.mu.RUnlock()

	for _, l := range listeners {
		l(session, diag)
	}
}
