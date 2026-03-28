package validate

import (
	"strings"
	"testing"

	"github.com/sambeau/kanbanzai/internal/model"
)

func TestEntryState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		kind      EntityKind
		wantState string
		wantOK    bool
	}{
		{
			name:      "plan",
			kind:      EntityPlan,
			wantState: "proposed",
			wantOK:    true,
		},
		{
			name:      "epic",
			kind:      EntityEpic,
			wantState: "proposed",
			wantOK:    true,
		},
		{
			name:      "feature",
			kind:      EntityFeature,
			wantState: "proposed",
			wantOK:    true,
		},
		{
			name:      "task",
			kind:      EntityTask,
			wantState: "queued",
			wantOK:    true,
		},
		{
			name:      "bug",
			kind:      EntityBug,
			wantState: "reported",
			wantOK:    true,
		},
		{
			name:      "decision",
			kind:      EntityDecision,
			wantState: "proposed",
			wantOK:    true,
		},
		{
			name:   "unknown",
			kind:   EntityKind("unknown"),
			wantOK: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotState, gotOK := EntryState(tt.kind)
			if gotOK != tt.wantOK {
				t.Fatalf("EntryState(%q) ok = %v, want %v", tt.kind, gotOK, tt.wantOK)
			}
			if gotState != tt.wantState {
				t.Fatalf("EntryState(%q) state = %q, want %q", tt.kind, gotState, tt.wantState)
			}
		})
	}
}

func TestIsTerminalState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		kind  EntityKind
		state string
		want  bool
	}{
		{name: "plan done", kind: EntityPlan, state: "done", want: false},
		{name: "plan superseded", kind: EntityPlan, state: "superseded", want: true},
		{name: "plan cancelled", kind: EntityPlan, state: "cancelled", want: true},
		{name: "epic done", kind: EntityEpic, state: "done", want: true},
		{name: "epic active", kind: EntityEpic, state: "active", want: false},
		{name: "feature done", kind: EntityFeature, state: "done", want: false},
		{name: "feature superseded", kind: EntityFeature, state: "superseded", want: true},
		{name: "feature cancelled", kind: EntityFeature, state: "cancelled", want: true},
		{name: "feature review", kind: EntityFeature, state: "review", want: false},
		{name: "feature reviewing", kind: EntityFeature, state: "reviewing", want: false},
		{name: "feature needs-rework", kind: EntityFeature, state: "needs-rework", want: false},
		{name: "task done", kind: EntityTask, state: "done", want: true},
		{name: "task active", kind: EntityTask, state: "active", want: false},
		{name: "bug closed", kind: EntityBug, state: "closed", want: true},
		{name: "bug duplicate", kind: EntityBug, state: "duplicate", want: true},
		{name: "bug not planned", kind: EntityBug, state: "not-planned", want: true},
		{name: "bug cannot reproduce", kind: EntityBug, state: "cannot-reproduce", want: false},
		{name: "decision rejected", kind: EntityDecision, state: "rejected", want: true},
		{name: "decision superseded", kind: EntityDecision, state: "superseded", want: true},
		{name: "decision accepted", kind: EntityDecision, state: "accepted", want: false},
		{name: "unknown kind", kind: EntityKind("unknown"), state: "done", want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := IsTerminalState(tt.kind, tt.state)
			if got != tt.want {
				t.Fatalf("IsTerminalState(%q, %q) = %v, want %v", tt.kind, tt.state, got, tt.want)
			}
		})
	}
}

