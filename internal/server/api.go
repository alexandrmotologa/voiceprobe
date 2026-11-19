package server

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alexandrmotologa/voiceprobe/internal/action"
	"github.com/alexandrmotologa/voiceprobe/internal/classifier"
	"github.com/alexandrmotologa/voiceprobe/internal/config"
	"github.com/alexandrmotologa/voiceprobe/internal/dsp"
	"github.com/alexandrmotologa/voiceprobe/internal/transport"
)

// ServerAPI provides administrative and operational REST endpoints.
type ServerAPI struct {
	cfg       *config.Config
	sm        *transport.SessionManager
	sseHub    *SSEHub
	startTime time.Time
}

// NewServerAPI initializes the API service.
func NewServerAPI(cfg *config.Config, sm *transport.SessionManager, sseHub *SSEHub) *ServerAPI {
	return &ServerAPI{
		cfg:       cfg,
		sm:        sm,
		sseHub:    sseHub,
		startTime: time.Now().UTC(),
	}
}

// RegisterRoutes attaches endpoints to an HTTP mux.
func (api *ServerAPI) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/status", api.corsMiddleware(api.handleStatus))
	mux.HandleFunc("/api/v1/policy", api.corsMiddleware(api.handlePolicy))
	mux.HandleFunc("/api/v1/calls", api.corsMiddleware(api.handleCalls))
	mux.HandleFunc("/api/v1/calls/", api.corsMiddleware(api.handleCallByID))
	mux.HandleFunc("/api/v1/analyze", api.corsMiddleware(api.handleAnalyze))
	mux.HandleFunc("/api/v1/simulate", api.corsMiddleware(api.handleSimulate))
	mux.HandleFunc("/api/v1/incidents", api.corsMiddleware(api.handleIncidents))
	mux.HandleFunc("/api/v1/incidents/", api.corsMiddleware(api.handleIncidentFile))
	mux.HandleFunc("/api/v1/events", api.sseHub.Handler())
	mux.HandleFunc(api.cfg.Server.TwilioPath, transport.HandleTwilioWebSocket(api.sm))
	mux.HandleFunc("/stream/browser-mic", transport.HandleBrowserMicWebSocket(api.sm))
}

func (api *ServerAPI) handlePolicy(w http.ResponseWriter, r *http.Request) {
	pe := api.sm.GetPolicyEngine()
	if pe == nil {
		http.Error(w, `{"error":"policy engine not configured"}`, http.StatusInternalServerError)
		return
	}

	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"active":   pe.GetActivePolicy(),
			"policies": pe.ListPolicies(),
		})
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			Policy string `json:"policy"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid JSON request"}`, http.StatusBadRequest)
			return
		}

		ok := pe.SetPolicy(action.SecurityPolicyName(req.Policy))
		if !ok {
			http.Error(w, `{"error":"unknown policy name"}`, http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "policy_updated",
			"active": pe.GetActivePolicy(),
		})
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func (api *ServerAPI) corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if api.cfg.Server.EnableCORS {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}
		}
		next(w, r)
	}
}

func (api *ServerAPI) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessions := api.sm.ListSessions()
	activeCount := len(sessions)

	status := map[string]interface{}{
		"status":          "operational",
		"service":         "voiceprobe",
		"version":         "1.0.0",
		"uptime_seconds":  time.Since(api.startTime).Seconds(),
		"active_sessions": activeCount,
		"config":          api.cfg,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (api *ServerAPI) handleCalls(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessions := api.sm.ListSessions()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessions)
}

func (api *ServerAPI) handleCallByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/calls/")
	parts := strings.Split(path, "/")
	callID := parts[0]

	sess, exists := api.sm.GetSession(callID)
	if !exists {
		http.Error(w, `{"error":"session not found"}`, http.StatusNotFound)
		return
	}

	if len(parts) > 1 && parts[1] == "drop" && r.Method == http.MethodPost {
		api.sm.RemoveSession(callID)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"dropped","call_id":"%s"}`, callID)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sess)
}

// FileAnalysisResult holds forensic diagnostics for an uploaded audio file.
type FileAnalysisResult struct {
	FileName           string                      `json:"file_name"`
	SampleRate         int                         `json:"sample_rate"`
	DurationSec        float64                     `json:"duration_sec"`
	TotalFrames        int                         `json:"total_frames"`
	VoicedFrames       int                         `json:"voiced_frames"`
	PeakScore          float64                     `json:"peak_score"`
	AverageScore       float64                     `json:"average_score"`
	OverallThreatLevel classifier.RiskLevel        `json:"overall_threat_level"`
	SummaryIndicators  []string                    `json:"summary_indicators"`
	Frames             []classifier.FrameDiagnosis `json:"frames"`
}

func (api *ServerAPI) handleAnalyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	file, header, err := r.FormFile("audio")
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid file upload: %v", err), http.StatusBadRequest)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "failed to read file content", http.StatusInternalServerError)
		return
	}

	samples, sampleRate, err := ParseWAVSamples(data)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to parse audio WAV: %v", err), http.StatusBadRequest)
		return
	}

	// Run full forensic analysis pass
	res := AnalyzeSamples(header.Filename, samples, sampleRate, api.cfg)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func (api *ServerAPI) handleSimulate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Type        string `json:"type"` // "human" or "cloned"
		DurationSec int    `json:"duration_sec"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.Type = "cloned"
		req.DurationSec = 6
	}
	if req.DurationSec <= 0 || req.DurationSec > 30 {
		req.DurationSec = 6
	}

	callID := fmt.Sprintf("sim-%s-%d", req.Type, time.Now().Unix()%100000)
	go RunCallSimulation(api.sm, req.Type, callID, req.DurationSec)

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status":"simulation_started","call_id":"%s","type":"%s","duration_sec":%d}`,
		callID, req.Type, req.DurationSec)
}

