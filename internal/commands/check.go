package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/hmans/beans/internal/gitutil"
	"github.com/hmans/beans/pkg/bean"
	"github.com/hmans/beans/pkg/beancore"
	"github.com/hmans/beans/pkg/config"
	"github.com/hmans/beans/internal/ui"
)

// unknownTypeBeans returns every bean whose type the current configuration
// does not carry. A config edit or a profile switch can leave beans behind
// that no longer match any known type.
func unknownTypeBeans(allBeans []*bean.Bean) []*bean.Bean {
	var out []*bean.Bean
	for _, b := range allBeans {
		if b.Type == "" {
			continue
		}
		if !cfg.IsValidType(b.Type) {
			out = append(out, b)
		}
	}
	return out
}

var (
	checkJSON   bool
	checkFix    bool
	checkStrict bool
)

type checkResult struct {
	Success        bool                      `json:"success"`
	ConfigErrors   []string                  `json:"config_errors"`
	BeanIssues     *beancore.LinkCheckResult `json:"bean_issues,omitempty"`
	Fixed          int                       `json:"fixed,omitempty"`
	PolicyWarnings []string                  `json:"policy_warnings,omitempty"`
}

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Validate configuration and bean integrity",
	Long: `Checks configuration and bean integrity, including:
- Configuration settings (colors, default type)
- Broken links (links to non-existent beans)
- Self-references (beans linking to themselves)
- Circular dependencies (cycles in blocks/parent relationships)

Use --fix to automatically remove broken links and self-references.
Note: Cycles cannot be auto-fixed and require manual intervention.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var configErrors []string
		var fixed int
		allBeans := core.All()

		// === Configuration checks ===
		if !checkJSON {
			fmt.Println(ui.Bold.Render("Configuration"))
		}

		// 1. Check statuses are defined (always true since hardcoded)
		if !checkJSON {
			fmt.Printf("  %s Statuses defined (%d hardcoded)\n", ui.Success.Render("✓"), len(config.DefaultStatuses))
		}

		// 2. Check default_status exists in statuses (always true since hardcoded)
		if !checkJSON {
			fmt.Printf("  %s Default status '%s' exists\n", ui.Success.Render("✓"), cfg.GetDefaultStatus())
		}

		// 2b. Check default_type is a valid hardcoded type
		if cfg.GetDefaultType() != "" && !cfg.IsValidType(cfg.GetDefaultType()) {
			configErrors = append(configErrors, fmt.Sprintf("default_type '%s' is not a valid type", cfg.GetDefaultType()))
		} else if cfg.GetDefaultType() != "" {
			if !checkJSON {
				fmt.Printf("  %s Default type '%s' is valid\n", ui.Success.Render("✓"), cfg.GetDefaultType())
			}
		}

		// 2b-2. Check every bean's type against the current configuration
		if stray := unknownTypeBeans(allBeans); len(stray) > 0 {
			for _, b := range stray {
				configErrors = append(configErrors,
					fmt.Sprintf("bean %s carries type '%s', which the configuration does not define", b.ID, b.Type))
			}
		} else if !checkJSON {
			fmt.Printf("  %s Every bean carries a known type\n", ui.Success.Render("✓"))
		}

		// 2c. Check agent.default_effort is a valid effort level
		if effort := cfg.GetDefaultEffort(); effort != "" && !config.IsValidEffortLevel(effort) {
			configErrors = append(configErrors, fmt.Sprintf("agent.default_effort '%s' is not valid (use low, medium, high, or max)", effort))
		} else if effort != "" {
			if !checkJSON {
				fmt.Printf("  %s Default effort '%s' is valid\n", ui.Success.Render("✓"), effort)
			}
		}

		// 3. Check all status colors are valid (hardcoded statuses). An
		// empty Color is not an invalid one -- it means "no explicit
		// colour, fall back to the muted default" (see ui.ResolveColor),
		// the same meaning it carries in a StatusOverride. Only a non-empty
		// value gets validated against the theme's tone names.
		for _, s := range config.DefaultStatuses {
			if s.Color != "" && !ui.IsValidColor(s.Color) {
				configErrors = append(configErrors, fmt.Sprintf("invalid color '%s' for status '%s'", s.Color, s.Name))
			}
		}
		if !checkJSON {
			colorErrors := 0
			for _, e := range configErrors {
				if len(e) > 13 && e[:13] == "invalid color" {
					colorErrors++
				}
			}
			if colorErrors == 0 {
				fmt.Printf("  %s All status colors valid\n", ui.Success.Render("✓"))
			}
		}

		// 4. Check all type colors are valid (hardcoded types). Same
		// reasoning as the status loop above: "task" deliberately carries
		// an empty Color in the ranked type scale (task 3), and an empty
		// colour is "uncoloured", not "invalid".
		for _, t := range config.DefaultTypes {
			if t.Color != "" && !ui.IsValidColor(t.Color) {
				configErrors = append(configErrors, fmt.Sprintf("invalid color '%s' for type '%s'", t.Color, t.Name))
			}
		}
		if !checkJSON {
			typeColorErrors := 0
			for _, e := range configErrors {
				if len(e) > 13 && e[:13] == "invalid color" {
					typeColorErrors++
				}
			}
			if typeColorErrors == 0 {
				fmt.Printf("  %s All type colors valid\n", ui.Success.Render("✓"))
			}
		}

	// 4. Check prefix consistency (configuration vs on-disk)
	prefixError := core.ValidatePrefixConsistency()
	if prefixError != "" {
		configErrors = append(configErrors, "prefix consistency: "+prefixError)
	} else if !checkJSON {
		fmt.Printf("  %s Prefix consistency valid\n", ui.Success.Render("✓"))
	}

		// Print config errors in human-readable mode
		if !checkJSON {
			for _, e := range configErrors {
				fmt.Printf("  %s %s\n", ui.Danger.Render("✗"), e)
			}
		}

		// === Bean link checks ===
		if !checkJSON {
			fmt.Println()
			fmt.Println(ui.Bold.Render("Bean Links"))
		}

		linkResult := core.CheckAllLinks()

		// Handle --fix mode
		if checkFix && (len(linkResult.BrokenLinks) > 0 || len(linkResult.SelfLinks) > 0) {
			fixedCount, err := core.FixBrokenLinks()
			if err != nil {
				return fmt.Errorf("fixing broken links: %w", err)
			}
			fixed = fixedCount

			if !checkJSON {
				for _, bl := range linkResult.BrokenLinks {
					fmt.Printf("  %s %s: removed broken link %s:%s\n", ui.Success.Render("✓"), bl.BeanID, bl.LinkType, bl.Target)
				}
				for _, sl := range linkResult.SelfLinks {
					fmt.Printf("  %s %s: removed self-reference in %s link\n", ui.Success.Render("✓"), sl.BeanID, sl.LinkType)
				}
			}

			// Clear the fixed issues from the result
			linkResult.BrokenLinks = []beancore.BrokenLink{}
			linkResult.SelfLinks = []beancore.SelfLink{}
		} else if !checkJSON {
			// Report issues without fixing
			for _, bl := range linkResult.BrokenLinks {
				fmt.Printf("  %s %s: broken link %s:%s\n", ui.Danger.Render("✗"), bl.BeanID, bl.LinkType, bl.Target)
			}
			for _, sl := range linkResult.SelfLinks {
				fmt.Printf("  %s %s: self-reference in %s link\n", ui.Danger.Render("✗"), sl.BeanID, sl.LinkType)
			}
		}

		// Cycles cannot be auto-fixed
		if !checkJSON {
			for _, c := range linkResult.Cycles {
				if checkFix {
					fmt.Printf("  %s Cannot auto-fix cycle: %s (via %s)\n", ui.Warning.Render("!"), formatCycle(c.Path), c.LinkType)
				} else {
					fmt.Printf("  %s Circular dependency: %s (via %s)\n", ui.Danger.Render("✗"), formatCycle(c.Path), c.LinkType)
				}
			}
		}

		// Show success if no issues
		if !checkJSON && !linkResult.HasIssues() && fixed == 0 {
			fmt.Printf("  %s No link issues found\n", ui.Success.Render("✓"))
		}

		// === Policy checks ===
		var policyWarnings []string
		if len(cfg.Beans.RequireFieldsOn) > 0 {
			if !checkJSON {
				fmt.Println()
				fmt.Println(ui.Bold.Render("Policy"))
			}

			commitField := cfg.GetCommitField()
			type commitRef struct {
				beanID string
				sha    string
			}
			var refs []commitRef
			var shas []string
			seenSha := make(map[string]bool)

			for _, b := range allBeans {
				if fields := cfg.RequiredFieldsFor(b.Status); len(fields) > 0 {
					missing := beancore.MissingRequiredFields(b, fields)
					if len(missing) > 0 {
						policyWarnings = append(policyWarnings, fmt.Sprintf("%s: status %q missing required field(s): %s", b.ID, b.Status, strings.Join(missing, ", ")))
					}
				}
				if v, ok := b.Extra[commitField]; ok {
					if sha, ok := v.(string); ok && sha != "" {
						refs = append(refs, commitRef{beanID: b.ID, sha: sha})
						if !seenSha[sha] {
							seenSha[sha] = true
							shas = append(shas, sha)
						}
					}
				}
			}

			if len(shas) > 0 {
				var exist map[string]bool
				ok := false
				if cwd, err := os.Getwd(); err == nil {
					exist, ok = gitutil.CommitsExist(cwd, shas)
				}
				if !ok {
					policyWarnings = append(policyWarnings, "commit verification skipped (not a git repository)")
				} else {
					for _, ref := range refs {
						if !exist[ref.sha] {
							policyWarnings = append(policyWarnings, fmt.Sprintf("%s: commit %s not found in this repository", ref.beanID, ref.sha))
						}
					}
				}
			}

			if !checkJSON {
				if len(policyWarnings) == 0 {
					fmt.Printf("  %s Field policy satisfied\n", ui.Success.Render("✓"))
				} else {
					for _, w := range policyWarnings {
						fmt.Printf("  %s %s\n", ui.Warning.Render("!"), w)
					}
				}
			}
		}

		// === Summary ===
		totalIssues := len(configErrors) + linkResult.TotalIssues()
		if checkStrict {
			totalIssues += len(policyWarnings)
		}

		if checkJSON {
			result := checkResult{
				Success:        totalIssues == 0,
				ConfigErrors:   configErrors,
				BeanIssues:     linkResult,
				Fixed:          fixed,
				PolicyWarnings: policyWarnings,
			}
			data, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(data))
		} else {
			fmt.Println()
			if totalIssues == 0 && fixed == 0 {
				fmt.Println(ui.Success.Render("All checks passed"))
			} else if totalIssues == 0 && fixed > 0 {
				fmt.Println(ui.Success.Render(fmt.Sprintf("Fixed %d issue(s)", fixed)))
			} else if fixed > 0 {
				// Some issues fixed, some remain (cycles)
				fmt.Println(ui.Warning.Render(fmt.Sprintf("Fixed %d issue(s), %d require manual intervention", fixed, totalIssues)))
			} else if totalIssues == 1 {
				fmt.Println(ui.Danger.Render("1 issue found"))
			} else {
				fmt.Println(ui.Danger.Render(fmt.Sprintf("%d issues found", totalIssues)))
			}
		}

		// Exit with error code if validation failed
		if totalIssues > 0 {
			os.Exit(1)
		}

		return nil
	},
}

func RegisterCheckCmd(root *cobra.Command) {
	checkCmd.Flags().BoolVar(&checkJSON, "json", false, "Output as JSON")
	checkCmd.Flags().BoolVar(&checkFix, "fix", false, "Automatically fix broken links and self-references")
	checkCmd.Flags().BoolVar(&checkStrict, "strict", false, "Count policy warnings as issues (exit 1)")
	root.AddCommand(checkCmd)
}