func TestIsKnownState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		kind  EntityKind
		state string
		want  bool
	}{
		{name: "epic entry state", kind: EntityEpic, state: "proposed", want: true},
		{name: "epic terminal state", kind: EntityEpic, state: "done", want: true},
		{name: "feature middle state", kind: EntityFeature, state: "needs-rework", want: true},
		{name: "task review state", kind: EntityTask, state: "needs-review", want: true},
		{name: "bug reopening state", kind: EntityBug, state: "cannot-reproduce", want: true},
		{name: "decision terminal state", kind: EntityDecision, state: "superseded", want: true},
		{name: "unknown state", kind: EntityBug, state: "mystery", want: false},
		{name: "unknown kind", kind: EntityKind("unknown"), state: "proposed", want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := IsKnownState(tt.kind, tt.state)
			if got != tt.want {
				t.Fatalf("IsKnownState(%q, %q) = %v, want %v", tt.kind, tt.state, got, tt.want)
			}
		})
	}
}

func TestCanTransition(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		kind EntityKind
		from string
		to   string
		want bool
	}{
		{
			name: "plan done to superseded",
			kind: EntityPlan,
			from: "done",
			to:   "superseded",
			want: true,
		},
		{
			name: "plan done to cancelled",
			kind: EntityPlan,
			from: "done",
			to:   "cancelled",
			want: true,
		},
		{
			name: "epic proposed to approved",
			kind: EntityEpic,
			from: "proposed",
			to:   "approved",
			want: true,
		},
		{
			name: "epic proposed to done is illegal",
			kind: EntityEpic,
			from: "proposed",
			to:   "done",
			want: false,
		},
		{
			name: "feature needs rework to in review",
			kind: EntityFeature,
			from: "needs-rework",
			to:   "in-review",
			want: true,
		},
		{
			name: "feature needs rework to review is illegal",
			kind: EntityFeature,
			from: "needs-rework",
			to:   "review",
			want: false,
		},
		{
			name: "feature done to superseded",
			kind: EntityFeature,
			from: "done",
			to:   "superseded",
			want: true,
		},
		{
			name: "feature done to cancelled",
			kind: EntityFeature,
			from: "done",
			to:   "cancelled",
			want: true,
		},
		{
			name: "task blocked to active",
			kind: EntityTask,
			from: "blocked",
			to:   "active",
			want: true,
		},
		{
			name: "task done to active is illegal",
			kind: EntityTask,
			from: "done",
			to:   "active",
			want: false,
		},
		{
			name: "bug triaged to planned",
			kind: EntityBug,
			from: "triaged",
			to:   "planned",
			want: true,
		},
		{
			name: "bug cannot reproduce to triaged",
			kind: EntityBug,
			from: "cannot-reproduce",
			to:   "triaged",
			want: true,
		},
		{
			name: "bug closed to triaged is illegal",
			kind: EntityBug,
			from: "closed",
			to:   "triaged",
			want: false,
		},
		{
			name: "decision accepted to superseded",
			kind: EntityDecision,
			from: "accepted",
			to:   "superseded",
			want: true,
		},
		{
			name: "decision rejected to accepted is illegal",
			kind: EntityDecision,
			from: "rejected",
			to:   "accepted",
			want: false,
		},
		// Plan full lifecycle (T3)
		{
			name: "plan proposed to designing",
			kind: EntityPlan,
			from: "proposed",
			to:   "designing",
			want: true,
		},
		{
			name: "plan designing to active",
			kind: EntityPlan,
			from: "designing",
			to:   "active",
			want: true,
		},
		{
			name: "plan active to reviewing",
			kind: EntityPlan,
			from: "active",
			to:   "reviewing",
			want: true,
		},
		{
			name: "plan active to done is illegal (must go through reviewing)",
			kind: EntityPlan,
			from: "active",
			to:   "done",
			want: false,
		},
		{
			name: "plan reviewing to done",
			kind: EntityPlan,
			from: "reviewing",
			to:   "done",
			want: true,
		},
		{
			name: "plan reviewing to active (rework)",
			kind: EntityPlan,
			from: "reviewing",
			to:   "active",
			want: true,
		},
		{
			name: "plan reviewing to superseded",
			kind: EntityPlan,
			from: "reviewing",
			to:   "superseded",
			want: true,
		},
		{
			name: "plan reviewing to cancelled",
			kind: EntityPlan,
			from: "reviewing",
			to:   "cancelled",
			want: true,
		},
		{
			name: "plan proposed to superseded",
			kind: EntityPlan,
			from: "proposed",
			to:   "superseded",
			want: true,
		},
		{
			name: "plan proposed to cancelled",
			kind: EntityPlan,
			from: "proposed",
			to:   "cancelled",
			want: true,
		},
		{
			name: "plan proposed to done is illegal",
			kind: EntityPlan,
			from: "proposed",
			to:   "done",
			want: false,
		},
		{
			name: "plan active to proposed is illegal (backward)",
			kind: EntityPlan,
			from: "active",
			to:   "proposed",
			want: false,
		},
		{
			name: "plan superseded to active is illegal (terminal)",
			kind: EntityPlan,
			from: "superseded",
			to:   "active",
			want: false,
		},
		{
			name: "plan cancelled to active is illegal (terminal)",
			kind: EntityPlan,
			from: "cancelled",
			to:   "active",
			want: false,
		},
		// Phase 2 Feature full lifecycle (T4)
		{
			name: "feature proposed to designing",
			kind: EntityFeature,
			from: "proposed",
			to:   "designing",
			want: true,
		},
		{
			name: "feature designing to specifying",
			kind: EntityFeature,
			from: "designing",
			to:   "specifying",
			want: true,
		},
		{
			name: "feature specifying to dev-planning",
			kind: EntityFeature,
			from: "specifying",
			to:   "dev-planning",
			want: true,
		},
		{
			name: "feature dev-planning to developing",
			kind: EntityFeature,
			from: "dev-planning",
			to:   "developing",
			want: true,
		},
		{
			name: "feature developing to done",
			kind: EntityFeature,
			from: "developing",
			to:   "done",
			want: false,
		},
		// Phase 2 review lifecycle transitions (AC-03, AC-05, AC-06, AC-07, AC-08, AC-10)
		{
			name: "feature developing to reviewing (AC-03)",
			kind: EntityFeature,
			from: "developing",
			to:   "reviewing",
			want: true,
		},
		{
			name: "feature reviewing to done (AC-05)",
			kind: EntityFeature,
			from: "reviewing",
			to:   "done",
			want: true,
		},
		{
			name: "feature reviewing to needs-rework (AC-06)",
			kind: EntityFeature,
			from: "reviewing",
			to:   "needs-rework",
			want: true,
		},
		{
			name: "feature needs-rework to developing (AC-07)",
			kind: EntityFeature,
			from: "needs-rework",
			to:   "developing",
			want: true,
		},
		{
			name: "feature needs-rework to reviewing quick-fix (AC-08)",
			kind: EntityFeature,
			from: "needs-rework",
			to:   "reviewing",
			want: true,
		},
		{
			name: "feature reviewing to superseded (AC-10)",
			kind: EntityFeature,
			from: "reviewing",
			to:   "superseded",
			want: true,
		},
		{
			name: "feature reviewing to cancelled (AC-10)",
			kind: EntityFeature,
			from: "reviewing",
			to:   "cancelled",
			want: true,
		},
		{
			name: "feature needs-rework to superseded (AC-10)",
			kind: EntityFeature,
			from: "needs-rework",
			to:   "superseded",
			want: true,
		},
		{
			name: "feature needs-rework to cancelled (AC-10)",
			kind: EntityFeature,
			from: "needs-rework",
			to:   "cancelled",
			want: true,
		},
		{
			name: "feature proposed to superseded shortcut",
			kind: EntityFeature,
			from: "proposed",
			to:   "superseded",
			want: true,
		},
		{
			name: "feature proposed to cancelled shortcut",
			kind: EntityFeature,
			from: "proposed",
			to:   "cancelled",
			want: true,
		},
		{
			name: "feature developing to proposed is illegal (backward)",
			kind: EntityFeature,
			from: "developing",
			to:   "proposed",
			want: false,
		},
		{
			name: "feature superseded to developing is illegal (terminal)",
			kind: EntityFeature,
			from: "superseded",
			to:   "developing",
			want: false,
		},
		{
			name: "self transition is illegal",
			kind: EntityFeature,
			from: "draft",
			to:   "draft",
			want: false,
		},
		{
			name: "unknown from state is illegal",
			kind: EntityEpic,
			from: "mystery",
			to:   "approved",
			want: false,
		},
		{
			name: "unknown to state is illegal",
			kind: EntityEpic,
			from: "proposed",
			to:   "mystery",
			want: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := CanTransition(tt.kind, tt.from, tt.to)
			if got != tt.want {
				t.Fatalf(
					"CanTransition(%q, %q, %q) = %v, want %v",
					tt.kind,
					tt.from,
					tt.to,
					got,
					tt.want,
				)
			}
		})
	}
}

