package actionlog

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// validCategories are the accepted scenario category values.
var validCategories = map[string]bool{
	"happy-path":                  true,
	"gate-failure-recovery":       true,
	"review-rework-loop":          true,
	"multi-feature-orchestration": true,
	"edge-case":                   true,
}

// Scenario describes a single evaluation scenario for live agent testing.
type Scenario struct {
	Name            string      `yaml:"name"`
	Description     string      `yaml:"description"`
	Category        string      `yaml:"category"`
	StartingState   interface{} `yaml:"starting_state"`
	ExpectedPattern interface{} `yaml:"expected_pattern"`
	SuccessCriteria []string    `yaml:"success_criteria"`
}

// LoadScenario reads a YAML scenario file and validates required fields.
func LoadScenario(path string) (*Scenario, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load scenario %s: %w", path, err)
	}

	var s Scenario
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse scenario %s: %w", path, err)
	}

	if err := validateScenario(&s, path); err != nil {
		return nil, err
	}

	return &s, nil
}

// validateScenario checks that all required fields are present and valid.
func validateScenario(s *Scenario, path string) error {
	if s.Name == "" {
		return fmt.Errorf("scenario %s: name is required", path)
	}
	if s.Description == "" {
		return fmt.Errorf("scenario %s: description is required", path)
	}
	if s.Category == "" {
		return fmt.Errorf("scenario %s: category is required", path)
	}
	if !validCategories[s.Category] {
		return fmt.Errorf("scenario %s: unknown category %q; valid: happy-path, gate-failure-recovery, review-rework-loop, multi-feature-orchestration, edge-case", path, s.Category)
	}
	if len(s.SuccessCriteria) == 0 {
		return fmt.Errorf("scenario %s: success_criteria must not be empty", path)
	}
	return nil
}
