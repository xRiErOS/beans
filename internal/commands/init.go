package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/hmans/beans/internal/gitutil"
	"github.com/hmans/beans/internal/output"
	"github.com/hmans/beans/pkg/beancore"
	"github.com/hmans/beans/pkg/config"
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
			return fmt.Errorf("--profile writes a .beans.yml, which --beans-path skips; use one or the other")
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
				types, ok := config.ProfileTypes(initProfile)
				if !ok {
					if initJSON {
						return output.Error(output.ErrValidation,
							fmt.Sprintf("unknown profile %q (must be %s)", initProfile, strings.Join(config.ProfileNames(), ", ")))
					}
					return fmt.Errorf("unknown profile %q (must be %s)", initProfile, strings.Join(config.ProfileNames(), ", "))
				}
				defaultCfg.Types = types
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