func TestValidateInitialState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		kind    EntityKind
		state   string
		wantErr bool
	}{
		{
			name:    "valid plan entry state",
			kind:    EntityPlan,
			state:   "proposed",
			wantErr: false,
		},
		{
			name:    "invalid plan non entry state",
			kind:    EntityPlan,
			state:   "active",
			wantErr: true,
		},
		{
			name:    "valid epic entry state",
			kind:    EntityEpic,
			state:   "proposed",
			wantErr: false,
		},
		{
			name:    "invalid epic non entry state",
			kind:    EntityEpic,
			state:   "approved",
			wantErr: true,
		},
		{
			name:    "valid bug entry state",
			kind:    EntityBug,
			state:   "reported",
			wantErr: false,
		},
		{
			name:    "invalid bug non entry state",
			kind:    EntityBug,
			state:   "triaged",
			wantErr: true,
		},
		{
			name:    "unknown kind",
			kind:    EntityKind("unknown"),
			state:   "proposed",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateInitialState(tt.kind, tt.state)
			if (err != nil) != tt.wantErr {
				t.Fatalf(
					"ValidateInitialState(%q, %q) error = %v, wantErr %v",
					tt.kind,
					tt.state,
					err,
					tt.wantErr,
				)
			}
		})
	}
}

