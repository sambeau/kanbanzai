package main

import (
	"fmt"
	"strings"

	"kanbanzai/internal/core"
	"kanbanzai/internal/id"
	"kanbanzai/internal/service"
)

const incidentUsageText = `Usage: kbz incident <subcommand> [options]

Subcommands:
  create      Create a new incident
    --slug          <slug>
    --title         <title>
    --severity      critical|high|medium|low
    --summary       <text>
    --reported_by   <user>
    --detected_at   <ISO 8601 timestamp>  (optional)

  list        List incidents
    --status    <status>    (optional filter)
    --severity  <severity>  (optional filter)

  show <id>   Show incident details
`

func runIncident(args []string, deps dependencies) error {
	if len(args) == 0 {
		return fmt.Errorf("missing incident subcommand\n\n%s", incidentUsageText)
	}

	switch args[0] {
	case "create":
		return runIncidentCreate(args[1:], deps)
	case "list":
		return runIncidentList(args[1:], deps)
	case "show":
		return runIncidentShow(args[1:], deps)
	default:
		return fmt.Errorf("unknown incident subcommand %q\n\n%s", args[0], incidentUsageText)
	}
}

func runIncidentCreate(args []string, deps dependencies) error {
	var slug, title, severity, summary, reportedBy, detectedAt string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--slug":
			if i+1 >= len(args) {
				return fmt.Errorf("--slug requires a value")
			}
			i++
			slug = args[i]
		case "--title":
			if i+1 >= len(args) {
				return fmt.Errorf("--title requires a value")
			}
			i++
			title = args[i]
		case "--severity":
			if i+1 >= len(args) {
				return fmt.Errorf("--severity requires a value")
			}
			i++
			severity = args[i]
		case "--summary":
			if i+1 >= len(args) {
				return fmt.Errorf("--summary requires a value")
			}
			i++
			summary = args[i]
		case "--reported_by":
			if i+1 >= len(args) {
				return fmt.Errorf("--reported_by requires a value")
			}
			i++
			reportedBy = args[i]
		case "--detected_at":
			if i+1 >= len(args) {
				return fmt.Errorf("--detected_at requires a value")
			}
			i++
			detectedAt = args[i]
		default:
			return fmt.Errorf("unknown flag %q\n\nUsage: kbz incident create [flags]", args[i])
		}
	}

	if slug == "" {
		return fmt.Errorf("--slug is required\n\nUsage: kbz incident create [flags]")
	}
	if title == "" {
		return fmt.Errorf("--title is required\n\nUsage: kbz incident create [flags]")
	}
	if severity == "" {
		return fmt.Errorf("--severity is required\n\nUsage: kbz incident create [flags]")
	}
	if summary == "" {
		return fmt.Errorf("--summary is required\n\nUsage: kbz incident create [flags]")
	}
	if reportedBy == "" {
		return fmt.Errorf("--reported_by is required\n\nUsage: kbz incident create [flags]")
	}

	svc := service.NewEntityService(core.StatePath())
	result, err := svc.CreateIncident(service.CreateIncidentInput{
		Slug:       slug,
		Title:      title,
		Severity:   severity,
		Summary:    summary,
		ReportedBy: reportedBy,
		DetectedAt: detectedAt,
	})
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(deps.stdout, "created incident\nid: %s\nslug: %s\npath: %s\n",
		id.FormatFullDisplay(result.ID), result.Slug, result.Path)
	return err
}

func runIncidentList(args []string, deps dependencies) error {
	var statusFilter, severityFilter string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--status":
			if i+1 >= len(args) {
				return fmt.Errorf("--status requires a value")
			}
			i++
			statusFilter = args[i]
		case "--severity":
			if i+1 >= len(args) {
				return fmt.Errorf("--severity requires a value")
			}
			i++
			severityFilter = args[i]
		default:
			return fmt.Errorf("unknown flag %q\n\nUsage: kbz incident list [--status <status>] [--severity <severity>]", args[i])
		}
	}

	svc := service.NewEntityService(core.StatePath())
	results, err := svc.ListIncidents(statusFilter, severityFilter)
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(deps.stdout, "listed incidents (%d)\n", len(results)); err != nil {
		return err
	}

	for _, r := range results {
		if _, err := fmt.Fprintf(deps.stdout, "%s\t%s\t%v\t%v\n",
			id.FormatFullDisplay(r.ID), r.Slug, r.State["status"], r.State["severity"]); err != nil {
			return err
		}
	}

	return nil
}

func runIncidentShow(args []string, deps dependencies) error {
	if len(args) == 0 {
		return fmt.Errorf("missing incident ID\n\nUsage: kbz incident show <incident-id>")
	}

	incidentID := strings.TrimSpace(args[0])

	svc := service.NewEntityService(core.StatePath())
	result, err := svc.Get("incident", incidentID, "")
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(deps.stdout,
		"type: %s\nid: %s\nslug: %s\nstatus: %v\nseverity: %v\ntitle: %v\nsummary: %v\n",
		result.Type,
		id.FormatFullDisplay(result.ID),
		result.Slug,
		result.State["status"],
		result.State["severity"],
		result.State["title"],
		result.State["summary"],
	)
	return err
}
