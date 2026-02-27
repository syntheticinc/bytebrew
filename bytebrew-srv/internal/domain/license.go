package domain

import "time"

// LicenseTier represents the subscription tier.
type LicenseTier string

const (
	TierTrial    LicenseTier = "trial"
	TierPersonal LicenseTier = "personal"
	TierTeams    LicenseTier = "teams"
)

// LicenseStatus represents the current license validation status.
type LicenseStatus string

const (
	LicenseActive  LicenseStatus = "active"
	LicenseGrace   LicenseStatus = "grace"
	LicenseBlocked LicenseStatus = "blocked"
)

// LicenseFeatures describes capabilities for a given tier.
type LicenseFeatures struct {
	FullAutonomy     bool
	ParallelAgents   int // -1 = unlimited, 1 = single
	ExploreCodebase  bool
	TraceSymbol      bool
	CodebaseIndexing bool
}

// LicenseInfo holds validated license information.
type LicenseInfo struct {
	UserID              string
	Email               string
	Tier                LicenseTier
	ExpiresAt           time.Time
	GraceUntil          time.Time
	Features            LicenseFeatures
	Status              LicenseStatus
	ProxyStepsRemaining int
	ProxyStepsLimit     int
	BYOKEnabled         bool
	MaxSeats            int
}

// BlockedLicense returns a license that blocks all functionality.
// Used when no valid license is present (missing, expired, tampered).
func BlockedLicense() *LicenseInfo {
	return &LicenseInfo{
		Tier:   "",
		Status: LicenseBlocked,
		Features: LicenseFeatures{
			FullAutonomy:     false,
			ParallelAgents:   0,
			ExploreCodebase:  false,
			TraceSymbol:      false,
			CodebaseIndexing: false,
		},
		BYOKEnabled: false,
	}
}
