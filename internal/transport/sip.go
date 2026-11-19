package transport

import (
	"bufio"
	"strings"

	"github.com/alexandrmotologa/voiceprobe/internal/action"
	"github.com/alexandrmotologa/voiceprobe/internal/classifier"
)

// SIPMessage represents a parsed SIP text envelope.
type SIPMessage struct {
	StartLine string
	Headers   map[string]string
	Body      string
	Raw       string
}

// ParseSIPMessage splits a raw SIP packet into headers and payload.
func ParseSIPMessage(raw string) *SIPMessage {
	msg := &SIPMessage{
		Headers: make(map[string]string),
		Raw:     raw,
	}

	scanner := bufio.NewScanner(strings.NewReader(raw))
	isFirstLine := true
	isBody := false
	var bodyBuilder strings.Builder

	for scanner.Scan() {
		line := scanner.Text()

		if isBody {
			bodyBuilder.WriteString(line)
			bodyBuilder.WriteString("\n")
			continue
		}

		if isFirstLine {
			msg.StartLine = strings.TrimSpace(line)
			isFirstLine = false
			continue
		}

		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			isBody = true
			continue
		}

		colonIdx := strings.Index(line, ":")
		if colonIdx != -1 {
			name := strings.TrimSpace(line[:colonIdx])
			val := strings.TrimSpace(line[colonIdx+1:])
			msg.Headers[strings.ToLower(name)] = val
		}
	}

	msg.Body = bodyBuilder.String()
	return msg
}

// CallID returns the Call-ID header value.
func (m *SIPMessage) CallID() string {
	return m.Headers["call-id"]
}

// MutateWithAlert injects X-Deepfake-Alert headers into the message string if high threat is detected.
func (m *SIPMessage) MutateWithAlert(diag classifier.FrameDiagnosis) string {
	return action.InjectSIPAlertHeader(m.Raw, diag)
}
