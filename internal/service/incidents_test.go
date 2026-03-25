package service

import (
	"reflect"
	"strings"
	"testing"
)

func TestEntityService_CreateIncident(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	got, err := svc.CreateIncident(CreateIncidentInput{
		Slug:       "prod-outage",
		Title:      "Production outage in API gateway",
		Severity:   "high",
		Summary:    "API gateway returning 503 for all requests",
		ReportedBy: "oncall-eng",
	})
	if err != nil {
		t.Fatalf("CreateIncident() error = %v", err)
	}

	if got.Type != "incident" {
		t.Fatalf("CreateIncident() type = %q, want %q", got.Type, "incident")
	}
	if !strings.HasPrefix(got.ID, "INC-") {
		t.Fatalf("CreateIncident() ID = %q, want prefix %q", got.ID, "INC-")
	}
	if got.Slug != "prod-outage" {
		t.Fatalf("CreateIncident() slug = %q, want %q", got.Slug, "prod-outage")
	}

	state := got.State
	if state["status"] != "reported" {
		t.Fatalf("CreateIncident() status = %v, want %q", state["status"], "reported")
	}
	if state["severity"] != "high" {
		t.Fatalf("CreateIncident() severity = %v, want %q", state["severity"], "high")
	}
	if state["title"] != "Production outage in API gateway" {
		t.Fatalf("CreateIncident() title = %v, want %q", state["title"], "Production outage in API gateway")
	}
	if state["summary"] != "API gateway returning 503 for all requests" {
		t.Fatalf("CreateIncident() summary = %v, want %q", state["summary"], "API gateway returning 503 for all requests")
	}
	if state["reported_by"] != "oncall-eng" {
		t.Fatalf("CreateIncident() reported_by = %v, want %q", state["reported_by"], "oncall-eng")
	}
}

func TestEntityService_CreateIncident_InvalidSeverity(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	_, err := svc.CreateIncident(CreateIncidentInput{
		Slug:       "bad-severity",
		Title:      "Test incident",
		Severity:   "extreme",
		Summary:    "Testing invalid severity",
		ReportedBy: "tester",
	})
	if err == nil {
		t.Fatal("CreateIncident() error = nil, want non-nil for invalid severity")
	}
	if !strings.Contains(err.Error(), "invalid incident severity") {
		t.Fatalf("CreateIncident() error = %q, want it to contain %q", err.Error(), "invalid incident severity")
	}
}

func TestEntityService_CreateIncident_MissingFields(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	_, err := svc.CreateIncident(CreateIncidentInput{
		Slug:       "",
		Title:      "Test incident",
		Severity:   "high",
		Summary:    "Missing slug",
		ReportedBy: "tester",
	})
	if err == nil {
		t.Fatal("CreateIncident() error = nil, want non-nil for empty slug")
	}
}

