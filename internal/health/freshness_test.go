package health

import (
	"testing"
	"time"
)

var fixedNow = time.Date(2025, 7, 30, 12, 0, 0, 0, time.UTC)

func TestClassifyFreshness_Fresh(t *testing.T) {
	t.Parallel()
	lv := fixedNow.AddDate(0, 0, -10)
	d := ClassifyFreshness(lv, false, 30, fixedNow)
	if d.Status != StatusFresh {
		t.Fatalf("expected StatusFresh, got %v", d.Status)
	}
	if d.DaysOverdue != 0 {
		t.Fatalf("expected DaysOverdue 0, got %d", d.DaysOverdue)
	}
	if d.LastVerified != lv {
		t.Fatalf("expected LastVerified %v, got %v", lv, d.LastVerified)
	}
}

func TestClassifyFreshness_Stale(t *testing.T) {
	t.Parallel()
	lv := fixedNow.AddDate(0, 0, -45)
	d := ClassifyFreshness(lv, false, 30, fixedNow)
	if d.Status != StatusStale {
		t.Fatalf("expected StatusStale, got %v", d.Status)
	}
	if d.DaysOverdue != 15 {
		t.Fatalf("expected DaysOverdue 15, got %d", d.DaysOverdue)
	}
	if d.LastVerified != lv {
		t.Fatalf("expected LastVerified %v, got %v", lv, d.LastVerified)
	}
}

func TestClassifyFreshness_NeverVerified(t *testing.T) {
	t.Parallel()
	d := ClassifyFreshness(time.Time{}, true, 30, fixedNow)
	if d.Status != StatusNeverVerified {
		t.Fatalf("expected StatusNeverVerified, got %v", d.Status)
	}
	if d.DaysOverdue != 0 {
		t.Fatalf("expected DaysOverdue 0, got %d", d.DaysOverdue)
	}
	if !d.LastVerified.IsZero() {
		t.Fatalf("expected zero LastVerified, got %v", d.LastVerified)
	}
}

func TestClassifyFreshness_ExactBoundary(t *testing.T) {
	t.Parallel()
	lv := fixedNow.AddDate(0, 0, -30)
	d := ClassifyFreshness(lv, false, 30, fixedNow)
	if d.Status != StatusFresh {
		t.Fatalf("expected StatusFresh at exact boundary, got %v", d.Status)
	}
	if d.DaysOverdue != 0 {
		t.Fatalf("expected DaysOverdue 0, got %d", d.DaysOverdue)
	}
}

func TestClassifyFreshness_OneDayOverdue(t *testing.T) {
	t.Parallel()
	lv := fixedNow.AddDate(0, 0, -31)
	d := ClassifyFreshness(lv, false, 30, fixedNow)
	if d.Status != StatusStale {
		t.Fatalf("expected StatusStale, got %v", d.Status)
	}
	if d.DaysOverdue != 1 {
		t.Fatalf("expected DaysOverdue 1, got %d", d.DaysOverdue)
	}
}

func TestClassifyFreshness_CustomWindow60(t *testing.T) {
	t.Parallel()
	lv := fixedNow.AddDate(0, 0, -31)
	d := ClassifyFreshness(lv, false, 60, fixedNow)
	if d.Status != StatusFresh {
		t.Fatalf("expected StatusFresh with 60-day window, got %v", d.Status)
	}
	if d.DaysOverdue != 0 {
		t.Fatalf("expected DaysOverdue 0, got %d", d.DaysOverdue)
	}
}

func TestClassifyFreshness_CustomWindow60_Stale(t *testing.T) {
	t.Parallel()
	lv := fixedNow.AddDate(0, 0, -61)
	d := ClassifyFreshness(lv, false, 60, fixedNow)
	if d.Status != StatusStale {
		t.Fatalf("expected StatusStale with 60-day window, got %v", d.Status)
	}
	if d.DaysOverdue != 1 {
		t.Fatalf("expected DaysOverdue 1, got %d", d.DaysOverdue)
	}
}

func TestValidateStalenessWindow(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		days    int
		wantErr bool
	}{
		{"positive", 30, false},
		{"zero", 0, true},
		{"negative", -5, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateStalenessWindow(tt.days)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateStalenessWindow(%d): got err=%v, wantErr=%v", tt.days, err, tt.wantErr)
			}
		})
	}
}

func TestFreshnessStatus_String(t *testing.T) {
	t.Parallel()
	tests := []struct {
		status FreshnessStatus
		want   string
	}{
		{StatusFresh, "fresh"},
		{StatusStale, "stale"},
		{StatusNeverVerified, "never-verified"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			if got := tt.status.String(); got != tt.want {
				t.Fatalf("FreshnessStatus(%d).String() = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}
