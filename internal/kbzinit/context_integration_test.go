//go:build integration

package kbzinit

import (
	"path/filepath"
	"testing"

	kbzcontext "github.com/sambeau/kanbanzai/internal/context"
)

// TestRun_ContextAssemble_ReviewerRole is an integration test verifying that
// context_assemble(role=reviewer) returns a non-empty packet after kbz init
// installs the reviewer role file. This exercises the full path from init →
// role file on disk → ProfileStore → Assemble.
//
// This test is in a separate file with the "integration" build tag because it
// transitively depends on internal/service (via internal/context), which may
// not compile during feature development when model fields are being renamed.
// Run with: go test -tags=integration ./internal/kbzinit/...
func TestRun_ContextAssemble_ReviewerRole(t *testing.T) {
	t.Parallel()
	dir := makeGitRepoNoCommits(t)

	in, _ := newTestInit(dir, "")
	if err := in.Run(Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Point the profile store at the roles directory created by kbz init.
	rolesDir := filepath.Join(dir, ".kbz", "context", "roles")
	profileStore := kbzcontext.NewProfileStore(rolesDir)

	result, err := kbzcontext.Assemble(
		kbzcontext.AssemblyInput{Role: "reviewer"},
		profileStore,
		nil, // knowledgeSvc — not needed for this test; Assemble handles nil safely
		nil, // entitySvc
		nil, // intelligenceSvc
	)
	if err != nil {
		t.Fatalf("Assemble(role=reviewer): %v", err)
	}
	if len(result.Items) == 0 {
		t.Error("Assemble(role=reviewer) returned an empty packet; expected at least one item (the role profile)")
	}
	if result.Role != "reviewer" {
		t.Errorf("result.Role = %q, want %q", result.Role, "reviewer")
	}
}
