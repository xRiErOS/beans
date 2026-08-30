package commands

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/hmans/beans/internal/output"
	"github.com/hmans/beans/internal/ui"
	"github.com/hmans/beans/pkg/bean"
	"github.com/hmans/beans/pkg/beangraph"
	"github.com/hmans/beans/pkg/config"
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
	Long: `Displays the full contents of one or more beans, including front matter and body.

The representation follows stdout. On a terminal the output is styled and the
body is rendered as markdown. When stdout is a pipe or a file, the output is the
raw markdown of the source file — the same text --raw produces, unpadded and
unwrapped, so it can be fed to a parser.`,
	Args: cobra.MinimumNArgs(1),
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
	return renderBeanDetail(b, cfg, resolveWidth(0, false, cfg)), nil
}

// renderBeanDetail lays out one bean for the terminal: the attribute header
// reads the same order vertically that the beans table reads horizontally --
// type, id, then title, then status and priority -- so type, id and title
// carry the type's colour and weight and the header reads as one unit.
//
// The body goes through ui.RenderMarkdown (Task 16) instead of glamour: that
// renderer emits no trailing padding and no painted backgrounds, which is
// exactly what glamour got wrong.
func renderBeanDetail(b *bean.Bean, cfg *config.Config, width int) string {
	var sb strings.Builder

	tint := ""
	bold := false
	if tc := cfg.GetType(b.Type); tc != nil {
		tint, bold = tc.Color, tc.Emphasis
	}
	head := lipgloss.NewStyle().Bold(bold)
	if tint != "" {
		head = head.Foreground(ui.ResolveColor(tint))
	}

	sb.WriteString(head.Render(b.Type) + "  " + head.Render(b.ID) + "\n")
	sb.WriteString(head.Render(b.Title) + "\n")

	var attrs []string
	if sc := cfg.GetStatus(b.Status); sc != nil {
		attrs = append(attrs, lipgloss.NewStyle().Foreground(ui.ResolveColor(sc.Color)).
			Bold(!sc.Archive).Render(b.Status))
	}
	if b.Priority != "" && b.Priority != "normal" {
		if pc := cfg.GetPriority(b.Priority); pc != nil {
			attrs = append(attrs, lipgloss.NewStyle().Foreground(ui.ResolveColor(pc.Color)).
				Render(b.Priority))
		}
	}
	if implicitStatus, implicitStatusFrom := core.ImplicitStatus(b.ID); implicitStatus != "" {
		attrs = append(attrs, ui.Muted.Render("↑"+implicitStatus+" (from "+implicitStatusFrom+")"))
	}
	if len(attrs) > 0 {
		sb.WriteString(strings.Join(attrs, "  ") + "\n")
	}

	if rel := formatRelationships(b); rel != "" {
		sb.WriteString(rel + "\n")
	}

	// created/updated: presentation-only metadata the old header carried as
	// muted text. Dropping glamour is the plan's only authorised behaviour
	// change; this stays, just moved under the reordered attribute header.
	var stamps []string
	if b.CreatedAt != nil {
		stamps = append(stamps, "created "+b.CreatedAt.Format("2006-01-02 15:04 UTC"))
	}
	if b.UpdatedAt != nil {
		stamps = append(stamps, "updated "+b.UpdatedAt.Format("2006-01-02 15:04 UTC"))
	}
	if len(stamps) > 0 {
		sb.WriteString(ui.Muted.Render(strings.Join(stamps, "  ")) + "\n")
	}

	sb.WriteString(ui.TreeLine.Render(strings.Repeat("─", width)) + "\n\n")

	if body := ui.RenderMarkdown(b.Body, min(width, 90)); body != "" {
		sb.WriteString(body + "\n")
	}
	if len(b.Tags) > 0 {
		parts := make([]string, len(b.Tags))
		for i, t := range b.Tags {
			parts[i] = "#" + t
		}
		sb.WriteString("\n  " + ui.Muted.Render(strings.Join(parts, " ")) + "\n")
	}
	return sb.String()
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
	showCmd.Flags().BoolVar(&showRaw, "raw", false, "Force raw markdown output even on a terminal (already the default off a terminal)")
	showCmd.Flags().BoolVar(&showBodyOnly, "body-only", false, "Output only the body content")
	showCmd.Flags().BoolVar(&showETagOnly, "etag-only", false, "Output only the etag")
	showCmd.MarkFlagsMutuallyExclusive("json", "raw", "body-only", "etag-only")
	root.AddCommand(showCmd)
}
