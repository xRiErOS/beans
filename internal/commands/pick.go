package commands

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/xRiErOS/beans/pkg/bean"
)

// errPickAborted is returned whenever nothing was selected -- the user
// cancelled, or reached the end of the picker without pressing enter on an
// item. R-11 AC3: stdout stays empty and the process exits non-zero either
// way, so callers cannot tell "cancelled" from "picked nothing" and don't
// need to.
var errPickAborted = errors.New("beans pick: no bean selected")

var pickCmd = &cobra.Command{
	Use:   "pick",
	Args:  cobra.NoArgs,
	Short: "Interactively pick a bean and print its ID",
	Long: `Draws an interactive, fuzzy-filterable list of every bean in the
resolved store on the controlling terminal. Selecting one prints exactly
that bean's ID to stdout and exits 0; aborting the picker, an empty
candidate list, or non-interactive stdin produce no stdout output and a
non-zero exit, so scripts can safely capture the result with command
substitution, e.g. id=$(beans pick).`,
	RunE: runPick,
}

// RegisterPickCmd adds the pick command to root. It intentionally does not
// call markPlumbing: pick is a directly-typed interactive verb, so it
// composes with RegisterCoreCommands's closing loop (register.go:101-108)
// and lands user-facing by falling into that default, the same way every
// other non-plumbing verb does.
func RegisterPickCmd(root *cobra.Command) {
	root.AddCommand(pickCmd)
}

// pickItem adapts a bean to bubbles/list's list.Item interface. It carries
// only the id and title -- the two fields the picker's list and its final
// stdout write need -- rather than a full *bean.Bean, so the picker and
// internal/tui's own pickers (parentpicker.go, blockingpicker.go, etc.)
// share no type between them (R-18: neither surface imports the other).
type pickItem struct {
	id, title string
}

func (i pickItem) Title() string       { return i.title }
func (i pickItem) Description() string { return i.id }
func (i pickItem) FilterValue() string { return i.title + " " + i.id }

// pickModel is the standalone bubbletea model driving the picker's
// terminal state machine: highlighting, filtering (delegated to the
// embedded list.Model), and the selected/aborted outcome the picker's
// caller reads back once the program returns.
type pickModel struct {
	list       list.Model
	selected   bool
	selectedID string
	aborted    bool
}

func newPickModel(beans []*bean.Bean) pickModel {
	items := make([]list.Item, len(beans))
	for i, b := range beans {
		items[i] = pickItem{id: b.ID, title: b.Title}
	}
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Pick a bean"
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	return pickModel{list: l}
}

func (m pickModel) Init() tea.Cmd { return nil }

func (m pickModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height)
		return m, nil
	case tea.KeyMsg:
		if m.list.FilterState() != list.Filtering {
			switch msg.String() {
			case "esc", "ctrl+c":
				m.aborted = true
				return m, tea.Quit
			case "enter":
				if item, ok := m.list.SelectedItem().(pickItem); ok {
					m.selected = true
					m.selectedID = item.id
				} else {
					m.aborted = true
				}
				return m, tea.Quit
			}
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View draws the picker. It returns a plain string for bubbletea to render
// wherever the program's output is configured -- runPick points that at
// the controlling tty, never at stdout -- so the model itself has no way
// to leak interactive chrome onto stdout even by mistake (R-11 AC2).
func (m pickModel) View() string {
	return m.list.View()
}

// runPick is pickCmd's RunE. Candidates come from the already-loaded store
// (R-16: core is populated exclusively by resolveBeansPath inside
// NewRootCmd's PersistentPreRunE; pick adds itself to no skip-list and
// opens no second, independent store-resolution path). Every interactive
// frame is drawn on the process's controlling terminal (opened explicitly
// as /dev/tty, never inherited stdin/stdout), so the only byte pick ever
// writes to stdout is the single selected ID on success.
func runPick(cmd *cobra.Command, _ []string) error {
	beans := core.All()
	if len(beans) == 0 {
		return errors.New("beans pick: store has no beans to pick from")
	}

	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return errors.New("beans pick: stdin is not a terminal")
	}

	sort.Slice(beans, func(i, j int) bool {
		return strings.ToLower(beans[i].Title) < strings.ToLower(beans[j].Title)
	})

	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("beans pick: opening controlling terminal: %w", err)
	}
	defer tty.Close()

	program := tea.NewProgram(newPickModel(beans), tea.WithInput(tty), tea.WithOutput(tty), tea.WithAltScreen())
	final, err := program.Run()
	if err != nil {
		return fmt.Errorf("beans pick: %w", err)
	}

	result, ok := final.(pickModel)
	if !ok || !result.selected {
		return errPickAborted
	}

	fmt.Fprintln(cmd.OutOrStdout(), result.selectedID)
	return nil
}
