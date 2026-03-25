package validate

import (
	"testing"

	"kanbanzai/internal/model"
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
			name: "plan active to done",
			kind: EntityPlan,
			from: "active",
			to:   "done",
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
