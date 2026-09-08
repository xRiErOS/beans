package commands

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/xRiErOS/beans/pkg/bean"
	"github.com/xRiErOS/beans/pkg/beancore"
	"github.com/xRiErOS/beans/pkg/config"
)

// setupCompletionTest installs a throwaway core with n beans into the
// package globals the completion functions read, mirroring
// setupStartTest's pattern (start_test.go).
func setupCompletionTest(t *testing.T, n int) []*bean.Bean {
	t.Helper()
	tmpDir := t.TempDir()
	beansDir := filepath.Join(tmpDir, ".beans")
	if err := os.MkdirAll(beansDir, 0755); err != nil {
		t.Fatalf("failed to create test .beans dir: %v", err)
	}

	testCfg := config.Default()
	testCore := beancore.New(beansDir, testCfg)
	if err := testCore.Load(); err != nil {
		t.Fatalf("failed to load core: %v", err)
	}

	oldCore, oldCfg := core, cfg
	core, cfg = testCore, testCfg
	t.Cleanup(func() { core, cfg = oldCore, oldCfg })

	beans := make([]*bean.Bean, 0, n)
	for i := range n {
		b := &bean.Bean{
			ID:     bean.Slugify("fixture") + "-" + string(rune('a'+i)),
			Title:  "Fixture bean",
			Status: "todo",
			Type:   "task",
			// A poison Body: SC-03 requires the candidate source never
			// reads it. If beanIDCandidates ever grows a Body reference,
			// this string would leak into a candidate and TestBeanIDCandidatesNeverExposeBody
			// below would catch it.
			Body: "POISON-BODY-MUST-NEVER-APPEAR-IN-COMPLETION-OUTPUT",
		}
		if err := core.Create(b); err != nil {
			t.Fatalf("core.Create() error = %v", err)
		}
		beans = append(beans, b)
	}
	return beans
}

// AC-02/SC-01: candidates come from the store, not an empty list.
func TestBeanIDCandidatesReturnsEveryBean(t *testing.T) {
	beans := setupCompletionTest(t, 5)

	got := beanIDCandidates()
	if len(got) != 5 {
		t.Fatalf("beanIDCandidates() returned %d candidates, want 5", len(got))
	}

	seen := make(map[string]bool, len(got))
	for _, c := range got {
		id := strings.SplitN(c, "\t", 2)[0]
		seen[id] = true
	}
	for _, b := range beans {
		if !seen[b.ID] {
			t.Errorf("candidate list missing bean id %q", b.ID)
		}
	}
}

// AC-03/SC-03 (substring class): the formatted candidate text never contains
// a bean's body content.
func TestBeanIDCandidatesNeverExposeBody(t *testing.T) {
	setupCompletionTest(t, 3)

	for _, c := range beanIDCandidates() {
		if strings.Contains(c, "POISON-BODY-MUST-NEVER-APPEAR-IN-COMPLETION-OUTPUT") {
			t.Fatalf("candidate %q leaks bean Body content", c)
		}
	}
}

// AC-03/SC-03 (structural class): a substring grep is undergoverned for an
// absence criterion, so this parses completion.go's AST and asserts that
// beanIDCandidates neither calls a body-loading/parsing operation (Get,
// GetFromArchive, Render, ReadFile, loadFromDisk) nor references a bean's
// Body field directly -- it may only walk the already-resident
// core.All() slice and read ID/Type/Title/Status. The Body-selector check
// exists because the call-name check alone does not cover a mutation that
// appends b.Body directly (a field read, not a call): confirmed by running
// that exact mutation, which left the call-name check green and only the
// substring test (TestBeanIDCandidatesNeverExposeBody) red -- SC-03 was
// covered only by the two tests together, not by this one alone, until
// this selector check closed the gap.
func TestBeanIDCandidatesSourceNeverCallsBodyLoader(t *testing.T) {
	disallowedCalls := map[string]bool{
		"Get":              true,
		"GetFromArchive":   true,
		"Render":           true,
		"ReadFile":         true,
		"loadFromDisk":     true,
		"loadBean":         true,
		"Load":             true,
		"parseFrontMatter": true,
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "completion.go", nil, parser.AllErrors)
	if err != nil {
		t.Fatalf("parsing completion.go: %v", err)
	}

	var found *ast.FuncDecl
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && fn.Name.Name == "beanIDCandidates" {
			found = fn
			break
		}
	}
	if found == nil {
		t.Fatal("completion.go declares no beanIDCandidates function")
	}

	ast.Inspect(found.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.CallExpr:
			var name string
			switch fn := node.Fun.(type) {
			case *ast.SelectorExpr:
				name = fn.Sel.Name
			case *ast.Ident:
				name = fn.Name
			}
			if disallowedCalls[name] {
				t.Errorf("beanIDCandidates calls disallowed body-loading function %q", name)
			}
		case *ast.SelectorExpr:
			if node.Sel.Name == "Body" {
				t.Errorf("beanIDCandidates references a bean's Body field directly")
			}
		}
		return true
	})
}

// AC-04 (unbounded verbs): complete/delete/scrap/show/start/tag declare
// cobra.MinimumNArgs(1) with no ceiling, so candidates keep coming no matter
// how many positions are already filled.
func TestCompletionUnboundedNeverStops(t *testing.T) {
	setupCompletionTest(t, 2)

	for _, argc := range []int{0, 1, 5} {
		args := make([]string, argc)
		got, directive := completionUnbounded(nil, args, "")
		if len(got) != 2 {
			t.Errorf("argc=%d: got %d candidates, want 2", argc, len(got))
		}
		if directive != cobra.ShellCompDirectiveNoFileComp {
			t.Errorf("argc=%d: directive = %v, want ShellCompDirectiveNoFileComp", argc, directive)
		}
	}
}