func (api *ServerAPI) handleIncidents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	dir := api.cfg.Actions.IncidentsDir
	entries, err := os.ReadDir(dir)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
		return
	}

	var list []map[string]interface{}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".json") {
			fullPath := filepath.Join(dir, e.Name())
			content, err := os.ReadFile(fullPath)
			if err == nil {
				var meta map[string]interface{}
				if json.Unmarshal(content, &meta) == nil {
					list = append(list, meta)
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func (api *ServerAPI) handleIncidentFile(w http.ResponseWriter, r *http.Request) {
	rawPath := strings.TrimPrefix(r.URL.Path, "/api/v1/incidents/")

	// If requesting compliance audit report for an incident
	if strings.HasSuffix(rawPath, "/report") {
		base := strings.TrimSuffix(rawPath, "/report")
		safeBase := filepath.Base(base)
		jsonName := strings.TrimSuffix(safeBase, ".wav") + ".json"
		fullJsonPath := filepath.Join(api.cfg.Actions.IncidentsDir, jsonName)

		content, err := os.ReadFile(fullJsonPath)
		if err != nil {
			http.Error(w, `{"error":"incident metadata not found"}`, http.StatusNotFound)
			return
		}

		var meta struct {
			CallID         string               `json:"call_id"`
			AudioWAVPath   string               `json:"audio_wav_path"`
			ThreatLevel    classifier.RiskLevel `json:"threat_level"`
			PeakScore      float64              `json:"peak_score"`
			Confidence     float64              `json:"confidence"`
			KeyIndicators  []string             `json:"key_indicators"`
		}
		if err := json.Unmarshal(content, &meta); err != nil {
			http.Error(w, `{"error":"corrupted metadata"}`, http.StatusInternalServerError)
			return
		}

		report, err := action.GenerateComplianceReport(
			meta.CallID,
			meta.AudioWAVPath,
			meta.PeakScore,
			meta.Confidence,
			meta.ThreatLevel,
			meta.KeyIndicators,
		)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"failed to generate report: %v"}`, err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(report)
		return
	}

	safeName := filepath.Base(rawPath)
	fullPath := filepath.Join(api.cfg.Actions.IncidentsDir, safeName)

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}

	http.ServeFile(w, r, fullPath)
}

// ParseWAVSamples extracts linear float32 mono audio samples from a standard WAV buffer.
func ParseWAVSamples(data []byte) ([]float32, int, error) {
	if len(data) < 44 {
		return nil, 0, fmt.Errorf("file too short for WAV header")
	}

	if string(data[0:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		return nil, 0, fmt.Errorf("invalid RIFF/WAVE header")
	}

	var sampleRate int
	var channels int
	var bitsPerSample int
	var dataOffset int
	var dataLen int

	pos := 12
	for pos+8 <= len(data) {
		chunkID := string(data[pos : pos+4])
		chunkSize := int(binary.LittleEndian.Uint32(data[pos+4 : pos+8]))
		pos += 8

		if chunkID == "fmt " {
			if pos+16 > len(data) {
				return nil, 0, fmt.Errorf("truncated fmt chunk")
			}
			channels = int(binary.LittleEndian.Uint16(data[pos+2 : pos+4]))
			sampleRate = int(binary.LittleEndian.Uint32(data[pos+4 : pos+8]))
			bitsPerSample = int(binary.LittleEndian.Uint16(data[pos+14 : pos+16]))
		} else if chunkID == "data" {
			dataOffset = pos
			dataLen = chunkSize
			break
		}

		pos += chunkSize
	}

	if dataOffset == 0 || dataLen == 0 {
		return nil, 0, fmt.Errorf("data chunk not found in WAV")
	}
	if dataOffset+dataLen > len(data) {
		dataLen = len(data) - dataOffset
	}

	bytesPerSample := bitsPerSample / 8
	if bytesPerSample <= 0 {
		bytesPerSample = 2
	}
	totalSamples := dataLen / (bytesPerSample * channels)
	out := make([]float32, totalSamples)

	for i := 0; i < totalSamples; i++ {
		offset := dataOffset + i*(bytesPerSample*channels)
		var val float32
		if bitsPerSample == 16 {
			raw := int16(binary.LittleEndian.Uint16(data[offset : offset+2]))
			val = float32(raw) / 32768.0
		} else if bitsPerSample == 8 {
			raw := int16(data[offset]) - 128
			val = float32(raw) / 128.0
		} else {
			// Default 16-bit
			raw := int16(binary.LittleEndian.Uint16(data[offset : offset+2]))
			val = float32(raw) / 32768.0
		}
		out[i] = val
	}

	if sampleRate == 0 {
		sampleRate = 8000
	}
	return out, sampleRate, nil
}

// AnalyzeSamples runs the full DSP and classifier pipeline on pre-loaded audio samples.
func AnalyzeSamples(fileName string, samples []float32, sampleRate int, cfg *config.Config) FileAnalysisResult {
	pipeline := dsp.NewPipeline(
		sampleRate,
		cfg.DSP.WindowDurationMs,
		cfg.DSP.HopDurationMs,
		cfg.DSP.MelBins,
		cfg.DSP.VADEnergyThreshold,
	)
	scorer := classifier.NewScorer(
		cfg.Detection.SuspiciousThreshold,
		cfg.Detection.CriticalThreshold,
		cfg.Detection.SmoothingAlpha,
		cfg.Detection.MinVoicedFrames,
	)
	session := scorer.NewSession("offline-analysis")

	windowSize := pipeline.WindowSize()
	hopSize := pipeline.HopSize()

	var frameDiags []classifier.FrameDiagnosis
	var peakScore, sumScore float64
	voicedCount := 0

	for pos := 0; pos+windowSize <= len(samples); pos += hopSize {
		chunk := samples[pos : pos+windowSize]
		features := pipeline.ExtractFeatures(chunk)
		diag := scorer.EvaluateFrame(session, features)
		frameDiags = append(frameDiags, diag)

		if diag.IsVoiced {
			voicedCount++
			sumScore += diag.SmoothedScore
			if diag.SmoothedScore > peakScore {
				peakScore = diag.SmoothedScore
			}
		}
	}

	avgScore := 0.0
	if voicedCount > 0 {
		avgScore = sumScore / float64(voicedCount)
	}

	// Classify overall threat based on sustained anomaly patterns (consistent with real-time session gating)
	maxConsecutiveHits := 0
	for _, diag := range frameDiags {
		if diag.ConsecutiveHits > maxConsecutiveHits {
			maxConsecutiveHits = diag.ConsecutiveHits
		}
	}

	var overallRisk classifier.RiskLevel = classifier.RiskClean
	if maxConsecutiveHits >= cfg.Detection.MinVoicedFrames || avgScore >= cfg.Detection.CriticalThreshold {
		overallRisk = classifier.RiskCritical
	} else if avgScore >= cfg.Detection.SuspiciousThreshold || maxConsecutiveHits >= 1 {
		overallRisk = classifier.RiskSuspicious
	}

	var summary []string
	if overallRisk == classifier.RiskCritical {
		summary = append(summary, "Elevated probability of synthetic voice cloning detected")
		summary = append(summary, "Persistent glottal phase irregularities observed across speech frames")
		summary = append(summary, "Absence of natural neuromuscular laryngeal micro-tremor")
	} else if overallRisk == classifier.RiskSuspicious {
		summary = append(summary, "Isolated acoustic fluctuations detected; within acceptable human boundary")
	} else {
		summary = append(summary, "Acoustic signatures match natural human phonation baselines")
	}

	return FileAnalysisResult{
		FileName:           fileName,
		SampleRate:         sampleRate,
		DurationSec:        float64(len(samples)) / float64(sampleRate),
		TotalFrames:        len(frameDiags),
		VoicedFrames:       voicedCount,
		PeakScore:          peakScore,
		AverageScore:       avgScore,
		OverallThreatLevel: overallRisk,
		SummaryIndicators:  summary,
		Frames:             frameDiags,
	}
}

// FormantFilter implements a 2-pole digital vocal tract resonator.
type FormantFilter struct {
	a1, a2 float64
	y1, y2 float64
}

func newFormantFilter(freq, bw, sampleRate float64) *FormantFilter {
	r := math.Exp(-math.Pi * bw / sampleRate)
	theta := 2.0 * math.Pi * freq / sampleRate
	return &FormantFilter{
		a1: 2.0 * r * math.Cos(theta),
		a2: -r * r,
	}
}

func (res *FormantFilter) process(x float64) float64 {
	y := x + res.a1*res.y1 + res.a2*res.y2
	res.y2 = res.y1
	res.y1 = y
	return y
}

// RunCallSimulation generates realistic streaming telephony packets in real time for testing.
func RunCallSimulation(sm *transport.SessionManager, simType string, callID string, durationSec int) {
	sampleRate := 8000.0
	session := sm.GetOrCreateSession(callID, "simulator", "G.711u", int(sampleRate))

	packetDurationMs := 50 // 50ms chunks (400 samples at 8000Hz)
	packetSamples := int(sampleRate) * packetDurationMs / 1000
	totalPackets := durationSec * 1000 / packetDurationMs

	r1 := newFormantFilter(650.0, 80.0, sampleRate)
	r2 := newFormantFilter(1250.0, 110.0, sampleRate)
	r3 := newFormantFilter(2400.0, 150.0, sampleRate)

	f0 := 150.0
	basePeriod := sampleRate / f0
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	ticker := time.NewTicker(time.Duration(packetDurationMs) * time.Millisecond)
	defer ticker.Stop()

	sampleGlobalIdx := 0
	cycleSampleIdx := 0
	currentCycleLen := int(math.Round(basePeriod))
	currentShimmer := 1.0

	for p := 0; p < totalPackets; p++ {
		<-ticker.C

		chunk := make([]float32, packetSamples)
		for i := 0; i < packetSamples; i++ {
			timeSec := float64(sampleGlobalIdx) / sampleRate
			cyclePos := math.Mod(timeSec, 1.8)

			if cyclePos > 1.5 {
				// Conversational breath pause
				chunk[i] = float32(0.002 * (rnd.Float64() - 0.5))
			} else {
				// Start a new glottal cycle if previous completed
				if cycleSampleIdx >= currentCycleLen {
					cycleSampleIdx = 0
					if simType == "cloned" {
						// Cloned vocoder: static cycle length (0.00% jitter), static amplitude (0.00% shimmer)
						currentCycleLen = int(math.Round(basePeriod))
						currentShimmer = 1.0
					} else {
						// Natural human speech: cycle-to-cycle neuromuscular jitter (~1.8%) and shimmer (~3.5%)
						jitterPerturb := (rnd.Float64() - 0.5) * 2.0 * (basePeriod * 0.018)
						currentCycleLen = int(math.Round(basePeriod + jitterPerturb))
						if currentCycleLen < 25 {
							currentCycleLen = int(basePeriod)
						}
						currentShimmer = 1.0 + (rnd.Float64()-0.5)*0.07
					}
				}

				// Generate Rosenberg glottal pulse
				var glottalPulse float64
				openPhase := float64(currentCycleLen) * 0.60
				if float64(cycleSampleIdx) < openPhase {
					glottalPulse = 0.5 * (1.0 - math.Cos(math.Pi*float64(cycleSampleIdx)/openPhase))
				} else {
					glottalPulse = math.Cos(0.5 * math.Pi * (float64(cycleSampleIdx) - openPhase) / (float64(currentCycleLen) - openPhase))
				}

				var excitation float64
				if simType == "cloned" {
					// Vocoder phase noise burst and high-frequency upsampling artifact
					phaseNoise := 0.04 * (rnd.Float64() - 0.5)
					excitation = (glottalPulse + phaseNoise) * currentShimmer * 0.20
					vocalOut := (r1.process(excitation)*0.5 + r2.process(excitation)*0.3 + r3.process(excitation)*0.15) * 0.15
					checkerboard := 0.06 * math.Sin(2.0*math.Pi*2900.0*timeSec)
					total := vocalOut + checkerboard
					if total > 0.95 {
						total = 0.95
					} else if total < -0.95 {
						total = -0.95
					}
					chunk[i] = float32(total)
				} else {
					// Clean human voice through acoustic resonators
					excitation = glottalPulse * currentShimmer * 0.20
					vocalOut := (r1.process(excitation)*0.5 + r2.process(excitation)*0.3 + r3.process(excitation)*0.15) * 0.15
					if vocalOut > 0.95 {
						vocalOut = 0.95
					} else if vocalOut < -0.95 {
						vocalOut = -0.95
					}
					chunk[i] = float32(vocalOut)
				}

				cycleSampleIdx++
			}
			sampleGlobalIdx++
		}

		sm.FeedAudio(session, chunk)
	}
}
