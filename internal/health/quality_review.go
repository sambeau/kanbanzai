package health

import "fmt"

// QualityReviewThreshold is the number of features at which an accumulated
// non-blocking finding pattern triggers a quality review signal.
const QualityReviewThreshold = 3

// CheckQualityReviewSignal scans features for quality review signals:
//   - Features that have reached the auto-validation cycle cap (blocked)
//   - Features near the cycle cap (maxCycles - 1 remaining)
//
// Tiers is a map of tier name → maxCycles for near-cap calculation.
// When the same escalation pattern (same tier + same blocked stage) appears
// on QualityReviewThreshold or more features, a quality review warning is
// raised in the health dashboard.
func CheckQualityReviewSignal(features []map[string]any, tiers map[string]int) CategoryResult {
	result := NewCategoryResult()

	// Count features blocked at cycle cap per tier.
	blockedByTier := make(map[string]int)
	// Count features near the cycle cap per tier.
	nearCapByTier := make(map[string]int)
	// Collect feature IDs for signal messages.
	blockedIDsByTier := make(map[string][]string)
	nearCapIDsByTier := make(map[string][]string)

	for _, f := range features {
		featureID, _ := f["id"].(string)
		if featureID == "" {
			continue
		}

		tier, _ := f["tier"].(string)
		if tier == "" {
			tier = "feature" // default tier
		}

		blockedReason, _ := f["blocked_reason"].(string)
		reviewCycle, _ := f["review_cycle"].(int)

		if blockedReason != "" {
			blockedByTier[tier]++
			blockedIDsByTier[tier] = append(blockedIDsByTier[tier], featureID)
		}

		// Near cap: review_cycle is at maxCycles-1 (one more cycle before escalation).
		// Compute per-tier using the actual tier configuration to avoid false
		// positives for tiers with max_cycles < 2 (e.g. critical max_cycles=0).
		maxCycles, hasCfg := tiers[tier]
		if !hasCfg {
			maxCycles = 2 // default for unconfigured tiers
		}
		if blockedReason == "" && maxCycles > 0 && reviewCycle >= maxCycles-1 {
			nearCapByTier[tier]++
			nearCapIDsByTier[tier] = append(nearCapIDsByTier[tier], featureID)
		}
	}

	// Signal: blocked features across tiers.
	for tier, count := range blockedByTier {
		if count >= QualityReviewThreshold {
			result.AddIssue(Issue{
				Severity: SeverityWarning,
				Message: fmt.Sprintf(
					"quality_review_signal: %d features in tier %q have reached the auto-validation cycle cap and are blocked pending human escalation: %v",
					count, tier, blockedIDsByTier[tier]),
			})
		} else if count > 0 {
			result.AddIssue(Issue{
				Severity: SeverityInfo,
				Message: fmt.Sprintf(
					"%d feature(s) in tier %q are blocked at cycle cap: %v",
					count, tier, blockedIDsByTier[tier]),
			})
		}
	}

	// Signal: features nearing the cap.
	for tier, count := range nearCapByTier {
		if count >= QualityReviewThreshold {
			result.AddIssue(Issue{
				Severity: SeverityInfo,
				Message: fmt.Sprintf(
					"quality_review_signal: %d features in tier %q are near their auto-validation cycle cap: %v",
					count, tier, nearCapIDsByTier[tier]),
			})
		}
	}

	return result
}
