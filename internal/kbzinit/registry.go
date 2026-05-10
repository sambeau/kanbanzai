package kbzinit

// ArtifactKind identifies the category of a file the installer manages.
type ArtifactKind string

const (
	WorkflowSkill       ArtifactKind = "workflow-skill"
	TaskSkill           ArtifactKind = "task-skill"
	Role                ArtifactKind = "role"
	AgentsMd            ArtifactKind = "agents-md"
	CopilotInstructions ArtifactKind = "copilot-instructions"
	StageBindings       ArtifactKind = "stage-bindings"
)

// VersionKind controls which version comparison strategy MarkerSpec uses.
type VersionKind string

const (
	IntCounter VersionKind = "int-counter"
	Semver     VersionKind = "semver"
)

// Decision is the outcome returned by compareManaged when deciding what
// action to take for an existing file.
type Decision int

const (
	Create    Decision = iota // file absent — create it
	Overwrite                 // file present, managed, older version — overwrite
	NoOp                      // file present, managed, same or newer version — leave alone
	WarnSkip                  // file present, not managed or unparseable — warn and skip
)

// MarkerSpec describes how to detect and parse the version marker in a
// managed file. Comment is the marker prefix (e.g. "# kanbanzai-managed:"),
// VersionKind selects the parsing strategy, and CurrentValue is the version
// the running binary embeds.
type MarkerSpec struct {
	Comment      string
	VersionKind  VersionKind
	CurrentValue string
}

// Artifact describes a single file that the installer manages. Both
// Required and Optional may be set to reflect the design §5.1 semantics:
// Required means the install fails if the artifact cannot be written;
// Optional means the artifact is skipped silently when its embed path is
// absent.
type Artifact struct {
	Name        string
	Kind        ArtifactKind
	EmbedPath   string
	InstallPath string
	Required    bool
	Optional    bool
	Marker      MarkerSpec
}
