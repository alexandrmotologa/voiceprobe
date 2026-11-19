package transport

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow cross-origin for telephony connectors and dev testing
	},
}

// TwilioMessage maps the JSON envelope received from Twilio Media Streams.
type TwilioMessage struct {
	Event     string `json:"event"`
	StreamSid string `json:"streamSid"`
	Start     *struct {
		CallSid     string `json:"callSid"`
		StreamSid   string `json:"streamSid"`
		MediaFormat struct {
			Encoding   string `json:"encoding"`
			SampleRate int    `json:"sampleRate"`
			Channels   int    `json:"channels"`
		} `json:"mediaFormat"`
	} `json:"start,omitempty"`
	Media *struct {
		Track     string `json:"track"`
		Chunk     string `json:"chunk"`
		Timestamp string `json:"timestamp"`
		Payload   string `json:"payload"` // Base64 encoded mu-law audio
	} `json:"media,omitempty"`
	Stop *struct {
		CallSid string `json:"callSid"`
	} `json:"stop,omitempty"`
}

// HandleTwilioWebSocket provides the HTTP handler for Twilio Media Streams.
func HandleTwilioWebSocket(sm *SessionManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("[Twilio WS] Upgrade error: %v", err)
			return
		}
		defer conn.Close()

		var currentCallID string
		var currentSession *CallSession

		for {
			_, msgBytes, err := conn.ReadMessage()
			if err != nil {
				if currentCallID != "" {
					log.Printf("[Twilio WS] Connection closed for call: %s", currentCallID)
				}
				break
			}

			var msg TwilioMessage
			if err := json.Unmarshal(msgBytes, &msg); err != nil {
				continue
			}

			switch msg.Event {
			case "connected":
				// Connection handshake established

			case "start":
				if msg.Start != nil && msg.Start.CallSid != "" {
					currentCallID = msg.Start.CallSid
					sampleRate := msg.Start.MediaFormat.SampleRate
					if sampleRate == 0 {
						sampleRate = 8000
					}
					currentSession = sm.GetOrCreateSession(currentCallID, "twilio", "G.711u", sampleRate)
					log.Printf("[Twilio WS] Call started: %s (sampleRate=%d)", currentCallID, sampleRate)
				}

			case "media":
				if currentSession != nil && msg.Media != nil && msg.Media.Payload != "" {
					muLawBytes, err := base64.StdEncoding.DecodeString(msg.Media.Payload)
					if err == nil && len(muLawBytes) > 0 {
						samples := DecodeMuLaw(muLawBytes)
						sm.FeedAudio(currentSession, samples)
					}
				}

			case "stop":
				if currentCallID != "" {
					log.Printf("[Twilio WS] Call stopped: %s", currentCallID)
					// Keep session in list so operator can view post-call forensics
				}
				return
			}
		}
	}
}

// HandleBrowserMicWebSocket allows web browser clients to stream live microphone audio
// directly to the voiceprobe DSP engine for immediate human vs deepfake verification.
func HandleBrowserMicWebSocket(sm *SessionManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("[Browser Mic WS] Upgrade error: %v", err)
			return
		}
		defer conn.Close()

		callID := r.URL.Query().Get("call_id")
		if callID == "" {
			callID = "mic-live"
		}

		session := sm.GetOrCreateSession(callID, "browser-mic", "PCM16", 8000)
		log.Printf("[Browser Mic WS] Client streaming live microphone: %s", callID)

		defer func() {
			log.Printf("[Browser Mic WS] Client disconnected: %s", callID)
		}()

		for {
			msgType, data, err := conn.ReadMessage()
			if err != nil {
				break
			}

			if msgType == websocket.BinaryMessage {
				// 16-bit signed PCM little-endian at 8000Hz
				numSamples := len(data) / 2
				if numSamples == 0 {
					continue
				}

				samples := make([]float32, numSamples)
				for i := 0; i < numSamples; i++ {
					raw := int16(data[2*i]) | (int16(data[2*i+1]) << 8)
					samples[i] = float32(raw) / 32768.0
				}

				sm.FeedAudio(session, samples)
			}
		}
	}
}