// AC-05: a bounded verb stops offering candidates once its declared arity
// is satisfied. Measured at the three points that matter for each n -- one
// position under the boundary, at it, and one past it -- because only the
// boundary itself proves anything.
// n=1 is the value every wired verb actually ships (order, update, graph,
// progress, roadmap, rename all call completionUpTo(1)); n=2 is measured
// too, on the same shared primitive, to show the logic generalizes rather
// than happening to work at the one value that ships.
func TestCompletionUpToStopsAtBoundary(t *testing.T) {
	setupCompletionTest(t, 4)

	for _, n := range []int{1, 2} {
		fn := completionUpTo(n)
		cases := []struct {
			argc      int
			wantEmpty bool
		}{
			{argc: n - 1, wantEmpty: false}, // under the boundary: still offering
			{argc: n, wantEmpty: true},      // at the boundary: arity satisfied
			{argc: n + 1, wantEmpty: true},  // past the boundary: stays stopped
		}
		for _, tc := range cases {
			args := make([]string, tc.argc)
			got, directive := fn(nil, args, "")
			if tc.wantEmpty && len(got) != 0 {
				t.Errorf("n=%d argc=%d: got %d candidates, want 0", n, tc.argc, len(got))
			}
			if !tc.wantEmpty && len(got) == 0 {
				t.Errorf("n=%d argc=%d: got 0 candidates, want > 0", n, tc.argc)
			}
			if directive != cobra.ShellCompDirectiveNoFileComp {
				t.Errorf("n=%d argc=%d: directive = %v, want ShellCompDirectiveNoFileComp", n, tc.argc, directive)
			}
		}
	}
}

// AC-01: each of the twelve verbs registers a ValidArgsFunction. promote is
// deliberately excluded -- neither of its positions names an existing bean
// (see the coordinator question posted 2026-09-08: promote.go:44,51 takes
// an artifact path and finding-ids, not bean IDs).
//
// Register funcs bind package-level singleton command/flag vars, so a
// second ad hoc Register call in this test binary panics in pflag with
// "flag redefined" for any verb lacking its own idempotency guard (only
// order.go and roadmap.go carry one). sharedTestRoot (error_shape_test.go)
// is this package's one process-wide RegisterCoreCommands call; every test
// that needs a fully wired root reuses it instead of registering again.
//
// A bare "ValidArgsFunction != nil" check observes only that something got
// registered -- a function that always returned (nil, ShellCompDirectiveError)
// would pass it. This calls the REGISTERED function through root.Find, at
// zero args (must offer candidates) and at one arg (must match the verb's
// declared binding: unbounded verbs keep offering, arity-1 verbs stop), so
// a swapped wiring -- completionUnbounded on a verb that ships
// completionUpTo(1), or vice versa -- turns exactly that verb's subtest red.
var idVerbCases = []struct {
	name      string
	unbounded bool
}{
	{"complete", true},
	{"delete", true},
	{"order", false},
	{"scrap", true},
	{"show", true},
	{"start", true},
	{"tag", true},
	{"update", false},
	{"graph", false},
	{"progress", false},
	{"roadmap", false},
	{"rename", false},
}

func TestEachIDVerbRegistersValidArgsFunction(t *testing.T) {
	setupCompletionTest(t, 3)
	root := sharedTestRoot(t)

	for _, tc := range idVerbCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd, _, err := root.Find([]string{tc.name})
			if err != nil {
				t.Fatalf("root.Find(%q): %v", tc.name, err)
			}
			if cmd.ValidArgsFunction == nil {
				t.Fatalf("verb %q has no ValidArgsFunction registered", tc.name)
			}

			atZero, _ := cmd.ValidArgsFunction(cmd, nil, "")
			if len(atZero) == 0 {
				t.Errorf("verb %q: ValidArgsFunction(0 args) returned no candidates", tc.name)
			}

			atOne, _ := cmd.ValidArgsFunction(cmd, []string{"beans-existing"}, "")
			switch {
			case tc.unbounded && len(atOne) == 0:
				t.Errorf("verb %q is unbounded: ValidArgsFunction(1 arg) returned no candidates, want more", tc.name)
			case !tc.unbounded && len(atOne) != 0:
				t.Errorf("verb %q stops at arity 1: ValidArgsFunction(1 arg) returned %d candidates, want 0", tc.name, len(atOne))
			}
		})
	}
}

// rename's second position is a new, not-yet-existing identifier (rename.go
// Long: "beans rename <id> <new-id> full new ID"), so it must not offer
// existing bean IDs even though cobra.MaximumNArgs(2) allows a second
// position at all.
func TestRenameSecondPositionOffersNoCandidates(t *testing.T) {
	setupCompletionTest(t, 3)

	root := sharedTestRoot(t)
	cmd, _, err := root.Find([]string{"rename"})
	if err != nil {
		t.Fatalf("root.Find(rename): %v", err)
	}

	got, _ := cmd.ValidArgsFunction(cmd, []string{"beans-existing"}, "")
	if len(got) != 0 {
		t.Errorf("rename position 2: got %d candidates, want 0 (new-id is not an existing bean)", len(got))
	}
}
