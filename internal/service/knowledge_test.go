package service

import (
	"strings"
	"testing"
	"time"
)

// newTestKnowledgeService creates a KnowledgeService backed by a temp directory.
func newTestKnowledgeService(t *testing.T) *KnowledgeService {
	t.Helper()
	root := t.TempDir()
	svc := NewKnowledgeService(root)
	// Fix clock for determinism
	fixed := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return fixed }
	return svc
}

// ─────────────────────────────────────────────────────────────────────────────
// Contribute
// ─────────────────────────────────────────────────────────────────────────────

func TestKnowledgeService_Contribute_Success(t *testing.T) {
	t.Parallel()
	svc := newTestKnowledgeService(t)

	record, dup, err := svc.Contribute(ContributeInput{
		Topic:     "API JSON Naming Convention",
		Content:   "Use camelCase for all JSON API field names.",
		Scope:     "project",
		Tier:      3,
		CreatedBy: "agent",
	})

	if err != nil {
		t.Fatalf("Contribute() error = %v", err)
	}
	if dup != nil {
		t.Fatal("Contribute() returned duplicate, want nil")
	}
	if !strings.HasPrefix(record.ID, "KE-") {
		t.Errorf("record.ID = %q, want KE- prefix", record.ID)
	}

	// Verify normalised topic
	if topic, _ := record.Fields["topic"].(string); topic != "api-json-naming-convention" {
		t.Errorf("Fields['topic'] = %q, want 'api-json-naming-convention'", topic)
	}
	if status, _ := record.Fields["status"].(string); status != "contributed" {
		t.Errorf("Fields['status'] = %q, want 'contributed'", status)
	}
	if tier := knowledgeFieldInt(record.Fields, "tier"); tier != 3 {
		t.Errorf("Fields['tier'] = %d, want 3", tier)
	}
	if ttl := knowledgeFieldInt(record.Fields, "ttl_days"); ttl != 30 {
		t.Errorf("Fields['ttl_days'] = %d, want 30 for tier 3", ttl)
	}
}

func TestKnowledgeService_Contribute_Tier2DefaultsTTL90(t *testing.T) {
	t.Parallel()
	svc := newTestKnowledgeService(t)

	record, _, err := svc.Contribute(ContributeInput{
		Topic:   "go-concurrency",
		Content: "Avoid goroutines unless there is a demonstrated need.",
		Scope:   "project",
		Tier:    2,
	})
	if err != nil {
		t.Fatalf("Contribute() error = %v", err)
	}
	if tier := knowledgeFieldInt(record.Fields, "tier"); tier != 2 {
		t.Errorf("Fields['tier'] = %d, want 2", tier)
	}
	if ttl := knowledgeFieldInt(record.Fields, "ttl_days"); ttl != 90 {
		t.Errorf("Fields['ttl_days'] = %d, want 90 for tier 2", ttl)
	}
}

func TestKnowledgeService_Contribute_InvalidTierDefaultsTo3(t *testing.T) {
	t.Parallel()
	svc := newTestKnowledgeService(t)

	record, _, err := svc.Contribute(ContributeInput{
		Topic:   "testing-conventions",
		Content: "Use table-driven tests for multiple related test scenarios.",
		Scope:   "project",
		Tier:    99, // invalid
	})
	if err != nil {
		t.Fatalf("Contribute() error = %v", err)
	}
	if tier := knowledgeFieldInt(record.Fields, "tier"); tier != 3 {
		t.Errorf("Fields['tier'] = %d, want 3 (default for invalid tier)", tier)
	}
}

func TestKnowledgeService_Contribute_MissingTopic(t *testing.T) {
	t.Parallel()
	svc := newTestKnowledgeService(t)

	_, _, err := svc.Contribute(ContributeInput{
		Topic:   "",
		Content: "Some content.",
		Scope:   "project",
	})
	if err == nil {
		t.Fatal("Contribute() expected error for empty topic, got nil")
	}
}

func TestKnowledgeService_Contribute_MissingContent(t *testing.T) {
	t.Parallel()
	svc := newTestKnowledgeService(t)

	_, _, err := svc.Contribute(ContributeInput{
		Topic:   "some-topic",
		Content: "",
		Scope:   "project",
	})
	if err == nil {
		t.Fatal("Contribute() expected error for empty content, got nil")
	}
}

