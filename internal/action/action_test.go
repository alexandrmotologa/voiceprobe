package action

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alexandrmotologa/voiceprobe/internal/classifier"
)

func TestInjectSIPAlertHeader(t *testing.T) {
	sipMsg := "INVITE sip:bob@biloxi.com SIP/2.0\r\nVia: SIP/2.0/UDP pc33.atlanta.com\r\n\r\nv=0\r\n"
	diagClean := classifier.FrameDiagnosis{
		RiskLevel: classifier.RiskClean,
	}
	cleanResult := InjectSIPAlertHeader(sipMsg, diagClean)
	if strings.Contains(cleanResult, "X-Deepfake-Alert") {
		t.Fatalf("expected no header injection for clean risk")
	}

	diagCrit := classifier.FrameDiagnosis{
		RiskLevel:     classifier.RiskCritical,
		SmoothedScore: 0.942,
		Confidence:    0.985,
	}
	critResult := InjectSIPAlertHeader(sipMsg, diagCrit)
	if !strings.Contains(critResult, "X-Deepfake-Alert: CRITICAL; score=0.942; confidence=0.985; model=voiceprobe-int8") {
		t.Fatalf("expected critical header in result, got:\n%s", critResult)
	}
	if !strings.Contains(critResult, "X-Deepfake-Recommendation: REQUIRE_MFA_STEPUP") {
		t.Fatalf("expected recommendation header in result")
	}
}

func TestWriteWAVAndIncident(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "vp_incidents_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	recorder, err := NewIncidentRecorder(tempDir, true)
	if err != nil {
		t.Fatal(err)
	}

	samples := make([]float32, 8000) // 1 second at 8000Hz
	for i := range samples {
		samples[i] = float32(0.5)
	}

	diag := classifier.FrameDiagnosis{
		RiskLevel:     classifier.RiskCritical,
		SmoothedScore: 0.91,
		Confidence:    0.96,
		KeyIndicators: []string{"High phase discontinuity (0.092)"},
	}

	meta, err := recorder.Record("call-999", samples, 8000, diag)
	if err != nil {
		t.Fatalf("record failed: %v", err)
	}
	if meta == nil {
		t.Fatal("expected non-nil metadata")
	}

	// Verify WAV exists and has reasonable size
	fi, err := os.Stat(meta.AudioWAVPath)
	if err != nil {
		t.Fatalf("wav file not found: %v", err)
	}
	expectedSize := 44 + 8000*2 // 44 bytes header + 16000 bytes PCM16 data
	if fi.Size() != int64(expectedSize) {
		t.Fatalf("expected file size %d, got %d", expectedSize, fi.Size())
	}

	// Verify JSON metadata exists
	jsonPath := strings.TrimSuffix(meta.AudioWAVPath, filepath.Ext(meta.AudioWAVPath)) + ".json"
	if _, err := os.Stat(jsonPath); err != nil {
		t.Fatalf("json file not found: %v", err)
	}
}
