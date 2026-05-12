package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sambeau/kanbanzai/internal/teststatus"
)

// ─── Test test subcommand dispatch ───────────────────────────────────────────

func TestTestCLIDispatch(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := run([]string{"test"}, makeTestDeps(&buf, ""))
	if err != nil && strings.Contains(err.Error(), "unknown command") {
		t.Errorf("test not dispatched; got unknown command error: %v", err)
	}
}

// ─── Test test status ────────────────────────────────────────────────────────

func TestTestStatusNoFile(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(origWd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := runTestStatus(nil, makeTestDeps(&buf, "")); err != nil {
		t.Fatalf("runTestStatus: %v", err)
	}

	var rec teststatus.Record
	if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
		t.Fatalf("unmarshal JSON: %v\noutput: %s", err, buf.String())
	}
	if rec.Result != teststatus.ResultUnknown {
		t.Errorf("Result: got %q, want %q", rec.Result, teststatus.ResultUnknown)
	}
}

func TestTestStatusWithRecord(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(origWd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	// Write a record first
	now := time.Now().Truncate(time.Second)
	orig := teststatus.Record{
		LastRun: &now,
		Result:  teststatus.ResultPass,
		Summary: "all tests passed",
	}
	if err := teststatus.WriteRecord(dir, orig); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := runTestStatus(nil, makeTestDeps(&buf, "")); err != nil {
		t.Fatalf("runTestStatus: %v", err)
	}

	var rec teststatus.Record
	if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
		t.Fatalf("unmarshal JSON: %v\noutput: %s", err, buf.String())
	}
	if rec.Result != teststatus.ResultPass {
		t.Errorf("Result: got %q, want %q", rec.Result, teststatus.ResultPass)
	}
	if rec.Summary != "all tests passed" {
		t.Errorf("Summary: got %q, want %q", rec.Summary, "all tests passed")
	}
}

// ─── Test test record ────────────────────────────────────────────────────────

