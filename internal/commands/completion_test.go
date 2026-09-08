package commands

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/xRiErOS/beans/pkg/bean"
	"github.com/xRiErOS/beans/pkg/beancore"
	"github.com/xRiErOS/beans/pkg/config"
)

// completionTestBinary is the compiled cmd/beans binary, built once in
// TestMain. Reaching the beans-9jtq bug requires cobra's real __complete
// request (completions.go's DisableFlagParsing path) driven through
// Execute()/ExecuteC(), which sharedTestRoot's in-process root.Find +
// ValidArgsFunction calls never exercise -- those skip straight past
// PersistentPreRunE's flag handling entirely.
var completionTestBinary string

func TestMain(m *testing.M) {
	binDir, err := os.MkdirTemp("", "beans-completion-bin")
	if err != nil {
		fmt.Fprintln(os.Stderr, "beans-completion-bin mkdtemp:", err)
		os.Exit(1)
	}

	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		fmt.Fprintln(os.Stderr, "resolve repo root:", err)
		os.RemoveAll(binDir)
		os.Exit(1)
	}

	completionTestBinary = filepath.Join(binDir, "beans")
	build := exec.Command("go", "build", "-o", completionTestBinary, "./cmd/beans")
	build.Dir = repoRoot
	if out, err := build.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "building beans binary: %v\n%s", err, out)
		os.RemoveAll(binDir)
		os.Exit(1)
	}

	// os.Exit below never runs deferred calls, so the build dir is
	// removed explicitly on every path out of this function instead of
	// relying on a defer that m.Run()'s exit code would skip.
	code := m.Run()
	os.RemoveAll(binDir)
	os.Exit(code)
}

// writeFixtureStore creates a real .beans store on disk (not the
// package-global core setupCompletionTest swaps in) with a single bean
// named "<idPrefix>-only", so a completion test driven through a
// subprocess can tell which store answered by grepping its stdout. The
// explicit Slug forces BuildFilename's double-dash "id--slug.md" form;
// without it, an ID containing a dash and no slug round-trips through
// ParseFilename's single-dash legacy fallback and gets cut at the first
// dash on reload -- a filename-format quirk unrelated to this bug that
// would otherwise corrupt the fixture IDs this test greps for.
func writeFixtureStore(t *testing.T, beansDir string, idPrefix string) {
	t.Helper()
	if err := os.MkdirAll(beansDir, 0755); err != nil {
		t.Fatalf("creating fixture store dir: %v", err)
	}
	c := beancore.New(beansDir, config.Default())
	if err := c.Load(); err != nil {
		t.Fatalf("loading fixture core: %v", err)
	}
	b := &bean.Bean{
		ID:     idPrefix + "-only",
		Slug:   "fixture",
		Title:  "Fixture bean",
		Status: "todo",
		Type:   "task",
	}
	if err := c.Create(b); err != nil {
		t.Fatalf("creating fixture bean: %v", err)
	}
}

