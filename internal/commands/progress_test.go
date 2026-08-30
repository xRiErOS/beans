package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/xRiErOS/beans/internal/ui"
	"github.com/xRiErOS/beans/pkg/bean"
	"github.com/xRiErOS/beans/pkg/beancore"
	"github.com/xRiErOS/beans/pkg/config"
)

// setupProgressTest installs a throwaway core and default config into the
// package globals progressCmd.RunE reads, mirroring setupMilestonesTest.
func setupProgressTest(t *testing.T) {
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
}

// resetProgressFlags clears every progress* package global the tests below
// touch and restores the pre-test value afterwards.
func resetProgressFlags(t *testing.T) {
	t.Helper()
	oldJSON := progressJSON
	progressJSON = false
	t.Cleanup(func() {
		progressJSON = oldJSON
	})
}

// captureProgressStdout redirects os.Stdout for the duration of fn and
// returns everything written to it.
func captureProgressStdout(t *testing.T, fn func()) []byte {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}

	orig := os.Stdout
	os.Stdout = w
	fn()
	os.Stdout = orig
	if err := w.Close(); err != nil {
		t.Fatalf("closing pipe write end: %v", err)
	}

	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("reading captured stdout: %v", err)
	}
	return data
}

// TestProgressCountsByStatus verifies plain-text output shows a count for
// every configured status (not just the four illustrated in the epic), and
// that the counts reflect the actual beans present.
func TestProgressCountsByStatus(t *testing.T) {
	setupProgressTest(t)
	resetProgressFlags(t)

	beans := []*bean.Bean{
		{ID: "beans-a1", Slug: bean.Slugify("A"), Title: "A", Status: "in-progress", Type: "task"},
		{ID: "beans-a2", Slug: bean.Slugify("B"), Title: "B", Status: "todo", Type: "task"},
		{ID: "beans-a3", Slug: bean.Slugify("C"), Title: "C", Status: "todo", Type: "task"},
		{ID: "beans-a4", Slug: bean.Slugify("D"), Title: "D", Status: "draft", Type: "task"},
		{ID: "beans-a5", Slug: bean.Slugify("E"), Title: "E", Status: "completed", Type: "task"},
		{ID: "beans-a6", Slug: bean.Slugify("F"), Title: "F", Status: "scrapped", Type: "task"},
	}
	for _, b := range beans {
		if err := core.Create(b); err != nil {
			t.Fatalf("core.Create(%s) error = %v", b.ID, err)
		}
	}

	out := captureProgressStdout(t, func() {
		if err := progressCmd.RunE(progressCmd, nil); err != nil {
			t.Fatalf("progressCmd.RunE() error = %v", err)
		}
	})

	got := string(out)
	for _, want := range []string{
		"In Progress: 1",
		"Todo: 2",
		"Draft: 1",
		"Completed: 1",
		"Scrapped: 1",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expected output to contain %q, got %q", want, got)
		}
	}
}

// TestProgressComputesPercent verifies the epic's own worked example:
// 2 in-progress + 15 todo + 23 completed + 3 scrapped -> 23/(2+15+23) = 57%.
func TestProgressComputesPercent(t *testing.T) {
	setupProgressTest(t)
	resetProgressFlags(t)

	seedProgressCounts(t, map[string]int{
		"in-progress": 2,
		"todo":        15,
		"completed":   23,
		"scrapped":    3,
	})

	out := captureProgressStdout(t, func() {
		if err := progressCmd.RunE(progressCmd, nil); err != nil {
			t.Fatalf("progressCmd.RunE() error = %v", err)
		}
	})

	got := string(out)
	if !strings.Contains(got, "57% complete") {
		t.Errorf("expected output to contain %q, got %q", "57% complete", got)
	}
}

// TestProgressRootArgScopesToDescendants verifies that a root ID argument
// limits the counted beans to the given bean's descendants (via
// buildChildrenIndex/descendants), not the whole workspace.
func TestProgressRootArgScopesToDescendants(t *testing.T) {
	setupProgressTest(t)
	resetProgressFlags(t)

	milestone := &bean.Bean{ID: "beans-mile1", Slug: bean.Slugify("Milestone"), Title: "Milestone", Status: "todo", Type: "milestone"}
	inScope := &bean.Bean{ID: "beans-task1", Slug: bean.Slugify("In scope"), Title: "In scope", Status: "completed", Type: "task", Parent: "beans-mile1"}
	outOfScope := &bean.Bean{ID: "beans-task2", Slug: bean.Slugify("Out of scope"), Title: "Out of scope", Status: "completed", Type: "task"}
	for _, b := range []*bean.Bean{milestone, inScope, outOfScope} {
		if err := core.Create(b); err != nil {
			t.Fatalf("core.Create(%s) error = %v", b.ID, err)
		}
	}

	progressJSON = true

	out := captureProgressStdout(t, func() {
		if err := progressCmd.RunE(progressCmd, []string{"beans-mile1"}); err != nil {
			t.Fatalf("progressCmd.RunE() error = %v", err)
		}
	})

	var result progressResult
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("decoding JSON output: %v; output = %s", err, out)
	}
	// Only the milestone's one descendant (in-scope task) should be
	// counted; the unrelated top-level task must be excluded.
	if result.Total != 1 {
		t.Errorf("expected total=1 (scoped to descendants), got %d", result.Total)
	}
	if result.Completed != 1 {
		t.Errorf("expected completed=1, got %d", result.Completed)
	}
}

