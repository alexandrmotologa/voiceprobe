package action

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/alexandrmotologa/voiceprobe/internal/classifier"
)

// IncidentMetadata records forensic audit details for a detected deepfake call.
type IncidentMetadata struct {
	CallID         string                   `json:"call_id"`
	Timestamp      time.Time                `json:"timestamp"`
	ThreatLevel    classifier.RiskLevel     `json:"threat_level"`
	PeakScore      float64                  `json:"peak_score"`
	Confidence     float64                  `json:"confidence"`
	SampleRate     int                      `json:"sample_rate"`
	SampleCount    int                      `json:"sample_count"`
	DurationSec    float64                  `json:"duration_sec"`
	KeyIndicators  []string                 `json:"key_indicators"`
	Diagnosis      classifier.FrameDiagnosis `json:"diagnosis"`
	AudioWAVPath   string                   `json:"audio_wav_path"`
}

// IncidentRecorder manages persistent audit trails for fraud investigations.
type IncidentRecorder struct {
	outputDir string
	enabled   bool
}

// NewIncidentRecorder initializes output directories.
func NewIncidentRecorder(outputDir string, enabled bool) (*IncidentRecorder, error) {
	if enabled {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return nil, err
		}
	}
	return &IncidentRecorder{
		outputDir: outputDir,
		enabled:   enabled,
	}, nil
}

// Record saves offending audio PCM samples as a WAV file with paired JSON forensic metadata.
func (r *IncidentRecorder) Record(callID string, samples []float32, sampleRate int, diag classifier.FrameDiagnosis) (*IncidentMetadata, error) {
	if !r.enabled {
		return nil, nil
	}

	timestamp := time.Now().UTC()
	safeID := fmt.Sprintf("%s_%s", callID, timestamp.Format("20060102_150405"))
	wavPath := filepath.Join(r.outputDir, safeID+".wav")
	jsonPath := filepath.Join(r.outputDir, safeID+".json")

	// 1. Write WAV audio
	if err := WriteWAV(wavPath, samples, sampleRate); err != nil {
		return nil, fmt.Errorf("failed to write forensic wav: %w", err)
	}

	// 2. Prepare metadata
	meta := &IncidentMetadata{
		CallID:        callID,
		Timestamp:     timestamp,
		ThreatLevel:   diag.RiskLevel,
		PeakScore:     diag.SmoothedScore,
		Confidence:    diag.Confidence,
		SampleRate:    sampleRate,
		SampleCount:   len(samples),
		DurationSec:   float64(len(samples)) / float64(sampleRate),
		KeyIndicators: diag.KeyIndicators,
		Diagnosis:     diag,
		AudioWAVPath:  wavPath,
	}

	jsonData, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return nil, err
	}

	if err := os.WriteFile(jsonPath, jsonData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write forensic json: %w", err)
	}

	return meta, nil
}

// WriteWAV writes linear PCM 16-bit mono audio samples to a standard RIFF/WAVE file.
func WriteWAV(filePath string, samples []float32, sampleRate int) error {
	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	numSamples := len(samples)
	numChannels := int16(1)
	bitsPerSample := int16(16)
	byteRate := int32(sampleRate * int(numChannels) * int(bitsPerSample/8))
	blockAlign := int16(numChannels * (bitsPerSample / 8))
	subchunk2Size := int32(numSamples * int(numChannels) * int(bitsPerSample/8))
	chunkSize := 36 + subchunk2Size

	// RIFF header
	if _, err := f.Write([]byte("RIFF")); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, chunkSize); err != nil {
		return err
	}
	if _, err := f.Write([]byte("WAVE")); err != nil {
		return err
	}

	// fmt subchunk
	if _, err := f.Write([]byte("fmt ")); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, int32(16)); err != nil { // Subchunk1Size
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, int16(1)); err != nil { // AudioFormat (1 = PCM)
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, numChannels); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, int32(sampleRate)); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, byteRate); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, blockAlign); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, bitsPerSample); err != nil {
		return err
	}

	// data subchunk
	if _, err := f.Write([]byte("data")); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, subchunk2Size); err != nil {
		return err
	}

	// Sample data in int16
	for _, s := range samples {
		// Clamp float32 [-1.0, 1.0] to int16 [-32768, 32767]
		val := s
		if val > 1.0 {
			val = 1.0
		} else if val < -1.0 {
			val = -1.0
		}
		int16Sample := int16(val * 32767.0)
		if err := binary.Write(f, binary.LittleEndian, int16Sample); err != nil {
			return err
		}
	}

	return nil
}
