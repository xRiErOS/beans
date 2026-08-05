package commands

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/hmans/beans/internal/output"
	"github.com/hmans/beans/internal/ui"
	"github.com/hmans/beans/pkg/bean"
	"github.com/hmans/beans/pkg/beangraph"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	showJSON     bool
	showRaw      bool
	showBodyOnly bool
	showETagOnly bool
)

var showCmd = &cobra.Command{
	Use:   "show <id> [id...]",
	Short: "Show a bean's contents",
	Long:  `Displays the full contents of one or more beans, including front matter and body.`,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		resolver := &beangraph.CoreResolver{Core: core}

		// Collect all beans
		var beans []*bean.Bean
		for _, id := range args {
			b, err := resolver.Bean(context.Background(), id)
			if err != nil {
				if showJSON {
					return output.Error(output.ErrNotFound, err.Error())
				}
				return fmt.Errorf("failed to find bean: %w", err)
			}
			if b == nil {
				if showJSON {
					return output.Error(output.ErrNotFound, fmt.Sprintf("bean not found: %s", id))
				}
				return fmt.Errorf("bean not found: %s", id)
			}
			beans = append(beans, b)
		}

		// JSON output
		if showJSON {
			if len(beans) == 1 {
				return output.SuccessSingle(beans[0])
			}
			return output.SuccessMultiple(beans)
		}

		// Raw markdown output (frontmatter + body)
		if showRaw {
			for i, b := range beans {
				if i > 0 {
					fmt.Print("\n---\n\n")
				}
				content, err := b.Render()
				if err != nil {
					return fmt.Errorf("failed to render bean: %w", err)
				}
				fmt.Print(string(content))
			}
			return nil
		}

		// Body only (no header, no styling)
		if showBodyOnly {
			for i, b := range beans {
				if i > 0 {
					fmt.Print("\n---\n\n")
				}
				fmt.Print(b.Body)
			}
			return nil
		}

		// ETag only (for easy extraction in scripts)
		if showETagOnly {
			for i, b := range beans {
				if i > 0 {
					fmt.Println()
				}
				fmt.Print(b.ETag())
			}
			return nil
		}

		// Default: styled for a terminal, raw markdown for a pipe or a file
		out, err := showOutputAll(beans, term.IsTerminal(int(os.Stdout.Fd())))
		if err != nil {
			return err
		}
		fmt.Print(out)

		return nil
	},
}

// showOutput returns the text for a single bean, choosing the representation
// from whether stdout is a terminal.
func showOutput(b *bean.Bean, isTTY bool) (string, error) {
	if !isTTY {
		content, err := b.Render()
		if err != nil {
			return "", fmt.Errorf("failed to render bean: %w", err)
		}
		return string(content), nil
	}
	return styledBeanOutput(b)
}

// showOutputAll joins the output of several beans with the separator that
// belongs to the chosen representation.
func showOutputAll(beans []*bean.Bean, isTTY bool) (string, error) {
	separator := "\n---\n\n"
	if isTTY {
		separator = "\n" + ui.Muted.Render(strings.Repeat("═", 60)) + "\n\n"
	}

	var out strings.Builder
	for i, b := range beans {
		if i > 0 {
			out.WriteString(separator)
		}
		text, err := showOutput(b, isTTY)
		if err != nil {
			return "", err
		}
		out.WriteString(text)
	}
	return out.String(), nil
}

// styledBeanOutput builds the styled representation of a single bean.
func styledBeanOutput(b *bean.Bean) (string, error) {
	statusCfg := cfg.GetStatus(b.Status)
	statusColor := "gray"
	if statusCfg != nil {
		statusColor = statusCfg.Color
	}
	isArchive := cfg.IsArchiveStatus(b.Status)

	var header strings.Builder
	header.WriteString(ui.ID.Render(b.ID))
	header.WriteString(" ")
	header.WriteString(ui.RenderStatusWithColor(b.Status, statusColor, isArchive))
	if implicitStatus, implicitStatusFrom := core.ImplicitStatus(b.ID); implicitStatus != "" {
		header.WriteString(" ")
		header.WriteString(ui.Muted.Render("↑" + implicitStatus + " (from " + implicitStatusFrom + ")"))
	}

	// Display type
	if b.Type != "" {
		typeCfg := cfg.GetType(b.Type)
		typeColor := "gray"
		if typeCfg != nil {
			typeColor = typeCfg.Color
		}
		header.WriteString(" ")
		header.WriteString(ui.RenderTypeWithColor(b.Type, typeColor))
	}

	if b.Priority != "" {
		priorityCfg := cfg.GetPriority(b.Priority)
		priorityColor := "gray"
		if priorityCfg != nil {
			priorityColor = priorityCfg.Color
		}
		header.WriteString(" ")
		header.WriteString(ui.RenderPriorityWithColor(b.Priority, priorityColor))
	}
	if len(b.Tags) > 0 {
		header.WriteString("  ")
		header.WriteString(ui.Muted.Render(strings.Join(b.Tags, ", ")))
	}
	if b.CreatedAt != nil {
		header.WriteString("  ")
		header.WriteString(ui.Muted.Render("created " + b.CreatedAt.Format("2006-01-02 15:04 UTC")))
	}
	if b.UpdatedAt != nil {
		header.WriteString("  ")
		header.WriteString(ui.Muted.Render("updated " + b.UpdatedAt.Format("2006-01-02 15:04 UTC")))
	}
	header.WriteString("\n")
	header.WriteString(ui.Title.Render(b.Title))

	// Display relationships
	if b.Parent != "" || len(b.Blocking) > 0 {
		header.WriteString("\n")
		header.WriteString(ui.Muted.Render(strings.Repeat("─", 50)))
		header.WriteString("\n")
		header.WriteString(formatRelationships(b))
	}

	header.WriteString("\n")
	header.WriteString(ui.Muted.Render(strings.Repeat("─", 50)))

	headerBox := lipgloss.NewStyle().
		MarginBottom(1).
		Render(header.String())

	var out strings.Builder
	out.WriteString(headerBox)
	out.WriteString("\n")

	// Render the body with Glamour
	if b.Body != "" {
		renderer, err := glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(80),
		)
		if err != nil {
			return "", fmt.Errorf("failed to create renderer: %w", err)
		}

		rendered, err := renderer.Render(b.Body)
		if err != nil {
			return "", fmt.Errorf("failed to render markdown: %w", err)
		}

		out.WriteString(rendered)
	}

	return out.String(), nil
}

// formatRelationships formats parent and blocks for display.
func formatRelationships(b *bean.Bean) string {
	var parts []string

	// Display parent
	if b.Parent != "" {
		parts = append(parts, fmt.Sprintf("%s %s",
			ui.Muted.Render("parent:"),
			ui.ID.Render(b.Parent)))
	}

	// Display blocking
	for _, target := range b.Blocking {
		parts = append(parts, fmt.Sprintf("%s %s",
			ui.Muted.Render("blocking:"),
			ui.ID.Render(target)))
	}
	return strings.Join(parts, "\n")
}

func RegisterShowCmd(root *cobra.Command) {
	showCmd.Flags().BoolVar(&showJSON, "json", false, "Output as JSON")
	showCmd.Flags().BoolVar(&showRaw, "raw", false, "Output raw markdown without styling")
	showCmd.Flags().BoolVar(&showBodyOnly, "body-only", false, "Output only the body content")
	showCmd.Flags().BoolVar(&showETagOnly, "etag-only", false, "Output only the etag")
	showCmd.MarkFlagsMutuallyExclusive("json", "raw", "body-only", "etag-only")
	root.AddCommand(showCmd)
}
