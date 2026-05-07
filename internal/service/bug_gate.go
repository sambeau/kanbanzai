package service

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sambeau/kanbanzai/internal/model"
)

// VerifierTimeout is the maximum duration a verifier sub-agent may run before
// the gate returns a timeout failure (NFR-401: 120s).
const VerifierTimeout = 120 * time.Second

// verifierDispatchTimes tracks when a verifier sub-agent was first dispatched
// for a given bug ID. Used by the verified→closed gate to detect timeouts.
var verifierDispatchTimes sync.Map // map[string]time.Time

// recordVerifierDispatch records that a verifier was dispatched for the given
// bug at the current time.
func recordVerifierDispatch(bugID string) {
	verifierDispatchTimes.Store(bugID, time.Now())
}

// isVerifierTimedOut checks whether a verifier dispatched for bugID has exceeded
// VerifierTimeout. Returns false if no dispatch was ever recorded.
func isVerifierTimedOut(bugID string) bool {
	v, ok := verifierDispatchTimes.Load(bugID)
	if !ok {
		return false
	}
	dispatchedAt := v.(time.Time)
	return time.Since(dispatchedAt) > VerifierTimeout
}

// ResetVerifierDispatchTime removes the recorded dispatch time for the given
// bug, allowing a fresh dispatch on the next gate check. Exported for tests.
func ResetVerifierDispatchTime(bugID string) {
	verifierDispatchTimes.Delete(bugID)
}

// DefaultBugMaxReviewCycles is the default review iteration cap for bugs.
// bug_fix tier uses this value; feature_equivalent tier doubles it.
const DefaultBugMaxReviewCycles = 2

// MaxReviewCyclesForTier returns the review cycle cap for a given bug tier.
// bug_fix: DefaultBugMaxReviewCycles (2)
// feature_equivalent: DefaultBugMaxReviewCycles * 2 (4)
func MaxReviewCyclesForTier(tier string) int {
	if tier == "feature_equivalent" {
		return DefaultBugMaxReviewCycles * 2
	}
	return DefaultBugMaxReviewCycles
}

// CheckBugTransitionGate checks the gate prerequisite for a specific (from, to)
// bug lifecycle transition. It returns a satisfied GateResult for ungated
// transitions and an unsatisfied GateResult when prerequisites are not met.
//
// Gate checks:
//   - needs-review→needs-rework: review cycle cap check
//   - verified→closed: verifier sub-agent dispatch (placeholder until P55 Component 7)
//   - All other transitions: ungated
func CheckBugTransitionGate(from, to string, bug *model.Bug, docSvc *DocumentService, entitySvc *EntityService) GateResult {
	transition := from + "→" + to
	switch transition {
	case string(model.BugStatusNeedsReview) + "→" + string(model.BugStatusNeedsRework):
		return checkBugReviewCycleCap(bug)

	case string(model.BugStatusVerified) + "→" + string(model.BugStatusClosed):
		return checkBugCloseOutVerification(bug, docSvc, entitySvc)

	default:
		// All other bug transitions are ungated.
		return GateResult{Satisfied: true}
	}
}

// checkBugReviewCycleCap checks whether the bug has exceeded its review
// iteration cap. When ReviewCycle >= MaxCycles, the gate blocks with
// ReviewCapReached=true.
func checkBugReviewCycleCap(bug *model.Bug) GateResult {
	maxCycles := MaxReviewCyclesForTier(bug.Tier)
	if bug.ReviewCycle >= maxCycles {
		return GateResult{
			Satisfied:        false,
			Reason:           fmt.Sprintf("Review iteration cap reached (%d/%d). Human decision required: accept with known issues, rework with revised scope, or close as not-planned.", bug.ReviewCycle, maxCycles),
			ReviewCapReached: true,
		}
	}
	return GateResult{Satisfied: true}
}