func TestValidateTransition(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		kind    EntityKind
		from    string
		to      string
		wantErr bool
	}{
		{
			name:    "valid epic transition",
			kind:    EntityEpic,
			from:    "approved",
			to:      "active",
			wantErr: false,
		},
		{
			name:    "invalid epic jump",
			kind:    EntityEpic,
			from:    "approved",
			to:      "done",
			wantErr: true,
		},
		{
			name:    "valid feature transition from needs rework to in progress",
			kind:    EntityFeature,
			from:    "needs-rework",
			to:      "in-progress",
			wantErr: false,
		},
		{
			name:    "invalid feature self transition",
			kind:    EntityFeature,
			from:    "review",
			to:      "review",
			wantErr: true,
		},
		{
			name:    "valid task transition",
			kind:    EntityTask,
			from:    "needs-review",
			to:      "done",
			wantErr: false,
		},
		{
			name:    "invalid task transition from terminal state",
			kind:    EntityTask,
			from:    "done",
			to:      "active",
			wantErr: true,
		},
		{
			name:    "valid bug reopening transition",
			kind:    EntityBug,
			from:    "cannot-reproduce",
			to:      "triaged",
			wantErr: false,
		},
		{
			name:    "invalid bug unknown destination",
			kind:    EntityBug,
			from:    "triaged",
			to:      "fixed",
			wantErr: true,
		},
		{
			name:    "valid decision transition",
			kind:    EntityDecision,
			from:    "proposed",
			to:      "accepted",
			wantErr: false,
		},
		{
			name:    "invalid decision transition from terminal state",
			kind:    EntityDecision,
			from:    "superseded",
			to:      "accepted",
			wantErr: true,
		},
		{
			name:    "unknown kind",
			kind:    EntityKind("unknown"),
			from:    "proposed",
			to:      "accepted",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateTransition(tt.kind, tt.from, tt.to)
			if (err != nil) != tt.wantErr {
				t.Fatalf(
					"ValidateTransition(%q, %q, %q) error = %v, wantErr %v",
					tt.kind,
					tt.from,
					tt.to,
					err,
					tt.wantErr,
				)
			}
		})
	}
}

