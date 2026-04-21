package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()

	if cfg.Version != "2" {
		t.Errorf("Version = %q, want %q", cfg.Version, "2")
	}
	if len(cfg.Prefixes) != 1 {
		t.Fatalf("len(Prefixes) = %d, want 1", len(cfg.Prefixes))
	}
	if cfg.Prefixes[0].Prefix != "P" {
		t.Errorf("Prefixes[0].Prefix = %q, want %q", cfg.Prefixes[0].Prefix, "P")
	}
	if cfg.Prefixes[0].Name != "Plan" {
		t.Errorf("Prefixes[0].Name = %q, want %q", cfg.Prefixes[0].Name, "Plan")
	}
	if cfg.Prefixes[0].Retired {
		t.Error("Prefixes[0].Retired = true, want false")
	}
}

func TestDefaultConfig_Phase3Fields(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()

	// BranchTracking defaults
	if cfg.BranchTracking.StaleAfterDays != 14 {
		t.Errorf("BranchTracking.StaleAfterDays = %d, want 14", cfg.BranchTracking.StaleAfterDays)
	}
	if cfg.BranchTracking.DriftWarningCommits != 50 {
		t.Errorf("BranchTracking.DriftWarningCommits = %d, want 50", cfg.BranchTracking.DriftWarningCommits)
	}
	if cfg.BranchTracking.DriftErrorCommits != 100 {
		t.Errorf("BranchTracking.DriftErrorCommits = %d, want 100", cfg.BranchTracking.DriftErrorCommits)
	}

	// Cleanup defaults
	if cfg.Cleanup.GracePeriodDays != 7 {
		t.Errorf("Cleanup.GracePeriodDays = %d, want 7", cfg.Cleanup.GracePeriodDays)
	}
	if !cfg.Cleanup.AutoDeleteRemoteBranch {
		t.Error("Cleanup.AutoDeleteRemoteBranch = false, want true")
	}

	// Knowledge TTL defaults
	if cfg.Knowledge.TTL.Tier3Days != 30 {
		t.Errorf("Knowledge.TTL.Tier3Days = %d, want 30", cfg.Knowledge.TTL.Tier3Days)
	}
	if cfg.Knowledge.TTL.Tier2Days != 90 {
		t.Errorf("Knowledge.TTL.Tier2Days = %d, want 90", cfg.Knowledge.TTL.Tier2Days)
	}

	// Knowledge Promotion defaults
	if cfg.Knowledge.Promotion.MinUseCount != 5 {
		t.Errorf("Knowledge.Promotion.MinUseCount = %d, want 5", cfg.Knowledge.Promotion.MinUseCount)
	}
	if cfg.Knowledge.Promotion.MaxMissCount != 0 {
		t.Errorf("Knowledge.Promotion.MaxMissCount = %d, want 0", cfg.Knowledge.Promotion.MaxMissCount)
	}
	if cfg.Knowledge.Promotion.MinConfidence != 0.7 {
		t.Errorf("Knowledge.Promotion.MinConfidence = %f, want 0.7", cfg.Knowledge.Promotion.MinConfidence)
	}

	// Knowledge Pruning defaults
	if cfg.Knowledge.Pruning.GracePeriodDays != 7 {
		t.Errorf("Knowledge.Pruning.GracePeriodDays = %d, want 7", cfg.Knowledge.Pruning.GracePeriodDays)
	}
}

