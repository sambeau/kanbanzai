package kbzinit

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/sambeau/kanbanzai/internal/validate"
)

// initCompleteFile is the sentinel file written as the final step of a
// successful init. Its presence indicates a complete, non-partial init.
const initCompleteFile = ".init-complete"

// SupportedSchemaVersion is the config schema version this binary understands.
const SupportedSchemaVersion = "2"

// LatestReleaseURL is the URL users should visit to download a newer binary.
const LatestReleaseURL = "https://github.com/kanbanzai/kanbanzai/releases/latest"

// Options holds the parsed CLI flags for kanbanzai init.
type Options struct {
	// DocsPath is the list of document root paths supplied via --docs-path.
	DocsPath []string
	// SkipSkills suppresses all skill file installation.
	SkipSkills bool
	// UpdateSkills causes init to only perform the skill update step.
	UpdateSkills bool
	// NonInteractive disables all prompts; errors instead of asking.
	NonInteractive bool
	// Name is the project name supplied via --name flag. If empty, the user
	// is prompted interactively (or the working directory name is used as default).
	Name string
	// SkipWorkDirs suppresses creation of work/ placeholder directories.
	SkipWorkDirs bool
	// SkipMCP suppresses writing .mcp.json and .zed/settings.json.
	SkipMCP bool
	// SkipRoles suppresses installation of context role files.
	SkipRoles bool
	// SkipAgentsMD suppresses writing AGENTS.md and .github/copilot-instructions.md.
	SkipAgentsMD bool
}

// Initializer runs the kanbanzai init command.
type Initializer struct {
	workDir string
	stdin   io.Reader
	stdout  io.Writer
	version string
}

// New creates a new Initializer for the given working directory.
func New(workDir string, stdin io.Reader, stdout io.Writer) *Initializer {
	return &Initializer{workDir: workDir, stdin: stdin, stdout: stdout, version: "dev"}
}

// WithVersion sets the binary version string used when writing skill file
// frontmatter. Call this after New if a non-"dev" version is known.
func (i *Initializer) WithVersion(v string) *Initializer {
	if v != "" {
		i.version = v
	}
	return i
}

// Run executes the init command with the given options.
func (i *Initializer) Run(opts Options) error {
	// Validate mutually exclusive flags before touching anything.
	if opts.UpdateSkills && opts.SkipSkills {
		return fmt.Errorf("--update-skills and --skip-skills are mutually exclusive and cannot be combined")
	}

	// Require a git repository.
	gitRoot, err := FindGitRoot(i.workDir)
	if err != nil {
		return fmt.Errorf("cannot initialise: this directory is not a Git repository. Run 'git init' first, then retry 'kanbanzai init'")
	}

	// Locate (or choose) the .kbz directory.
	kbzDir, kbzExists := i.findKbzDir(gitRoot)
	configPath := filepath.Join(kbzDir, "config.yaml")

	// --update-skills: only update skill files, skip everything else.
	if opts.UpdateSkills {
		fmt.Fprintln(i.stdout, "Updating skill files...")
		if err := i.installSkills(gitRoot); err != nil {
			return err
		}
		// Also update managed role files.
		if err := i.updateManagedRoles(kbzDir); err != nil {
			return err
		}
		fmt.Fprintln(i.stdout, "Skill update complete.")
		return nil
	}

	// Determine new vs existing project state.
	isNew, err := i.isNewProject(gitRoot, kbzExists)
	if err != nil {
		return err
	}

	if isNew {
		return i.runNewProject(opts, kbzDir, configPath)
	}
	return i.runExistingProject(opts, kbzDir, configPath, kbzExists)
}