func TestIncidentLifecycle_ValidTransitions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		from string
		to   string
	}{
		{
			name: "reported to triaged",
			from: string(model.IncidentStatusReported),
			to:   string(model.IncidentStatusTriaged),
		},
		{
			name: "reported to closed",
			from: string(model.IncidentStatusReported),
			to:   string(model.IncidentStatusClosed),
		},
		{
			name: "triaged to investigating",
			from: string(model.IncidentStatusTriaged),
			to:   string(model.IncidentStatusInvestigating),
		},
		{
			name: "triaged to closed",
			from: string(model.IncidentStatusTriaged),
			to:   string(model.IncidentStatusClosed),
		},
		{
			name: "investigating to root-cause-identified",
			from: string(model.IncidentStatusInvestigating),
			to:   string(model.IncidentStatusRootCauseIdentified),
		},
		{
			name: "investigating to closed",
			from: string(model.IncidentStatusInvestigating),
			to:   string(model.IncidentStatusClosed),
		},
		{
			name: "root-cause-identified to mitigated",
			from: string(model.IncidentStatusRootCauseIdentified),
			to:   string(model.IncidentStatusMitigated),
		},
		{
			name: "root-cause-identified to investigating (root cause revised)",
			from: string(model.IncidentStatusRootCauseIdentified),
			to:   string(model.IncidentStatusInvestigating),
		},
		{
			name: "root-cause-identified to closed",
			from: string(model.IncidentStatusRootCauseIdentified),
			to:   string(model.IncidentStatusClosed),
		},
		{
			name: "mitigated to resolved",
			from: string(model.IncidentStatusMitigated),
			to:   string(model.IncidentStatusResolved),
		},
		{
			name: "mitigated to investigating (mitigation incomplete)",
			from: string(model.IncidentStatusMitigated),
			to:   string(model.IncidentStatusInvestigating),
		},
		{
			name: "mitigated to closed",
			from: string(model.IncidentStatusMitigated),
			to:   string(model.IncidentStatusClosed),
		},
		{
			name: "resolved to closed",
			from: string(model.IncidentStatusResolved),
			to:   string(model.IncidentStatusClosed),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if !CanTransition(EntityIncident, tt.from, tt.to) {
				t.Fatalf(
					"CanTransition(%q, %q, %q) = false, want true",
					EntityIncident, tt.from, tt.to,
				)
			}
			if err := ValidateTransition(EntityIncident, tt.from, tt.to); err != nil {
				t.Fatalf(
					"ValidateTransition(%q, %q, %q) unexpected error: %v",
					EntityIncident, tt.from, tt.to, err,
				)
			}
		})
	}
}

func TestIncidentLifecycle_InvalidTransitions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		from string
		to   string
	}{
		{
			name: "reported to investigating (skip triaged)",
			from: string(model.IncidentStatusReported),
			to:   string(model.IncidentStatusInvestigating),
		},
		{
			name: "closed to reported (terminal state)",
			from: string(model.IncidentStatusClosed),
			to:   string(model.IncidentStatusReported),
		},
		{
			name: "resolved to reported (skip back)",
			from: string(model.IncidentStatusResolved),
			to:   string(model.IncidentStatusReported),
		},
		{
			name: "triaged to resolved (skip intermediate)",
			from: string(model.IncidentStatusTriaged),
			to:   string(model.IncidentStatusResolved),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if CanTransition(EntityIncident, tt.from, tt.to) {
				t.Fatalf(
					"CanTransition(%q, %q, %q) = true, want false",
					EntityIncident, tt.from, tt.to,
				)
			}
			if err := ValidateTransition(EntityIncident, tt.from, tt.to); err == nil {
				t.Fatalf(
					"ValidateTransition(%q, %q, %q) expected error, got nil",
					EntityIncident, tt.from, tt.to,
				)
			}
		})
	}
}