// checkBugCloseOutVerification handles the verified→closed gate.
//
// FR-413: Until P55 Component 7 is implemented (the verifier role file exists),
// the gate is a pass-through placeholder that logs "verifier not yet implemented".
//
// FR-414: When .kbz/roles/verifier.yaml exists, the gate switches to full
// verifier dispatch automatically — no code change required.
func checkBugCloseOutVerification(bug *model.Bug, docSvc *DocumentService, entitySvc *EntityService) GateResult {
	// FR-414: Auto-detect whether the verifier role file exists.
	// We check relative to the current working directory, which is the repo root
	// in the server process.
	verifierRolePath := filepath.Join(".kbz", "roles", "verifier.yaml")
	if _, err := os.Stat(verifierRolePath); os.IsNotExist(err) {
		// FR-413: Placeholder — verifier not yet implemented.
		log.Printf("[INFO] verifier not yet implemented — see F4 and P55 Component 7")
		return GateResult{
			Satisfied: true,
			Reason:    "verifier not yet implemented — see F4 and P55 Component 7",
		}
	}

	// FR-412: If a verifier was dispatched previously, check whether it has
	// exceeded VerifierTimeout. If so, return a timeout failure. If a verifier
	// is still running (dispatched but not yet timed out), signal that the
	// verifier is still needed without re-recording the dispatch time.
	if _, ok := verifierDispatchTimes.Load(bug.ID); ok {
		if isVerifierTimedOut(bug.ID) {
			log.Printf("[WARN] verifier sub-agent timed out for %s (limit: %v)", bug.ID, VerifierTimeout)
			return GateResult{
				Satisfied:        false,
				Reason:           fmt.Sprintf("verifier sub-agent timed out after %v — re-dispatch or investigate", VerifierTimeout),
				VerifierTimedOut: true,
			}
		}
		// Verifier still running — keep signalling NeedsVerifier without
		// recording a fresh dispatch time.
		return dispatchBugVerifier(bug, docSvc, entitySvc)
	}

	// FR-401 to FR-408: First-time verifier dispatch.
	// Record the dispatch time and signal the orchestrator to spawn the verifier.
	recordVerifierDispatch(bug.ID)
	return dispatchBugVerifier(bug, docSvc, entitySvc)
}

// dispatchBugVerifier generates a verifier dispatch prompt and returns a
// GateResult that signals the orchestrator to spawn the verifier sub-agent.
//
// The Prompt field in the GateResult carries the assembled handoff prompt
// for the orchestrator to pass to spawn_agent. The orchestrator is responsible
// for:
//  1. Spawning the verifier with the prompt
//  2. Collecting the structured JSON report
//  3. Calling parseAndApplyVerifierReport to get the final gate result
func dispatchBugVerifier(bug *model.Bug, _ *DocumentService, _ *EntityService) GateResult {
	prompt := buildVerifierPrompt(bug)

	return GateResult{
		Satisfied:        false,
		Reason:           "verifier dispatch required",
		VerifierPrompt:   prompt,
		NeedsVerifier:    true,
	}
}

// buildVerifierPrompt assembles the handoff prompt for the verifier sub-agent.
// It includes the bug ID, slug, the 8-item DoD checklist, and instructions
// to produce the structured JSON report defined in FR-405.
func buildVerifierPrompt(bug *model.Bug) string {
	return fmt.Sprintf(`# Bug Close-Out Verification

**Role:** verifier
**Skill:** verify-closeout
**Bug:** %s (%s)

## Definition of Done Checklist (8 items)

Execute each verification action independently. Do not trust entity state claims — re-run commands even when state suggests they should pass.
Produce the structured JSON report at the end. Do not converse or ask questions.

| # | DoD Item | Verification Action |
|---|----------|---------------------|
| 1 | Fix verified | Read the bug entity. Confirm verification field is populated and non-empty (minimum 10 characters). |
| 2 | Changes committed | Run 'git status --porcelain'. Confirm output is empty. |
| 3 | Temp files removed | Run 'git ls-files --others --exclude-standard'. Confirm no untracked files outside work/ and docs/. |
| 4 | Tests pass | Run 'go test ./...' on the worktree branch. Confirm exit code is 0. |
| 5 | Code reviewed | Call doc(action: "list", owner: "%s", type: "report"). Confirm at least one report document exists with status approved or draft. |
| 6 | Full lifecycle | Read the bug entity. Confirm current status is verified and the bug reached it via needs-review (no skipped stages). |
| 7 | Landed on main | If a worktree exists, run 'git merge-base --is-ancestor <branch> main'. Confirm exit code is 0. If no worktree, confirm git branch --contains HEAD includes main. |
| 8 | Worktree cleaned up | Run 'git worktree list'. Confirm no entry exists for this bug. Run 'git branch | grep %s'. Confirm no output. |

## Output Format

Produce exactly this JSON, wrapped in a markdown code block:

` + "```json" + `
{
  "bug_id": "%s",
  "checked_at": "<RFC 3339 timestamp>",
  "verdict": "pass" or "fail",
  "items": [
    {
      "dod_item": 1,
      "description": "Fix verified against expected behaviour",
      "verdict": "pass" or "fail",
      "evidence": "<output of verification action or failure reason>"
    },
    ...
  ]
}
` + "```" + `

## Instructions

1. Adopt the verifier role identity from .kbz/roles/verifier.yaml
2. Follow the verify-closeout skill procedure from .kbz/skills/verify-closeout/SKILL.md
3. Run each of the 8 verification actions independently
4. Every pass verdict must include concrete evidence (command output, entity field, document reference)
5. Every fail verdict must include the reason for failure
6. Write the report as a markdown file at work/reviews/verify-%s-%s.md
7. Register the report with doc(action: "register", path: "work/reviews/verify-%s-%s.md", type: "report", owner: "%s", title: "Close-Out Verification Report: %s")
8. Return the structured JSON as your response
`, bug.ID, bug.Slug,
		bug.ID,
		bug.ID,
		bug.ID,
		bug.ID, bug.Slug,
		bug.ID, bug.Slug, bug.ID, bug.Slug)
}

