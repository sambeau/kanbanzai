package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/sambeau/kanbanzai/internal/teststatus"
)

const testUsageText = `kanbanzai test <subcommand> [flags]

Manage test suite status tracking.

Subcommands:
  status     Print the current test status record as JSON
  record     Write a test status record from an exit code and output
  run        Run 'go test ./...' and record the result
  verify     Check staleness; re-run if stale or last result was fail/unknown
  force-fail Write a manual failure record

Examples:
  kbz test status
  kbz test record --result=0
  kbz test record --result=1 --output="FAIL TestFoo"
  kbz test run
  kbz test verify
  kbz test force-fail --summary="intentional fail for review gate"
`

func runTest(args []string, deps dependencies) error {
	if len(args) == 0 || wantsHelp(args) {
		fmt.Fprint(deps.stdout, testUsageText)
		return nil
	}

	switch args[0] {
	case "status":
		return runTestStatus(args[1:], deps)
	case "record":
		return runTestRecord(args[1:], deps)
	case "run":
		return runTestRun(args[1:], deps)
	case "verify":
		return runTestVerify(args[1:], deps)
	case "force-fail":
		return runTestForceFail(args[1:], deps)
	default:
		return fmt.Errorf("unknown test subcommand %q\n\n%s", args[0], testUsageText)
	}
}

// ─── status ──────────────────────────────────────────────────────────────────