func TestIncidentLifecycle_EntryState(t *testing.T) {
	t.Parallel()

	gotState, gotOK := EntryState(EntityIncident)
	if !gotOK {
		t.Fatal("EntryState(incident) ok = false, want true")
	}
	want := string(model.IncidentStatusReported)
	if gotState != want {
		t.Fatalf("EntryState(incident) state = %q, want %q", gotState, want)
	}
}

func TestIncidentLifecycle_TerminalState(t *testing.T) {
	t.Parallel()

	terminal := string(model.IncidentStatusClosed)
	if !IsTerminalState(EntityIncident, terminal) {
		t.Fatalf("IsTerminalState(%q, %q) = false, want true", EntityIncident, terminal)
	}

	nonTerminal := []string{
		string(model.IncidentStatusReported),
		string(model.IncidentStatusTriaged),
		string(model.IncidentStatusInvestigating),
		string(model.IncidentStatusRootCauseIdentified),
		string(model.IncidentStatusMitigated),
		string(model.IncidentStatusResolved),
	}
	for _, state := range nonTerminal {
		if IsTerminalState(EntityIncident, state) {
			t.Fatalf("IsTerminalState(%q, %q) = true, want false", EntityIncident, state)
		}
	}
}

func TestEntryStateOrPanic_ReturnsEntryState(t *testing.T) {
	t.Parallel()

	got := EntryStateOrPanic(EntityEpic)
	if got != "proposed" {
		t.Fatalf("EntryStateOrPanic(%q) = %q, want %q", EntityEpic, got, "proposed")
	}
}

func TestEntryStateOrPanic_PanicsOnUnknownKind(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("EntryStateOrPanic(unknown) did not panic")
		}
	}()

	EntryStateOrPanic(EntityKind("unknown"))
}