func TestKnowledgeService_Contribute_MissingScope(t *testing.T) {
	t.Parallel()
	svc := newTestKnowledgeService(t)

	_, _, err := svc.Contribute(ContributeInput{
		Topic:   "some-topic",
		Content: "Some content.",
		Scope:   "",
	})
	if err == nil {
		t.Fatal("Contribute() expected error for empty scope, got nil")
	}
}

func TestKnowledgeService_Contribute_ExactTopicDuplicate(t *testing.T) {
	t.Parallel()
	svc := newTestKnowledgeService(t)

	input := ContributeInput{
		Topic:   "go-error-wrapping",
		Content: "Always wrap errors with context using fmt.Errorf with %w.",
		Scope:   "project",
	}
	if _, _, err := svc.Contribute(input); err != nil {
		t.Fatalf("first Contribute() error = %v", err)
	}

	// Same topic, same scope → duplicate
	_, dup, err := svc.Contribute(ContributeInput{
		Topic:   "Go Error Wrapping", // different case/format → same normalised topic
		Content: "Wrap errors with context.",
		Scope:   "project",
	})
	if err == nil {
		t.Fatal("Contribute() expected error for duplicate topic, got nil")
	}
	if dup == nil {
		t.Fatal("Contribute() expected duplicate pointer, got nil")
	}
}

func TestKnowledgeService_Contribute_NearDuplicateRejected(t *testing.T) {
	t.Parallel()
	svc := newTestKnowledgeService(t)

	if _, _, err := svc.Contribute(ContributeInput{
		Topic:   "json-naming",
		Content: "Always use camelCase for JSON API field names in REST responses",
		Scope:   "project",
	}); err != nil {
		t.Fatalf("first Contribute() error = %v", err)
	}

	// Very similar content in same scope → near-duplicate rejection
	_, dup, err := svc.Contribute(ContributeInput{
		Topic:   "json-field-naming",
		Content: "Always use camelCase for JSON API field names in HTTP responses",
		Scope:   "project",
	})
	if err == nil {
		t.Fatal("Contribute() expected error for near-duplicate, got nil")
	}
	if dup == nil {
		t.Fatal("Contribute() expected duplicate pointer for near-duplicate, got nil")
	}
}

func TestKnowledgeService_Contribute_DifferentScopeNotDuplicate(t *testing.T) {
	t.Parallel()
	svc := newTestKnowledgeService(t)

	if _, _, err := svc.Contribute(ContributeInput{
		Topic:   "api-json-naming",
		Content: "Use camelCase for JSON API field names.",
		Scope:   "project",
	}); err != nil {
		t.Fatalf("first Contribute() error = %v", err)
	}

	// Same topic but different scope → NOT a duplicate
	_, dup, err := svc.Contribute(ContributeInput{
		Topic:   "api-json-naming",
		Content: "Use camelCase for JSON API field names.",
		Scope:   "backend-profile",
	})
	if err != nil {
		t.Fatalf("Contribute() in different scope should succeed, got error = %v", err)
	}
	if dup != nil {
		t.Fatal("Contribute() in different scope should not return duplicate")
	}
}

