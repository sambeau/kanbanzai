// Package teststatus provides a persistent record of the most recent test suite
// run, stored as .kbz/state/test-status.yaml. It is used by AI agents and workflow
// tools to determine whether the test suite is healthy before starting new work.
package teststatus

import "time"

// Result represents the outcome of a test suite run.
type Result string

const (
	ResultPass    Result = "pass"
	ResultFail    Result = "fail"
	ResultUnknown Result = "unknown"
)

// Failure records a single failing test within a test suite run.
type Failure struct {
	Package string `yaml:"package"`
	Test    string `yaml:"test"`
	Message string `yaml:"message"`
}

// Record is the persisted test-status payload written to
// .kbz/state/test-status.yaml.
type Record struct {
	LastRun  *time.Time `yaml:"last_run"`
	Result   Result     `yaml:"result"`
	Summary  string     `yaml:"summary"`
	Failures []Failure  `yaml:"failures"`
	Runner   string     `yaml:"runner,omitempty"`
	Trigger  string     `yaml:"trigger,omitempty"`
}