func TestValidNextStates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		kind       EntityKind
		from       string
		wantStates []string
	}{
		{
			name:       "plan proposed has designing, cancelled, superseded",
			kind:       EntityPlan,
			from:       "proposed",
			wantStates: []string{"cancelled", "designing", "superseded"},
		},
		{
			name:       "plan active has reviewing, cancelled, superseded",
			kind:       EntityPlan,
			from:       "active",
			wantStates: []string{"cancelled", "reviewing", "superseded"},
		},
		{
			name:       "plan reviewing has active, cancelled, done, superseded",
			kind:       EntityPlan,
			from:       "reviewing",
			wantStates: []string{"active", "cancelled", "done", "superseded"},
		},
		{
			name:       "task queued has duplicate, not-planned, ready",
			kind:       EntityTask,
			from:       string(model.TaskStatusQueued),
			wantStates: []string{"duplicate", "not-planned", "ready"},
		},
		{
			name:       "task active has many transitions",
			kind:       EntityTask,
			from:       string(model.TaskStatusActive),
			wantStates: []string{"blocked", "done", "duplicate", "needs-review", "needs-rework", "not-planned"},
		},
		{
			name:       "task blocked only goes to active",
			kind:       EntityTask,
			from:       string(model.TaskStatusBlocked),
			wantStates: []string{"active"},
		},
		{
			name:       "bug reported has duplicate, triaged",
			kind:       EntityBug,
			from:       string(model.BugStatusReported),
			wantStates: []string{"duplicate", "triaged"},
		},
		{
			name:       "feature proposed phase2",
			kind:       EntityFeature,
			from:       string(model.FeatureStatusProposed),
			wantStates: []string{"cancelled", "designing", "specifying", "superseded"},
		},
		{
			name:       "feature developing includes reviewing (AC-11)",
			kind:       EntityFeature,
			from:       string(model.FeatureStatusDeveloping),
			wantStates: []string{"cancelled", "dev-planning", "reviewing", "superseded"},
		},
		{
			name:       "feature reviewing includes done and needs-rework (AC-12)",
			kind:       EntityFeature,
			from:       string(model.FeatureStatusReviewing),
			wantStates: []string{"cancelled", "done", "needs-rework", "superseded"},
		},
		{
			name:       "feature needs-rework includes developing and reviewing (AC-13)",
			kind:       EntityFeature,
			from:       string(model.FeatureStatusNeedsRework),
			wantStates: []string{"cancelled", "developing", "in-progress", "in-review", "reviewing", "superseded"},
		},
		{
			name:       "terminal task done returns nil",
			kind:       EntityTask,
			from:       string(model.TaskStatusDone),
			wantStates: nil,
		},
		{
			name:       "terminal task not-planned returns nil",
			kind:       EntityTask,
			from:       string(model.TaskStatusNotPlanned),
			wantStates: nil,
		},
		{
			name:       "terminal bug duplicate returns nil",
			kind:       EntityBug,
			from:       string(model.BugStatusDuplicate),
			wantStates: nil,
		},
		{
			name:       "unknown kind returns nil",
			kind:       EntityKind("unknown"),
			from:       "proposed",
			wantStates: nil,
		},
		{
			name:       "unknown state returns nil",
			kind:       EntityTask,
			from:       "nonexistent",
			wantStates: nil,
		},
		{
			name:       "decision proposed has accepted, rejected",
			kind:       EntityDecision,
			from:       string(model.DecisionStatusProposed),
			wantStates: []string{"accepted", "rejected"},
		},
		{
			name:       "decision terminal superseded returns nil",
			kind:       EntityDecision,
			from:       string(model.DecisionStatusSuperseded),
			wantStates: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ValidNextStates(tt.kind, tt.from)

			if tt.wantStates == nil {
				if got != nil {
					t.Errorf("ValidNextStates(%s, %q) = %v, want nil", tt.kind, tt.from, got)
				}
				return
			}

			if len(got) != len(tt.wantStates) {
				t.Fatalf("ValidNextStates(%s, %q) returned %d states %v, want %d states %v",
					tt.kind, tt.from, len(got), got, len(tt.wantStates), tt.wantStates)
			}

			for i, want := range tt.wantStates {
				if got[i] != want {
					t.Errorf("ValidNextStates(%s, %q)[%d] = %q, want %q", tt.kind, tt.from, i, got[i], want)
				}
			}
		})
	}
}

func TestValidateTransition_ErrorContainsValidStates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		kind           EntityKind
		from           string
		to             string
		wantSubstrings []string
	}{
		{
			name: "plan proposed to done shows valid transitions",
			kind: EntityPlan,
			from: "proposed",
			to:   "done",
			wantSubstrings: []string{
				`valid transitions from "proposed"`,
				"designing",
				"cancelled",
				"superseded",
			},
		},
		{
			name: "task queued to done shows valid transitions",
			kind: EntityTask,
			from: string(model.TaskStatusQueued),
			to:   string(model.TaskStatusDone),
			wantSubstrings: []string{
				`valid transitions from "queued"`,
				"ready",
				"not-planned",
				"duplicate",
			},
		},
		{
			name: "feature proposed to done shows valid transitions",
			kind: EntityFeature,
			from: string(model.FeatureStatusProposed),
			to:   string(model.FeatureStatusDone),
			wantSubstrings: []string{
				`valid transitions from "proposed"`,
				"designing",
				"specifying",
			},
		},
		{
			name: "bug reported to closed shows valid transitions",
			kind: EntityBug,
			from: string(model.BugStatusReported),
			to:   string(model.BugStatusClosed),
			wantSubstrings: []string{
				`valid transitions from "reported"`,
				"triaged",
			},
		},
		{
			name: "feature developing to done names reviewing as valid (AC-14)",
			kind: EntityFeature,
			from: string(model.FeatureStatusDeveloping),
			to:   string(model.FeatureStatusDone),
			wantSubstrings: []string{
				`valid transitions from "developing"`,
				"reviewing",
			},
		},
		{
			name: "feature reviewing invalid transition includes valid alternatives (AC-15)",
			kind: EntityFeature,
			from: string(model.FeatureStatusReviewing),
			to:   string(model.FeatureStatusDeveloping),
			wantSubstrings: []string{
				`valid transitions from "reviewing"`,
				"done",
				"needs-rework",
			},
		},
		{
			name: "feature needs-rework invalid transition includes valid alternatives (AC-15)",
			kind: EntityFeature,
			from: string(model.FeatureStatusNeedsRework),
			to:   string(model.FeatureStatusDone),
			wantSubstrings: []string{
				`valid transitions from "needs-rework"`,
				"developing",
				"reviewing",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateTransition(tt.kind, tt.from, tt.to)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			msg := err.Error()
			for _, substr := range tt.wantSubstrings {
				if !strings.Contains(msg, substr) {
					t.Errorf("error message %q does not contain %q", msg, substr)
				}
			}
		})
	}
}

