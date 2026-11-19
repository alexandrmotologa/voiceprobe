package action

import (
	"testing"

	"github.com/alexandrmotologa/voiceprobe/internal/classifier"
)

func TestPolicyEngine(t *testing.T) {
	pe := NewPolicyEngine()

	defaultPolicy := pe.GetActivePolicy()
	if defaultPolicy.Name != PolicyStandard {
		t.Fatalf("expected default policy to be STANDARD, got %s", defaultPolicy.Name)
	}

	ok := pe.SetPolicy(PolicyHighValueWire)
	if !ok {
		t.Fatal("failed to set HIGH_VALUE_WIRE policy")
	}

	active := pe.GetActivePolicy()
	if active.Name != PolicyHighValueWire {
		t.Fatalf("expected active policy HIGH_VALUE_WIRE, got %s", active.Name)
	}
	if active.CriticalThreshold != 0.55 {
		t.Fatalf("expected critical threshold 0.55, got %f", active.CriticalThreshold)
	}

	// Test invalid policy name
	if pe.SetPolicy("NON_EXISTENT") {
		t.Fatal("expected failure on non-existent policy")
	}
}

func TestComplianceReport(t *testing.T) {
	report, err := GenerateComplianceReport(
		"test-call-123",
		"",
		0.94,
		0.98,
		classifier.RiskCritical,
		[]string{"Suppressed pitch jitter", "High phase discontinuity"},
	)
	if err != nil {
		t.Fatalf("failed to generate compliance report: %v", err)
	}

	if report.CallID != "test-call-123" {
		t.Fatalf("unexpected call ID: %s", report.CallID)
	}
	if report.OverallThreatLevel != classifier.RiskCritical {
		t.Fatalf("unexpected threat level: %s", report.OverallThreatLevel)
	}
	if report.DigitalSignature == "" {
		t.Fatal("expected non-empty cryptographic digital signature")
	}
}
