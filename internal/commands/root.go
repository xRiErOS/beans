package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hmans/beans/internal/gitutil"
	"github.com/hmans/beans/internal/output"
	"github.com/hmans/beans/internal/ui"
	"github.com/hmans/beans/pkg/beancore"
	"github.com/hmans/beans/pkg/config"
	"github.com/spf13/cobra"
)

var core *beancore.Core
var cfg *config.Config
var beansPath string
var configPath string

// NewRootCmd creates the root cobra command with shared persistent flags
// and core initialization logic.
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "beans",
		Short: "A file-based issue tracker for AI-first workflows",
		// Reporting a failure is done once, in reportExecutionError. Cobra
		// otherwise prints the error itself and follows it with the whole
		// usage block — measured at 33 lines behind a one-line failure, and
		// identical for a runtime outcome and a typo'd flag. Both switches
		// are conjunctions over command and root, so setting them here
		// covers every subcommand of all three binaries.
		SilenceErrors: true,
		SilenceUsage:  true,
		Long: `Beans is a lightweight issue tracker that stores issues as markdown files.
Track your work alongside your code and supercharge your coding agent with
a full view of your project.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Skip core initialization for init, prime, and version commands
			if cmd.Name() == "init" || cmd.Name() == "prime" || cmd.Name() == "version" {
				return nil
			}

			var err error

			// Load configuration
			if configPath != "" {
				cfg, err = config.Load(configPath)
				if err != nil {
					return fmt.Errorf("loading config from %s: %w", configPath, err)
				}
			} else {
				cwd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("getting current directory: %w", err)
				}
				cfg, err = config.LoadFromDirectory(cwd)
				if err != nil {
					return fmt.Errorf("loading config: %w", err)
				}
			}

			// The theme is process wide and must be in force before anything
			// renders, so this is the single place that applies it.
			ui.SetTheme(cfg.GetTheme())

			feedTypeTables(cfg)
			feedStatusPriorityTables(cfg)

			root, err := resolveBeansPath(beansPath, cfg)
			if err != nil {
				return err
			}

			core = beancore.New(root, cfg)
			if err := core.Load(); err != nil {
				return fmt.Errorf("loading beans: %w", err)
			}

			return nil
		},
	}

	rootCmd.PersistentFlags().StringVar(&beansPath, "beans-path", "", "Path to data directory (overrides config and BEANS_PATH env var)")
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "Path to config file (default: searches upward for .beans.yml)")

	return rootCmd
}

// feedTypeTables fills internal/ui's process-wide type-short and
// type-column-width tables from the merged type list. internal/ui must not
// import pkg/config, so this is the one place - analogous to a SetTheme
// call - where the config-derived values cross into it.
func feedTypeTables(c *config.Config) {
	shorts := make(map[string]string)
	longest := 0
	for _, t := range c.TypeList() {
		shorts[t.Name] = c.ShortOf(t.Name)
		// Cells, not bytes: this value becomes a column width, and a type
		// named "Änderung" is 8 cells in 9 bytes.
		if w := ui.DisplayWidth(t.Name); w > longest {
			longest = w
		}
	}
	ui.SetTypeShorts(shorts)
	// One trailing space keeps the column from touching the next one, which is
	// what the old literal 10 did for "milestone" at nine characters.
	ui.SetTypeColumnWidths(3, longest+1)
}

// feedStatusPriorityTables fills internal/ui's process-wide status-short and
// priority-symbol tables from the merged status/priority lists. Same
// reasoning as feedTypeTables: internal/ui must not import pkg/config, so
// this is where the config-derived values cross into it.
func feedStatusPriorityTables(c *config.Config) {
	shorts := make(map[string]string, len(c.StatusList()))
	for _, s := range c.StatusList() {
		shorts[s.Name] = c.ShortOfStatus(s.Name)
	}
	ui.SetStatusShorts(shorts)

	symbols := make(map[string]string, len(c.PriorityList()))
	for _, p := range c.PriorityList() {
		symbols[p.Name] = p.Symbol
	}
	ui.SetPrioritySymbols(symbols)
}

// resolveBeansPath determines the beans data directory path.
// Precedence: --beans-path flag > .beans.yml (found upward or named via
// --config) > BEANS_PATH env var > default.
//
// A .beans.yml is the repository's own declaration of where its store lives,
// so it outranks BEANS_PATH: the env var is commonly exported by direnv and
// inherited into unrelated repositories, where honouring it would silently
// redirect every call to a foreign store. Without such a declaration the env
// var is still the next-best answer. The anchor is part of that declaration
// and is applied while resolving it (see resolveAnchoredPath).
//
// In worktrees, the CLI uses the worktree's local .beans/ directory.
// beans-serve watches worktree .beans/ dirs and merges changes into
// runtime state, so the UI stays up-to-date without writing to main.
// A repository that wants one shared store instead opts out of that with
// `beans.anchor: repo-root` in its .beans.yml (see resolveAnchoredPath).
func resolveBeansPath(flagPath string, c *config.Config) (string, error) {
	envPath := os.Getenv("BEANS_PATH")
	// A found .beans.yml, and only a found one, demotes the env var. A
	// defaulted config carries no declaration to honour.
	useEnv := envPath != "" && (c == nil || c.ConfigFile() == "")
	explicitOverride := flagPath != "" || useEnv

	var root string
	switch {
	case flagPath != "":
		root = flagPath
	case useEnv:
		root = envPath
	default:
		anchored, err := resolveAnchoredPath(c)
		if err != nil {
			return "", err
		}
		root = anchored
	}

	if info, statErr := os.Stat(root); statErr != nil || !info.IsDir() {
		// A missing/empty live directory next to a non-empty .beans.bak-*
		// sibling is not "no store here" -- it's an interrupted stageAndSwap
		// (rename.go: the swap is two os.Rename calls with .beans absent in
		// between). Core.Load's own detectOrphanBackup repairs or warns
		// about exactly that case, but only gets the chance to run if this
		// precheck doesn't fail first.
		if beancore.HasOrphanBackup(root) {
			return root, nil
		}
		if explicitOverride {
			return "", fmt.Errorf("beans path does not exist or is not a directory: %s", root)
		}
		return "", fmt.Errorf("no .beans directory found at %s (run 'beans init' to create one)", root)
	}

	return root, nil
}

// resolveAnchoredPath applies the config's beans.anchor setting.
//
// With AnchorRepoRoot, a call from a secondary worktree resolves to the main
// worktree's store, so worktrees of one repository share it. Outside a git repo
// — and in the main worktree, where there is nothing to redirect — the config
// file's own directory stays the anchor.
func resolveAnchoredPath(c *config.Config) (string, error) {
	switch c.Beans.Anchor {
	case "":
		return c.ResolveBeansPath(), nil
	case config.AnchorRepoRoot:
		mainRoot, isSecondary := gitutil.MainWorktreeRoot(c.ConfigDir())
		if !isSecondary {
			return c.ResolveBeansPath(), nil
		}
		return filepath.Join(mainRoot, c.Beans.Path), nil
	default:
		return "", config.ValidateAnchor(c.Beans.Anchor)
	}
}

// reportExecutionError writes the one shape a failure takes: the error line
// and a pointer to the manual, both on stderr. It is cobra's own idiom for an
// unknown command (command.go:1124-1133), adopted here for every error class
// rather than inventing a second policy.
//
// The suppression is keyed on the error already carrying a machine-readable
// document, not on --json being present. That way a --json consumer gets
// exactly one artifact, while an error raised before a command reached the
// output layer — a broken config, a missing store — still reaches the user.
func reportExecutionError(cmd *cobra.Command, err error) {
	if output.Emitted(err) {
		return
	}
	cmd.PrintErrln(cmd.ErrPrefix(), err.Error())
	cmd.PrintErrf("Run '%v --help' for usage.\n", cmd.CommandPath())
}

// Execute runs the given root command and exits on error.
func Execute(rootCmd *cobra.Command) {
	cmd, err := rootCmd.ExecuteC()
	if err != nil {
		reportExecutionError(cmd, err)
		os.Exit(1)
	}
}
