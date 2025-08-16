package stokercr

import (
	"bytes"
	"context"
)

type ImageScanner interface {
	Scan(ctx context.Context, b bytes.Buffer) ([]Vuln, error)
}

type Vuln struct {
	ID          string
	PackageID   string
	Title       string
	Description string
	Status      VulnStatus
	Severity    VulnSeverity
}

type VulnStatus int

var (
	StatusNames = []string{
		"unknown",
		"not_affected",
		"affected",
		"fixed",
		"under_investigation",
		"will_not_fix",
		"fix_deferred",
		"end_of_life",
	}
)

const (
	StatusUnknown VulnStatus = iota
	StatusNotAffected
	StatusAffected
	StatusFixed
	StatusUnderInvestigation
	StatusWillNotFix
	StatusFixDeferred
	StatusEndOfLife
)

func NewStatus(status string) VulnStatus {
	for i, s := range StatusNames {
		if status == s {
			return VulnStatus(i)
		}
	}

	return StatusUnknown
}

func (s VulnStatus) String() string {
	if int(s) < 0 || int(s) >= len(StatusNames) {
		return StatusNames[StatusUnknown]
	}
	return StatusNames[s]
}

type VulnSeverity int

const (
	SeverityUnknown VulnSeverity = iota
	SeverityLow
	SeverityMedium
	SeverityHigh
	SeverityCritical
)

var (
	SeverityNames = []string{
		"UNKNOWN",
		"LOW",
		"MEDIUM",
		"HIGH",
		"CRITICAL",
	}
)

func NewSeverity(severity string) VulnSeverity {
	for i, name := range SeverityNames {
		if severity == name {
			return VulnSeverity(i)
		}
	}

	return SeverityUnknown
}

func (s VulnSeverity) String() string {
	if int(s) < 0 || int(s) >= len(SeverityNames) {
		return "UNKNOWN"
	}
	return SeverityNames[s]
}
