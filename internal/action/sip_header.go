package action

import (
	"fmt"
	"strings"

	"github.com/alexandrmotologa/voiceprobe/internal/classifier"
)

// InjectSIPAlertHeader inserts custom security headers into raw SIP messages.
// If the call risk is CRITICAL or SUSPICIOUS, it injects X-Deepfake-Alert and step-up headers.
func InjectSIPAlertHeader(sipMsg string, diag classifier.FrameDiagnosis) string {
	if diag.RiskLevel == classifier.RiskClean {
		return sipMsg
	}

	header := fmt.Sprintf("X-Deepfake-Alert: %s; score=%.3f; confidence=%.3f; model=voiceprobe-int8\r\n",
		diag.RiskLevel, diag.SmoothedScore, diag.Confidence)

	if diag.RiskLevel == classifier.RiskCritical {
		header += "X-Deepfake-Recommendation: REQUIRE_MFA_STEPUP\r\n"
	}

	// Insert before first empty line marking header-body separation
	splitIdx := strings.Index(sipMsg, "\r\n\r\n")
	if splitIdx != -1 {
		return sipMsg[:splitIdx] + "\r\n" + header + sipMsg[splitIdx+2:]
	}

	// Fallback to LF only
	splitIdx = strings.Index(sipMsg, "\n\n")
	if splitIdx != -1 {
		return sipMsg[:splitIdx] + "\n" + strings.ReplaceAll(header, "\r\n", "\n") + sipMsg[splitIdx+1:]
	}

	return sipMsg + header
}
