package main

import (
	"fmt"
	"strings"
	"time"

	"kanbanzai/internal/checkpoint"
	"kanbanzai/internal/core"
	"kanbanzai/internal/id"
)

const checkpointUsageText = `Usage: kbz checkpoint <subcommand> [options]

Subcommands:
  create      Create a new human checkpoint
    --question    <text>   The decision or question requiring human input (required)
    --context     <text>   Background information to help the human answer (required)
    --summary     <text>   Brief state of the orchestration session (required)
    --by          <user>   Identity of the creating agent

  respond <id> <response>
              Record a response to a pending checkpoint

  list        List all checkpoints
    --status  pending|responded   Filter by status
`

func runCheckpoint(args []string, deps dependencies) error {
	if len(args) == 0 {
		return fmt.Errorf("missing checkpoint subcommand\n\n%s", checkpointUsageText)
	}

	switch args[0] {
	case "create":
		return runCheckpointCreate(args[1:], deps)
	case "respond":
		return runCheckpointRespond(args[1:], deps)
	case "list":
		return runCheckpointList(args[1:], deps)
	default:
		return fmt.Errorf("unknown checkpoint subcommand %q\n\n%s", args[0], checkpointUsageText)
	}
}

// ─── create ──────────────────────────────────────────────────────────────────

func runCheckpointCreate(args []string, deps dependencies) error {
	var question, ctx, summary, createdBy string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--question":
			if i+1 >= len(args) {
				return fmt.Errorf("--question requires a value")
			}
			i++
			question = args[i]
		case "--context":
			if i+1 >= len(args) {
				return fmt.Errorf("--context requires a value")
			}
			i++
			ctx = args[i]
		case "--summary":
			if i+1 >= len(args) {
				return fmt.Errorf("--summary requires a value")
			}
			i++
			summary = args[i]
		case "--by":
			if i+1 >= len(args) {
				return fmt.Errorf("--by requires a value")
			}
			i++
			createdBy = args[i]
		default:
			return fmt.Errorf("unknown flag %q\n\nUsage: kbz checkpoint create [flags]", args[i])
		}
	}

	if question == "" {
		return fmt.Errorf("--question is required")
	}
	if ctx == "" {
		return fmt.Errorf("--context is required")
	}
	if summary == "" {
		return fmt.Errorf("--summary is required")
	}
	if createdBy == "" {
		createdBy = "cli"
	}

	stateRoot := core.StatePath()
	store := checkpoint.NewStore(stateRoot)

	record := checkpoint.Record{
		Question:             question,
		Context:              ctx,
		OrchestrationSummary: summary,
		Status:               checkpoint.StatusPending,
		CreatedAt:            time.Now().UTC(),
		CreatedBy:            createdBy,
	}

	created, err := store.Create(record)
	if err != nil {
		return fmt.Errorf("create checkpoint: %w", err)
	}

	_, err = fmt.Fprintf(deps.stdout, "created checkpoint\nid: %s\nstatus: %s\n",
		id.FormatFullDisplay(created.ID), created.Status)
	return err
}

// ─── respond ─────────────────────────────────────────────────────────────────

func runCheckpointRespond(args []string, deps dependencies) error {
	if len(args) < 2 {
		return fmt.Errorf("missing checkpoint ID or response\n\nUsage: kbz checkpoint respond <id> <response>")
	}

	checkpointID := args[0]
	response := strings.Join(args[1:], " ")

	stateRoot := core.StatePath()
	store := checkpoint.NewStore(stateRoot)

	record, err := store.Get(checkpointID)
	if err != nil {
		return fmt.Errorf("checkpoint not found: %s", checkpointID)
	}

	now := time.Now().UTC()
	record.Status = checkpoint.StatusResponded
	record.RespondedAt = &now
	record.Response = &response

	updated, err := store.Update(record)
	if err != nil {
		return fmt.Errorf("update checkpoint: %w", err)
	}

	_, err = fmt.Fprintf(deps.stdout, "responded to checkpoint\nid: %s\nstatus: %s\n",
		id.FormatFullDisplay(updated.ID), updated.Status)
	return err
}

// ─── list ────────────────────────────────────────────────────────────────────

func runCheckpointList(args []string, deps dependencies) error {
	var statusFilter string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--status":
			if i+1 >= len(args) {
				return fmt.Errorf("--status requires a value")
			}
			i++
			statusFilter = args[i]
		default:
			return fmt.Errorf("unknown flag %q\n\nUsage: kbz checkpoint list [--status pending|responded]", args[i])
		}
	}

	stateRoot := core.StatePath()
	store := checkpoint.NewStore(stateRoot)

	records, err := store.List(statusFilter)
	if err != nil {
		return fmt.Errorf("list checkpoints: %w", err)
	}

	_, err = fmt.Fprintf(deps.stdout, "listed checkpoints (%d)\n", len(records))
	if err != nil {
		return err
	}

	for _, r := range records {
		q := r.Question
		if len(q) > 60 {
			q = q[:57] + "..."
		}
		_, err = fmt.Fprintf(deps.stdout, "%s\t%s\t%s\n",
			id.FormatFullDisplay(r.ID), r.Status, q)
		if err != nil {
			return err
		}
	}

	return nil
}