// VerifierReport is the structured JSON report returned by the verifier sub-agent.
// See FR-405 for the schema.
type VerifierReport struct {
	BugID     string              `json:"bug_id"`
	CheckedAt string              `json:"checked_at"`
	Verdict   string              `json:"verdict"` // "pass" or "fail"
	Items     []VerifierReportItem `json:"items"`
}

// VerifierReportItem is a single DoD item check result.
type VerifierReportItem struct {
	DoDItem     int    `json:"dod_item"`
	Description string `json:"description"`
	Verdict     string `json:"verdict"` // "pass" or "fail"
	Evidence    string `json:"evidence"`
}

// ParseVerifierReport parses a verifier's JSON report from a raw string.
// It handles the case where the JSON is wrapped in a markdown code block.
func ParseVerifierReport(raw string) (*VerifierReport, error) {
	// Strip markdown code block delimiters if present.
	// The verifier is instructed to wrap JSON in ```json ... ```
	jsonStr := raw
	// Simple heuristic: find the first '{' and last '}'
	start := 0
	end := len(jsonStr)
	for i, c := range jsonStr {
		if c == '{' {
			start = i
			break
		}
	}
	for i := len(jsonStr) - 1; i >= 0; i-- {
		if jsonStr[i] == '}' {
			end = i + 1
			break
		}
	}
	if start < end {
		jsonStr = jsonStr[start:end]
	}

	var report VerifierReport
	if err := json.Unmarshal([]byte(jsonStr), &report); err != nil {
		return nil, fmt.Errorf("parse verifier report: %w", err)
	}
	return &report, nil
}

// ApplyVerifierReport evaluates a parsed verifier report and returns the
// final GateResult for the verified→closed transition.
//
// FR-409: If verdict is "pass", the transition is allowed.
// If verdict is "fail", the transition is blocked with itemised failures.
func ApplyVerifierReport(report *VerifierReport) GateResult {
	if report.Verdict == "pass" {
		return GateResult{
			Satisfied: true,
			Reason:    "close-out verification passed: all 8 DoD items satisfied",
		}
	}

	// Build itemised failure reason (FR-410).
	var failures []string
	for _, item := range report.Items {
		if item.Verdict == "fail" {
			failures = append(failures, fmt.Sprintf("%d (%s)", item.DoDItem, item.Description))
		}
	}

	reason := fmt.Sprintf("close-out verification failed: DoD items %s — see verifier report for details",
		joinItems(failures))

	return GateResult{
		Satisfied: false,
		Reason:    reason,
	}
}

// joinItems joins item descriptions with commas.
func joinItems(items []string) string {
	if len(items) == 0 {
		return ""
	}
	result := items[0]
	for i := 1; i < len(items); i++ {
		result += ", " + items[i]
	}
	return result
}
