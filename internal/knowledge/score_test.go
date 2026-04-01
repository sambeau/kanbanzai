package knowledge

import (
	"fmt"
	"math"
	"testing"
	"time"
)

func scoredTestEntry(id string, confidence float64, confirmedDaysAgo int, status string, now time.Time) MatchedEntry {
	confirmed := now.Add(-time.Duration(confirmedDaysAgo) * 24 * time.Hour)
	return MatchedEntry{
		ID:          id,
		Topic:       "topic-" + id,
		Content:     "content-" + id,
		Status:      status,
		Confidence:  confidence,
		ConfirmedAt: confirmed,
		CreatedAt:   confirmed,
	}
}

func TestScoreEntry_RecentHigherThanOld(t *testing.T) {
	t.Parallel()

	now := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	recent := scoredTestEntry("recent", 0.8, 0, "confirmed", now)
	old := scoredTestEntry("old", 0.8, 90, "confirmed", now)

	recentScore := ScoreEntry(recent, now)
	oldScore := ScoreEntry(old, now)

	if recentScore <= oldScore {
		t.Errorf("recent score %.4f should be > old score %.4f", recentScore, oldScore)
	}
}

func TestScoreEntry_HighConfidenceStaleBeatsLowConfidenceRecent(t *testing.T) {
	t.Parallel()

	now := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	stale := scoredTestEntry("stale", 0.9, 150, "confirmed", now)
	recent := scoredTestEntry("recent", 0.3, 0, "confirmed", now)

	staleScore := ScoreEntry(stale, now)
	recentScore := ScoreEntry(recent, now)

	if staleScore <= recentScore {
		t.Errorf("stale high-confidence score %.4f should be > recent low-confidence score %.4f", staleScore, recentScore)
	}
}

func TestScoreEntry_UnconfirmedLowerThanConfirmed(t *testing.T) {
	t.Parallel()

	now := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	confirmed := scoredTestEntry("confirmed", 0.8, 10, "confirmed", now)
	contributed := scoredTestEntry("contributed", 0.8, 10, "contributed", now)

	confirmedScore := ScoreEntry(confirmed, now)
	contributedScore := ScoreEntry(contributed, now)

	if confirmedScore <= contributedScore {
		t.Errorf("confirmed score %.4f should be > contributed score %.4f", confirmedScore, contributedScore)
	}
}

func TestScoreEntry_Deterministic(t *testing.T) {
	t.Parallel()

	now := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	entry := scoredTestEntry("stable", 0.7, 30, "confirmed", now)

	score1 := ScoreEntry(entry, now)
	score2 := ScoreEntry(entry, now)

	if score1 != score2 {
		t.Errorf("scores should be identical: %.4f != %.4f", score1, score2)
	}
}

func TestScoreEntry_ZeroConfirmedAt(t *testing.T) {
	t.Parallel()

	now := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	entry := MatchedEntry{
		ID:         "zero-conf",
		Topic:      "topic-zero",
		Content:    "content-zero",
		Status:     "contributed",
		Confidence: 0.8,
		CreatedAt:  now.Add(-30 * 24 * time.Hour),
	}

	withCreatedAt := scoredTestEntry("ref", 0.8, 30, "contributed", now)

	score := ScoreEntry(entry, now)
	refScore := ScoreEntry(withCreatedAt, now)

	if math.Abs(score-refScore) > 1e-9 {
		t.Errorf("zero ConfirmedAt should fall back to CreatedAt: got %.4f, want %.4f", score, refScore)
	}
}

func TestScoreEntry_ZeroConfirmedAtAndCreatedAt(t *testing.T) {
	t.Parallel()

	now := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	entry := MatchedEntry{
		ID:         "zero-both",
		Topic:      "topic-zero-both",
		Content:    "content-zero-both",
		Status:     "contributed",
		Confidence: 0.6,
	}

	// daysSinceConfirmed defaults to 365
	// recency = 1.0 / (1.0 + 365/90) = 1.0 / 5.0556 ≈ 0.1978
	// boost = 0.8
	// score = 0.6 * 0.1978 * 0.8 ≈ 0.0949
	score := ScoreEntry(entry, now)
	expected := 0.6 * (1.0 / (1.0 + 365.0/90.0)) * 0.8

	if math.Abs(score-expected) > 1e-9 {
		t.Errorf("zero timestamps should use 365 days: got %.4f, want %.4f", score, expected)
	}
}