// TestProgressRootArgScopesToDescendantsWithPrefix is a regression test
// for a bug where the root argument passed the raw (possibly short) value
// into descendants() instead of the resolved bean's full ID.
// buildChildrenIndex keys its map by full bean IDs (b.Parent is always a
// full ID), so looking up a short ID there missed and silently produced
// 0/0 -- with no error. setupProgressTest uses config.Default(), whose
// prefix is "" (short IDs == full IDs there), so that helper alone can't
// catch this; this test installs a config with a real prefix and resolves
// a short root ID argument against it.
func TestProgressRootArgScopesToDescendantsWithPrefix(t *testing.T) {
	tmpDir := t.TempDir()
	beansDir := filepath.Join(tmpDir, ".beans")
	if err := os.MkdirAll(beansDir, 0755); err != nil {
		t.Fatalf("failed to create test .beans dir: %v", err)
	}

	testCfg := config.DefaultWithPrefix("beans-")
	testCore := beancore.New(beansDir, testCfg)
	if err := testCore.Load(); err != nil {
		t.Fatalf("failed to load core: %v", err)
	}

	oldCore, oldCfg := core, cfg
	core, cfg = testCore, testCfg
	t.Cleanup(func() { core, cfg = oldCore, oldCfg })

	resetProgressFlags(t)

	milestone := &bean.Bean{ID: "beans-mile1", Slug: bean.Slugify("Milestone"), Title: "Milestone", Status: "todo", Type: "milestone"}
	inScope := &bean.Bean{ID: "beans-task1", Slug: bean.Slugify("In scope"), Title: "In scope", Status: "completed", Type: "task", Parent: "beans-mile1"}
	outOfScope := &bean.Bean{ID: "beans-task2", Slug: bean.Slugify("Out of scope"), Title: "Out of scope", Status: "completed", Type: "task"}
	for _, b := range []*bean.Bean{milestone, inScope, outOfScope} {
		if err := core.Create(b); err != nil {
			t.Fatalf("core.Create(%s) error = %v", b.ID, err)
		}
	}

	// "mile1" is a short ID that requires prefix-normalization ("beans-"
	// prepended) to resolve to the full "beans-mile1" bean ID.
	progressJSON = true

	out := captureProgressStdout(t, func() {
		if err := progressCmd.RunE(progressCmd, []string{"mile1"}); err != nil {
			t.Fatalf("progressCmd.RunE() error = %v", err)
		}
	})

	var result progressResult
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("decoding JSON output: %v; output = %s", err, out)
	}
	if result.Total != 1 {
		t.Errorf("expected total=1 (scoped to descendants via short root ID), got %d", result.Total)
	}
	if result.Completed != 1 {
		t.Errorf("expected completed=1, got %d", result.Completed)
	}
}

// TestProgressRootArgErrorsOnUnknownID verifies that a root ID argument
// that does not resolve to any bean returns an error instead of silently
// reporting 0/0 progress (resolver.Bean returns (nil, nil) for unknown
// IDs, so this must be checked explicitly).
func TestProgressRootArgErrorsOnUnknownID(t *testing.T) {
	setupProgressTest(t)
	resetProgressFlags(t)

	err := progressCmd.RunE(progressCmd, []string{"beans-doesnotexist"})
	if err == nil {
		t.Fatal("expected an error for an unknown root ID argument, got nil")
	}
}

// TestProgressRejectsMoreThanOneRootArg verifies that Args wires
// cobra.MaximumNArgs(1), rejecting two positional arguments.
func TestProgressRejectsMoreThanOneRootArg(t *testing.T) {
	if err := progressCmd.Args(progressCmd, []string{"a", "b"}); err == nil {
		t.Fatal("expected an error for more than one root argument, got nil")
	}
}

