package commands

import (
	_ "embed"
	"os"
	"sort"
	"text/template"

	"github.com/hmans/beans/pkg/config"
	"github.com/spf13/cobra"
)

//go:embed prompt.tmpl
var agentPromptTemplate string

// requiredFieldsEntry is one status's required-field list, for deterministic
// (sorted-by-status) template rendering of a map.
type requiredFieldsEntry struct {
	Status string
	Fields []string
}

// promptData holds all data needed to render the prompt template.
type promptData struct {
	Types      []config.TypeConfig
	Statuses   []config.StatusConfig
	Priorities []config.PriorityConfig

	// RequiredFields lists this project's beans.require_fields_on policy,
	// sorted by status; empty when no policy is configured.
	RequiredFields []requiredFieldsEntry
	// CommitField is the configured beans.commit_field (or its default).
	CommitField string
	// CommitFieldGated is true when CommitField is among some status's
	// required fields, i.e. `--commit` resolution actually applies here.
	CommitFieldGated bool
}

var primeCmd = &cobra.Command{
	Use:   "prime",
	Short: "Output instructions for AI coding agents",
	Long:  `Outputs a prompt that primes AI coding agents on how to use the beans CLI to manage project issues.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load the project's actual config so the required-fields policy
		// section reflects what's really configured here, not static prose.
		// If no explicit path is given and no project can be found, stay
		// silent (agents outside a beans project get no prompt).
		var loadedCfg *config.Config
		switch {
		case configPath != "":
			c, err := config.Load(configPath)
			if err != nil {
				return nil // Silently exit on error
			}
			loadedCfg = c
		case beansPath != "":
			cwd, err := os.Getwd()
			if err != nil {
				return nil // Silently exit on error
			}
			c, err := config.LoadFromDirectory(cwd)
			if err != nil {
				return nil // Silently exit on error
			}
			loadedCfg = c
		default:
			cwd, err := os.Getwd()
			if err != nil {
				return nil // Silently exit on error
			}
			configFile, err := config.FindConfig(cwd)
			if err != nil || configFile == "" {
				// No config file found - silently exit
				return nil
			}
			c, err := config.LoadFromDirectory(cwd)
			if err != nil {
				return nil // Silently exit on error
			}
			loadedCfg = c
		}

		tmpl, err := template.New("prompt").Parse(agentPromptTemplate)
		if err != nil {
			return err
		}

		commitField := loadedCfg.GetCommitField()
		commitFieldGated := false

		statuses := make([]string, 0, len(loadedCfg.Beans.RequireFieldsOn))
		for status := range loadedCfg.Beans.RequireFieldsOn {
			statuses = append(statuses, status)
		}
		sort.Strings(statuses)

		requiredFields := make([]requiredFieldsEntry, 0, len(statuses))
		for _, status := range statuses {
			fields := loadedCfg.Beans.RequireFieldsOn[status]
			requiredFields = append(requiredFields, requiredFieldsEntry{Status: status, Fields: fields})
			for _, f := range fields {
				if f == commitField {
					commitFieldGated = true
				}
			}
		}

		data := promptData{
			Types:            config.DefaultTypes,
			Statuses:         config.DefaultStatuses,
			Priorities:       config.DefaultPriorities,
			RequiredFields:   requiredFields,
			CommitField:      commitField,
			CommitFieldGated: commitFieldGated,
		}

		return tmpl.Execute(os.Stdout, data)
	},
}

func RegisterPrimeCmd(root *cobra.Command) {
	root.AddCommand(primeCmd)
}