func TestKnowledgeService_Contribute_WithTags(t *testing.T) {
	t.Parallel()
	svc := newTestKnowledgeService(t)

	record, _, err := svc.Contribute(ContributeInput{
		Topic:   "go-imports",
		Content: "Use goimports to organise imports automatically.",
		Scope:   "project",
		Tags:    []string{"go", "formatting"},
	})
	if err != nil {
		t.Fatalf("Contribute() error = %v", err)
	}

	tags := knowledgeFieldStrings(record.Fields, "tags")
	if len(tags) != 2 {
		t.Errorf("Fields['tags'] len = %d, want 2", len(tags))
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Get
// ─────────────────────────────────────────────────────────────────────────────

func TestKnowledgeService_Get_Found(t *testing.T) {
	t.Parallel()
	svc := newTestKnowledgeService(t)

	record, _, err := svc.Contribute(ContributeInput{
		Topic:   "get-test-topic",
		Content: "Content for get test.",
		Scope:   "project",
	})
	if err != nil {
		t.Fatalf("Contribute() error = %v", err)
	}

	got, err := svc.Get(record.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.ID != record.ID {
		t.Errorf("Get() ID = %q, want %q", got.ID, record.ID)
	}
}

func TestKnowledgeService_Get_NotFound(t *testing.T) {
	t.Parallel()
	svc := newTestKnowledgeService(t)

	_, err := svc.Get("KE-DOESNOTEXIST")
	if err == nil {
		t.Fatal("Get() expected error for missing entry, got nil")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// List
// ─────────────────────────────────────────────────────────────────────────────

func TestKnowledgeService_List_All(t *testing.T) {
	t.Parallel()
	svc := newTestKnowledgeService(t)

	contents := []string{
		"Error wrapping uses fmt.Errorf with the %w verb for context propagation.",
		"Database connections require explicit Close calls to avoid resource leaks.",
		"HTTP handlers validate input at the boundary before passing to service layer.",
	}
	for i, topic := range []string{"topic-a", "topic-b", "topic-c"} {
		_, _, err := svc.Contribute(ContributeInput{
			Topic:   topic,
			Content: contents[i],
			Scope:   "project",
			Tier:    []int{2, 3, 3}[i],
		})
		if err != nil {
			t.Fatalf("Contribute(%s) error = %v", topic, err)
		}
	}

	records, err := svc.List(KnowledgeFilters{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(records) != 3 {
		t.Errorf("List() count = %d, want 3", len(records))
	}
}

func TestKnowledgeService_List_ExcludesRetiredByDefault(t *testing.T) {
	t.Parallel()
	svc := newTestKnowledgeService(t)

	r1, _, err := svc.Contribute(ContributeInput{Topic: "active-topic", Content: "Active.", Scope: "project"})
	if err != nil {
		t.Fatalf("Contribute() error = %v", err)
	}
	r2, _, err := svc.Contribute(ContributeInput{Topic: "retired-topic", Content: "Will be retired.", Scope: "project"})
	if err != nil {
		t.Fatalf("Contribute() error = %v", err)
	}
	_ = r1

	if _, err := svc.Retire(r2.ID, "no longer needed"); err != nil {
		t.Fatalf("Retire() error = %v", err)
	}

	records, err := svc.List(KnowledgeFilters{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(records) != 1 {
		t.Errorf("List() returned %d records, want 1 (retired excluded)", len(records))
	}
}

func TestKnowledgeService_List_IncludeRetired(t *testing.T) {
	t.Parallel()
	svc := newTestKnowledgeService(t)

	r, _, _ := svc.Contribute(ContributeInput{Topic: "to-retire", Content: "Content.", Scope: "project"})
	svc.Retire(r.ID, "obsolete") //nolint:errcheck

	records, err := svc.List(KnowledgeFilters{IncludeRetired: true})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(records) != 1 {
		t.Errorf("List(IncludeRetired) returned %d records, want 1", len(records))
	}
}

func TestKnowledgeService_List_FilterByScope(t *testing.T) {
	t.Parallel()
	svc := newTestKnowledgeService(t)

	svc.Contribute(ContributeInput{Topic: "scope-a-topic", Content: "Scope A content.", Scope: "scope-a"}) //nolint:errcheck
	svc.Contribute(ContributeInput{Topic: "scope-b-topic", Content: "Scope B content.", Scope: "scope-b"}) //nolint:errcheck

	records, err := svc.List(KnowledgeFilters{Scope: "scope-a"})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(records) != 1 {
		t.Errorf("List(Scope=scope-a) = %d, want 1", len(records))
	}
	if s, _ := records[0].Fields["scope"].(string); s != "scope-a" {
		t.Errorf("record scope = %q, want 'scope-a'", s)
	}
}

func TestKnowledgeService_List_FilterByTier(t *testing.T) {
	t.Parallel()
	svc := newTestKnowledgeService(t)

	svc.Contribute(ContributeInput{Topic: "tier2-topic", Content: "Tier 2.", Scope: "project", Tier: 2}) //nolint:errcheck
	svc.Contribute(ContributeInput{Topic: "tier3-topic", Content: "Tier 3.", Scope: "project", Tier: 3}) //nolint:errcheck

	records, err := svc.List(KnowledgeFilters{Tier: 2})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(records) != 1 {
		t.Errorf("List(Tier=2) = %d, want 1", len(records))
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Update
// ─────────────────────────────────────────────────────────────────────────────

func TestKnowledgeService_Update_ResetsCounters(t *testing.T) {
	t.Parallel()
	svc := newTestKnowledgeService(t)

	record, _, err := svc.Contribute(ContributeInput{
		Topic:   "update-test",
		Content: "Original content.",
		Scope:   "project",
	})
	if err != nil {
		t.Fatalf("Contribute() error = %v", err)
	}

	// Simulate some usage
	svc.ContextReport("TASK-001", []string{record.ID}, nil) //nolint:errcheck

	updated, err := svc.Update(record.ID, "Updated content.")
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if content, _ := updated.Fields["content"].(string); content != "Updated content." {
		t.Errorf("Fields['content'] = %q, want 'Updated content.'", content)
	}
	if uc := knowledgeFieldInt(updated.Fields, "use_count"); uc != 0 {
		t.Errorf("Fields['use_count'] = %d, want 0 after update", uc)
	}
	if mc := knowledgeFieldInt(updated.Fields, "miss_count"); mc != 0 {
		t.Errorf("Fields['miss_count'] = %d, want 0 after update", mc)
	}
}

func TestKnowledgeService_Update_NotFound(t *testing.T) {
	t.Parallel()
	svc := newTestKnowledgeService(t)

	_, err := svc.Update("KE-NOTFOUND", "New content.")
	if err == nil {
		t.Fatal("Update() expected error for missing entry, got nil")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Confirm
// ─────────────────────────────────────────────────────────────────────────────

func TestKnowledgeService_Confirm_FromContributed(t *testing.T) {
	t.Parallel()
	svc := newTestKnowledgeService(t)

	record, _, err := svc.Contribute(ContributeInput{
		Topic:   "confirm-test",
		Content: "Content to confirm.",
		Scope:   "project",
	})
	if err != nil {
		t.Fatalf("Contribute() error = %v", err)
	}

	confirmed, err := svc.Confirm(record.ID)
	if err != nil {
		t.Fatalf("Confirm() error = %v", err)
	}
	if status, _ := confirmed.Fields["status"].(string); status != "confirmed" {
		t.Errorf("Fields['status'] = %q, want 'confirmed'", status)
	}
}

func TestKnowledgeService_Confirm_FromRetired_Fails(t *testing.T) {
	t.Parallel()
	svc := newTestKnowledgeService(t)

	record, _, _ := svc.Contribute(ContributeInput{Topic: "retire-then-confirm", Content: "Content.", Scope: "project"})
	svc.Retire(record.ID, "done") //nolint:errcheck

	_, err := svc.Confirm(record.ID)
	if err == nil {
		t.Fatal("Confirm() from retired should return error")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Flag
// ─────────────────────────────────────────────────────────────────────────────

func TestKnowledgeService_Flag_TransitionsToDisputed(t *testing.T) {
	t.Parallel()
	svc := newTestKnowledgeService(t)

	record, _, err := svc.Contribute(ContributeInput{
		Topic:   "flag-test",
		Content: "Content to flag.",
		Scope:   "project",
	})
	if err != nil {
		t.Fatalf("Contribute() error = %v", err)
	}

	flagged, err := svc.Flag(record.ID, "seems wrong")
	if err != nil {
		t.Fatalf("Flag() error = %v", err)
	}

	if status, _ := flagged.Fields["status"].(string); status != "disputed" {
		t.Errorf("Fields['status'] = %q, want 'disputed' after first flag", status)
	}
	if mc := knowledgeFieldInt(flagged.Fields, "miss_count"); mc != 1 {
		t.Errorf("Fields['miss_count'] = %d, want 1", mc)
	}
}

func TestKnowledgeService_Flag_AutoRetireAt2(t *testing.T) {
	t.Parallel()
	svc := newTestKnowledgeService(t)

	record, _, _ := svc.Contribute(ContributeInput{Topic: "auto-retire-test", Content: "Content.", Scope: "project"})

	svc.Flag(record.ID, "wrong once") //nolint:errcheck

	retired, err := svc.Flag(record.ID, "wrong twice")
	if err != nil {
		t.Fatalf("second Flag() error = %v", err)
	}
	if status, _ := retired.Fields["status"].(string); status != "retired" {
		t.Errorf("Fields['status'] = %q, want 'retired' after second flag", status)
	}
	wantReason := "auto-retired: miss_count reached 2 (wrong twice)"
	if reason, _ := retired.Fields["deprecated_reason"].(string); reason != wantReason {
		t.Errorf("Fields['deprecated_reason'] = %q, want %q", reason, wantReason)
	}
}

func TestKnowledgeService_Flag_AlreadyRetired_Fails(t *testing.T) {
	t.Parallel()
	svc := newTestKnowledgeService(t)

	record, _, _ := svc.Contribute(ContributeInput{Topic: "flag-retired", Content: "Content.", Scope: "project"})
	svc.Retire(record.ID, "done") //nolint:errcheck

	_, err := svc.Flag(record.ID, "late flag")
	if err == nil {
		t.Fatal("Flag() on retired entry should return error")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Retire
// ─────────────────────────────────────────────────────────────────────────────

func TestKnowledgeService_Retire_WithReason(t *testing.T) {
	t.Parallel()
	svc := newTestKnowledgeService(t)

	record, _, _ := svc.Contribute(ContributeInput{Topic: "retire-me", Content: "Content.", Scope: "project"})

	retired, err := svc.Retire(record.ID, "superseded by new policy")
	if err != nil {
		t.Fatalf("Retire() error = %v", err)
	}
	if status, _ := retired.Fields["status"].(string); status != "retired" {
		t.Errorf("Fields['status'] = %q, want 'retired'", status)
	}
	if reason, _ := retired.Fields["deprecated_reason"].(string); reason != "superseded by new policy" {
		t.Errorf("Fields['deprecated_reason'] = %q", reason)
	}
}

func TestKnowledgeService_Retire_AlreadyRetired_Fails(t *testing.T) {
	t.Parallel()
	svc := newTestKnowledgeService(t)

	record, _, _ := svc.Contribute(ContributeInput{Topic: "retire-twice", Content: "Content.", Scope: "project"})
	svc.Retire(record.ID, "first") //nolint:errcheck

	_, err := svc.Retire(record.ID, "second")
	if err == nil {
		t.Fatal("Retire() on already-retired entry should return error")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Promote
// ─────────────────────────────────────────────────────────────────────────────

func TestKnowledgeService_Promote_Tier3To2(t *testing.T) {
	t.Parallel()
	svc := newTestKnowledgeService(t)

	record, _, err := svc.Contribute(ContributeInput{
		Topic:   "promote-test",
		Content: "Content to promote.",
		Scope:   "project",
		Tier:    3,
	})
	if err != nil {
		t.Fatalf("Contribute() error = %v", err)
	}
	if tier := knowledgeFieldInt(record.Fields, "tier"); tier != 3 {
		t.Fatalf("initial tier = %d, want 3", tier)
	}

	promoted, err := svc.Promote(record.ID)
	if err != nil {
		t.Fatalf("Promote() error = %v", err)
	}
	if tier := knowledgeFieldInt(promoted.Fields, "tier"); tier != 2 {
		t.Errorf("Fields['tier'] = %d, want 2 after promotion", tier)
	}
	if ttl := knowledgeFieldInt(promoted.Fields, "ttl_days"); ttl != 90 {
		t.Errorf("Fields['ttl_days'] = %d, want 90 after promotion", ttl)
	}
}

func TestKnowledgeService_Promote_AlreadyTier2_Fails(t *testing.T) {
	t.Parallel()
	svc := newTestKnowledgeService(t)

	record, _, _ := svc.Contribute(ContributeInput{Topic: "already-tier2", Content: "Content.", Scope: "project", Tier: 2})

	_, err := svc.Promote(record.ID)
	if err == nil {
		t.Fatal("Promote() on tier-2 entry should return error")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// ContextReport
// ─────────────────────────────────────────────────────────────────────────────

func TestKnowledgeService_ContextReport_IncrementsUseCount(t *testing.T) {
	t.Parallel()
	svc := newTestKnowledgeService(t)

	record, _, _ := svc.Contribute(ContributeInput{Topic: "report-use", Content: "Content.", Scope: "project"})

	if err := svc.ContextReport("TASK-001", []string{record.ID}, nil); err != nil {
		t.Fatalf("ContextReport() error = %v", err)
	}

	got, _ := svc.Get(record.ID)
	if uc := knowledgeFieldInt(got.Fields, "use_count"); uc != 1 {
		t.Errorf("use_count = %d, want 1", uc)
	}
}

func TestKnowledgeService_ContextReport_AutoConfirmAt3Uses(t *testing.T) {
	t.Parallel()
	svc := newTestKnowledgeService(t)

	record, _, _ := svc.Contribute(ContributeInput{Topic: "auto-confirm", Content: "Content.", Scope: "project"})

	// Three successive reports with the same entry
	for i := 0; i < 3; i++ {
		svc.ContextReport("TASK-001", []string{record.ID}, nil) //nolint:errcheck
	}

	got, _ := svc.Get(record.ID)
	if status, _ := got.Fields["status"].(string); status != "confirmed" {
		t.Errorf("status = %q, want 'confirmed' after 3 uses with 0 misses", status)
	}
}

func TestKnowledgeService_ContextReport_FlaggedAutoRetireAt2(t *testing.T) {
	t.Parallel()
	svc := newTestKnowledgeService(t)

	record, _, _ := svc.Contribute(ContributeInput{Topic: "report-flag", Content: "Content.", Scope: "project"})

	svc.ContextReport("TASK-001", nil, []FlaggedEntry{{EntryID: record.ID, Reason: "wrong"}})       //nolint:errcheck
	svc.ContextReport("TASK-002", nil, []FlaggedEntry{{EntryID: record.ID, Reason: "still wrong"}}) //nolint:errcheck

	got, _ := svc.Get(record.ID)
	if status, _ := got.Fields["status"].(string); status != "retired" {
		t.Errorf("status = %q, want 'retired' after 2 context-report flags", status)
	}
}

func TestKnowledgeService_ContextReport_SkipsMissingEntries(t *testing.T) {
	t.Parallel()
	svc := newTestKnowledgeService(t)

	// Should not error even if the entry IDs don't exist
	err := svc.ContextReport("TASK-001", []string{"KE-NOTEXIST1", "KE-NOTEXIST2"}, nil)
	if err != nil {
		t.Errorf("ContextReport() with missing entries error = %v, want nil", err)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// ParseFlaggedEntries
// ─────────────────────────────────────────────────────────────────────────────

func TestParseFlaggedEntries(t *testing.T) {
	t.Parallel()

	t.Run("valid JSON", func(t *testing.T) {
		t.Parallel()
		raw := `[{"entry_id":"KE-001","reason":"wrong"},{"entry_id":"KE-002","reason":"outdated"}]`
		entries, err := ParseFlaggedEntries(raw)
		if err != nil {
			t.Fatalf("ParseFlaggedEntries() error = %v", err)
		}
		if len(entries) != 2 {
			t.Fatalf("len = %d, want 2", len(entries))
		}
		if entries[0].EntryID != "KE-001" {
			t.Errorf("entries[0].EntryID = %q, want 'KE-001'", entries[0].EntryID)
		}
		if entries[1].Reason != "outdated" {
			t.Errorf("entries[1].Reason = %q, want 'outdated'", entries[1].Reason)
		}
	})

	t.Run("empty string", func(t *testing.T) {
		t.Parallel()
		entries, err := ParseFlaggedEntries("")
		if err != nil {
			t.Fatalf("ParseFlaggedEntries('') error = %v", err)
		}
		if len(entries) != 0 {
			t.Errorf("len = %d, want 0 for empty input", len(entries))
		}
	})

	t.Run("whitespace only", func(t *testing.T) {
		t.Parallel()
		entries, err := ParseFlaggedEntries("   ")
		if err != nil {
			t.Fatalf("ParseFlaggedEntries(whitespace) error = %v", err)
		}
		if len(entries) != 0 {
			t.Errorf("len = %d, want 0 for whitespace input", len(entries))
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		t.Parallel()
		_, err := ParseFlaggedEntries("not json")
		if err == nil {
			t.Fatal("ParseFlaggedEntries(invalid) expected error, got nil")
		}
	})
}