// TestProgressJSONCarriesResolvedRootID verifies that --json with a short
// root ID argument returns the resolved full ID in the "root" field, never
// the typed short form.
func TestProgressJSONCarriesResolvedRootID(t *testing.T) {
	tmpDir := t.TempDir()
	beansDir := filepath.Join(tmpDir, ".beans")
	if err := os.MkdirAll(beansDir, 0755); err != nil {
		t.Fatalf("failed to create test .beans dir: %v", err)
	}

	testCfg := config.DefaultWithPrefix("beans-")
	testCore := beancore.New(beansDir, testCfg)
	if err := testCore.Load(); err != nil {
		t.Fatalf("failed to load core: %v", err)
	}

	oldCore, oldCfg := core, cfg
	core, cfg = testCore, testCfg
	t.Cleanup(func() { core, cfg = oldCore, oldCfg })

	resetProgressFlags(t)

	milestone := &bean.Bean{ID: "beans-mile1", Slug: bean.Slugify("Milestone"), Title: "Milestone", Status: "todo", Type: "milestone"}
	inScope := &bean.Bean{ID: "beans-task1", Slug: bean.Slugify("In scope"), Title: "In scope", Status: "completed", Type: "task", Parent: "beans-mile1"}
	for _, b := range []*bean.Bean{milestone, inScope} {
		if err := core.Create(b); err != nil {
			t.Fatalf("core.Create(%s) error = %v", b.ID, err)
		}
	}

	progressJSON = true

	out := captureProgressStdout(t, func() {
		if err := progressCmd.RunE(progressCmd, []string{"mile1"}); err != nil {
			t.Fatalf("progressCmd.RunE() error = %v", err)
		}
	})

	var result progressResult
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("decoding JSON output: %v; output = %s", err, out)
	}
	if result.Root != "beans-mile1" {
		t.Errorf("expected root=%q (resolved full ID), got %q", "beans-mile1", result.Root)
	}
}

// TestProgressJSONOmitsRootWhenUnscoped verifies that an unscoped --json
// run has no "root" key at all, proving the omitempty tag.
func TestProgressJSONOmitsRootWhenUnscoped(t *testing.T) {
	setupProgressTest(t)
	resetProgressFlags(t)
	progressJSON = true

	task := &bean.Bean{ID: "beans-a1", Slug: bean.Slugify("A"), Title: "A", Status: "completed", Type: "task"}
	if err := core.Create(task); err != nil {
		t.Fatalf("core.Create error = %v", err)
	}

	out := captureProgressStdout(t, func() {
		if err := progressCmd.RunE(progressCmd, nil); err != nil {
			t.Fatalf("progressCmd.RunE() error = %v", err)
		}
	})

	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatalf("decoding JSON output: %v; output = %s", err, out)
	}
	if _, ok := m["root"]; ok {
		t.Errorf("expected no %q key in unscoped JSON output, got %s", "root", out)
	}
}

// TestProgressScopedPlainTextNamesTheRoot verifies that a scoped plain-text
// run prints a header line naming the root bean before the status lines.
func TestProgressScopedPlainTextNamesTheRoot(t *testing.T) {
	setupProgressTest(t)
	resetProgressFlags(t)

	milestone := &bean.Bean{ID: "beans-mile1", Slug: bean.Slugify("Milestone"), Title: "Milestone", Status: "todo", Type: "milestone"}
	child := &bean.Bean{ID: "beans-task1", Slug: bean.Slugify("Child"), Title: "Child", Status: "completed", Type: "task", Parent: "beans-mile1"}
	for _, b := range []*bean.Bean{milestone, child} {
		if err := core.Create(b); err != nil {
			t.Fatalf("core.Create(%s) error = %v", b.ID, err)
		}
	}

	out := captureProgressStdout(t, func() {
		if err := progressCmd.RunE(progressCmd, []string{"beans-mile1"}); err != nil {
			t.Fatalf("progressCmd.RunE() error = %v", err)
		}
	})

	got := stripANSITest(string(out))
	firstLine := strings.SplitN(got, "\n", 2)[0]
	if !strings.Contains(firstLine, "beans-mile1") || !strings.Contains(firstLine, "Milestone") {
		t.Errorf("expected first line to name the root bean, got %q", firstLine)
	}
}