func TestTestRecordPass(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(origWd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	err := runTestRecord([]string{"--result=0"}, makeTestDeps(&buf, ""))
	if err != nil {
		t.Fatalf("runTestRecord: %v", err)
	}
	if !strings.Contains(buf.String(), "pass") {
		t.Errorf("output %q does not contain pass", buf.String())
	}

	rec, err := teststatus.ReadRecord(dir)
	if err != nil {
		t.Fatalf("ReadRecord: %v", err)
	}
	if rec.Result != teststatus.ResultPass {
		t.Errorf("Result: got %q, want %q", rec.Result, teststatus.ResultPass)
	}
}

func TestTestRecordFail(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(origWd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	output := "--- FAIL: TestFoo (0.01s)\n    foo_test.go:10: expected 42, got 0\nFAIL\n"
	var buf bytes.Buffer
	err := runTestRecord([]string{"--result=1", "--output=" + output}, makeTestDeps(&buf, ""))
	if err != nil {
		t.Fatalf("runTestRecord: %v", err)
	}

	rec, err := teststatus.ReadRecord(dir)
	if err != nil {
		t.Fatalf("ReadRecord: %v", err)
	}
	if rec.Result != teststatus.ResultFail {
		t.Errorf("Result: got %q, want %q", rec.Result, teststatus.ResultFail)
	}
	if len(rec.Failures) == 0 {
		t.Fatal("expected failures to be parsed")
	}
	if rec.Failures[0].Test != "TestFoo" {
		t.Errorf("First failure test: got %q, want %q", rec.Failures[0].Test, "TestFoo")
	}
}

func TestTestRecordNoFlagsError(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := runTestRecord(nil, makeTestDeps(&buf, ""))
	if err == nil {
		t.Fatal("expected error for missing --result flag")
	}
	if !strings.Contains(err.Error(), "--result") {
		t.Errorf("error %q does not mention --result", err.Error())
	}
}

// ─── Test test force-fail ────────────────────────────────────────────────────

func TestTestForceFail(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(origWd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	err := runTestForceFail([]string{"--summary=manual override for review gate"}, makeTestDeps(&buf, ""))
	if err != nil {
		t.Fatalf("runTestForceFail: %v", err)
	}
	if !strings.Contains(buf.String(), "forced test failure") {
		t.Errorf("output %q does not mention forced failure", buf.String())
	}

	rec, err := teststatus.ReadRecord(dir)
	if err != nil {
		t.Fatalf("ReadRecord: %v", err)
	}
	if rec.Result != teststatus.ResultFail {
		t.Errorf("Result: got %q, want %q", rec.Result, teststatus.ResultFail)
	}
	if rec.Summary != "manual override for review gate" {
		t.Errorf("Summary: got %q, want %q", rec.Summary, "manual override for review gate")
	}
	if rec.Trigger != "force-fail" {
		t.Errorf("Trigger: got %q, want %q", rec.Trigger, "force-fail")
	}
}

func TestTestForceFailNoSummary(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := runTestForceFail(nil, makeTestDeps(&buf, ""))
	if err == nil {
		t.Fatal("expected error for missing --summary flag")
	}
	if !strings.Contains(err.Error(), "--summary") {
		t.Errorf("error %q does not mention --summary", err.Error())
	}
}

// ─── Test parseFailures ──────────────────────────────────────────────────────

func TestParseFailures_Empty(t *testing.T) {
	t.Parallel()
	failures := parseFailures("")
	if len(failures) != 0 {
		t.Errorf("expected 0 failures, got %d", len(failures))
	}
}

func TestParseFailures_NoFailures(t *testing.T) {
	t.Parallel()
	output := "ok  github.com/example/pkg 0.123s\n?   github.com/example/cmd [no test files]\n"
	failures := parseFailures(output)
	if len(failures) != 0 {
		t.Errorf("expected 0 failures, got %d: %+v", len(failures), failures)
	}
}

func TestParseFailures_HeaderOnly(t *testing.T) {
	t.Parallel()
	output := `--- FAIL: TestBar (0.01s)
    bar_test.go:15: something went wrong
FAIL
`
	failures := parseFailures(output)
	if len(failures) == 0 {
		t.Fatal("expected at least 1 failure")
	}
	if failures[0].Test != "TestBar" {
		t.Errorf("Test: got %q, want %q", failures[0].Test, "TestBar")
	}
}

func TestParseFailures_BuildFailure(t *testing.T) {
	t.Parallel()
	output := `# github.com/example/pkg
pkg.go:10:2: undefined: x
FAIL    github.com/example/pkg [build failed]
`
	failures := parseFailures(output)
	if len(failures) == 0 {
		t.Fatal("expected at least 1 failure")
	}
	if failures[0].Package != "github.com/example/pkg" {
		t.Errorf("Package: got %q, want %q", failures[0].Package, "github.com/example/pkg")
	}
}

func TestParseFailures_BasicFailLine(t *testing.T) {
	t.Parallel()
	output := `ok  	github.com/example/pkg1	0.123s
--- FAIL: TestSomething (0.01s)
    something_test.go:10: expected true, got false
FAIL
FAIL	github.com/example/pkg1	0.234s
`
	failures := parseFailures(output)
	if len(failures) == 0 {
		t.Fatal("expected failures")
	}
	found := false
	for _, f := range failures {
		if f.Test == "TestSomething" {
			found = true
			if f.Package != "github.com/example/pkg1" {
				t.Errorf("Package: got %q, want %q", f.Package, "github.com/example/pkg1")
			}
			break
		}
	}
	if !found {
		t.Errorf("TestSomething not found in failures: %+v", failures)
	}
}

// ─── Test buildRecord ────────────────────────────────────────────────────────

func TestBuildRecord_Pass(t *testing.T) {
	t.Parallel()
	rec := buildRecord(".", "0", "", "cli")
	if rec.Result != teststatus.ResultPass {
		t.Errorf("Result: got %q, want %q", rec.Result, teststatus.ResultPass)
	}
	if rec.LastRun == nil {
		t.Error("LastRun is nil")
	}
}

func TestBuildRecord_Fail(t *testing.T) {
	t.Parallel()
	rec := buildRecord(".", "1", "--- FAIL: TestX\nFAIL\n", "cli")
	if rec.Result != teststatus.ResultFail {
		t.Errorf("Result: got %q, want %q", rec.Result, teststatus.ResultFail)
	}
	if len(rec.Failures) == 0 {
		t.Fatal("expected failures to be parsed")
	}
}

func TestBuildRecord_FailNoOutput(t *testing.T) {
	t.Parallel()
	rec := buildRecord(".", "2", "", "cli")
	if rec.Result != teststatus.ResultFail {
		t.Errorf("Result: got %q, want %q", rec.Result, teststatus.ResultFail)
	}
	if rec.Summary != "exit code 2" {
		t.Errorf("Summary: got %q, want %q", rec.Summary, "exit code 2")
	}
}

// ─── Test wiring in main.go ──────────────────────────────────────────────────

func TestTestCommandDispatched(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := run([]string{"test"}, makeTestDeps(&buf, ""))
	if err != nil {
		if strings.Contains(err.Error(), "unknown command") {
			t.Errorf("'test' not dispatched: %v", err)
		}
		// Other errors (e.g. usage) are fine — just means subcommand routing works.
	}
}

// ─── Test verify behaviors ───────────────────────────────────────────────────

func TestTestVerifyNoRecord(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(origWd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	// Without a record, verify should try to run tests. Since this is a temp dir
	// with no Go module, it will fail — but we just verify the run was attempted.
	var buf bytes.Buffer
	err := runTestVerify(nil, makeTestDeps(&buf, ""))
	if err == nil {
		// If there's no error, verify succeeded (unlikely in empty dir, but acceptable)
		return
	}
	// Error should reference running tests or go test
	if !strings.Contains(err.Error(), "go test") && !strings.Contains(buf.String(), "running") {
		t.Errorf("expected test run attempt; output: %s, err: %v", buf.String(), err)
	}
}

func TestTestVerifyCurrentPassing(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(origWd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	// Create a go file with old timestamp to avoid staleness
	goFile := filepath.Join(dir, "dummy.go")
	if err := os.WriteFile(goFile, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Write a passing record with a recent last run
	now := time.Now()
	rec := teststatus.Record{
		LastRun: &now,
		Result:  teststatus.ResultPass,
		Summary: "all tests passed",
	}
	if err := teststatus.WriteRecord(dir, rec); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	err := runTestVerify(nil, makeTestDeps(&buf, ""))
	if err != nil {
		t.Fatalf("runTestVerify: %v", err)
	}

	if !strings.Contains(buf.String(), "current and passing") {
		t.Errorf("output %q does not mention current and passing", buf.String())
	}
}

// ─── Test JSON output format ─────────────────────────────────────────────────

func TestTestStatusJSONFormat(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(origWd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	now := time.Now().Truncate(time.Second)
	orig := teststatus.Record{
		LastRun: &now,
		Result:  teststatus.ResultFail,
		Summary: "2 tests failed",
		Failures: []teststatus.Failure{
			{Package: "github.com/example/pkg", Test: "TestBar", Message: "--- FAIL: TestBar"},
		},
		Runner:  "cli",
		Trigger: "record",
	}
	if err := teststatus.WriteRecord(dir, orig); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := runTestStatus(nil, makeTestDeps(&buf, "")); err != nil {
		t.Fatalf("runTestStatus: %v", err)
	}

	// Verify valid JSON
	var got teststatus.Record
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, buf.String())
	}

	// Verify all fields round-trip
	if got.Result != teststatus.ResultFail {
		t.Errorf("Result: got %q, want %q", got.Result, teststatus.ResultFail)
	}
	if got.Summary != "2 tests failed" {
		t.Errorf("Summary: got %q, want %q", got.Summary, "2 tests failed")
	}
	if got.Runner != "cli" {
		t.Errorf("Runner: got %q, want %q", got.Runner, "cli")
	}
	if got.Trigger != "record" {
		t.Errorf("Trigger: got %q, want %q", got.Trigger, "record")
	}
	if len(got.Failures) != 1 {
		t.Fatalf("len(Failures): got %d, want 1", len(got.Failures))
	}
	if got.Failures[0].Test != "TestBar" {
		t.Errorf("Failure.Test: got %q, want %q", got.Failures[0].Test, "TestBar")
	}
}
