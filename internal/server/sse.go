package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/alexandrmotologa/voiceprobe/internal/classifier"
	"github.com/alexandrmotologa/voiceprobe/internal/transport"
)

// TelemetryEvent carries live telemetry for frontend charts and meters.
type TelemetryEvent struct {
	CallID             string                    `json:"call_id"`
	Timestamp          time.Time                 `json:"timestamp"`
	InstantScore       float64                   `json:"instant_score"`
	SmoothedScore      float64                   `json:"smoothed_score"`
	RiskLevel          classifier.RiskLevel      `json:"risk_level"`
	Confidence         float64                   `json:"confidence"`
	PhaseDiscontinuity float64                   `json:"phase_discontinuity"`
	SpectralFlux       float64                   `json:"spectral_flux"`
	SpectralCentroid   float64                   `json:"spectral_centroid"`
	Jitter             float64                   `json:"jitter"`
	Shimmer            float64                   `json:"shimmer"`
	F0Hz               float64                   `json:"f0_hz"`
	SpectralTilt       float64                   `json:"spectral_tilt"`
	FormantDispersion  float64                   `json:"formant_dispersion"`
	THDPercent         float64                   `json:"thd_percent"`
	LoudspeakerAnomaly bool                      `json:"loudspeaker_anomaly"`
	VADEnergy          float64                   `json:"vad_energy"`
	IsVoiced           bool                      `json:"is_voiced"`
	ConsecutiveHits    int                       `json:"consecutive_hits"`
	KeyIndicators      []string                  `json:"key_indicators"`
	MelEnergies        []float64                 `json:"mel_energies,omitempty"`
}

// SSEHub broadcasts real-time telemetry to subscribed web browsers.
type SSEHub struct {
	mu      sync.RWMutex
	clients map[chan []byte]bool
}

// NewSSEHub allocates the connection pool.
func NewSSEHub() *SSEHub {
	return &SSEHub{
		clients: make(map[chan []byte]bool),
	}
}

// AddClient registers a browser HTTP connection.
func (h *SSEHub) AddClient(ch chan []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[ch] = true
}

// RemoveClient unregisters a disconnected browser.
func (h *SSEHub) RemoveClient(ch chan []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, ch)
	close(ch)
}

// Broadcast serializes and dispatches an event to all connected browsers.
func (h *SSEHub) Broadcast(event TelemetryEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(h.clients) == 0 {
		return
	}

	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	msg := fmt.Sprintf("event: telemetry\ndata: %s\n\n", data)
	bytesMsg := []byte(msg)

	for ch := range h.clients {
		select {
		case ch <- bytesMsg:
		default:
			// Non-blocking drop if client is slow
		}
	}
}

// Handler handles the SSE endpoint (/api/v1/events).
func (h *SSEHub) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		clientChan := make(chan []byte, 64)
		h.AddClient(clientChan)
		defer h.RemoveClient(clientChan)

		// Send initial keep-alive ping
		fmt.Fprintf(w, ": ping\n\n")
		flusher.Flush()

		ctx := r.Context()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-clientChan:
				if !ok {
					return
				}
				if _, err := w.Write(msg); err != nil {
					return
				}
				flusher.Flush()
			}
		}
	}
}

// AttachToSessionManager binds the SSE hub to receive telemetry from all active calls.
func (h *SSEHub) AttachToSessionManager(sm *transport.SessionManager) {
	sm.AddListener(func(session *transport.CallSession, diag classifier.FrameDiagnosis) {
		var melCopy []float64
		if diag.Features != nil && len(diag.Features.MelEnergies) > 0 {
			melCopy = make([]float64, len(diag.Features.MelEnergies))
			copy(melCopy, diag.Features.MelEnergies)
		}

		var phaseDisc, flux, centroid, jitter, shimmer, f0, energy, tilt, formDisp, thd float64
		var loudAnomaly bool
		if diag.Features != nil {
			phaseDisc = diag.Features.PhaseDiscontinuity
			flux = diag.Features.SpectralFlux
			centroid = diag.Features.SpectralCentroid
			jitter = diag.Features.Jitter
			shimmer = diag.Features.Shimmer
			f0 = diag.Features.F0Hz
			energy = diag.Features.VAD.Energy
			tilt = diag.Features.SpectralTilt
			formDisp = diag.Features.FormantDispersion
			thd = diag.Features.THDPercent
			loudAnomaly = diag.Features.LoudspeakerAnomaly
		}

		h.Broadcast(TelemetryEvent{
			CallID:             session.CallID,
			Timestamp:          time.Now().UTC(),
			InstantScore:       diag.InstantScore,
			SmoothedScore:      diag.SmoothedScore,
			RiskLevel:          diag.RiskLevel,
			Confidence:         diag.Confidence,
			PhaseDiscontinuity: phaseDisc,
			SpectralFlux:       flux,
			SpectralCentroid:   centroid,
			Jitter:             jitter,
			Shimmer:            shimmer,
			F0Hz:               f0,
			SpectralTilt:       tilt,
			FormantDispersion:  formDisp,
			THDPercent:         thd,
			LoudspeakerAnomaly: loudAnomaly,
			VADEnergy:          energy,
			IsVoiced:           diag.IsVoiced,
			ConsecutiveHits:    diag.ConsecutiveHits,
			KeyIndicators:      diag.KeyIndicators,
			MelEnergies:        melCopy,
		})
	})
}