func runTestStatus(args []string, deps dependencies) error {
	repoRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	rec, err := teststatus.ReadRecord(repoRoot)
	if err != nil {
		return fmt.Errorf("read test status: %w", err)
	}

	enc := json.NewEncoder(deps.stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(rec); err != nil {
		return fmt.Errorf("encode JSON: %w", err)
	}
	return nil
}

// ─── record ──────────────────────────────────────────────────────────────────

func runTestRecord(args []string, deps dependencies) error {
	flags, err := parseFlags(args)
	if err != nil {
		return fmt.Errorf("parse flags: %w\n\n%s", err, testUsageText)
	}

	resultStr := flags["result"]
	if resultStr == "" {
		return fmt.Errorf("--result is required\n\n%s", testUsageText)
	}

	output := flags["output"]

	repoRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	rec := buildRecord(repoRoot, resultStr, output, "cli")
	if err := teststatus.WriteRecord(repoRoot, rec); err != nil {
		return fmt.Errorf("write test status: %w", err)
	}

	fmt.Fprintf(deps.stdout, "recorded test status: %s\n", rec.Result)
	return nil
}

// ─── run ─────────────────────────────────────────────────────────────────────

func runTestRun(args []string, deps dependencies) error {
	fmt.Fprintf(deps.stdout, "running go test ./...\n")

	repoRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = repoRoot
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err = cmd.Run()
	elapsed := time.Since(start)

	// Combine output
	var outputBuf bytes.Buffer
	if stdout.Len() > 0 {
		outputBuf.WriteString(stdout.String())
	}
	if stderr.Len() > 0 {
		if stdout.Len() > 0 {
			outputBuf.WriteString("\n")
		}
		outputBuf.WriteString(stderr.String())
	}
	output := outputBuf.String()

	resultStr := "0"
	summary := fmt.Sprintf("all tests passed in %s", elapsed.Round(time.Second))
	if err != nil {
		resultStr = "1"
		summary = fmt.Sprintf("tests failed in %s", elapsed.Round(time.Second))
	}

	rec := buildRecord(repoRoot, resultStr, output, "go-test")
	rec.Summary = summary

	if err := teststatus.WriteRecord(repoRoot, rec); err != nil {
		return fmt.Errorf("write test status: %w", err)
	}

	fmt.Fprintf(deps.stdout, "\nrecorded test status: %s (%s)\n", rec.Result, elapsed.Round(time.Second))

	if err != nil {
		return fmt.Errorf("go test failed")
	}
	return nil
}

// ─── verify ──────────────────────────────────────────────────────────────────

func runTestVerify(args []string, deps dependencies) error {
	repoRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	rec, err := teststatus.ReadRecord(repoRoot)
	if err != nil {
		return fmt.Errorf("read test status: %w", err)
	}

	if rec.Result == teststatus.ResultUnknown {
		fmt.Fprintf(deps.stdout, "no previous test record found — running tests\n")
		return runTestRun(nil, deps)
	}

	stale, err := teststatus.IsStale(repoRoot, rec)
	if err != nil {
		return fmt.Errorf("check staleness: %w", err)
	}

	if stale {
		fmt.Fprintf(deps.stdout, "test status is stale — re-running\n")
		return runTestRun(nil, deps)
	}

	if rec.Result == teststatus.ResultFail {
		fmt.Fprintf(deps.stdout, "last test result was %s — re-running\n", rec.Result)
		return runTestRun(nil, deps)
	}

	fmt.Fprintf(deps.stdout, "test status is current and passing\n")
	enc := json.NewEncoder(deps.stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(rec); err != nil {
		return fmt.Errorf("encode JSON: %w", err)
	}
	return nil
}

// ─── force-fail ──────────────────────────────────────────────────────────────

func runTestForceFail(args []string, deps dependencies) error {
	flags, err := parseFlags(args)
	if err != nil {
		return fmt.Errorf("parse flags: %w\n\n%s", err, testUsageText)
	}

	summary := flags["summary"]
	if summary == "" {
		return fmt.Errorf("--summary is required\n\n%s", testUsageText)
	}

	repoRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	now := time.Now().UTC()
	rec := teststatus.Record{
		LastRun: &now,
		Result:  teststatus.ResultFail,
		Summary: summary,
		Runner:  "cli",
		Trigger: "force-fail",
	}

	if err := teststatus.WriteRecord(repoRoot, rec); err != nil {
		return fmt.Errorf("write test status: %w", err)
	}

	fmt.Fprintf(deps.stdout, "recorded forced test failure: %s\n", summary)
	return nil
}

// ─── helpers ─────────────────────────────────────────────────────────────────

// buildRecord constructs a teststatus.Record from a result code string (exit
// code), optional test output, and trigger name. It parses the output for
// failing test names and packages.
func buildRecord(repoRoot, resultStr, output, trigger string) teststatus.Record {
	now := time.Now().UTC()
	rec := teststatus.Record{
		LastRun: &now,
		Runner:  "cli",
		Trigger: trigger,
	}

	switch resultStr {
	case "0":
		rec.Result = teststatus.ResultPass
		rec.Summary = "all tests passed"
	default:
		rec.Result = teststatus.ResultFail
		rec.Summary = fmt.Sprintf("exit code %s", resultStr)
		if output != "" {
			// Truncate summary to first meaningful line
			lines := strings.Split(strings.TrimSpace(output), "\n")
			for _, line := range lines {
				trimmed := strings.TrimSpace(line)
				if trimmed != "" {
					rec.Summary = trimmed
					break
				}
			}
		}
		rec.Failures = parseFailures(output)
	}

	return rec
}

// failLineRE matches lines like:
//
//	--- FAIL: TestName (0.00s)
//	    package_test.go:123: message
//	FAIL    package/path [build failed]
//	FAIL
var (
	failHeaderRE = regexp.MustCompile(`^\s*---\s+FAIL:\s+(Test\S+)`)
	failPackageRE = regexp.MustCompile(`^FAIL\s+(\S+)`)
	failBuildRE   = regexp.MustCompile(`^FAIL\s+(\S+)\s+\[build failed\]`)
)

// parseFailures extracts Failure entries from go test output.
func parseFailures(output string) []teststatus.Failure {
	if output == "" {
		return nil
	}

	var failures []teststatus.Failure
	var currentPkg string
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		// Track the current package from "ok" lines so failures know their package.
		if strings.HasPrefix(line, "ok  ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				currentPkg = parts[1]
			}
		}
		if strings.HasPrefix(line, "?   ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				currentPkg = parts[1]
			}
		}

		// --- FAIL: TestName
		if m := failHeaderRE.FindStringSubmatch(line); m != nil {
			failures = append(failures, teststatus.Failure{
				Package: currentPkg,
				Test:    m[1],
				Message: strings.TrimSpace(line),
			})
		}

		// FAIL    package/path [build failed]
		if m := failBuildRE.FindStringSubmatch(line); m != nil {
			failures = append(failures, teststatus.Failure{
				Package: m[1],
				Test:    "",
				Message: "build failed",
			})
		}

		// FAIL    package/path
		if m := failPackageRE.FindStringSubmatch(line); m != nil {
			failures = append(failures, teststatus.Failure{
				Package: m[1],
				Test:    "",
				Message: "test failed",
			})
		}
	}

	// Deduplicate: keep only the first occurrence of each Package+Test combination.
	seen := make(map[string]bool)
	var unique []teststatus.Failure
	for _, f := range failures {
		key := filepath.Join(f.Package, f.Test)
		if seen[key] {
			continue
		}
		seen[key] = true
		unique = append(unique, f)
	}

	return unique
}
