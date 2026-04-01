package health

import (
	"fmt"
	"time"
)

// FreshnessStatus classifies a last_verified timestamp.
type FreshnessStatus int

const (
	StatusFresh FreshnessStatus = iota
	StatusStale
	StatusNeverVerified
)

// String returns a human-readable name for the status.
func (s FreshnessStatus) String() string {
	switch s {
	case StatusFresh:
		return "fresh"
	case StatusStale:
		return "stale"
	case StatusNeverVerified:
		return "never-verified"
	default:
		return "unknown"
	}
}

// StalenessDetail holds the classification result with context for messages.
type StalenessDetail struct {
	Status       FreshnessStatus
	DaysOverdue  int       // 0 if fresh or never-verified
	LastVerified time.Time // zero if never-verified
}

// ClassifyFreshness returns the staleness classification for a given timestamp.
// If isZero is true, the file has no last_verified field (never verified).
// window is the staleness threshold in days (must be positive).
func ClassifyFreshness(lastVerified time.Time, isZero bool, window int, now time.Time) StalenessDetail {
	if isZero {
		return StalenessDetail{Status: StatusNeverVerified}
	}
	daysSince := int(now.Sub(lastVerified).Hours() / 24)
	if daysSince > window {
		return StalenessDetail{
			Status:       StatusStale,
			DaysOverdue:  daysSince - window,
			LastVerified: lastVerified,
		}
	}
	return StalenessDetail{
		Status:       StatusFresh,
		LastVerified: lastVerified,
	}
}

// ValidateStalenessWindow returns an error if days is not positive.
func ValidateStalenessWindow(days int) error {
	if days <= 0 {
		return fmt.Errorf("staleness window must be positive, got %d", days)
	}
	return nil
}
