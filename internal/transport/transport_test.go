package transport

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/alexandrmotologa/voiceprobe/internal/classifier"
	"github.com/alexandrmotologa/voiceprobe/internal/config"
)

func TestG711Decode(t *testing.T) {
	// 0xFF in mu-law is quiet/zero
	uLawBytes := []byte{0xFF, 0x7F, 0x00}
	decoded := DecodeMuLaw(uLawBytes)
	if len(decoded) != 3 {
		t.Fatalf("expected 3 decoded samples, got %d", len(decoded))
	}
	for _, s := range decoded {
		if s < -1.0 || s > 1.0 {
			t.Fatalf("sample out of range [-1.0, 1.0]: %f", s)
		}
	}

	aLawBytes := []byte{0xD5, 0x55, 0x00}
	decodedA := DecodeALaw(aLawBytes)
	if len(decodedA) != 3 {
		t.Fatalf("expected 3 decoded samples, got %d", len(decodedA))
	}
}

func TestParseRTP(t *testing.T) {
	// Construct valid 12-byte RTP packet with 4-byte payload
	packet := make([]byte, 16)
	packet[0] = 0x80 // V=2, P=0, X=0, CC=0
	packet[1] = 0x00 // M=0, PT=0 (PCMU)
	binary.BigEndian.PutUint16(packet[2:4], 1024)
	binary.BigEndian.PutUint32(packet[4:8], 160000)
	binary.BigEndian.PutUint32(packet[8:12], 987654)
	copy(packet[12:], []byte{0xFF, 0xFE, 0xFD, 0xFC})

	hdr, payload, err := ParseRTP(packet)
	if err != nil {
		t.Fatalf("ParseRTP failed: %v", err)
	}
	if hdr.Version != 2 {
		t.Fatalf("expected version 2, got %d", hdr.Version)
	}
	if hdr.PayloadType != 0 {
		t.Fatalf("expected PT 0, got %d", hdr.PayloadType)
	}
	if hdr.SequenceNumber != 1024 {
		t.Fatalf("expected seq 1024, got %d", hdr.SequenceNumber)
	}
	if len(payload) != 4 {
		t.Fatalf("expected payload len 4, got %d", len(payload))
	}
}

func TestParseSIPMessage(t *testing.T) {
	raw := "INVITE sip:alice@bank.net SIP/2.0\r\nVia: SIP/2.0/UDP 10.0.0.1:5060\r\nCall-ID: call-abc-123\r\nFrom: <sip:bob@external.com>\r\n\r\nv=0\r\n"
	msg := ParseSIPMessage(raw)
	if msg.StartLine != "INVITE sip:alice@bank.net SIP/2.0" {
		t.Fatalf("unexpected start line: %s", msg.StartLine)
	}
	if msg.CallID() != "call-abc-123" {
		t.Fatalf("expected call-id 'call-abc-123', got '%s'", msg.CallID())
	}
}

func TestSessionManagerFeedAudio(t *testing.T) {
	cfg := config.DefaultConfig()
	sm := NewSessionManager(cfg, nil, nil)

	var receivedDiags []classifier.FrameDiagnosis
	sm.AddListener(func(session *CallSession, diag classifier.FrameDiagnosis) {
		receivedDiags = append(receivedDiags, diag)
	})

	sess := sm.GetOrCreateSession("test-call-stream", "test", "L16", 8000)
	if sess == nil {
		t.Fatal("session must not be nil")
	}

	// Feed 400ms of audio (3200 samples) at 8000Hz
	// Window is 300ms (2400 samples), hop is 150ms (1200 samples)
	// Feeding 3200 samples should trigger at least 1 hop evaluation!
	samples := make([]float32, 3200)
	for i := range samples {
		samples[i] = float32(0.5 * math.Sin(2.0*math.Pi*200.0*float64(i)/8000.0))
	}

	sm.FeedAudio(sess, samples)

	if len(receivedDiags) == 0 {
		t.Fatalf("expected at least 1 diagnostic evaluation after feeding 3200 samples")
	}

	active := sm.ListSessions()
	if len(active) != 1 {
		t.Fatalf("expected 1 active session, got %d", len(active))
	}
}