// runBeansCompletion runs the compiled binary with cwd, args, and any
// extraEnv entries ("KEY=value") appended to a copy of the current
// environment with any inherited BEANS_PATH stripped first -- so a
// developer's real BEANS_PATH can never leak into a result, and a test
// that wants BEANS_PATH set does so explicitly via extraEnv. Returns
// stdout and the exit error (nil on success). Only stdout is used for
// assertions: cobra's __complete Run prints candidates and the trailing
// ":<directive>" line there, never on stderr.
func runBeansCompletion(t *testing.T, cwd string, extraEnv []string, args []string) (string, error) {
	t.Helper()
	cmd := exec.Command(completionTestBinary, args...)
	cmd.Dir = cwd
	baseEnv := os.Environ()
	filtered := make([]string, 0, len(baseEnv)+len(extraEnv))
	for _, kv := range baseEnv {
		if strings.HasPrefix(kv, "BEANS_PATH=") {
			continue
		}
		filtered = append(filtered, kv)
	}
	cmd.Env = append(filtered, extraEnv...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if testing.Verbose() {
		t.Logf("stderr: %s", stderr.String())
	}
	return stdout.String(), err
}

// AC-01/AC-02: beans --beans-path <X> __complete show "" must yield
// candidates from <X>, not from a store discovered via cwd -- even when
// that foreign cwd carries its own .beans.yml. A foreign .beans.yml is the
// case that matters (see beans-9jtq "Known failure mode"): directory
// discovery and the flag coincide when the test runs from inside the
// flag's own store, so a test that never leaves that store's directory
// cannot observe this bug (the mistake beans-hh5z SC-01 made).
func TestCompletionBeansPathOverridesForeignCwdDiscovery(t *testing.T) {
	flagStoreBeans := filepath.Join(t.TempDir(), ".beans")
	writeFixtureStore(t, flagStoreBeans, "flagstore")

	foreignCwd := t.TempDir()
	writeFixtureStore(t, filepath.Join(foreignCwd, ".beans"), "cwdstore")
	if err := os.WriteFile(filepath.Join(foreignCwd, ".beans.yml"), []byte("beans:\n  path: .beans\n"), 0644); err != nil {
		t.Fatalf("writing foreign .beans.yml: %v", err)
	}

	out, err := runBeansCompletion(t, foreignCwd, nil, []string{"--beans-path", flagStoreBeans, "__complete", "show", ""})
	if err != nil {
		t.Fatalf("__complete show \"\" failed: %v\nstdout: %s", err, out)
	}

	if !strings.Contains(out, "flagstore-only") {
		t.Errorf("completion output = %q, want the --beans-path store's candidate (flagstore-only)", out)
	}
	if strings.Contains(out, "cwdstore-only") {
		t.Errorf("completion output = %q, must not contain the foreign cwd's own store's candidate (cwdstore-only) -- the flag must win", out)
	}
}

// The store-unresolvable half of the same contract: an invalid
// --beans-path must never fall back to the foreign cwd's perfectly valid
// store. "No candidates" is the only acceptable outcome.
func TestCompletionUnresolvableStoreYieldsNoCandidates(t *testing.T) {
	foreignCwd := t.TempDir()
	writeFixtureStore(t, filepath.Join(foreignCwd, ".beans"), "cwdstore")
	if err := os.WriteFile(filepath.Join(foreignCwd, ".beans.yml"), []byte("beans:\n  path: .beans\n"), 0644); err != nil {
		t.Fatalf("writing foreign .beans.yml: %v", err)
	}

	missingStore := filepath.Join(t.TempDir(), "does-not-exist")
	out, err := runBeansCompletion(t, foreignCwd, nil, []string{"--beans-path", missingStore, "__complete", "show", ""})
	if err == nil {
		t.Fatalf("__complete show \"\" with an unresolvable --beans-path succeeded, want failure; stdout = %q", out)
	}
	if strings.Contains(out, "cwdstore-only") {
		t.Errorf("completion output = %q, must not fall back to the foreign cwd's own store when --beans-path is unresolvable", out)
	}
}

// SC-01/AC-01's second required measurement point: a foreign cwd with NO
// .beans.yml at all, so directory discovery has nothing of its own to find
// there except BEANS_PATH. The flag must win over that divergence too, not
// only over a discovered .beans.yml (TestCompletionBeansPathOverridesForeignCwdDiscovery
// above covers that first axis).
func TestCompletionBeansPathOverridesEnvWithNoConfigFile(t *testing.T) {
	flagStoreBeans := filepath.Join(t.TempDir(), ".beans")
	writeFixtureStore(t, flagStoreBeans, "flagstore2")

	envStoreBeans := filepath.Join(t.TempDir(), ".beans")
	writeFixtureStore(t, envStoreBeans, "envstore")

	// A bare foreign cwd: no .beans.yml, no .beans/ of its own, so the
	// only thing directory-based discovery could ever fall back to here
	// is BEANS_PATH.
	foreignCwd := t.TempDir()

	out, err := runBeansCompletion(t, foreignCwd, []string{"BEANS_PATH=" + envStoreBeans},
		[]string{"--beans-path", flagStoreBeans, "__complete", "show", ""})
	if err != nil {
		t.Fatalf("__complete show \"\" failed: %v\nstdout: %s", err, out)
	}

	if !strings.Contains(out, "flagstore2-only") {
		t.Errorf("completion output = %q, want the --beans-path store's candidate (flagstore2-only)", out)
	}
	if strings.Contains(out, "envstore-only") {
		t.Errorf("completion output = %q, must not contain BEANS_PATH's candidate (envstore-only) -- the flag must win", out)
	}
}

// SC-02: --config is the second raw-arg-recovered flag alongside
// --beans-path (root.go's completionAwareRootFlags handles both). Without
// it, __complete would resolve a .beans.yml named explicitly via --config
// only for the regular command path, silently falling back to directory
// discovery for completion -- offering a third store's config (and
// through it, its beans.path) while --beans-path/--config both point
// elsewhere. Measured from a foreign cwd with its own unrelated
// .beans.yml, so directory discovery and --config cannot coincide.
func TestCompletionConfigFlagOverridesForeignCwdDiscovery(t *testing.T) {
	configStoreBeans := filepath.Join(t.TempDir(), ".beans")
	writeFixtureStore(t, configStoreBeans, "configstore")
	configDir := t.TempDir()
	configPath := filepath.Join(configDir, "third.beans.yml")
	configYAML := "beans:\n  path: " + configStoreBeans + "\n"
	if err := os.WriteFile(configPath, []byte(configYAML), 0644); err != nil {
		t.Fatalf("writing --config file: %v", err)
	}

	foreignCwd := t.TempDir()
	writeFixtureStore(t, filepath.Join(foreignCwd, ".beans"), "cwdstore2")
	if err := os.WriteFile(filepath.Join(foreignCwd, ".beans.yml"), []byte("beans:\n  path: .beans\n"), 0644); err != nil {
		t.Fatalf("writing foreign .beans.yml: %v", err)
	}

	out, err := runBeansCompletion(t, foreignCwd, nil, []string{"--config", configPath, "__complete", "show", ""})
	if err != nil {
		t.Fatalf("__complete show \"\" failed: %v\nstdout: %s", err, out)
	}

	if !strings.Contains(out, "configstore-only") {
		t.Errorf("completion output = %q, want the --config store's candidate (configstore-only)", out)
	}
	if strings.Contains(out, "cwdstore2-only") {
		t.Errorf("completion output = %q, must not contain the foreign cwd's own store's candidate (cwdstore2-only) -- --config must win", out)
	}
}

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
		// beans-sfle AC-01: the directive must also request KeepOrder so
		// the shell preserves beanIDCandidates' actionable-first ordering.
		if directive != completionDirective {
			t.Errorf("argc=%d: directive = %v, want %v", argc, directive, completionDirective)
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
			// beans-sfle AC-01: KeepOrder must survive both the
			// candidate-producing and the exhausted-position branch.
			if directive != completionDirective {
				t.Errorf("n=%d argc=%d: directive = %v, want %v", n, tc.argc, directive, completionDirective)
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

// setupCompletionTestWithStatuses installs a throwaway core with one bean
// per entry in statuses, mirroring setupCompletionTest but letting the
// caller control each bean's status instead of hardcoding "todo" -- needed
// to fixture the actionable/non-actionable status boundary beans-sfle
// AC-01 sorts across.
func setupCompletionTestWithStatuses(t *testing.T, ids, statuses []string) []*bean.Bean {
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

	beans := make([]*bean.Bean, 0, len(statuses))
	for i, status := range statuses {
		b := &bean.Bean{
			ID:     ids[i],
			Title:  ids[i],
			Status: status,
			Type:   "task",
		}
		if err := testCore.Create(b); err != nil {
			t.Fatalf("core.Create(%s) error = %v", ids[i], err)
		}
		beans = append(beans, b)
	}
	return beans
}

// TestBeanIDCandidatesActionableFirstOrder pins beans-sfle AC-01:
// todo/in-progress candidates must precede draft, and draft must precede
// completed/scrapped.
//
// Fixture shape justification (Fehlklasse B54): the coordinator's own
// measurement found that with only 3 pairwise-distinct elements the
// expected actionable-first order occurred by chance in 47/60 (78%) runs
// with NO sorting at all, and a "comparator always returns false" mutation
// was only red in 8/10 runs. With 5 pairwise-distinct-enough entries
// spanning the actionable/non-actionable boundary, the same measurement
// found the chance rate drop to 62% and the mutation red 10/10. This
// fixture uses all 5 statuses (todo, in-progress, draft, completed,
// scrapped) and inserts them in the OPPOSITE of the required order --
// completed/scrapped first, actionable last -- so core.All()'s natural
// insertion order is the worst case for an accidental pass: a passing
// assertion below can only be explained by the sort actually running.
func TestBeanIDCandidatesActionableFirstOrder(t *testing.T) {
	ids := []string{"beans-zzscr1", "beans-zzcmp1", "beans-zzdrf1", "beans-zzip01", "beans-zztd01"}
	statuses := []string{"scrapped", "completed", "draft", "in-progress", "todo"}
	setupCompletionTestWithStatuses(t, ids, statuses)

	candidates := beanIDCandidates()

	rankOf := func(id string) int {
		for i, cand := range candidates {
			if strings.SplitN(cand, "\t", 2)[0] == id {
				return i
			}
		}
		t.Fatalf("candidate %s not found in %v", id, candidates)
		return -1
	}

	scrappedRank := rankOf("beans-zzscr1")
	completedRank := rankOf("beans-zzcmp1")
	draftRank := rankOf("beans-zzdrf1")
	inProgressRank := rankOf("beans-zzip01")
	todoRank := rankOf("beans-zztd01")

	for _, actionable := range []int{todoRank, inProgressRank} {
		if actionable >= draftRank {
			t.Errorf("actionable rank %d must precede draft rank %d", actionable, draftRank)
		}
		if actionable >= completedRank || actionable >= scrappedRank {
			t.Errorf("actionable rank %d must precede completed/scrapped ranks %d/%d", actionable, completedRank, scrappedRank)
		}
	}
	if draftRank >= completedRank {
		t.Errorf("draft rank %d must precede completed rank %d", draftRank, completedRank)
	}
	if draftRank >= scrappedRank {
		t.Errorf("draft rank %d must precede scrapped rank %d", draftRank, scrappedRank)
	}

	// AC-02, restated for this fixture: sorting must not drop a candidate.
	if len(candidates) != len(ids) {
		t.Fatalf("got %d candidates, want %d (a candidate was dropped): %v", len(candidates), len(ids), candidates)
	}
	for _, id := range ids {
		rankOf(id) // fails the test via t.Fatalf if id is missing
	}
}

// TestCompletionUnboundedKeepsOrderDirective proves the ValidArgsFunctions
// request ShellCompDirectiveKeepOrder, without which a shell would discard
// beanIDCandidates' actionable-first ordering by re-sorting alphabetically.
func TestCompletionUnboundedKeepsOrderDirective(t *testing.T) {
	setupCompletionTest(t, 1)

	_, directive := completionUnbounded(nil, nil, "")
	if directive&cobra.ShellCompDirectiveKeepOrder == 0 {
		t.Errorf("completionUnbounded directive %v missing ShellCompDirectiveKeepOrder", directive)
	}
}

// acceptsPositionalArg reports whether cmd's own cobra.Args validator
// allows at least one positional argument, probed by calling it with
// increasing argument counts (a command's minimum arity is not otherwise
// knowable from the outside: promote.go declares MinimumNArgs(2), so a
// single-argument probe alone would wrongly call it argument-less). A nil
// Args validator is cobra's own "no restriction" default and always
// accepts. This is the sole scoping applied before the directive walk
// below: a verb that never accepts an argument (list, version, init, ...)
// can never trigger the shell's file-name completion fallback at all, so
// whether it nominally reports ShellCompDirectiveDefault is not this
// bug's domain.
func acceptsPositionalArg(cmd *cobra.Command) bool {
	if cmd.Args == nil {
		return true
	}
	for n := 1; n <= 5; n++ {
		probe := make([]string, n)
		for i := range probe {
			probe[i] = "x"
		}
		if cmd.Args(cmd, probe) == nil {
			return true
		}
	}
	return false
}

// lastDirectiveLine finds cobra's trailing ":<directive>" line in
// __complete output and parses the numeric directive it names.
func lastDirectiveLine(out string) (int, bool) {
	for _, line := range strings.Split(out, "\n") {
		if !strings.HasPrefix(line, ":") {
			continue
		}
		if n, err := strconv.Atoi(strings.TrimPrefix(line, ":")); err == nil {
			return n, true
		}
	}
	return 0, false
}

// TestOnlyPromoteFallsToDefaultDirective is beans-12cb AC-03: it derives
// the verb set mechanically by walking the actual command tree
// (sharedTestRoot(t).Commands()), never from a hand-maintained verb list
// -- a second such list would itself violate the guard beans-t7sv SC-03
// exists to enforce. For every verb whose own Args validator accepts at
// least one positional argument (acceptsPositionalArg above), it runs the
// real compiled binary's `beans __complete <verb> ""` and reads the
// trailing ":<directive>" line cobra always prints. Numeric 0 is
// ShellCompDirectiveDefault, the value that makes a shell fall back to
// file-name completion in the cwd -- the beans-12cb bug. The set of verbs
// still answering it after this leaf's fix must be exactly {promote}:
// promote's first positional is a real file path (promote.go:44, read
// via os.ReadFile at :53), where file-name completion is CORRECT, and
// Wave 1 already excluded it from its own coverage for the same reason
// (TestEachIDVerbRegistersValidArgsFunction's promote exclusion above).
// Any other verb appearing here is a regression to fix, not a gap to
// document.
func TestOnlyPromoteFallsToDefaultDirective(t *testing.T) {
	root := sharedTestRoot(t)

	storeDir := t.TempDir()
	writeFixtureStore(t, filepath.Join(storeDir, ".beans"), "verbscan")

	var defaulted []string
	for _, cmd := range root.Commands() {
		if !acceptsPositionalArg(cmd) {
			continue
		}
		name := cmd.Name()
		out, err := runBeansCompletion(t, storeDir, nil, []string{"__complete", name, ""})
		if err != nil {
			t.Fatalf("__complete %s \"\": %v\nstdout: %s", name, err, out)
		}
		directive, ok := lastDirectiveLine(out)
		if !ok {
			t.Fatalf("__complete %s \"\": no \":<directive>\" line in output %q", name, out)
		}
		if directive == 0 {
			defaulted = append(defaulted, name)
		}
	}

	sort.Strings(defaulted)
	want := []string{"promote"}
	if !reflect.DeepEqual(defaulted, want) {
		t.Fatalf("verbs falling to ShellCompDirectiveDefault = %v, want exactly %v", defaulted, want)
	}
}
