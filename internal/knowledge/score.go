package knowledge

import (
	"sort"
	"time"
)

type ScoredEntry struct {
	MatchedEntry
	Score float64
}

type ExcludedEntry struct {
	ID    string
	Topic string
}

func ScoreEntry(entry MatchedEntry, now time.Time) float64 {
	ref := entry.ConfirmedAt
	if ref.IsZero() {
		ref = entry.CreatedAt
	}

	daysSinceConfirmed := now.Sub(ref).Hours() / 24
	if ref.IsZero() {
		daysSinceConfirmed = 365
	}
	if daysSinceConfirmed < 0 {
		daysSinceConfirmed = 0
	}

	recencyMultiplier := 1.0 / (1.0 + daysSinceConfirmed/90.0)

	confirmedBoost := 0.8
	if entry.Status == "confirmed" {
		confirmedBoost = 1.0
	}

	return entry.Confidence * recencyMultiplier * confirmedBoost
}

func RankAndCap(entries []MatchedEntry, now time.Time, maxEntries int) ([]ScoredEntry, []ExcludedEntry) {
	scored := make([]ScoredEntry, len(entries))
	for i, e := range entries {
		scored[i] = ScoredEntry{
			MatchedEntry: e,
			Score:        ScoreEntry(e, now),
		}
	}

	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].Score != scored[j].Score {
			return scored[i].Score < scored[j].Score
		}
		return scored[i].ID < scored[j].ID
	})

	if len(scored) <= maxEntries {
		return scored, nil
	}

	cutoff := len(scored) - maxEntries
	excluded := make([]ExcludedEntry, cutoff)
	for i, e := range scored[:cutoff] {
		excluded[i] = ExcludedEntry{
			ID:    e.ID,
			Topic: e.Topic,
		}
	}

	return scored[cutoff:], excluded
}
