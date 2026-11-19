package server

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alexandrmotologa/voiceprobe/internal/config"
	"github.com/alexandrmotologa/voiceprobe/internal/transport"
)

func TestServerAPIStatusAndCalls(t *testing.T) {
	cfg := config.DefaultConfig()
	sm := transport.NewSessionManager(cfg, nil, nil)
	sse := NewSSEHub()
	api := NewServerAPI(cfg, sm, sse)

	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	// 1. Test /api/v1/status
	req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var statusResp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &statusResp); err != nil {
		t.Fatalf("failed to decode status response: %v", err)
	}
	if statusResp["service"] != "voiceprobe" {
		t.Fatalf("unexpected service name: %v", statusResp["service"])
	}

	// 2. Test /api/v1/calls (initially empty)
	reqCalls := httptest.NewRequest(http.MethodGet, "/api/v1/calls", nil)
	recCalls := httptest.NewRecorder()
	mux.ServeHTTP(recCalls, reqCalls)

	if recCalls.Code != http.StatusOK {
		t.Fatalf("expected calls status 200, got %d", recCalls.Code)
	}
}

func TestServerAnalyzeMultipart(t *testing.T) {
	cfg := config.DefaultConfig()
	sm := transport.NewSessionManager(cfg, nil, nil)
	sse := NewSSEHub()
	api := NewServerAPI(cfg, sm, sse)

	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	// Create test in-memory WAV file
	wavBuf := new(bytes.Buffer)
	samples := make([]float32, 4000) // 0.5s audio
	for i := range samples {
		samples[i] = float32(0.5)
	}
	tempWav, err := osCreateTempWAV(samples, 8000)
	if err != nil {
		t.Fatal(err)
	}

	// Prepare multipart form
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("audio", "sample_test.wav")
	if err != nil {
		t.Fatal(err)
	}
	part.Write(tempWav)
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/analyze", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected analyze status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var analysis FileAnalysisResult
	if err := json.Unmarshal(rec.Body.Bytes(), &analysis); err != nil {
		t.Fatalf("failed to parse analysis json: %v", err)
	}
	if analysis.SampleRate != 8000 {
		t.Fatalf("expected sample rate 8000, got %d", analysis.SampleRate)
	}
	_ = wavBuf
}

func osCreateTempWAV(samples []float32, sampleRate int) ([]byte, error) {
	buf := new(bytes.Buffer)
	// Write minimal WAV header
	buf.WriteString("RIFF")
	subchunk2Size := uint32(len(samples) * 2)
	chunkSize := uint32(36 + subchunk2Size)
	buf.WriteByte(byte(chunkSize))
	buf.WriteByte(byte(chunkSize >> 8))
	buf.WriteByte(byte(chunkSize >> 16))
	buf.WriteByte(byte(chunkSize >> 24))
	buf.WriteString("WAVEfmt ")
	// subchunk1Size = 16
	buf.Write([]byte{16, 0, 0, 0})
	// audioFormat = 1 (PCM)
	buf.Write([]byte{1, 0})
	// channels = 1
	buf.Write([]byte{1, 0})
	// sampleRate
	srBytes := make([]byte, 4)
	srBytes[0] = byte(sampleRate)
	srBytes[1] = byte(sampleRate >> 8)
	srBytes[2] = byte(sampleRate >> 16)
	srBytes[3] = byte(sampleRate >> 24)
	buf.Write(srBytes)
	// byteRate
	byteRate := sampleRate * 2
	buf.Write([]byte{byte(byteRate), byte(byteRate >> 8), byte(byteRate >> 16), byte(byteRate >> 24)})
	// blockAlign = 2
	buf.Write([]byte{2, 0})
	// bitsPerSample = 16
	buf.Write([]byte{16, 0})
	// data chunk
	buf.WriteString("data")
	buf.Write([]byte{byte(subchunk2Size), byte(subchunk2Size >> 8), byte(subchunk2Size >> 16), byte(subchunk2Size >> 24)})
	for _, s := range samples {
		intVal := int16(s * 32767.0)
		buf.Write([]byte{byte(intVal), byte(intVal >> 8)})
	}
	return buf.Bytes(), nil
}

func TestSSEHub(t *testing.T) {
	hub := NewSSEHub()
	ch := make(chan []byte, 10)
	hub.AddClient(ch)

	hub.Broadcast(TelemetryEvent{
		CallID:        "call-sse-test",
		SmoothedScore: 0.88,
		Timestamp:     time.Now().UTC(),
	})

	select {
	case msg := <-ch:
		if len(msg) == 0 {
			t.Fatal("expected non-empty SSE message")
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for SSE broadcast")
	}

	hub.RemoveClient(ch)
}
