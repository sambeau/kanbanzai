package main

import (
	"fmt"

	"kanbanzai/internal/kbzinit"
)

const initUsageText = `kanbanzai init [flags]

Initialise a Git repository for use with Kanbanzai.

Creates .kbz/config.yaml with a default prefix registry and document roots.
For new projects (no commits, no .kbz/) also creates work/ placeholder
directories. Skill file installation is managed separately.

Flags:
  --docs-path <path>    Document root path (repeatable). Suppresses the
                        interactive prompt on existing projects. Each value
                        becomes one entry in documents.roots.
                        Default: work/design, work/spec, work/dev,
                                 work/research, work/reports (new projects)

  --non-interactive     Use defaults and error instead of prompting.
                        Requires --docs-path if an existing project has no
                        config.yaml.
                        Default: false

  --update-skills       Perform only the skill update step. Skips config
                        writing and work/ directory creation.
                        Mutually exclusive with --skip-skills.
                        Default: false

  --skip-skills         Do not install or update any skill files.
                        Takes precedence over --update-skills.
                        Mutually exclusive with --update-skills.
                        Default: false

  --skip-work-dirs      Do not create work/ placeholder directories.
                        Has no effect on existing projects.
                        Default: false

Example:
  kanbanzai init
  kanbanzai init --docs-path work/docs --non-interactive
  kanbanzai init --skip-skills
  kanbanzai init --update-skills
`

func runInit(args []string, deps dependencies) error {
	var opts kbzinit.Options

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--docs-path":
			if i+1 >= len(args) {
				return fmt.Errorf("--docs-path requires a value\n\n%s", initUsageText)
			}
			i++
			opts.DocsPath = append(opts.DocsPath, args[i])
		case "--non-interactive":
			opts.NonInteractive = true
		case "--update-skills":
			opts.UpdateSkills = true
		case "--skip-skills":
			opts.SkipSkills = true
		case "--skip-work-dirs":
			opts.SkipWorkDirs = true
		case "-h", "--help":
			fmt.Fprint(deps.stdout, initUsageText)
			return nil
		default:
			return fmt.Errorf("unknown flag %q\n\n%s", args[i], initUsageText)
		}
	}

	initializer := kbzinit.New(".", deps.stdin, deps.stdout)
	return initializer.Run(opts)
}
