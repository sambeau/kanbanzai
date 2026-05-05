package main

import (
	"context"
	"fmt"

	"github.com/sambeau/kanbanzai/internal/config"
	"github.com/sambeau/kanbanzai/internal/core"
	"github.com/sambeau/kanbanzai/internal/resolution"
	"github.com/sambeau/kanbanzai/internal/service"
)

const docUsageText = `Usage: kbz doc <subcommand> [options]

Subcommands:
  register <path>     Register a document with the system
    --type <type>     Document type: design, specification, dev-plan, research, report, policy
    --title <title>   Human-readable title
    --owner <id>      Optional parent Plan or Feature ID
    --by <user>       Creator identity (auto-resolved if omitted)

  approve <id|path>   Approve a document record by ID or file path
    --by <user>       Approver identity (auto-resolved if omitted)

  list                List all document records
    --type <type>     Filter by type
    --status <status> Filter by status: draft, approved, superseded
    --owner <id>      Filter by owner entity ID
`

func runDoc(args []string, deps dependencies) error {
	if len(args) == 0 {
		return fmt.Errorf("missing doc subcommand\n\n%s", docUsageText)
	}

	switch args[0] {
	case "register":
		return runDocRegister(args[1:], deps)
	case "approve":
		return runDocApprove(args[1:], deps)
	case "list":
		return runDocList(args[1:], deps)
	default:
		return fmt.Errorf("unknown doc subcommand %q\n\n%s", args[0], docUsageText)
	}
}

// ─── register ────────────────────────────────────────────────────────────────

func runDocRegister(args []string, deps dependencies) error {
	if len(args) == 0 {
		return fmt.Errorf("missing document path\n\nUsage: kbz doc register <path> [--type <type>] [--title <title>] [--owner <id>] [--by <user>]")
	}

	path := args[0]
	remaining := args[1:]

	var docType, title, owner, createdByRaw string

	for i := 0; i < len(remaining); i++ {
		switch remaining[i] {
		case "--type":
			if i+1 >= len(remaining) {
				return fmt.Errorf("--type requires a value")
			}
			i++
			docType = remaining[i]
		case "--title":
			if i+1 >= len(remaining) {
				return fmt.Errorf("--title requires a value")
			}
			i++
			title = remaining[i]
		case "--owner":
			if i+1 >= len(remaining) {
				return fmt.Errorf("--owner requires a value")
			}
			i++
			owner = remaining[i]
		case "--by":
			if i+1 >= len(remaining) {
				return fmt.Errorf("--by requires a value")
			}
			i++
			createdByRaw = remaining[i]
		default:
			return fmt.Errorf("unknown flag %q\n\nUsage: kbz doc register <path> [flags]", remaining[i])
		}
	}

	if docType == "" {
		return fmt.Errorf("--type is required\n\nUsage: kbz doc register <path> --type <type> --title <title>")
	}
	if title == "" {
		return fmt.Errorf("--title is required\n\nUsage: kbz doc register <path> --type <type> --title <title>")
	}

	createdBy, err := config.ResolveIdentity(createdByRaw)
	if err != nil {
		return err
	}

	stateRoot := core.StatePath()
	repoRoot := "."
	docSvc := service.NewDocumentService(stateRoot, repoRoot)

	result, err := docSvc.SubmitDocument(service.SubmitDocumentInput{
		Path:      path,
		Type:      docType,
		Title:     title,
		Owner:     owner,
		CreatedBy: createdBy,
	})
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(deps.stdout, "registered document\nid: %s\ntitle: %s\npath: %s\nstatus: %s\n",
		result.ID, result.Title, result.Path, result.Status)
	return err
}

// ─── approve ─────────────────────────────────────────────────────────────────

func runDocApprove(args []string, deps dependencies) error {
	if len(args) == 0 {
		return fmt.Errorf("missing document ID or path\n\nUsage: kbz doc approve <id|path> [--by <user>]")
	}

	target := args[0]
	remaining := args[1:]

	var approvedBy string

	for i := 0; i < len(remaining); i++ {
		switch remaining[i] {
		case "--by":
			if i+1 >= len(remaining) {
				return fmt.Errorf("--by requires a value")
			}
			i++
			approvedBy = remaining[i]
		default:
			return fmt.Errorf("unknown flag %q\n\nUsage: kbz doc approve <id|path> [--by <user>]", remaining[i])
		}
	}

	stateRoot := core.StatePath()
	repoRoot := "."
	docSvc := service.NewDocumentService(stateRoot, repoRoot)

	// Resolve document ID from target, supporting both file paths and IDs.
	docID, err := resolveDocApproveTarget(target, docSvc)
	if err != nil {
		return err
	}

	resolvedBy, err := config.ResolveIdentity(approvedBy)
	if err != nil {
		return err
	}

	result, err := docSvc.ApproveDocument(service.ApproveDocumentInput{
		ID:         docID,
		ApprovedBy: resolvedBy,
	})
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(deps.stdout, "approved document\nid: %s\ntitle: %s\nstatus: %s\n",
		result.ID, result.Title, result.Status)
	return err
}

// resolveDocApproveTarget resolves a target string (which may be a file path
// or a document ID) to a document ID using lexical disambiguation.
func resolveDocApproveTarget(target string, docSvc *service.DocumentService) (string, error) {
	kind := resolution.Disambiguate(target)

	if kind == resolution.ResolvePath {
		result, err := docSvc.LookupByPath(context.Background(), target)
		if err != nil {
			return "", err
		}
		if result.ID == "" {
			return "", fmt.Errorf("file is not registered: %s", target)
		}
		return result.ID, nil
	}

	// For ResolveEntity, ResolvePlanPrefix, or ResolveNone, treat as a
	// document ID and let the service layer validate.
	return target, nil
}

// ─── list ────────────────────────────────────────────────────────────────────

func runDocList(args []string, deps dependencies) error {
	var docType, status, owner string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--type":
			if i+1 >= len(args) {
				return fmt.Errorf("--type requires a value")
			}
			i++
			docType = args[i]
		case "--status":
			if i+1 >= len(args) {
				return fmt.Errorf("--status requires a value")
			}
			i++
			status = args[i]
		case "--owner":
			if i+1 >= len(args) {
				return fmt.Errorf("--owner requires a value")
			}
			i++
			owner = args[i]
		default:
			return fmt.Errorf("unknown flag %q\n\nUsage: kbz doc list [--type <type>] [--status <status>] [--owner <id>]", args[i])
		}
	}

	stateRoot := core.StatePath()
	repoRoot := "."
	docSvc := service.NewDocumentService(stateRoot, repoRoot)

	results, err := docSvc.ListDocuments(service.DocumentFilters{
		Type:   docType,
		Status: status,
		Owner:  owner,
	})
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(deps.stdout, "listed documents (%d)\n", len(results))
	if err != nil {
		return err
	}

	for _, r := range results {
		_, err = fmt.Fprintf(deps.stdout, "%s\t%s\t%s\t%s\n", r.ID, r.Status, r.Type, r.Title)
		if err != nil {
			return err
		}
	}

	return nil
}
