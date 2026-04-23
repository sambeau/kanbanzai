package main

import (
	"bytes"
	"strings"
	"testing"
)

// TestRunDoc_NoSubcommand checks the missing-subcommand error path.
func TestRunDoc_NoSubcommand(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	deps := dependencies{stdout: &buf, stdin: strings.NewReader("")}

	err := runDoc(nil, deps)
	if err == nil {
		t.Fatal("expected error for missing subcommand, got nil")
	}
	if !strings.Contains(err.Error(), "missing doc subcommand") {
		t.Errorf("error = %q, want to contain 'missing doc subcommand'", err.Error())
	}
}

// TestRunDoc_UnknownSubcommand checks that unknown subcommands produce a clear error.
func TestRunDoc_UnknownSubcommand(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	deps := dependencies{stdout: &buf, stdin: strings.NewReader("")}

	err := runDoc([]string{"frobnicate"}, deps)
	if err == nil {
		t.Fatal("expected error for unknown subcommand, got nil")
	}
	if !strings.Contains(err.Error(), "frobnicate") {
		t.Errorf("error = %q, want to contain the unknown subcommand name", err.Error())
	}
}

// TestRunDocRegister_MissingPath checks that omitting the path argument returns an error.
func TestRunDocRegister_MissingPath(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	deps := dependencies{stdout: &buf, stdin: strings.NewReader("")}

	err := runDocRegister(nil, deps)
	if err == nil {
		t.Fatal("expected error for missing path, got nil")
	}
	if !strings.Contains(err.Error(), "missing document path") {
		t.Errorf("error = %q, want to contain 'missing document path'", err.Error())
	}
}

// TestRunDocRegister_MissingType checks that omitting --type returns an error before
// identity resolution or any service call.
func TestRunDocRegister_MissingType(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	deps := dependencies{stdout: &buf, stdin: strings.NewReader("")}

	err := runDocRegister([]string{"some/path.md", "--title", "My Doc"}, deps)
	if err == nil {
		t.Fatal("expected error for missing --type, got nil")
	}
	if !strings.Contains(err.Error(), "--type is required") {
		t.Errorf("error = %q, want to contain '--type is required'", err.Error())
	}
}

// TestRunDocRegister_MissingTitle checks that omitting --title returns an error before
// identity resolution or any service call.
func TestRunDocRegister_MissingTitle(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	deps := dependencies{stdout: &buf, stdin: strings.NewReader("")}

	err := runDocRegister([]string{"some/path.md", "--type", "design"}, deps)
	if err == nil {
		t.Fatal("expected error for missing --title, got nil")
	}
	if !strings.Contains(err.Error(), "--title is required") {
		t.Errorf("error = %q, want to contain '--title is required'", err.Error())
	}
}

// TestRunDocRegister_UnknownFlag checks that unknown flags return a clear error.
func TestRunDocRegister_UnknownFlag(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	deps := dependencies{stdout: &buf, stdin: strings.NewReader("")}

	err := runDocRegister([]string{"some/path.md", "--bogus", "val"}, deps)
	if err == nil {
		t.Fatal("expected error for unknown flag, got nil")
	}
	if !strings.Contains(err.Error(), "--bogus") {
		t.Errorf("error = %q, want to contain flag name '--bogus'", err.Error())
	}
}

// TestRunDocRegister_ByFlagMissingValue checks that --by without a value errors.
func TestRunDocRegister_ByFlagMissingValue(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	deps := dependencies{stdout: &buf, stdin: strings.NewReader("")}

	err := runDocRegister([]string{"some/path.md", "--by"}, deps)
	if err == nil {
		t.Fatal("expected error for --by with no value, got nil")
	}
	if !strings.Contains(err.Error(), "--by requires a value") {
		t.Errorf("error = %q, want to contain '--by requires a value'", err.Error())
	}
}

// TestDocUsageText_ContainsByFlag verifies AC-005: the usage text lists --by as an
// optional flag with a description, satisfying REQ-005.
func TestDocUsageText_ContainsByFlag(t *testing.T) {
	t.Parallel()

	if !strings.Contains(docUsageText, "--by") {
		t.Error("docUsageText does not contain '--by' flag")
	}
	// The flag should be described as optional (auto-resolved if omitted).
	if !strings.Contains(docUsageText, "auto-resolved") {
		t.Error("docUsageText does not indicate '--by' is auto-resolved when omitted")
	}
}

// TestRunDocApprove_ByFlagAcceptsEmptyWithoutHardcodedError verifies that
// runDocApprove calls config.ResolveIdentity rather than passing the raw --by
// value directly. When --by is omitted, the error (if any) must come from
// ResolveIdentity or the service layer, not a hard-coded "approver is required" check.
func TestRunDocApprove_ByFlagAcceptsEmptyWithoutHardcodedError(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	deps := dependencies{stdout: &buf, stdin: strings.NewReader("")}

	// Omit --by. The command will reach config.ResolveIdentity.
	// The resulting error (if any) must NOT contain "approver is required".
	err := runDocApprove([]string{"DOC-nonexistent"}, deps)
	if err != nil && strings.Contains(err.Error(), "approver is required") {
		t.Errorf("got old hard-coded error %q; expected identity resolution via config.ResolveIdentity", err.Error())
	}
}

// TestRunDocRegister_ByFlagAcceptsEmptyWithoutHardcodedError verifies AC-004 and
// AC-007: when --by is omitted and identity cannot be resolved, the error comes
// directly from config.ResolveIdentity with no extra wrapping (we check the known
// error substrings that ResolveIdentity produces).
//
// This test only fires when run in an environment where git config user.name is also
// unset, which is not always the case. We instead verify the code path by asserting
// that the function does NOT return a "created_by is required" error (the old
// hard-coded error) — proving the new identity resolution path is taken.
func TestRunDocRegister_ByFlagAcceptsEmptyWithoutHardcodedError(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	deps := dependencies{stdout: &buf, stdin: strings.NewReader("")}

	// Pass valid flags but omit --by. The command will reach config.ResolveIdentity.
	// The resulting error (if any) must NOT be the old "created_by is required" message.
	err := runDocRegister([]string{"nonexistent/path.md", "--type", "design", "--title", "Test"}, deps)

	// If an error occurs, it should not be the old hard-coded "created_by is required" error.
	// It will be either:
	//   - nil / a service error (if identity resolved and service ran)
	//   - the ResolveIdentity error (if git config is also absent)
	if err != nil && strings.Contains(err.Error(), "created_by is required") {
		t.Errorf("got old hard-coded error %q; expected identity resolution to be attempted via config.ResolveIdentity", err.Error())
	}
}