func TestEntityService_UpdateIncident_StatusTransition(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	created, err := svc.CreateIncident(CreateIncidentInput{
		Slug:       "status-transition",
		Title:      "Transition test",
		Severity:   "medium",
		Summary:    "Testing status transition",
		ReportedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreateIncident() error = %v", err)
	}

	if created.State["status"] != "reported" {
		t.Fatalf("initial status = %v, want %q", created.State["status"], "reported")
	}

	updated, err := svc.UpdateIncident(UpdateIncidentInput{
		ID:     created.ID,
		Slug:   created.Slug,
		Status: "triaged",
	})
	if err != nil {
		t.Fatalf("UpdateIncident() error = %v", err)
	}

	if updated.State["status"] != "triaged" {
		t.Fatalf("UpdateIncident() status = %v, want %q", updated.State["status"], "triaged")
	}
}

func TestEntityService_UpdateIncident_InvalidTransition(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	created, err := svc.CreateIncident(CreateIncidentInput{
		Slug:       "bad-transition",
		Title:      "Invalid transition test",
		Severity:   "low",
		Summary:    "Testing invalid transition",
		ReportedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreateIncident() error = %v", err)
	}

	if created.State["status"] != "reported" {
		t.Fatalf("initial status = %v, want %q", created.State["status"], "reported")
	}

	// reported → investigating is not a valid transition (must go through triaged first)
	_, err = svc.UpdateIncident(UpdateIncidentInput{
		ID:     created.ID,
		Slug:   created.Slug,
		Status: "investigating",
	})
	if err == nil {
		t.Fatal("UpdateIncident() error = nil, want non-nil for invalid transition reported→investigating")
	}
}

func TestEntityService_UpdateIncident_FieldsOnly(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	created, err := svc.CreateIncident(CreateIncidentInput{
		Slug:       "field-update",
		Title:      "Field update test",
		Severity:   "medium",
		Summary:    "Testing field-only update",
		ReportedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreateIncident() error = %v", err)
	}

	updated, err := svc.UpdateIncident(UpdateIncidentInput{
		ID:       created.ID,
		Slug:     created.Slug,
		Severity: "critical",
	})
	if err != nil {
		t.Fatalf("UpdateIncident() error = %v", err)
	}

	if updated.State["severity"] != "critical" {
		t.Fatalf("UpdateIncident() severity = %v, want %q", updated.State["severity"], "critical")
	}
	// Status should remain unchanged
	if updated.State["status"] != "reported" {
		t.Fatalf("UpdateIncident() status = %v, want %q (unchanged)", updated.State["status"], "reported")
	}
}

func TestEntityService_ListIncidents(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	// Create first incident (stays in "reported")
	_, err := svc.CreateIncident(CreateIncidentInput{
		Slug:       "incident-one",
		Title:      "First incident",
		Severity:   "high",
		Summary:    "First test incident",
		ReportedBy: "tester",
	})
	if err != nil {
		t.Fatalf("first CreateIncident() error = %v", err)
	}

	// Create second incident and advance to "triaged"
	second, err := svc.CreateIncident(CreateIncidentInput{
		Slug:       "incident-two",
		Title:      "Second incident",
		Severity:   "low",
		Summary:    "Second test incident",
		ReportedBy: "tester",
	})
	if err != nil {
		t.Fatalf("second CreateIncident() error = %v", err)
	}

	_, err = svc.UpdateIncident(UpdateIncidentInput{
		ID:     second.ID,
		Slug:   second.Slug,
		Status: "triaged",
	})
	if err != nil {
		t.Fatalf("UpdateIncident() to triaged error = %v", err)
	}

	// List all — should return 2
	all, err := svc.ListIncidents("", "")
	if err != nil {
		t.Fatalf("ListIncidents('', '') error = %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("ListIncidents('', '') returned %d results, want 2", len(all))
	}

	// List with status "reported" — should return 1
	reported, err := svc.ListIncidents("reported", "")
	if err != nil {
		t.Fatalf("ListIncidents('reported', '') error = %v", err)
	}
	if len(reported) != 1 {
		t.Fatalf("ListIncidents('reported', '') returned %d results, want 1", len(reported))
	}

	// List with status "triaged" — should return 1
	triaged, err := svc.ListIncidents("triaged", "")
	if err != nil {
		t.Fatalf("ListIncidents('triaged', '') error = %v", err)
	}
	if len(triaged) != 1 {
		t.Fatalf("ListIncidents('triaged', '') returned %d results, want 1", len(triaged))
	}
}

func TestEntityService_LinkBug(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	incident, err := svc.CreateIncident(CreateIncidentInput{
		Slug:       "link-bug-test",
		Title:      "Link bug test incident",
		Severity:   "high",
		Summary:    "Testing bug linking",
		ReportedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreateIncident() error = %v", err)
	}

	bugResult, err := svc.CreateBug(CreateBugInput{
		Slug:       "test-bug",
		Title:      "Test bug",
		ReportedBy: "tester",
		Observed:   "broken",
		Expected:   "working",
		Severity:   "medium",
		Priority:   "medium",
		Type:       "implementation-defect",
	})
	if err != nil {
		t.Fatalf("CreateBug() error = %v", err)
	}

	linked, err := svc.LinkBug(LinkBugInput{
		IncidentID: incident.ID,
		BugID:      bugResult.ID,
	})
	if err != nil {
		t.Fatalf("LinkBug() error = %v", err)
	}

	linkedBugs, ok := linked.State["linked_bugs"]
	if !ok {
		t.Fatal("LinkBug() state missing linked_bugs field")
	}

	bugList, ok := linkedBugs.([]string)
	if !ok {
		t.Fatalf("linked_bugs has unexpected type %T", linkedBugs)
	}
	if len(bugList) != 1 {
		t.Fatalf("linked_bugs has %d entries, want 1", len(bugList))
	}
	if bugList[0] != bugResult.ID {
		t.Fatalf("linked_bugs[0] = %q, want %q", bugList[0], bugResult.ID)
	}
}

func TestEntityService_LinkBug_Idempotent(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	incident, err := svc.CreateIncident(CreateIncidentInput{
		Slug:       "idempotent-link",
		Title:      "Idempotent link test",
		Severity:   "medium",
		Summary:    "Testing idempotent linking",
		ReportedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreateIncident() error = %v", err)
	}

	bugResult, err := svc.CreateBug(CreateBugInput{
		Slug:       "test-bug",
		Title:      "Test bug",
		ReportedBy: "tester",
		Observed:   "broken",
		Expected:   "working",
		Severity:   "medium",
		Priority:   "medium",
		Type:       "implementation-defect",
	})
	if err != nil {
		t.Fatalf("CreateBug() error = %v", err)
	}

	input := LinkBugInput{
		IncidentID: incident.ID,
		BugID:      bugResult.ID,
	}

	// Link the bug twice
	_, err = svc.LinkBug(input)
	if err != nil {
		t.Fatalf("first LinkBug() error = %v", err)
	}

	secondLink, err := svc.LinkBug(input)
	if err != nil {
		t.Fatalf("second LinkBug() error = %v", err)
	}

	linkedBugs, ok := secondLink.State["linked_bugs"]
	if !ok {
		t.Fatal("second LinkBug() state missing linked_bugs field")
	}

	bugCount := 0
	switch v := linkedBugs.(type) {
	case []string:
		bugCount = len(v)
	case []any:
		bugCount = len(v)
	default:
		t.Fatalf("linked_bugs has unexpected type %T", linkedBugs)
	}
	if bugCount != 1 {
		t.Fatalf("linked_bugs has %d entries after duplicate link, want 1", bugCount)
	}
}

func TestEntityService_IncidentRoundTrip(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	created, err := svc.CreateIncident(CreateIncidentInput{
		Slug:       "round-trip",
		Title:      "Round trip test",
		Severity:   "low",
		Summary:    "Testing round-trip serialization",
		ReportedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreateIncident() error = %v", err)
	}

	// Load via Get
	loaded, err := svc.Get(created.Type, created.ID, created.Slug)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if loaded.Type != created.Type {
		t.Fatalf("Get() type = %q, want %q", loaded.Type, created.Type)
	}
	if loaded.ID != created.ID {
		t.Fatalf("Get() ID = %q, want %q", loaded.ID, created.ID)
	}
	if loaded.Slug != created.Slug {
		t.Fatalf("Get() slug = %q, want %q", loaded.Slug, created.Slug)
	}

	if !reflect.DeepEqual(loaded.State, created.State) {
		t.Fatalf("Get() state mismatch after round-trip\nwant: %#v\ngot:  %#v", created.State, loaded.State)
	}
}
