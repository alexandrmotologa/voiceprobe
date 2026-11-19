package config

import (
	"encoding/json"
	"os"
	"time"
)

// Config represents runtime configuration parameters for voiceprobe.
type Config struct {
	Server    ServerConfig    `json:"server"`
	DSP       DSPConfig       `json:"dsp"`
	Detection DetectionConfig `json:"detection"`
	Actions   ActionsConfig   `json:"actions"`
}

// ServerConfig defines network listening settings.
type ServerConfig struct {
	HTTPPort    int    `json:"http_port"`
	TwilioPath  string `json:"twilio_path"`
	RTPPort     int    `json:"rtp_port"`
	SIPPort     int    `json:"sip_port"`
	EnableCORS  bool   `json:"enable_cors"`
}

// DSPConfig defines acoustic signal processing parameters.
type DSPConfig struct {
	SampleRate         int     `json:"sample_rate"`          // e.g. 8000 or 16000
	WindowDurationMs   int     `json:"window_duration_ms"`   // e.g. 300ms
	HopDurationMs      int     `json:"hop_duration_ms"`      // e.g. 150ms
	MelBins            int     `json:"mel_bins"`             // e.g. 40
	FFTSize            int     `json:"fft_size"`             // e.g. 2048
	VADEnergyThreshold float64 `json:"vad_energy_threshold"` // Minimum energy ratio for voiced speech
}

// DetectionConfig holds thresholds for classification and alerts.
type DetectionConfig struct {
	SuspiciousThreshold float64 `json:"suspicious_threshold"` // e.g. 0.40
	CriticalThreshold   float64 `json:"critical_threshold"`   // e.g. 0.75
	SmoothingAlpha      float64 `json:"smoothing_alpha"`      // Exponential moving average weight (0.0 to 1.0)
	MinVoicedFrames     int     `json:"min_voiced_frames"`    // Minimum active frames before alerting
}

// ActionsConfig specifies outbound alerting actions.
type ActionsConfig struct {
	WebhookEnabled     bool          `json:"webhook_enabled"`
	WebhookURL         string        `json:"webhook_url"`
	WebhookTimeout     time.Duration `json:"webhook_timeout"`
	SIPHeaderInjection bool          `json:"sip_header_injection"`
	RecordIncidents    bool          `json:"record_incidents"`
	IncidentsDir       string        `json:"incidents_dir"`
}

// DefaultConfig returns safe, pre-calibrated defaults for production telephony.
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			HTTPPort:   7979,
			TwilioPath: "/stream/twilio",
			RTPPort:    5004,
			SIPPort:    5060,
			EnableCORS: true,
		},
		DSP: DSPConfig{
			SampleRate:         8000,
			WindowDurationMs:   300,
			HopDurationMs:      150,
			MelBins:            40,
			FFTSize:            2048,
			VADEnergyThreshold: 0.0008,
		},
		Detection: DetectionConfig{
			SuspiciousThreshold: 0.40,
			CriticalThreshold:   0.75,
			SmoothingAlpha:      0.35,
			MinVoicedFrames:     3,
		},
		Actions: ActionsConfig{
			WebhookEnabled:     false,
			WebhookURL:         "http://127.0.0.1:9000/webhook/fraud-alert",
			WebhookTimeout:     2 * time.Second,
			SIPHeaderInjection: true,
			RecordIncidents:    true,
			IncidentsDir:       "./data/incidents",
		},
	}
}

// LoadFromFile loads configuration from a JSON file.
func LoadFromFile(filePath string) (*Config, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	cfg := DefaultConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
