package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunTask_NoSubcommand_PrintsHelp(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	deps := dependencies{
		stdout: &buf,
		stdin:  strings.NewReader(""),
	}

	err := runTask(nil, deps)
	if err != nil {
		t.Fatalf("runTask(nil) error = %v, want nil", err)
	}
	stdout := buf.String()
	if !strings.Contains(stdout, "task") || !strings.Contains(stdout, "review") {
		t.Fatalf("stdout missing task usage:\n%s", stdout)
	}
}

func TestRunTask_UnknownSubcommand(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	deps := dependencies{
		stdout: &buf,
		stdin:  strings.NewReader(""),
	}

	err := runTask([]string{"frobnicate"}, deps)
	if err == nil {
		t.Fatal("expected error for unknown subcommand, got nil")
	}
	if !strings.Contains(err.Error(), "unknown task subcommand") {
		t.Errorf("error = %q, want to contain %q", err.Error(), "unknown task subcommand")
	}
	if !strings.Contains(err.Error(), "frobnicate") {
		t.Errorf("error = %q, want to contain the unknown command name", err.Error())
	}
}

func TestRunTask_ReviewMissingID(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	deps := dependencies{
		stdout: &buf,
		stdin:  strings.NewReader(""),
	}

	err := runTaskReview(nil, deps)
	if err == nil {
		t.Fatal("expected error for missing task ID, got nil")
	}
	if !strings.Contains(err.Error(), "missing task ID") {
		t.Errorf("error = %q, want to contain %q", err.Error(), "missing task ID")
	}
}
