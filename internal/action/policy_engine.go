package action

import (
	"sync"
)

// SecurityPolicyName identifies active banking defense postures.
type SecurityPolicyName string

const (
	PolicyStandard            SecurityPolicyName = "STANDARD"
	PolicyHighValueWire       SecurityPolicyName = "HIGH_VALUE_WIRE"
	PolicyExecutiveProtection SecurityPolicyName = "EXECUTIVE_PROTECTION"
)

// ActionTrigger defines response behaviors when an acoustic fraud threshold is reached.
type ActionTrigger string

const (
	ActionLogOnly            ActionTrigger = "LOG_ONLY"
	ActionInjectSipHeader    ActionTrigger = "INJECT_SIP_HEADER"
	ActionRequireStepUpMFA   ActionTrigger = "REQUIRE_STEP_UP_MFA"
	ActionAutoTerminateCall  ActionTrigger = "AUTO_TERMINATE_CALL"
)

// SecurityPolicy defines operational thresholds and automated response protocols.
type SecurityPolicy struct {
	Name                SecurityPolicyName `json:"name"`
	Description         string             `json:"description"`
	SuspiciousThreshold float64            `json:"suspicious_threshold"`
	CriticalThreshold   float64            `json:"critical_threshold"`
	MinVoicedHits       int                `json:"min_voiced_hits"`
	Action              ActionTrigger      `json:"action"`
	MaxWireAmountEUR    float64            `json:"max_wire_amount_eur,omitempty"`
}

// PolicyEngine manages active risk thresholds and determines real-time banking responses.
type PolicyEngine struct {
	mu            sync.RWMutex
	currentPolicy SecurityPolicy
	policies      map[SecurityPolicyName]SecurityPolicy
}

// NewPolicyEngine creates a policy engine initialized with Standard banking protection.
func NewPolicyEngine() *PolicyEngine {
	policies := map[SecurityPolicyName]SecurityPolicy{
		PolicyStandard: {
			Name:                PolicyStandard,
			Description:         "Default banking call center surveillance. Alerts operators via SIP headers without call interruption.",
			SuspiciousThreshold: 0.40,
			CriticalThreshold:   0.75,
			MinVoicedHits:       3,
			Action:              ActionInjectSipHeader,
			MaxWireAmountEUR:    10000.0,
		},
		PolicyHighValueWire: {
			Name:                PolicyHighValueWire,
			Description:         "High-value funds transfer mode (> 10,000 EUR). Tighter acoustic tolerances, triggers mandatory biometric MFA step-up.",
			SuspiciousThreshold: 0.30,
			CriticalThreshold:   0.55,
			MinVoicedHits:       2,
			Action:              ActionRequireStepUpMFA,
			MaxWireAmountEUR:    100000.0,
		},
		PolicyExecutiveProtection: {
			Name:                PolicyExecutiveProtection,
			Description:         "C-suite executive impersonation prevention. Immediate automated call termination upon synthetic signature verification.",
			SuspiciousThreshold: 0.25,
			CriticalThreshold:   0.50,
			MinVoicedHits:       2,
			Action:              ActionAutoTerminateCall,
			MaxWireAmountEUR:    0.0,
		},
	}

	return &PolicyEngine{
		currentPolicy: policies[PolicyStandard],
		policies:      policies,
	}
}

// GetActivePolicy returns the currently enforced risk policy.
func (pe *PolicyEngine) GetActivePolicy() SecurityPolicy {
	pe.mu.RLock()
	defer pe.mu.RUnlock()
	return pe.currentPolicy
}

// SetPolicy switches the active banking policy.
func (pe *PolicyEngine) SetPolicy(name SecurityPolicyName) bool {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	p, exists := pe.policies[name]
	if !exists {
		return false
	}
	pe.currentPolicy = p
	return true
}

// ListPolicies returns all pre-calibrated policy postures.
func (pe *PolicyEngine) ListPolicies() []SecurityPolicy {
	pe.mu.RLock()
	defer pe.mu.RUnlock()

	list := make([]SecurityPolicy, 0, len(pe.policies))
	for _, p := range pe.policies {
		list = append(list, p)
	}
	return list
}