// findKbzDir searches from workDir up to gitRoot for an existing .kbz directory.
// Returns the path to use and whether an existing one was found.
func (i *Initializer) findKbzDir(gitRoot string) (string, bool) {
	absWork, err := filepath.Abs(i.workDir)
	if err != nil {
		absWork = i.workDir
	}
	absGitRoot, err := filepath.Abs(gitRoot)
	if err != nil {
		absGitRoot = gitRoot
	}

	current := absWork
	for {
		candidate := filepath.Join(current, ".kbz")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, true
		}
		if current == absGitRoot {
			break
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	// No existing .kbz found — will create at the git root.
	return filepath.Join(absGitRoot, ".kbz"), false
}

// isNewProject returns true when the project has no prior Kanbanzai state and
// no git commits, meaning init should run non-interactively with defaults.
func (i *Initializer) isNewProject(gitRoot string, kbzExists bool) (bool, error) {
	if kbzExists {
		return false, nil
	}

	hasCommits, err := HasCommits(gitRoot)
	if err != nil {
		return false, fmt.Errorf("check git history: %w", err)
	}
	return !hasCommits, nil
}

// runNewProject handles the new project path: write config, create work/ dirs,
// and install skill files. Uses a temp-dir-then-rename approach for atomicity:
// if any step fails, the partial .kbz/ directory is removed.
func (i *Initializer) runNewProject(opts Options, kbzDir, configPath string) error {
	roots := DefaultDocumentRoots()
	baseDir := filepath.Dir(kbzDir)

	// Resolve the project name before creating any directories.
	name, err := i.resolveProjectName(opts)
	if err != nil {
		return err
	}

	// Write config to a temporary directory first.
	tmpDir, err := os.MkdirTemp(baseDir, ".kbz-tmp-*")
	if err != nil {
		return fmt.Errorf("cannot create temporary directory: check that the current user has write access to this directory")
	}

	// Clean up the temp dir on any failure.
	success := false
	defer func() {
		if !success {
			os.RemoveAll(tmpDir)
			// Also remove the final kbzDir if it was partially created.
			os.RemoveAll(kbzDir)
		}
	}()

	if err := WriteInitConfig(tmpDir, name, roots); err != nil {
		return err
	}

	// Atomically rename temp dir to final .kbz/.
	if err := os.Rename(tmpDir, kbzDir); err != nil {
		return fmt.Errorf("cannot create '.kbz/' directory: %w", err)
	}
	fmt.Fprintf(i.stdout, "Created %s\n", configPath)

	if !opts.SkipWorkDirs {
		if err := i.createWorkDirs(baseDir, roots); err != nil {
			return err
		}
		if err := i.writeWorkReadme(baseDir); err != nil {
			return err
		}
	}

	if !opts.SkipSkills {
		if err := i.installSkills(baseDir); err != nil {
			return err
		}
	}

	if !opts.SkipMCP {
		if err := i.writeMCPConfig(baseDir); err != nil {
			return err
		}
		if err := i.writeZedConfig(baseDir, true); err != nil {
			return err
		}
	}

	if !opts.SkipAgentsMD {
		if err := i.writeAgentsMD(baseDir); err != nil {
			return err
		}
		if err := i.writeCopilotInstructions(baseDir); err != nil {
			return err
		}
	}

	if !opts.SkipRoles {
		if err := i.installRoles(kbzDir); err != nil {
			return err
		}
	}

	// Write the sentinel file as the very last step.
	sentinelPath := filepath.Join(kbzDir, initCompleteFile)
	if err := os.WriteFile(sentinelPath, []byte{}, 0o644); err != nil {
		return fmt.Errorf("cannot write sentinel '%s': %w", sentinelPath, err)
	}

	success = true
	fmt.Fprintln(i.stdout, "Initialisation complete.")
	return nil
}

// runExistingProject handles the existing project path: validate/write config
// (if absent), and install/update skill files.
// kbzExisted reports whether .kbz/ was present before this run. When false
// (first-time kanbanzai init on a project that already has commits), work/
// directories, work/README.md, and .zed/settings.json are created — the same
// behaviour as runNewProject.
func (i *Initializer) runExistingProject(opts Options, kbzDir, configPath string, kbzExisted bool) error {
	baseDir := filepath.Dir(kbzDir)

	// Detect partial init: .kbz/ exists but sentinel is absent.
	sentinelPath := filepath.Join(kbzDir, initCompleteFile)
	if _, err := os.Stat(sentinelPath); os.IsNotExist(err) {
		fmt.Fprintf(i.stdout, "Warning: previous init appears incomplete (no '%s' sentinel). Re-running init to complete setup.\n", initCompleteFile)
	}

	fmt.Fprintln(i.stdout, "Existing project detected.")

	// workRoots holds the document roots written to config on this run.
	// It is only set when a new config is created (i.e. on a first-time init),
	// and is used below to create the matching work/ directories.
	var workRoots []DocumentRoot

	// Check whether config.yaml already exists.
	configExists := false
	if _, err := os.Stat(configPath); err == nil {
		configExists = true
	}

	if configExists {
		// Validate the existing config.
		if err := i.validateExistingConfig(opts, kbzDir, configPath); err != nil {
			return err
		}
	} else {
		// No config — we need document roots.
		roots, err := i.resolveDocumentRoots(opts)
		if err != nil {
			return err
		}
		workRoots = roots

		name, err := i.resolveProjectName(opts)
		if err != nil {
			return err
		}
		if err := WriteInitConfig(kbzDir, name, roots); err != nil {
			return err
		}
		fmt.Fprintf(i.stdout, "Created %s\n", configPath)
	}

	// On a first-time init (no prior .kbz/), create work/ directories and README
	// using the same roots that were written to config. On re-runs, leave existing
	// work/ alone.
	if !kbzExisted && !opts.SkipWorkDirs && len(workRoots) > 0 {
		if err := i.createWorkDirs(baseDir, workRoots); err != nil {
			return err
		}
		if err := i.writeWorkReadme(baseDir); err != nil {
			return err
		}
	}

	if !opts.SkipSkills {
		if err := i.installSkills(baseDir); err != nil {
			return err
		}
	}

	if !opts.SkipMCP {
		if err := i.writeMCPConfig(baseDir); err != nil {
			return err
		}
		if err := i.writeZedConfig(baseDir, !kbzExisted); err != nil {
			return err
		}
	}

	if !opts.SkipAgentsMD {
		if err := i.writeAgentsMD(baseDir); err != nil {
			return err
		}
		if err := i.writeCopilotInstructions(baseDir); err != nil {
			return err
		}
	}

	if !opts.SkipRoles {
		if err := i.installRoles(kbzDir); err != nil {
			return err
		}
	}

	// Write/refresh the sentinel file.
	if err := os.WriteFile(sentinelPath, []byte{}, 0o644); err != nil {
		return fmt.Errorf("cannot write sentinel '%s': %w", sentinelPath, err)
	}

	fmt.Fprintln(i.stdout, "Done.")
	return nil
}

// validateExistingConfig reads the existing config.yaml and enforces schema
// version and validity rules.
func (i *Initializer) validateExistingConfig(opts Options, kbzDir, configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("cannot read '%s': check that the current user has read access to this directory", configPath)
	}

	// First check: is it valid YAML at all?
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return i.handleInvalidConfig(opts, kbzDir, configPath, "the file contains invalid YAML")
	}
	if raw == nil {
		return i.handleInvalidConfig(opts, kbzDir, configPath, "the file is empty or contains no recognisable configuration")
	}

	// Extract and check the schema version.
	versionRaw := raw["version"]
	configVersion := fmt.Sprintf("%v", versionRaw)
	if configVersion == "<nil>" || configVersion == "" {
		configVersion = "0"
	}

	if isNewerSchemaVersion(configVersion, SupportedSchemaVersion) {
		return fmt.Errorf(
			"'%s' was written by a newer version of Kanbanzai (config schema version %s, this binary supports %s). "+
				"Download the latest release from %s",
			configPath, configVersion, SupportedSchemaVersion, LatestReleaseURL,
		)
	}

	// Config exists and is at a supported schema version — nothing to do.
	fmt.Fprintf(i.stdout, "Config already exists at %s (schema version %s).\n", configPath, configVersion)
	return nil
}

