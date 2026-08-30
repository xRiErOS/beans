package commands

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xRiErOS/beans/internal/gitutil"
	"github.com/xRiErOS/beans/internal/output"
	"github.com/xRiErOS/beans/pkg/beancore"
	"github.com/xRiErOS/beans/pkg/config"
)

var initJSON bool
var initProfile string

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a beans project",
	Long:  `Creates a .beans directory and .beans.yml config file in the current directory.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var projectDir string
		var beansDir string
		var dirName string

		if beansPath != "" && initProfile != "" {
			msg := "--profile writes a .beans.yml, which --beans-path skips; use one or the other"
			if initJSON {
				return output.Error(output.ErrValidation, msg)
			}
			return errors.New(msg)
		}

		// Resolve the profile before any side effect (beancore.Init creates
		// .beans/ and its .gitignore): a typo must not leave a half-initialised
		// project behind.
		var profileTypes []config.TypeOverride
		if initProfile != "" {
			types, ok := config.ProfileTypes(initProfile)
			if !ok {
				msg := fmt.Sprintf("unknown profile %q (must be %s)", initProfile, strings.Join(config.ProfileNames(), ", "))
				if initJSON {
					return output.Error(output.ErrValidation, msg)
				}
				return errors.New(msg)
			}
			profileTypes = types
		}

		if beansPath != "" {
			// Use explicit path for beans directory
			beansDir = beansPath
			// Create the directory using Core.Init to set up .gitignore
			core := beancore.New(beansDir, nil)
			if err := core.Init(); err != nil {
				if initJSON {
					return output.Error(output.ErrFileError, err.Error())
				}
				return fmt.Errorf("failed to create directory: %w", err)
			}
			// Skip creating .beans.yml when --beans-path is explicit:
			// the path is already known, and writing a config to the parent
			// directory could pollute unrelated locations (e.g. /tmp).
		} else {
			// Use current working directory
			dir, err := os.Getwd()
			if err != nil {
				if initJSON {
					return output.Error(output.ErrFileError, err.Error())
				}
				return err
			}

			if err := beancore.Init(dir); err != nil {
				if initJSON {
					return output.Error(output.ErrFileError, err.Error())
				}
				return fmt.Errorf("failed to initialize: %w", err)
			}

			projectDir = dir
			beansDir = filepath.Join(dir, ".beans")
			dirName = filepath.Base(dir)

			// Create default config file with directory name as prefix
			// Config is saved at project root (not inside .beans/)
			defaultCfg := config.DefaultWithPrefix(dirName + "-")
			defaultCfg.Project.Name = dirName
			if initProfile != "" {
				defaultCfg.Types = profileTypes
				// A profile gives a project exactly its own types: switch off
				// the merge onto the built-in defaults, not just override them.
				defaultCfg.TypesExclusive = true
				// The default type must exist in the chosen profile: todo has no
				// milestone, complex has no plain feature leaf.
				defaultCfg.Beans.DefaultType = "task"
			}
			defaultCfg.SetConfigDir(projectDir)

			// Auto-detect the remote's default branch if we're in a git repo
			if baseRef, ok := gitutil.DefaultRemoteBranch(projectDir, "origin"); ok {
				defaultCfg.Worktree.BaseRef = baseRef
			}
			if err := defaultCfg.Save(projectDir); err != nil {
				if initJSON {
					return output.Error(output.ErrFileError, err.Error())
				}
				return fmt.Errorf("failed to create config: %w", err)
			}
		}

		if initJSON {
			return output.SuccessInit(beansDir)
		}

		fmt.Println("Initialized beans project")
		return nil
	},
}

func RegisterInitCmd(root *cobra.Command) {
	initCmd.Flags().BoolVar(&initJSON, "json", false, "Output as JSON")
	initCmd.Flags().StringVar(&initProfile, "profile", "",
		fmt.Sprintf("Type profile to write out (%s)", strings.Join(config.ProfileNames(), ", ")))
	root.AddCommand(initCmd)
}
