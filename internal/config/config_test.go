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
	if cfg.Prefixes[0].Label != "Plan" {
		t.Errorf("Prefixes[0].Label = %q, want %q", cfg.Prefixes[0].Label, "Plan")
	}
	if cfg.Prefixes[0].Retired {
		t.Error("Prefixes[0].Retired = true, want false")
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
					{Prefix: "P", Label: "Plan"},
				},
			},
			wantErr: false,
		},
		{
			name: "multiple prefixes",
			cfg: Config{
				Version: "2",
				Prefixes: []PrefixEntry{
					{Prefix: "P", Label: "Plan"},
					{Prefix: "F", Label: "Feature Plan"},
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
					{Prefix: "P", Label: "Plan"},
					{Prefix: "P", Label: "Another Plan"},
				},
			},
			wantErr: true,
		},
		{
			name: "all retired",
			cfg: Config{
				Version: "2",
				Prefixes: []PrefixEntry{
					{Prefix: "P", Label: "Plan", Retired: true},
				},
			},
			wantErr: true,
		},
		{
			name: "some retired",
			cfg: Config{
				Version: "2",
				Prefixes: []PrefixEntry{
					{Prefix: "P", Label: "Plan", Retired: true},
					{Prefix: "X", Label: "Active"},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid prefix format",
			cfg: Config{
				Version: "2",
				Prefixes: []PrefixEntry{
					{Prefix: "PP", Label: "Invalid"},
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
			{Prefix: "P", Label: "Plan"},
			{Prefix: "X", Label: "Extra", Retired: true},
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
			{Prefix: "P", Label: "Plan"},
			{Prefix: "X", Label: "Extra", Retired: true},
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
			{Prefix: "P", Label: "Plan"},
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
			{Prefix: "P", Label: "Plan"},
			{Prefix: "X", Label: "Extra", Retired: true},
			{Prefix: "A", Label: "Active"},
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
	err := cfg.AddPrefix("F", "Feature Plan")
	if err != nil {
		t.Fatalf("AddPrefix(F) error = %v", err)
	}

	if !cfg.IsValidPrefix("F") {
		t.Error("F should be valid after adding")
	}

	// Try to add duplicate
	err = cfg.AddPrefix("F", "Duplicate")
	if err == nil {
		t.Error("AddPrefix(F) should fail for duplicate")
	}

	// Try to add invalid prefix
	err = cfg.AddPrefix("PP", "Invalid")
	if err == nil {
		t.Error("AddPrefix(PP) should fail for invalid format")
	}
}

func TestConfig_RetirePrefix(t *testing.T) {
	t.Parallel()

	cfg := Config{
		Version: "2",
		Prefixes: []PrefixEntry{
			{Prefix: "P", Label: "Plan"},
			{Prefix: "X", Label: "Extra"},
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
			{Prefix: "P", Label: "Plan"},
			{Prefix: "T", Label: "Test"},
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