// handleInvalidConfig deals with a config.yaml that cannot be parsed. In
// interactive mode it prompts the user to overwrite; in --non-interactive mode
// it overwrites without prompting.
func (i *Initializer) handleInvalidConfig(opts Options, kbzDir, configPath, reason string) error {
	msg := fmt.Sprintf("'%s' is not a valid Kanbanzai config file: %s.", configPath, reason)

	// Use the provided name or fall back to the working directory basename.
	// We do not prompt here — the user is already being asked about overwriting.
	name := opts.Name
	if name == "" {
		name = filepath.Base(i.workDir)
	}

	if opts.NonInteractive {
		// Overwrite silently in non-interactive mode.
		fmt.Fprintln(i.stdout, msg)
		fmt.Fprintln(i.stdout, "Overwriting with a fresh default config (--non-interactive).")
		roots := DefaultDocumentRoots()
		if len(opts.DocsPath) > 0 {
			roots = docPathsToRoots(opts.DocsPath)
		}
		return WriteInitConfig(kbzDir, name, roots)
	}

	// Interactive: ask the user.
	fmt.Fprintln(i.stdout, msg)
	fmt.Fprint(i.stdout, "Overwrite with a fresh default config? [y/N]: ")

	scanner := bufio.NewScanner(i.stdin)
	answer := ""
	if scanner.Scan() {
		answer = strings.TrimSpace(strings.ToLower(scanner.Text()))
	}

	if answer != "y" && answer != "yes" {
		return fmt.Errorf("config file is invalid and overwrite was declined. Fix '%s' manually or re-run with --non-interactive to overwrite automatically", configPath)
	}

	roots := DefaultDocumentRoots()
	if len(opts.DocsPath) > 0 {
		roots = docPathsToRoots(opts.DocsPath)
	}
	return WriteInitConfig(kbzDir, name, roots)
}

