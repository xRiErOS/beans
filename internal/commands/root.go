package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hmans/beans/internal/gitutil"
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

// resolveBeansPath determines the beans data directory path.
// Precedence: --beans-path flag > BEANS_PATH env var > beans.anchor > config default.
//
// In worktrees, the CLI uses the worktree's local .beans/ directory.
// beans-serve watches worktree .beans/ dirs and merges changes into
// runtime state, so the UI stays up-to-date without writing to main.
// A repository that wants one shared store instead opts out of that with
// `beans.anchor: repo-root` in its .beans.yml (see resolveAnchoredPath).
func resolveBeansPath(flagPath string, c *config.Config) (string, error) {
	explicitOverride := flagPath != "" || os.Getenv("BEANS_PATH") != ""

	var root string
	if flagPath != "" {
		root = flagPath
	} else if envPath := os.Getenv("BEANS_PATH"); envPath != "" {
		root = envPath
	} else {
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

// Execute runs the given root command and exits on error.
func Execute(rootCmd *cobra.Command) {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
