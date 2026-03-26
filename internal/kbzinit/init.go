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
)

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
	// SkipWorkDirs suppresses creation of work/ placeholder directories.
	SkipWorkDirs bool
}

// Initializer runs the kanbanzai init command.
type Initializer struct {
	workDir string
	stdin   io.Reader
	stdout  io.Writer
}

// New creates a new Initializer for the given working directory.
func New(workDir string, stdin io.Reader, stdout io.Writer) *Initializer {
	return &Initializer{workDir: workDir, stdin: stdin, stdout: stdout}
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
		// TODO(FEAT-01KMKRQSD1TKK): embed and install skill files here
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
	return i.runExistingProject(opts, kbzDir, configPath)
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
// and stub in skill installation.
func (i *Initializer) runNewProject(opts Options, kbzDir, configPath string) error {
	roots := DefaultDocumentRoots()

	if err := WriteInitConfig(kbzDir, roots); err != nil {
		return err
	}
	fmt.Fprintf(i.stdout, "Created %s\n", configPath)

	if !opts.SkipWorkDirs {
		if err := i.createWorkDirs(filepath.Dir(kbzDir), roots); err != nil {
			return err
		}
	}

	if !opts.SkipSkills {
		// TODO(FEAT-01KMKRQSD1TKK): embed and install skill files here
	}

	fmt.Fprintln(i.stdout, "Initialisation complete.")
	return nil
}

// runExistingProject handles the existing project path: validate/write config
// (if absent), and stub in skill updates.
func (i *Initializer) runExistingProject(opts Options, kbzDir, configPath string) error {
	fmt.Fprintln(i.stdout, "Existing project detected.")

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

		if err := WriteInitConfig(kbzDir, roots); err != nil {
			return err
		}
		fmt.Fprintf(i.stdout, "Created %s\n", configPath)
	}

	if !opts.SkipSkills {
		// TODO(FEAT-01KMKRQSD1TKK): embed and install skill files here
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
	versionRaw, _ := raw["version"]
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

	if opts.NonInteractive {
		// Overwrite silently in non-interactive mode.
		fmt.Fprintln(i.stdout, msg)
		fmt.Fprintln(i.stdout, "Overwriting with a fresh default config (--non-interactive).")
		roots := DefaultDocumentRoots()
		if len(opts.DocsPath) > 0 {
			roots = docPathsToRoots(opts.DocsPath)
		}
		return WriteInitConfig(kbzDir, roots)
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
	return WriteInitConfig(kbzDir, roots)
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
	fmt.Fprint(i.stdout, "Document root path (e.g. work/docs): ")

	scanner := bufio.NewScanner(i.stdin)
	if scanner.Scan() {
		path := strings.TrimSpace(scanner.Text())
		if path == "" {
			return nil, fmt.Errorf("document root path cannot be empty")
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
