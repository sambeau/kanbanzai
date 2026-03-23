// Package config provides project configuration management for Kanbanzai.
// This includes the prefix registry for Plan IDs and other project settings.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"

	"kanbanzai/internal/core"
)

// ConfigFile is the configuration filename.
const ConfigFile = "config.yaml"

// PrefixEntry represents a single prefix in the registry.
type PrefixEntry struct {
	// Prefix is a single non-digit Unicode rune used as Plan ID prefix.
	Prefix string `yaml:"prefix"`
	// Name is a human-readable name for the prefix.
	Name string `yaml:"name"`
	// Description is an optional longer description of the prefix purpose.
	Description string `yaml:"description,omitempty"`
	// Retired indicates this prefix is no longer used for new Plans.
	Retired bool `yaml:"retired,omitempty"`
}

// Config is the project configuration structure stored in .kbz/config.yaml.
type Config struct {
	// Version is the configuration schema version.
	Version string `yaml:"version"`
	// Prefixes is the registry of Plan ID prefixes.
	Prefixes []PrefixEntry `yaml:"prefixes"`
}

// DefaultConfig returns a new Config with sensible defaults.
// The default prefix 'P' for Plan is included.
func DefaultConfig() Config {
	return Config{
		Version: "2",
		Prefixes: []PrefixEntry{
			{Prefix: "P", Name: "Plan"},
		},
	}
}

// Load loads the configuration from the default location.
func Load() (*Config, error) {
	return LoadFrom(filepath.Join(core.RootPath(), ConfigFile))
}

// LoadFrom loads the configuration from the specified path.
func LoadFrom(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("config file not found: %s (run 'kbz init' or create it manually)", path)
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

// LoadOrDefault loads the configuration, returning defaults if not found.
func LoadOrDefault() *Config {
	cfg, err := Load()
	if err != nil {
		def := DefaultConfig()
		return &def
	}
	return cfg
}

// Save saves the configuration to the default location.
func (c *Config) Save() error {
	return c.SaveTo(filepath.Join(core.RootPath(), ConfigFile))
}

// SaveTo saves the configuration to the specified path.
func (c *Config) SaveTo(path string) error {
	if err := c.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

// Validate checks that the configuration is valid.
func (c *Config) Validate() error {
	if len(c.Prefixes) == 0 {
		return errors.New("at least one prefix is required")
	}

	seen := make(map[string]bool)
	hasActive := false

	for i, entry := range c.Prefixes {
		if err := ValidatePrefix(entry.Prefix); err != nil {
			return fmt.Errorf("prefix %d: %w", i, err)
		}

		if entry.Name == "" {
			return fmt.Errorf("prefix %d: name is required", i)
		}

		if seen[entry.Prefix] {
			return fmt.Errorf("duplicate prefix: %q", entry.Prefix)
		}
		seen[entry.Prefix] = true

		if !entry.Retired {
			hasActive = true
		}
	}

	if !hasActive {
		return errors.New("at least one non-retired prefix is required")
	}

	return nil
}

// ValidatePrefix checks that a prefix is a valid single non-digit Unicode rune.
func ValidatePrefix(prefix string) error {
	if prefix == "" {
		return errors.New("prefix cannot be empty")
	}

	runes := []rune(prefix)
	if len(runes) != 1 {
		return fmt.Errorf("prefix must be exactly one character, got %d", len(runes))
	}

	r := runes[0]
	if unicode.IsDigit(r) {
		return fmt.Errorf("prefix cannot be a digit: %q", prefix)
	}

	return nil
}

// IsValidPrefix returns true if the prefix is declared in the registry.
func (c *Config) IsValidPrefix(prefix string) bool {
	for _, entry := range c.Prefixes {
		if entry.Prefix == prefix {
			return true
		}
	}
	return false
}

// IsActivePrefix returns true if the prefix is declared and not retired.
func (c *Config) IsActivePrefix(prefix string) bool {
	for _, entry := range c.Prefixes {
		if entry.Prefix == prefix {
			return !entry.Retired
		}
	}
	return false
}

// GetPrefixEntry returns the prefix entry for the given prefix, or nil if not found.
func (c *Config) GetPrefixEntry(prefix string) *PrefixEntry {
	for i := range c.Prefixes {
		if c.Prefixes[i].Prefix == prefix {
			return &c.Prefixes[i]
		}
	}
	return nil
}

// ActivePrefixes returns all non-retired prefixes.
func (c *Config) ActivePrefixes() []PrefixEntry {
	var result []PrefixEntry
	for _, entry := range c.Prefixes {
		if !entry.Retired {
			result = append(result, entry)
		}
	}
	return result
}

// AddPrefix adds a new prefix to the registry.
func (c *Config) AddPrefix(prefix, name, description string) error {
	if err := ValidatePrefix(prefix); err != nil {
		return err
	}

	if c.IsValidPrefix(prefix) {
		return fmt.Errorf("prefix already exists: %q", prefix)
	}

	c.Prefixes = append(c.Prefixes, PrefixEntry{
		Prefix:      prefix,
		Name:        strings.TrimSpace(name),
		Description: strings.TrimSpace(description),
	})

	return nil
}

// RetirePrefix marks a prefix as retired.
func (c *Config) RetirePrefix(prefix string) error {
	entry := c.GetPrefixEntry(prefix)
	if entry == nil {
		return fmt.Errorf("prefix not found: %q", prefix)
	}

	if entry.Retired {
		return fmt.Errorf("prefix already retired: %q", prefix)
	}

	// Check if this would leave no active prefixes
	activeCount := 0
	for _, e := range c.Prefixes {
		if !e.Retired && e.Prefix != prefix {
			activeCount++
		}
	}

	if activeCount == 0 {
		return errors.New("cannot retire the last active prefix")
	}

	entry.Retired = true
	return nil
}

// NextPlanNumber returns the next available number for a given prefix.
// This is determined by scanning existing Plan IDs and finding the maximum.
// The planIDScanner function should return all existing Plan IDs.
func (c *Config) NextPlanNumber(prefix string, planIDScanner func() ([]string, error)) (int, error) {
	if !c.IsValidPrefix(prefix) {
		return 0, fmt.Errorf("unknown prefix: %q", prefix)
	}

	ids, err := planIDScanner()
	if err != nil {
		return 0, fmt.Errorf("scan plan IDs: %w", err)
	}

	maxNum := 0
	for _, id := range ids {
		p, numStr, _ := parsePlanIDParts(id)
		if p != prefix {
			continue
		}

		var num int
		if _, err := fmt.Sscanf(numStr, "%d", &num); err == nil {
			if num > maxNum {
				maxNum = num
			}
		}
	}

	return maxNum + 1, nil
}

// parsePlanIDParts extracts prefix, number, and slug from a Plan ID.
// This is a local helper that mirrors model.ParsePlanID to avoid import cycles.
// TODO: Consider extracting shared Plan ID parsing logic to a leaf package to eliminate duplication.
func parsePlanIDParts(id string) (prefix, number, slug string) {
	if len(id) < 4 {
		return "", "", ""
	}

	runes := []rune(id)
	if runes[0] >= '0' && runes[0] <= '9' {
		return "", "", ""
	}

	prefix = string(runes[0])
	digitEnd := 1
	for digitEnd < len(runes) && runes[digitEnd] >= '0' && runes[digitEnd] <= '9' {
		digitEnd++
	}

	if digitEnd == 1 || digitEnd >= len(runes) || runes[digitEnd] != '-' {
		return "", "", ""
	}

	number = string(runes[1:digitEnd])
	slug = string(runes[digitEnd+1:])

	return prefix, number, slug
}
