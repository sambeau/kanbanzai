package service

import (
	"strings"
	"testing"

	"github.com/sambeau/kanbanzai/internal/config"
)

func TestEntityService_ResolvePlanByNumber(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	writeTestPlan(t, svc, "P1-basic-plan")
	writeTestPlan(t, svc, "P2-another-plan")
	writeTestPlan(t, svc, "M1-meta-plan")

	cfg := config.Config{
		Prefixes: []config.PrefixEntry{
			{Prefix: "P", Name: "Plan"},
			{Prefix: "M", Name: "Meta"},
			{Prefix: "X", Name: "Old", Retired: true},
		},
	}

	tests := []struct {
		name       string
		prefix     string
		number     string
		wantID     string
		wantSlug   string
		wantErr    bool
		errContain string
	}{
		{
			name:     "matching plan P1",
			prefix:   "P",
			number:   "1",
			wantID:   "P1-basic-plan",
			wantSlug: "basic-plan",
		},
		{
			name:     "matching plan P2",
			prefix:   "P",
			number:   "2",
			wantID:   "P2-another-plan",
			wantSlug: "another-plan",
		},
		{
			name:     "matching plan M1",
			prefix:   "M",
			number:   "1",
			wantID:   "M1-meta-plan",
			wantSlug: "meta-plan",
		},
		{
			// AC-011: valid prefix but no matching number
			name:       "valid prefix no matching number",
			prefix:     "P",
			number:     "99",
			wantErr:    true,
			errContain: "no plan found",
		},
		{
			name:       "unknown prefix error message",
			prefix:     "Z",
			number:     "1",
			wantErr:    true,
			errContain: "unknown plan prefix",
		},
		{
			name:       "unknown prefix lists valid prefixes",
			prefix:     "Z",
			number:     "1",
			wantErr:    true,
			errContain: "P",
		},
		{
			// retired prefix is not active
			name:       "retired prefix treated as unknown",
			prefix:     "X",
			number:     "1",
			wantErr:    true,
			errContain: "unknown plan prefix",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotID, gotSlug, err := svc.ResolvePlanByNumber(cfg, tt.prefix, tt.number)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ResolvePlanByNumber() error = nil, want error containing %q", tt.errContain)
				}
				if !strings.Contains(err.Error(), tt.errContain) {
					t.Fatalf("ResolvePlanByNumber() error = %q, want error containing %q", err, tt.errContain)
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolvePlanByNumber() error = %v", err)
			}
			if gotID != tt.wantID {
				t.Errorf("ResolvePlanByNumber() id = %q, want %q", gotID, tt.wantID)
			}
			if gotSlug != tt.wantSlug {
				t.Errorf("ResolvePlanByNumber() slug = %q, want %q", gotSlug, tt.wantSlug)
			}
		})
	}
}