func TestPlanLifecycle_ReviewingTransitions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		from   string
		to     string
		wantOK bool
	}{
		{name: "active to reviewing succeeds", from: "active", to: "reviewing", wantOK: true},
		{name: "reviewing to done succeeds", from: "reviewing", to: "done", wantOK: true},
		{name: "reviewing to active succeeds (rework path)", from: "reviewing", to: "active", wantOK: true},
		{name: "reviewing to superseded succeeds", from: "reviewing", to: "superseded", wantOK: true},
		{name: "reviewing to cancelled succeeds", from: "reviewing", to: "cancelled", wantOK: true},
		{name: "active to done fails (must go through reviewing)", from: "active", to: "done", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateTransition(EntityPlan, tt.from, tt.to)
			if tt.wantOK && err != nil {
				t.Fatalf("expected plan transition %s → %s to succeed, got error: %v", tt.from, tt.to, err)
			}
			if !tt.wantOK && err == nil {
				t.Fatalf("expected plan transition %s → %s to fail, but it succeeded", tt.from, tt.to)
			}
		})
	}
}

func TestPlanLifecycle_ActiveToDoneErrorMessage(t *testing.T) {
	t.Parallel()

	err := ValidateTransition(EntityPlan, "active", "done")
	if err == nil {
		t.Fatal("expected error for plan active → done, got nil")
	}

	msg := err.Error()
	if !strings.Contains(msg, "reviewing") {
		t.Errorf("error message %q does not contain \"reviewing\" in valid transitions list", msg)
	}
}

// TestPlanLifecycle_ActiveCannotSkipReviewing verifies that plans cannot skip
// the reviewing state when transitioning from active to done.
func TestPlanLifecycle_ActiveCannotSkipReviewing(t *testing.T) {
	t.Parallel()

	// active → done should fail
	if CanTransition(EntityPlan, "active", "done") {
		t.Error("CanTransition(plan, active, done) = true; want false")
	}

	err := ValidateTransition(EntityPlan, "active", "done")
	if err == nil {
		t.Error("ValidateTransition(plan, active, done) = nil; want error")
	}

	// active → reviewing should succeed
	if !CanTransition(EntityPlan, "active", "reviewing") {
		t.Error("CanTransition(plan, active, reviewing) = false; want true")
	}
}

// TestPlanLifecycle_FullLifecyclePath verifies the happy path through the
// complete plan lifecycle: proposed → designing → active → reviewing → done.
func TestPlanLifecycle_FullLifecyclePath(t *testing.T) {
	t.Parallel()

	path := []string{"proposed", "designing", "active", "reviewing", "done"}

	for i := 0; i < len(path)-1; i++ {
		from, to := path[i], path[i+1]
		if err := ValidateTransition(EntityPlan, from, to); err != nil {
			t.Errorf("ValidateTransition(plan, %q, %q) = %v; want nil", from, to, err)
		}
	}
}
