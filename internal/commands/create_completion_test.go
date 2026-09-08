package commands

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// beans-u93j: `beans create ` + TAB used to answer with no candidates and
// no explanation (completionNoFileComp, beans-12cb). These tests cover
// create's own ValidArgsFunction, which now surfaces a cobra ActiveHelp
// hint in that zero-args case instead of staying silent.

// TestCreateValidArgsHintsOnEmptyArgs is AC-01/AC-02: with no positional
// argument yet, createValidArgs must return exactly one candidate --
// cobra's "_activeHelp_ "-prefixed pseudo-candidate (AppendActiveHelp),
// never a real, insertable candidate -- alongside
// ShellCompDirectiveNoFileComp so the shell's filename fallback stays
// blocked.
func TestCreateValidArgsHintsOnEmptyArgs(t *testing.T) {
	candidates, directive := createValidArgs(createCmd, nil, "")

	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("directive = %v, want ShellCompDirectiveNoFileComp", directive)
	}
	if len(candidates) != 1 {
		t.Fatalf("candidates = %v, want exactly one ActiveHelp pseudo-candidate", candidates)
	}
	if !strings.HasPrefix(candidates[0], "_activeHelp_ ") {
		t.Errorf("candidate = %q, want an ActiveHelp-marked line (cobra.AppendActiveHelp), not a real completion value", candidates[0])
	}
	// AC-02: an ActiveHelp line's payload is never a bare/empty value --
	// that is precisely the shape (an empty-value real candidate) the
	// Integration points section measured as unsafe to insert.
	if strings.TrimSpace(strings.TrimPrefix(candidates[0], "_activeHelp_ ")) == "" {
		t.Errorf("candidate = %q, ActiveHelp payload must not be empty", candidates[0])
	}
}

// TestCreateValidArgsSilentOnceTitleStarted: once the user has typed the
// first word of the title, create has nothing further to suggest --
// completionNoFileComp's rationale (create's positional is free-form text,
// never a bean-ID or file path) still applies, and repeating the hint on
// every subsequent word would be noise, not help.
func TestCreateValidArgsSilentOnceTitleStarted(t *testing.T) {
	candidates, directive := createValidArgs(createCmd, []string{"Fix"}, "")

	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("directive = %v, want ShellCompDirectiveNoFileComp", directive)
	}
	if len(candidates) != 0 {
		t.Errorf("candidates = %v, want none once a title word is already typed", candidates)
	}
}

// TestCreateHintNamesEveryRegisteredFlag is SC-02/AC-03: the hint must
// name every flag create.go actually registers, derived live from the
// shared createCmd's own flag set (never a hand-copied second list) --
// filtered only by the two structural exclusions that are never
// "create's own primary flags": cobra's own injected --help, and any
// flag inherited from the root command's PersistentFlags (--config,
// --beans-path). A newly added create-specific flag that createHint
// fails to surface would fail this test.
func TestCreateHintNamesEveryRegisteredFlag(t *testing.T) {
	root := sharedTestRoot(t)
	createSub, _, err := root.Find([]string{"create"})
	if err != nil {
		t.Fatalf("finding create command: %v", err)
	}

	rootPersistent := map[string]bool{}
	root.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		rootPersistent[f.Name] = true
	})

	var expected []string
	createSub.Flags().VisitAll(func(f *pflag.Flag) {
		if f.Name == "help" || rootPersistent[f.Name] {
			return
		}
		expected = append(expected, "--"+f.Name)
	})
	if len(expected) == 0 {
		t.Fatal("create registers no flags of its own -- test setup is broken")
	}

	hint := createHint()
	for _, name := range expected {
		if !strings.Contains(hint, name) {
			t.Errorf("hint %q does not mention registered flag %q", hint, name)
		}
	}
}

// TestCreateHintMentionsTitle is AC-01's other half: the hint must not
// only list flags but also name the expected title argument itself.
func TestCreateHintMentionsTitle(t *testing.T) {
	if !strings.Contains(createHint(), "title") {
		t.Errorf("hint %q does not mention the expected title argument", createHint())
	}
}

// TestCreateCompleteRawOutputNeverInsertsText exercises the real compiled
// binary's `__complete create ""` -- the exact request the PO's `beans
// create ` + TAB triggers -- and inspects cobra's raw wire format: one
// line per candidate, then a trailing ":<directive>" line. AC-02 requires
// that pressing TAB never inserts text into the command line; the only
// candidate line present must be the ActiveHelp pseudo-candidate, which
// the shipped zsh completion script (zsh_completions.go) renders via
// `compadd -x` -- zsh's display-only form that is never inserted or
// selectable -- specifically because it never reaches the `[ -n "$comp"
// ]` branch that feeds ordinary, insertable completions. A bare/empty
// candidate line here would be exactly the unsafe shape Integration
// point 1 warned against, since that same zsh script only guards empty
// *non-ActiveHelp* lines by skipping them -- this test pins that no such
// line is ever emitted in the first place.
func TestCreateCompleteRawOutputNeverInsertsText(t *testing.T) {
	cwd := t.TempDir()
	writeFixtureStore(t, filepath.Join(cwd, ".beans"), "rawout")

	out, err := runBeansCompletion(t, cwd, nil, []string{"__complete", "create", ""})
	if err != nil {
		t.Fatalf("__complete create \"\" failed: %v\nstdout: %s", err, out)
	}

	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) < 2 {
		t.Fatalf("__complete create \"\" output = %q, want at least a candidate line and a directive line", out)
	}
	directiveLine := lines[len(lines)-1]
	candidateLines := lines[:len(lines)-1]

	if !strings.HasPrefix(directiveLine, ":") {
		t.Fatalf("last line = %q, want cobra's \":<directive>\" line", directiveLine)
	}

	sawActiveHelp := false
	for _, line := range candidateLines {
		if strings.HasPrefix(line, "_activeHelp_ ") {
			sawActiveHelp = true
			continue
		}
		if line == "" {
			t.Errorf("candidate line is empty -- an empty-value candidate would insert nothing to select but is exactly the unsafe shape the leaf's Integration points warned against")
		} else {
			t.Errorf("unexpected non-ActiveHelp candidate %q for a bare `create` + TAB with no title typed", line)
		}
	}
	if !sawActiveHelp {
		t.Errorf("output %q never carries an ActiveHelp (\"_activeHelp_ \") line", out)
	}
}