// TestProgressJSONOutputHasNoBarString verifies that --json returns raw
// counts/percent only -- no bar/progress-bar string field, since bars are
// presentation-only.
func TestProgressJSONOutputHasNoBarString(t *testing.T) {
	setupProgressTest(t)
	resetProgressFlags(t)
	progressJSON = true

	task := &bean.Bean{ID: "beans-a1", Slug: bean.Slugify("A"), Title: "A", Status: "completed", Type: "task"}
	if err := core.Create(task); err != nil {
		t.Fatalf("core.Create error = %v", err)
	}

	out := captureProgressStdout(t, func() {
		if err := progressCmd.RunE(progressCmd, nil); err != nil {
			t.Fatalf("progressCmd.RunE() error = %v", err)
		}
	})

	if strings.Contains(string(out), "━") { // "━"
		t.Errorf("expected JSON output to NOT contain a bar string, got %q", out)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(out, &raw); err != nil {
		t.Fatalf("decoding JSON output: %v; output = %s", err, out)
	}
	for _, forbidden := range []string{"bar", "progressBar", "progress_bar"} {
		if _, ok := raw[forbidden]; ok {
			t.Errorf("expected JSON output to NOT contain a %q field, got %s", forbidden, out)
		}
	}
	for _, required := range []string{"counts", "completed", "total", "percent"} {
		if _, ok := raw[required]; !ok {
			t.Errorf("expected JSON output to contain a %q field, got %s", required, out)
		}
	}
}

// seedProgressCounts creates n beans of the given status for each entry in
// counts, so tests can assert on aggregate counts without caring about
// individual bean identity.
func seedProgressCounts(t *testing.T, counts map[string]int) {
	t.Helper()
	i := 0
	for status, n := range counts {
		for j := 0; j < n; j++ {
			i++
			id := fmt.Sprintf("beans-%s-%d", status, i)
			title := fmt.Sprintf("%s %d", status, i)
			b := &bean.Bean{ID: id, Slug: bean.Slugify(title), Title: title, Status: status, Type: "task"}
			if err := core.Create(b); err != nil {
				t.Fatalf("core.Create error = %v", err)
			}
		}
	}
}

// withTrueColor forces lipgloss's default renderer to TrueColor for the
// duration of the test and restores whatever profile was in force before.
// lipgloss.SetColorProfile exists "mostly for testing purposes" per its own
// doc comment: without it, output captured under `go test` (no controlling
// tty) renders with no ANSI codes at all, which would make colour-bearing
// assertions vacuously true regardless of whether the code under test
// actually applies any colour.
func withTrueColor(t *testing.T) {
	t.Helper()
	old := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(old) })
}

// stripANSITest removes CSI colour/style escape sequences, leaving the text
// a plain-text assertion can match against regardless of the active theme
// or colour profile.
func stripANSITest(s string) string {
	return ansiEscapeTest.ReplaceAllString(s, "")
}

var ansiEscapeTest = regexp.MustCompile("\x1b\\[[0-9;]*m")

// runProgressInTestStore sets up a throwaway store with one bean in each
// configured status and returns the plain-text `beans progress` output.
func runProgressInTestStore(t *testing.T) string {
	t.Helper()
	setupProgressTest(t)
	resetProgressFlags(t)

	for _, sc := range cfg.StatusList() {
		id := fmt.Sprintf("beans-%s1", strings.ReplaceAll(sc.Name, "-", ""))
		title := sc.Name
		b := &bean.Bean{ID: id, Slug: bean.Slugify(title), Title: title, Status: sc.Name, Type: "task"}
		if err := core.Create(b); err != nil {
			t.Fatalf("core.Create(%s) error = %v", b.ID, err)
		}
	}

	return string(captureProgressStdout(t, func() {
		if err := progressCmd.RunE(progressCmd, nil); err != nil {
			t.Fatalf("progressCmd.RunE() error = %v", err)
		}
	}))
}

// TestProgressUsesTheConfiguredStatusColours verifies status lines are
// coloured through ui.ResolveColor against the active theme, not a
// hardcoded palette. Mocha's green for "todo" must appear.
func TestProgressUsesTheConfiguredStatusColours(t *testing.T) {
	withTrueColor(t)
	t.Cleanup(func() { ui.SetTheme("mocha") })
	ui.SetTheme("mocha")

	out := runProgressInTestStore(t)
	if !strings.Contains(out, "166;227;161") {
		t.Errorf("progress does not draw todo in the theme's green:\n%q", out)
	}
}

// TestProgressLabelsEveryConfiguredStatus verifies every configured status
// still gets a labelled line, independent of the colouring added on top.
func TestProgressLabelsEveryConfiguredStatus(t *testing.T) {
	out := stripANSITest(runProgressInTestStore(t))
	for _, want := range []string{"Todo", "Draft", "Completed", "Scrapped", "In Progress"} {
		if !strings.Contains(out, want) {
			t.Errorf("progress is missing the %q line:\n%s", want, out)
		}
	}
}