func TestRankAndCap_ExceedsLimit(t *testing.T) {
	t.Parallel()

	now := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	entries := make([]MatchedEntry, 15)
	for i := range entries {
		entries[i] = scoredTestEntry(
			fmt.Sprintf("e-%02d", i),
			0.5+float64(i)*0.03,
			(14-i)*10,
			"confirmed",
			now,
		)
	}

	surfaced, excluded := RankAndCap(entries, now, 10)

	if len(surfaced) != 10 {
		t.Errorf("surfaced count = %d, want 10", len(surfaced))
	}
	if len(excluded) != 5 {
		t.Errorf("excluded count = %d, want 5", len(excluded))
	}
}

func TestRankAndCap_UnderLimit(t *testing.T) {
	t.Parallel()

	now := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	entries := make([]MatchedEntry, 8)
	for i := range entries {
		entries[i] = scoredTestEntry(
			fmt.Sprintf("e-%02d", i),
			0.5+float64(i)*0.05,
			i*10,
			"confirmed",
			now,
		)
	}

	surfaced, excluded := RankAndCap(entries, now, 10)

	if len(surfaced) != 8 {
		t.Errorf("surfaced count = %d, want 8", len(surfaced))
	}
	if len(excluded) != 0 {
		t.Errorf("excluded count = %d, want 0", len(excluded))
	}
}

func TestRankAndCap_ZeroEntries(t *testing.T) {
	t.Parallel()

	now := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)

	surfaced, excluded := RankAndCap(nil, now, 10)

	if len(surfaced) != 0 {
		t.Errorf("surfaced count = %d, want 0", len(surfaced))
	}
	if len(excluded) != 0 {
		t.Errorf("excluded count = %d, want 0", len(excluded))
	}
}

func TestRankAndCap_OrderingHighestLast(t *testing.T) {
	t.Parallel()

	now := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	entries := []MatchedEntry{
		scoredTestEntry("low", 0.3, 60, "contributed", now),
		scoredTestEntry("mid", 0.6, 30, "confirmed", now),
		scoredTestEntry("high", 0.9, 0, "confirmed", now),
	}

	surfaced, _ := RankAndCap(entries, now, 10)

	if len(surfaced) != 3 {
		t.Fatalf("surfaced count = %d, want 3", len(surfaced))
	}

	last := surfaced[len(surfaced)-1]
	for i, s := range surfaced[:len(surfaced)-1] {
		if s.Score > last.Score {
			t.Errorf("surfaced[%d].Score=%.4f > last.Score=%.4f; highest should be last", i, s.Score, last.Score)
		}
	}
}

func TestRankAndCap_ExcludedHaveIDsAndTopics(t *testing.T) {
	t.Parallel()

	now := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	entries := make([]MatchedEntry, 5)
	for i := range entries {
		entries[i] = scoredTestEntry(
			fmt.Sprintf("e-%02d", i),
			0.1+float64(i)*0.2,
			(4-i)*30,
			"confirmed",
			now,
		)
	}

	_, excluded := RankAndCap(entries, now, 3)

	if len(excluded) != 2 {
		t.Fatalf("excluded count = %d, want 2", len(excluded))
	}
	for _, ex := range excluded {
		if ex.ID == "" {
			t.Error("excluded entry has empty ID")
		}
		if ex.Topic == "" {
			t.Error("excluded entry has empty Topic")
		}
	}
}

func TestRankAndCap_DeterministicTieBreaking(t *testing.T) {
	t.Parallel()

	now := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	entries := []MatchedEntry{
		scoredTestEntry("c-entry", 0.8, 30, "confirmed", now),
		scoredTestEntry("a-entry", 0.8, 30, "confirmed", now),
		scoredTestEntry("b-entry", 0.8, 30, "confirmed", now),
	}

	surfaced, _ := RankAndCap(entries, now, 10)

	if len(surfaced) != 3 {
		t.Fatalf("surfaced count = %d, want 3", len(surfaced))
	}

	if surfaced[0].ID != "a-entry" {
		t.Errorf("surfaced[0].ID = %q, want %q", surfaced[0].ID, "a-entry")
	}
	if surfaced[1].ID != "b-entry" {
		t.Errorf("surfaced[1].ID = %q, want %q", surfaced[1].ID, "b-entry")
	}
	if surfaced[2].ID != "c-entry" {
		t.Errorf("surfaced[2].ID = %q, want %q", surfaced[2].ID, "c-entry")
	}
}