func TestValidatePrefix(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		prefix  string
		wantErr bool
	}{
		{"P", false},
		{"X", false},
		{"α", false}, // Unicode letter
		{"日", false}, // CJK character
		{"1", true},  // Digit
		{"9", true},  // Digit
		{"", true},   // Empty
		{"PP", true}, // Multiple characters
		{"ab", true}, // Multiple characters
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.prefix, func(t *testing.T) {
			t.Parallel()
			err := ValidatePrefix(tc.prefix)
			if (err != nil) != tc.wantErr {
				t.Errorf("ValidatePrefix(%q) error = %v, wantErr %v", tc.prefix, err, tc.wantErr)
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: Config{
				Version: "2",
				Prefixes: []PrefixEntry{
					{Prefix: "P", Name: "Plan"},
				},
			},
			wantErr: false,
		},
		{
			name: "multiple prefixes",
			cfg: Config{
				Version: "2",
				Prefixes: []PrefixEntry{
					{Prefix: "P", Name: "Plan"},
					{Prefix: "F", Name: "Feature Plan"},
				},
			},
			wantErr: false,
		},
		{
			name: "no prefixes",
			cfg: Config{
				Version:  "2",
				Prefixes: []PrefixEntry{},
			},
			wantErr: true,
		},
		{
			name: "duplicate prefix",
			cfg: Config{
				Version: "2",
				Prefixes: []PrefixEntry{
					{Prefix: "P", Name: "Plan"},
					{Prefix: "P", Name: "Another Plan"},
				},
			},
			wantErr: true,
		},
		{
			name: "all retired",
			cfg: Config{
				Version: "2",
				Prefixes: []PrefixEntry{
					{Prefix: "P", Name: "Plan", Retired: true},
				},
			},
			wantErr: true,
		},
		{
			name: "some retired",
			cfg: Config{
				Version: "2",
				Prefixes: []PrefixEntry{
					{Prefix: "P", Name: "Plan", Retired: true},
					{Prefix: "X", Name: "Active"},
				},
			},
			wantErr: false,
		},
		{
			name: "empty name",
			cfg: Config{
				Version: "2",
				Prefixes: []PrefixEntry{
					{Prefix: "P", Name: ""},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid prefix format",
			cfg: Config{
				Version: "2",
				Prefixes: []PrefixEntry{
					{Prefix: "PP", Name: "Invalid"},
				},
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.cfg.Validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestConfig_IsValidPrefix(t *testing.T) {
	t.Parallel()

	cfg := Config{
		Version: "2",
		Prefixes: []PrefixEntry{
			{Prefix: "P", Name: "Plan"},
			{Prefix: "X", Name: "Extra", Retired: true},
		},
	}

	if !cfg.IsValidPrefix("P") {
		t.Error("IsValidPrefix(P) = false, want true")
	}
	if !cfg.IsValidPrefix("X") {
		t.Error("IsValidPrefix(X) = false, want true (even though retired)")
	}
	if cfg.IsValidPrefix("Y") {
		t.Error("IsValidPrefix(Y) = true, want false")
	}
}

func TestConfig_IsActivePrefix(t *testing.T) {
	t.Parallel()

	cfg := Config{
		Version: "2",
		Prefixes: []PrefixEntry{
			{Prefix: "P", Name: "Plan"},
			{Prefix: "X", Name: "Extra", Retired: true},
		},
	}

	if !cfg.IsActivePrefix("P") {
		t.Error("IsActivePrefix(P) = false, want true")
	}
	if cfg.IsActivePrefix("X") {
		t.Error("IsActivePrefix(X) = true, want false (retired)")
	}
	if cfg.IsActivePrefix("Y") {
		t.Error("IsActivePrefix(Y) = true, want false (not declared)")
	}
}

func TestConfig_GetPrefixEntry(t *testing.T) {
	t.Parallel()

	cfg := Config{
		Version: "2",
		Prefixes: []PrefixEntry{
			{Prefix: "P", Name: "Plan"},
		},
	}

	entry := cfg.GetPrefixEntry("P")
	if entry == nil {
		t.Fatal("GetPrefixEntry(P) = nil, want non-nil")
	}
	if entry.Prefix != "P" {
		t.Errorf("entry.Prefix = %q, want %q", entry.Prefix, "P")
	}

	entry = cfg.GetPrefixEntry("X")
	if entry != nil {
		t.Errorf("GetPrefixEntry(X) = %v, want nil", entry)
	}
}

func TestConfig_ActivePrefixes(t *testing.T) {
	t.Parallel()

	cfg := Config{
		Version: "2",
		Prefixes: []PrefixEntry{
			{Prefix: "P", Name: "Plan"},
			{Prefix: "X", Name: "Extra", Retired: true},
			{Prefix: "A", Name: "Active"},
		},
	}

	active := cfg.ActivePrefixes()
	if len(active) != 2 {
		t.Fatalf("len(ActivePrefixes()) = %d, want 2", len(active))
	}

	// Check that retired prefix is not included
	for _, p := range active {
		if p.Prefix == "X" {
			t.Error("ActivePrefixes() includes retired prefix X")
		}
	}
}

func TestConfig_AddPrefix(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()

	// Add a new prefix
	err := cfg.AddPrefix("F", "Feature Plan", "")
	if err != nil {
		t.Fatalf("AddPrefix(F) error = %v", err)
	}

	if !cfg.IsValidPrefix("F") {
		t.Error("F should be valid after adding")
	}

	// Try to add duplicate
	err = cfg.AddPrefix("F", "Duplicate", "")
	if err == nil {
		t.Error("AddPrefix(F) should fail for duplicate")
	}

	// Try to add invalid prefix
	err = cfg.AddPrefix("PP", "Invalid", "")
	if err == nil {
		t.Error("AddPrefix(PP) should fail for invalid format")
	}
}

func TestConfig_RetirePrefix(t *testing.T) {
	t.Parallel()

	cfg := Config{
		Version: "2",
		Prefixes: []PrefixEntry{
			{Prefix: "P", Name: "Plan"},
			{Prefix: "X", Name: "Extra"},
		},
	}

	// Retire one prefix
	err := cfg.RetirePrefix("X")
	if err != nil {
		t.Fatalf("RetirePrefix(X) error = %v", err)
	}

	if cfg.IsActivePrefix("X") {
		t.Error("X should not be active after retiring")
	}
	if !cfg.IsValidPrefix("X") {
		t.Error("X should still be valid after retiring")
	}

	// Try to retire already retired
	err = cfg.RetirePrefix("X")
	if err == nil {
		t.Error("RetirePrefix(X) should fail for already retired")
	}

	// Try to retire non-existent
	err = cfg.RetirePrefix("Y")
	if err == nil {
		t.Error("RetirePrefix(Y) should fail for non-existent")
	}

	// Try to retire last active prefix
	err = cfg.RetirePrefix("P")
	if err == nil {
		t.Error("RetirePrefix(P) should fail when it's the last active")
	}
}

func TestConfig_SaveAndLoad(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")

	// Create and save config
	cfg := Config{
		Version: "2",
		Prefixes: []PrefixEntry{
			{Prefix: "P", Name: "Plan"},
			{Prefix: "T", Name: "Test"},
		},
	}

	err := cfg.SaveTo(cfgPath)
	if err != nil {
		t.Fatalf("SaveTo() error = %v", err)
	}

	// Load and verify
	loaded, err := LoadFrom(cfgPath)
	if err != nil {
		t.Fatalf("LoadFrom() error = %v", err)
	}

	if loaded.Version != cfg.Version {
		t.Errorf("Version = %q, want %q", loaded.Version, cfg.Version)
	}
	if len(loaded.Prefixes) != len(cfg.Prefixes) {
		t.Fatalf("len(Prefixes) = %d, want %d", len(loaded.Prefixes), len(cfg.Prefixes))
	}
	for i, p := range loaded.Prefixes {
		if p.Prefix != cfg.Prefixes[i].Prefix {
			t.Errorf("Prefixes[%d].Prefix = %q, want %q", i, p.Prefix, cfg.Prefixes[i].Prefix)
		}
	}
}

func TestConfig_Phase3FieldsParseCorrectly(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")

	yamlContent := `version: "2"
prefixes:
  - prefix: P
    name: Plan
branch_tracking:
  stale_after_days: 21
  drift_warning_commits: 75
  drift_error_commits: 150
cleanup:
  grace_period_days: 14
  auto_delete_remote_branch: false
knowledge:
  ttl:
    tier_3_days: 45
    tier_2_days: 120
  promotion:
    min_use_count: 10
    max_miss_count: 2
    min_confidence: 0.8
  pruning:
    grace_period_days: 14
`
	if err := os.WriteFile(cfgPath, []byte(yamlContent), 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	loaded, err := LoadFrom(cfgPath)
	if err != nil {
		t.Fatalf("LoadFrom() error = %v", err)
	}

	// Verify BranchTracking
	if loaded.BranchTracking.StaleAfterDays != 21 {
		t.Errorf("BranchTracking.StaleAfterDays = %d, want 21", loaded.BranchTracking.StaleAfterDays)
	}
	if loaded.BranchTracking.DriftWarningCommits != 75 {
		t.Errorf("BranchTracking.DriftWarningCommits = %d, want 75", loaded.BranchTracking.DriftWarningCommits)
	}
	if loaded.BranchTracking.DriftErrorCommits != 150 {
		t.Errorf("BranchTracking.DriftErrorCommits = %d, want 150", loaded.BranchTracking.DriftErrorCommits)
	}

	// Verify Cleanup
	if loaded.Cleanup.GracePeriodDays != 14 {
		t.Errorf("Cleanup.GracePeriodDays = %d, want 14", loaded.Cleanup.GracePeriodDays)
	}
	if loaded.Cleanup.AutoDeleteRemoteBranch {
		t.Error("Cleanup.AutoDeleteRemoteBranch = true, want false")
	}

	// Verify Knowledge TTL
	if loaded.Knowledge.TTL.Tier3Days != 45 {
		t.Errorf("Knowledge.TTL.Tier3Days = %d, want 45", loaded.Knowledge.TTL.Tier3Days)
	}
	if loaded.Knowledge.TTL.Tier2Days != 120 {
		t.Errorf("Knowledge.TTL.Tier2Days = %d, want 120", loaded.Knowledge.TTL.Tier2Days)
	}

	// Verify Knowledge Promotion
	if loaded.Knowledge.Promotion.MinUseCount != 10 {
		t.Errorf("Knowledge.Promotion.MinUseCount = %d, want 10", loaded.Knowledge.Promotion.MinUseCount)
	}
	if loaded.Knowledge.Promotion.MaxMissCount != 2 {
		t.Errorf("Knowledge.Promotion.MaxMissCount = %d, want 2", loaded.Knowledge.Promotion.MaxMissCount)
	}
	if loaded.Knowledge.Promotion.MinConfidence != 0.8 {
		t.Errorf("Knowledge.Promotion.MinConfidence = %f, want 0.8", loaded.Knowledge.Promotion.MinConfidence)
	}

	// Verify Knowledge Pruning
	if loaded.Knowledge.Pruning.GracePeriodDays != 14 {
		t.Errorf("Knowledge.Pruning.GracePeriodDays = %d, want 14", loaded.Knowledge.Pruning.GracePeriodDays)
	}
}

func TestConfig_Phase3FieldsDefaultWhenMissing(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")

	// Minimal config without Phase 3 fields
	yamlContent := `version: "2"
prefixes:
  - prefix: P
    name: Plan
`
	if err := os.WriteFile(cfgPath, []byte(yamlContent), 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	loaded, err := LoadFrom(cfgPath)
	if err != nil {
		t.Fatalf("LoadFrom() error = %v", err)
	}

	// When fields are missing, defaults are now merged by LoadFrom
	defaults := DefaultBranchTrackingConfig()
	if loaded.BranchTracking.StaleAfterDays != defaults.StaleAfterDays {
		t.Errorf("BranchTracking.StaleAfterDays = %d, want %d", loaded.BranchTracking.StaleAfterDays, defaults.StaleAfterDays)
	}
	if loaded.Cleanup.GracePeriodDays != DefaultCleanupConfig().GracePeriodDays {
		t.Errorf("Cleanup.GracePeriodDays = %d, want %d", loaded.Cleanup.GracePeriodDays, DefaultCleanupConfig().GracePeriodDays)
	}
	if loaded.Knowledge.TTL.Tier3Days != DefaultKnowledgeConfig().TTL.Tier3Days {
		t.Errorf("Knowledge.TTL.Tier3Days = %d, want %d", loaded.Knowledge.TTL.Tier3Days, DefaultKnowledgeConfig().TTL.Tier3Days)
	}
}

func TestConfig_Phase3Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		modify  func(*Config)
		wantErr string
	}{
		{
			name:   "valid defaults",
			modify: func(c *Config) {},
		},
		{
			name:    "negative stale_after_days",
			modify:  func(c *Config) { c.BranchTracking.StaleAfterDays = -1 },
			wantErr: "stale_after_days must be non-negative",
		},
		{
			name:    "negative drift_warning_commits",
			modify:  func(c *Config) { c.BranchTracking.DriftWarningCommits = -1 },
			wantErr: "drift_warning_commits must be non-negative",
		},
		{
			name:    "negative drift_error_commits",
			modify:  func(c *Config) { c.BranchTracking.DriftErrorCommits = -1 },
			wantErr: "drift_error_commits must be non-negative",
		},
		{
			name: "warning >= error commits",
			modify: func(c *Config) {
				c.BranchTracking.DriftWarningCommits = 100
				c.BranchTracking.DriftErrorCommits = 50
			},
			wantErr: "drift_warning_commits must be less than drift_error_commits",
		},
		{
			name:    "negative grace_period_days",
			modify:  func(c *Config) { c.Cleanup.GracePeriodDays = -1 },
			wantErr: "grace_period_days must be non-negative",
		},
		{
			name:    "negative ttl tier_3_days",
			modify:  func(c *Config) { c.Knowledge.TTL.Tier3Days = -1 },
			wantErr: "tier_3_days must be non-negative",
		},
		{
			name:    "confidence > 1",
			modify:  func(c *Config) { c.Knowledge.Promotion.MinConfidence = 1.5 },
			wantErr: "min_confidence must be between 0 and 1",
		},
		{
			name:    "confidence < 0",
			modify:  func(c *Config) { c.Knowledge.Promotion.MinConfidence = -0.1 },
			wantErr: "min_confidence must be between 0 and 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := DefaultConfig()
			tt.modify(&cfg)
			err := cfg.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("Validate() expected error containing %q, got nil", tt.wantErr)
				} else if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("Validate() error = %v, want containing %q", err, tt.wantErr)
				}
			}
		})
	}
}

func TestConfig_Phase3DefaultsMerging(t *testing.T) {
	t.Parallel()

	// Simulate a pre-Phase 3 config file (only prefixes, no Phase 3 fields)
	configYAML := `version: "2"
prefixes:
  - prefix: P
    name: Plan
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(configYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom() unexpected error: %v", err)
	}

	// Verify defaults were merged
	defaults := DefaultBranchTrackingConfig()
	if cfg.BranchTracking.StaleAfterDays != defaults.StaleAfterDays {
		t.Errorf("StaleAfterDays = %d, want %d", cfg.BranchTracking.StaleAfterDays, defaults.StaleAfterDays)
	}
	if cfg.BranchTracking.DriftWarningCommits != defaults.DriftWarningCommits {
		t.Errorf("DriftWarningCommits = %d, want %d", cfg.BranchTracking.DriftWarningCommits, defaults.DriftWarningCommits)
	}
	if cfg.Cleanup.GracePeriodDays != DefaultCleanupConfig().GracePeriodDays {
		t.Errorf("GracePeriodDays = %d, want %d", cfg.Cleanup.GracePeriodDays, DefaultCleanupConfig().GracePeriodDays)
	}
	if cfg.Knowledge.Promotion.MinConfidence != DefaultKnowledgeConfig().Promotion.MinConfidence {
		t.Errorf("MinConfidence = %f, want %f", cfg.Knowledge.Promotion.MinConfidence, DefaultKnowledgeConfig().Promotion.MinConfidence)
	}
}

func TestLoadFrom_NotFound(t *testing.T) {
	t.Parallel()

	_, err := LoadFrom("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("LoadFrom() should fail for non-existent file")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want to contain 'not found'", err.Error())
	}
}

func TestLoadFrom_InvalidYAML(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")

	// Write invalid YAML
	err := os.WriteFile(cfgPath, []byte("this is not: valid: yaml: content"), 0o644)
	if err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	_, err = LoadFrom(cfgPath)
	if err == nil {
		t.Error("LoadFrom() should fail for invalid YAML")
	}
}

func TestLoadFrom_InvalidConfig(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")

	// Write valid YAML but invalid config (no prefixes)
	err := os.WriteFile(cfgPath, []byte("version: \"2\"\nprefixes: []\n"), 0o644)
	if err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	_, err = LoadFrom(cfgPath)
	if err == nil {
		t.Error("LoadFrom() should fail for invalid config")
	}
	if !strings.Contains(err.Error(), "invalid config") {
		t.Errorf("error = %q, want to contain 'invalid config'", err.Error())
	}
}

func TestLoadOrDefault(t *testing.T) {
	t.Parallel()

	// When config doesn't exist, should return default
	cfg := LoadOrDefault()
	if cfg == nil {
		t.Fatal("LoadOrDefault() = nil")
	}
	if cfg.Version != "2" {
		t.Errorf("Version = %q, want %q (default)", cfg.Version, "2")
	}
	if len(cfg.Prefixes) == 0 {
		t.Error("Prefixes should not be empty")
	}
}

func TestConfig_NextPlanNumber(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()

	// Scanner that returns no existing IDs
	emptyScanner := func() ([]string, error) {
		return nil, nil
	}

	num, err := cfg.NextPlanNumber("P", emptyScanner)
	if err != nil {
		t.Fatalf("NextPlanNumber() error = %v", err)
	}
	if num != 1 {
		t.Errorf("NextPlanNumber() = %d, want 1", num)
	}

	// Scanner that returns some existing IDs
	existingScanner := func() ([]string, error) {
		return []string{"P1-first", "P3-third", "P2-second"}, nil
	}

	num, err = cfg.NextPlanNumber("P", existingScanner)
	if err != nil {
		t.Fatalf("NextPlanNumber() error = %v", err)
	}
	if num != 4 {
		t.Errorf("NextPlanNumber() = %d, want 4", num)
	}

	// Scanner with different prefix - should ignore those
	mixedScanner := func() ([]string, error) {
		return []string{"P1-plan", "X5-other", "P10-big"}, nil
	}

	num, err = cfg.NextPlanNumber("P", mixedScanner)
	if err != nil {
		t.Fatalf("NextPlanNumber() error = %v", err)
	}
	if num != 11 {
		t.Errorf("NextPlanNumber() = %d, want 11", num)
	}

	// Unknown prefix should fail
	_, err = cfg.NextPlanNumber("Z", emptyScanner)
	if err == nil {
		t.Error("NextPlanNumber(Z) should fail for unknown prefix")
	}
}

func TestParsePlanIDParts(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		id         string
		wantPrefix string
		wantNumber string
		wantSlug   string
	}{
		{"P1-basic", "P", "1", "basic"},
		{"P12-multi-word", "P", "12", "multi-word"},
		{"X99-test", "X", "99", "test"},
		{"P1-a", "P", "1", "a"},
		{"FEAT-123", "", "", ""},   // Not a Plan ID
		{"1P-invalid", "", "", ""}, // Starts with digit
		{"P-no-num", "", "", ""},   // No number
		{"", "", "", ""},           // Empty
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.id, func(t *testing.T) {
			t.Parallel()
			prefix, number, slug := parsePlanIDParts(tc.id)
			if prefix != tc.wantPrefix {
				t.Errorf("prefix = %q, want %q", prefix, tc.wantPrefix)
			}
			if number != tc.wantNumber {
				t.Errorf("number = %q, want %q", number, tc.wantNumber)
			}
			if slug != tc.wantSlug {
				t.Errorf("slug = %q, want %q", slug, tc.wantSlug)
			}
		})
	}
}

// TestConfig_MCPConfigParseFromYAML verifies that the mcp section (preset and groups)
// is correctly parsed from a YAML config file (spec §27.1, task A.2).
func TestConfig_MCPConfigParseFromYAML(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		yaml       string
		wantPreset string
		wantGroups map[string]bool
	}{
		{
			name: "preset_only",
			yaml: `version: "2"
prefixes:
  - prefix: P
    name: Plan
mcp:
  preset: orchestration
`,
			wantPreset: "orchestration",
			wantGroups: nil,
		},
		{
			name: "groups_only",
			yaml: `version: "2"
prefixes:
  - prefix: P
    name: Plan
mcp:
  groups:
    core: true
    planning: false
    knowledge: true
`,
			wantPreset: "",
			wantGroups: map[string]bool{
				"core":      true,
				"planning":  false,
				"knowledge": true,
			},
		},
		{
			name: "preset_and_groups",
			yaml: `version: "2"
prefixes:
  - prefix: P
    name: Plan
mcp:
  preset: minimal
  groups:
    checkpoints: true
`,
			wantPreset: "minimal",
			wantGroups: map[string]bool{"checkpoints": true},
		},
		{
			name: "no_mcp_section",
			yaml: `version: "2"
prefixes:
  - prefix: P
    name: Plan
`,
			wantPreset: "",
			wantGroups: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			path := filepath.Join(dir, "config.yaml")
			if err := os.WriteFile(path, []byte(tt.yaml), 0o644); err != nil {
				t.Fatalf("write config: %v", err)
			}

			cfg, err := LoadFrom(path)
			if err != nil {
				t.Fatalf("LoadFrom() error = %v", err)
			}

			if cfg.MCP.Preset != tt.wantPreset {
				t.Errorf("MCP.Preset = %q, want %q", cfg.MCP.Preset, tt.wantPreset)
			}

			if len(tt.wantGroups) == 0 {
				if len(cfg.MCP.Groups) != 0 {
					t.Errorf("MCP.Groups = %v, want empty", cfg.MCP.Groups)
				}
			} else {
				for k, want := range tt.wantGroups {
					got, ok := cfg.MCP.Groups[k]
					if !ok {
						t.Errorf("MCP.Groups[%q] missing, want %v", k, want)
					} else if got != want {
						t.Errorf("MCP.Groups[%q] = %v, want %v", k, got, want)
					}
				}
			}
		})
	}
}

// TestConfig_MCPConfigRoundTrip verifies that an MCPConfig survives a save/load cycle
// without mutation (YAML serialisation correctness).
func TestConfig_MCPConfigRoundTrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	original := Config{
		Version:  "2",
		Prefixes: []PrefixEntry{{Prefix: "P", Name: "Plan"}},
		MCP: MCPConfig{
			Preset: "orchestration",
			Groups: map[string]bool{
				GroupCheckpoints: true,
				GroupKnowledge:   false,
			},
		},
	}

	if err := original.SaveTo(path); err != nil {
		t.Fatalf("SaveTo() error = %v", err)
	}

	loaded, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom() error = %v", err)
	}

	if loaded.MCP.Preset != original.MCP.Preset {
		t.Errorf("MCP.Preset = %q, want %q", loaded.MCP.Preset, original.MCP.Preset)
	}
	for k, want := range original.MCP.Groups {
		got, ok := loaded.MCP.Groups[k]
		if !ok {
			t.Errorf("MCP.Groups[%q] missing after round-trip", k)
		} else if got != want {
			t.Errorf("MCP.Groups[%q] = %v, want %v", k, got, want)
		}
	}
}



// ─── MergeConfig.RequireGitHubPR ─────────────────────────────────────────────

// minimalValidYAML is the smallest valid config YAML (one prefix required).
const minimalValidYAML = "version: \"2\"\nprefixes:\n  - prefix: P\n    name: Plan\n"

// TestMergeConfig_RequireGitHubPR_Unmarshal_True verifies AC-001.
func TestMergeConfig_RequireGitHubPR_Unmarshal_True(t *testing.T) {
	t.Parallel()
	data := minimalValidYAML + "merge:\n  require_github_pr: true\n"
	cfg, err := LoadFrom(writeTempConfig(t, []byte(data)))
	if err != nil {
		t.Fatalf("LoadFrom() error = %v", err)
	}
	if cfg.Merge.RequireGitHubPR == nil {
		t.Fatal("RequireGitHubPR is nil, want non-nil")
	}
	if !*cfg.Merge.RequireGitHubPR {
		t.Errorf("*RequireGitHubPR = false, want true")
	}
}

// TestMergeConfig_RequireGitHubPR_Unmarshal_Absent verifies AC-002.
func TestMergeConfig_RequireGitHubPR_Unmarshal_Absent(t *testing.T) {
	t.Parallel()
	cfg, err := LoadFrom(writeTempConfig(t, []byte(minimalValidYAML)))
	if err != nil {
		t.Fatalf("LoadFrom() error = %v", err)
	}
	if cfg.Merge.RequireGitHubPR != nil {
		t.Errorf("RequireGitHubPR = %v, want nil", *cfg.Merge.RequireGitHubPR)
	}
}

// writeTempConfig writes YAML bytes to a temp file and returns its path.
func writeTempConfig(t *testing.T, data []byte) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "config-*.yaml")
	if err != nil {
		t.Fatalf("create temp config: %v", err)
	}
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	return f.Name()
}

// TestRequiresGitHubPR_NilPointer verifies AC-003.
func TestRequiresGitHubPR_NilPointer(t *testing.T) {
	t.Parallel()
	var m MergeConfig
	if m.RequiresGitHubPR() {
		t.Error("RequiresGitHubPR() = true, want false for nil pointer")
	}
}

// TestRequiresGitHubPR_FalsePointer verifies AC-004.
func TestRequiresGitHubPR_FalsePointer(t *testing.T) {
	t.Parallel()
	v := false
	m := MergeConfig{RequireGitHubPR: &v}
	if m.RequiresGitHubPR() {
		t.Error("RequiresGitHubPR() = true, want false for *false pointer")
	}
}

// TestRequiresGitHubPR_TruePointer verifies AC-005.
func TestRequiresGitHubPR_TruePointer(t *testing.T) {
	t.Parallel()
	v := true
	m := MergeConfig{RequireGitHubPR: &v}
	if !m.RequiresGitHubPR() {
		t.Error("RequiresGitHubPR() = false, want true for *true pointer")
	}
}
