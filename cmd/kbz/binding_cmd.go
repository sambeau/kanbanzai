package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/sambeau/kanbanzai/internal/binding"
	"github.com/sambeau/kanbanzai/internal/core"
)

const bindingDoctorUsageText = `Usage: kbz binding doctor [--file <path>]

  Validates .kbz/stage-bindings.yaml against the binding schema and role
  file references. Reports all errors and warnings.

  Options:
    --file <path>  Path to the binding file (default: .kbz/stage-bindings.yaml)

  Exit codes:
    0 — validation passed (no errors)
    1 — validation errors found

  This is a diagnostic-only command — it does not start the server or
  modify any files.
`

func runBinding(args []string, deps dependencies) error {
	if len(args) == 0 {
		return fmt.Errorf("missing binding subcommand\n\n%s", bindingDoctorUsageText)
	}

	switch args[0] {
	case "doctor":
		return runBindingDoctor(args[1:], deps)
	default:
		return fmt.Errorf("unknown binding subcommand %q\n\n%s", args[0], bindingDoctorUsageText)
	}
}

func runBindingDoctor(args []string, deps dependencies) error {
	bindingPath := filepath.Join(core.InstanceRootDir, "stage-bindings.yaml")

	// Support --file override for testing.
	if len(args) >= 2 && args[0] == "--file" {
		bindingPath = args[1]
		args = args[2:]
	}

	bf, errs := binding.LoadBindingFile(bindingPath)
	if len(errs) > 0 {
		fmt.Fprintln(deps.stdout, "Errors loading binding file:")
		for _, e := range errs {
			fmt.Fprintf(deps.stdout, "  ERROR: %s\n", e)
		}
		return fmt.Errorf("binding file load failed")
	}

	// Build a role checker that verifies roles exist as files on disk.
	// Use the directory containing the binding file as the roles root.
	bindingDir := filepath.Dir(bindingPath)
	roleChecker := roleFileCheckerAt(bindingDir)

	result := binding.ValidateBindingFile(bf, roleChecker)

	printBindingDoctorResults(deps.stdout, result)

	if len(result.Errors) > 0 {
		return fmt.Errorf("binding file validation failed with %d error(s)", len(result.Errors))
	}
	return nil
}

// roleFileCheckerAt returns a binding.RoleChecker that checks whether a role
// ID has a corresponding YAML file in the roles/ subdirectory of kbzDir.
func roleFileCheckerAt(kbzDir string) binding.RoleChecker {
	rolesDir := filepath.Join(kbzDir, "roles")
	return func(id string) bool {
		_, err := os.Stat(filepath.Join(rolesDir, id+".yaml"))
		return err == nil
	}
}

func printBindingDoctorResults(w io.Writer, result *binding.ValidationResult) {
	if len(result.Errors) == 0 && len(result.Warnings) == 0 {
		fmt.Fprintln(w, "stage-bindings.yaml is valid — no errors or warnings.")
		return
	}

	if len(result.Errors) > 0 {
		fmt.Fprintf(w, "Validation errors: %d\n", len(result.Errors))
		for _, e := range result.Errors {
			fmt.Fprintf(w, "  ERROR: %s\n", e)
		}
		fmt.Fprintln(w)
	}

	if len(result.Warnings) > 0 {
		fmt.Fprintf(w, "Validation warnings: %d\n", len(result.Warnings))
		for _, warn := range result.Warnings {
			fmt.Fprintf(w, "  WARN:  %s\n", warn)
		}
	}

	if len(result.Errors) == 0 {
		fmt.Fprintln(w, "\nstage-bindings.yaml passed validation (warnings only).")
	} else {
		fmt.Fprintln(w, "\nRun 'kbz init --upgrade' to restore a known-good binding file if this is a consumer install.")
	}
}