// resolveDocumentRoots determines the document roots for an existing project
// that has no config.yaml yet.
func (i *Initializer) resolveDocumentRoots(opts Options) ([]DocumentRoot, error) {
	if len(opts.DocsPath) > 0 {
		return docPathsToRoots(opts.DocsPath), nil
	}

	if opts.NonInteractive {
		return nil, fmt.Errorf(
			"--non-interactive requires --docs-path when no config.yaml exists: " +
				"the document root cannot be determined automatically. " +
				"Re-run with --docs-path <path> to specify where your workflow documents live",
		)
	}

	// Interactive prompt.
	fmt.Fprintln(i.stdout, "No config.yaml found. Where are your workflow documents?")
	fmt.Fprint(i.stdout, "Document root path (press Enter for standard work/ layout): ")

	scanner := bufio.NewScanner(i.stdin)
	if scanner.Scan() {
		path := strings.TrimSpace(scanner.Text())
		if path == "" {
			return DefaultDocumentRoots(), nil
		}
		return docPathsToRoots([]string{path}), nil
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}
	return nil, fmt.Errorf("no document root path provided")
}

// createWorkDirs creates the work/ placeholder directories with .gitkeep files.
// baseDir is typically the git root. It skips directories that already exist.
// resolveProjectName returns the project name to use when writing config.yaml.
// Priority: explicit opt > interactive prompt (with default = workDir basename).
// Returns an error in non-interactive mode when no name is supplied.
func (i *Initializer) resolveProjectName(opts Options) (string, error) {
	if opts.Name != "" {
		name, err := validate.ValidateName(opts.Name)
		if err != nil {
			return "", err
		}
		return name, nil
	}
	defaultName := filepath.Base(i.workDir)
	if opts.NonInteractive {
		return "", fmt.Errorf("--name is required in non-interactive mode")
	}
	fmt.Fprintf(i.stdout, "Project name [%s]: ", defaultName)
	scanner := bufio.NewScanner(i.stdin)
	if scanner.Scan() {
		if input := strings.TrimSpace(scanner.Text()); input != "" {
			name, err := validate.ValidateName(input)
			if err != nil {
				return "", err
			}
			return name, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("reading project name: %w", err)
	}
	name, err := validate.ValidateName(defaultName)
	if err != nil {
		return "", err
	}
	return name, nil
}

func (i *Initializer) createWorkDirs(baseDir string, roots []DocumentRoot) error {
	for _, root := range roots {
		dir := filepath.Join(baseDir, root.Path)
		if _, err := os.Stat(dir); err == nil {
			// Already exists — leave it alone.
			continue
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("cannot create directory '%s': check that the current user has write access to this directory", dir)
		}
		gitkeep := filepath.Join(dir, ".gitkeep")
		if err := os.WriteFile(gitkeep, []byte{}, 0o644); err != nil {
			return fmt.Errorf("cannot write '%s': check that the current user has write access to this directory", gitkeep)
		}
		fmt.Fprintf(i.stdout, "Created %s/.gitkeep\n", root.Path)
	}
	return nil
}

// writeWorkReadme writes work/README.md to baseDir if it does not already exist.
func (i *Initializer) writeWorkReadme(baseDir string) error {
	readmePath := filepath.Join(baseDir, "work", "README.md")
	if _, err := os.Stat(readmePath); err == nil {
		// Already exists — leave it alone.
		return nil
	}
	// If work/ itself doesn't exist (custom --docs-path without a work/ root),
	// skip silently — the README is only meaningful alongside the standard layout.
	if _, err := os.Stat(filepath.Join(baseDir, "work")); os.IsNotExist(err) {
		return nil
	}
	content := `# work/

Workflow documents for this project. Register all documents with kanbanzai after creation.

| Directory | Type | Contents |
|---|---|---|
| ` + "`design/`" + ` | design | Architecture decisions, technical vision, policies |
| ` + "`spec/`" + ` | specification | Acceptance criteria and binding contracts |
| ` + "`plan/`" + ` | plan | Project planning: roadmaps, scope, decision logs |
| ` + "`dev/`" + ` | dev-plan | Feature implementation plans and task breakdowns |
| ` + "`research/`" + ` | research | Analysis, exploration, background reading |
| ` + "`report/`" + ` | report | Audit reports, post-mortems, general reports |
| ` + "`review/`" + ` | report | Feature and plan review reports |
| ` + "`retro/`" + ` | retrospective | Retrospective synthesis documents |

AI agents: see the ` + "`kanbanzai-documents`" + ` skill for registration instructions.
`
	if err := os.WriteFile(readmePath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("cannot write 'work/README.md': check that the current user has write access to this directory")
	}
	fmt.Fprintln(i.stdout, "Created work/README.md")
	return nil
}

// docPathsToRoots converts a list of path strings into DocumentRoot entries,
// inferring the default_type from the path basename.
func docPathsToRoots(paths []string) []DocumentRoot {
	roots := make([]DocumentRoot, 0, len(paths))
	for _, p := range paths {
		roots = append(roots, DocumentRoot{
			Path:        p,
			DefaultType: InferDocType(p),
		})
	}
	return roots
}

// isNewerSchemaVersion returns true if configVersion is strictly greater than
// binaryVersion, using integer comparison when both parse as integers.
func isNewerSchemaVersion(configVersion, binaryVersion string) bool {
	cv, err1 := strconv.Atoi(configVersion)
	bv, err2 := strconv.Atoi(binaryVersion)
	if err1 != nil || err2 != nil {
		// Fall back to lexicographic comparison.
		return configVersion > binaryVersion
	}
	return cv > bv
}
