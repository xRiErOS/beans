package commands

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hmans/beans/internal/output"
	"github.com/hmans/beans/pkg/bean"
	"github.com/hmans/beans/pkg/beancore"
	"github.com/hmans/beans/pkg/config"
)

// setupCompleteTest installs a throwaway core and default config into the
// package globals completeCmd.RunE reads, and returns a bean already persisted
// in that core.
func setupCompleteTest(t *testing.T) *bean.Bean {
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

	b := &bean.Bean{
		ID:     "beans-test1",
		Slug:   bean.Slugify("A test bean"),
		Title:  "A test bean",
		Status: "todo",
		Type:   "task",
	}
	if err := core.Create(b); err != nil {
		t.Fatalf("core.Create() error = %v", err)
	}
	return b
}

// resetCompleteFlags clears every complete* package global the tests below touch
// and restores the pre-test values afterwards, so tests stay isolated despite
// sharing the completeCmd package-level flag vars.
func resetCompleteFlags(t *testing.T) {
	t.Helper()
	oldJSON, oldSummary := completeJSON, completeSummary
	oldCommit, oldSet := completeCommit, completeSet
	completeJSON, completeSummary = false, ""
	completeCommit, completeSet = "", nil
	t.Cleanup(func() {
		completeJSON, completeSummary = oldJSON, oldSummary
		completeCommit, completeSet = oldCommit, oldSet
	})
}

// TestCompleteSetsStatus verifies that beans complete <id> sets status to completed.
func TestCompleteSetsStatus(t *testing.T) {
	b := setupCompleteTest(t)
	resetCompleteFlags(t)

	if err := completeCmd.RunE(completeCmd, []string{b.ID}); err != nil {
		t.Fatalf("completeCmd.RunE() error = %v", err)
	}

	got, err := core.Get(b.ID)
	if err != nil {
		t.Fatalf("core.Get() error = %v", err)
	}
	if got.Status != "completed" {
		t.Errorf("bean status = %q, want %q", got.Status, "completed")
	}
}

// TestCompleteWithSummaryAppendsSection verifies that --summary appends a properly formatted section.
func TestCompleteWithSummaryAppendsSection(t *testing.T) {
	b := setupCompleteTest(t)
	resetCompleteFlags(t)

	completeSummary = "Implemented via PR #42"
	if err := completeCmd.RunE(completeCmd, []string{b.ID}); err != nil {
		t.Fatalf("completeCmd.RunE() error = %v", err)
	}

	got, err := core.Get(b.ID)
	if err != nil {
		t.Fatalf("core.Get() error = %v", err)
	}
	if got.Status != "completed" {
		t.Errorf("bean status = %q, want %q", got.Status, "completed")
	}
	if !strings.Contains(got.Body, "## Summary of Changes") {
		t.Errorf("body does not contain '## Summary of Changes', body = %q", got.Body)
	}
	if !strings.Contains(got.Body, "Implemented via PR #42") {
		t.Errorf("body does not contain summary text, body = %q", got.Body)
	}
}

// TestCompleteRejectsUnknownID verifies that unknown bean ID returns an error.
func TestCompleteRejectsUnknownID(t *testing.T) {
	setupCompleteTest(t)
	resetCompleteFlags(t)

	if err := completeCmd.RunE(completeCmd, []string{"beans-notfound"}); err == nil {
		t.Errorf("completeCmd.RunE() expected error for unknown ID, got nil")
	}
}

// TestCompleteJSONOutput verifies that --json flag produces valid JSON with expected shape.
func TestCompleteJSONOutput(t *testing.T) {
	b := setupCompleteTest(t)
	resetCompleteFlags(t)

	// Capture stdout to verify JSON output
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}

	oldStdout := os.Stdout
	os.Stdout = w

	completeJSON = true
	runErr := completeCmd.RunE(completeCmd, []string{b.ID})

	os.Stdout = oldStdout
	if err := w.Close(); err != nil {
		t.Fatalf("closing pipe write end: %v", err)
	}

	captured, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("reading captured stdout: %v", err)
	}

	if runErr != nil {
		t.Fatalf("completeCmd.RunE() error = %v", runErr)
	}

	// Parse the JSON output
	dec := json.NewDecoder(bytes.NewReader(captured))
	var resp output.Response
	if err := dec.Decode(&resp); err != nil {
		t.Fatalf("decoding JSON output error = %v; output = %s", err, captured)
	}

	// Verify the response structure
	if !resp.Success {
		t.Errorf("response success = false, want true")
	}
	if resp.Bean == nil {
		t.Errorf("response bean = nil, want *bean.Bean")
	}
	if resp.Bean != nil && resp.Bean.ID != b.ID {
		t.Errorf("response bean ID = %q, want %q", resp.Bean.ID, b.ID)
	}
	if resp.Bean != nil && resp.Bean.Status != "completed" {
		t.Errorf("response bean status = %q, want %q", resp.Bean.Status, "completed")
	}
	if resp.Message != "Bean completed" {
		t.Errorf("response message = %q, want %q", resp.Message, "Bean completed")
	}

	// Also verify the bean was actually persisted with correct status
	got, err := core.Get(b.ID)
	if err != nil {
		t.Fatalf("core.Get() error = %v", err)
	}
	if got.Status != "completed" {
		t.Errorf("persisted bean status = %q, want %q", got.Status, "completed")
	}
}

// setupCompleteTestWithCommitPolicy is like setupCompleteTest but installs a
// require_fields_on policy requiring "commit" on completion.
func setupCompleteTestWithCommitPolicy(t *testing.T) *bean.Bean {
	t.Helper()
	tmpDir := t.TempDir()
	beansDir := filepath.Join(tmpDir, ".beans")
	if err := os.MkdirAll(beansDir, 0755); err != nil {
		t.Fatalf("failed to create test .beans dir: %v", err)
	}

	testCfg := config.Default()
	testCfg.Beans.RequireFieldsOn = map[string][]string{"completed": {"commit"}}
	testCore := beancore.New(beansDir, testCfg)
	if err := testCore.Load(); err != nil {
		t.Fatalf("failed to load core: %v", err)
	}

	oldCore, oldCfg := core, cfg
	core, cfg = testCore, testCfg
	t.Cleanup(func() { core, cfg = oldCore, oldCfg })

	b := &bean.Bean{
		ID:     "beans-test1",
		Slug:   bean.Slugify("A test bean"),
		Title:  "A test bean",
		Status: "todo",
		Type:   "task",
	}
	if err := core.Create(b); err != nil {
		t.Fatalf("core.Create() error = %v", err)
	}
	return b
}

// TestCompleteUnderPolicyWithoutCommitFails verifies that under an active
// require_fields_on policy, completing without --commit fails with the
// POLICY_VIOLATION JSON code.
func TestCompleteUnderPolicyWithoutCommitFails(t *testing.T) {
	b := setupCompleteTestWithCommitPolicy(t)
	resetCompleteFlags(t)

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	oldStdout := os.Stdout
	os.Stdout = w

	completeJSON = true
	runErr := completeCmd.RunE(completeCmd, []string{b.ID})

	os.Stdout = oldStdout
	if err := w.Close(); err != nil {
		t.Fatalf("closing pipe write end: %v", err)
	}
	captured, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("reading captured stdout: %v", err)
	}

	if runErr == nil {
		t.Fatal("completeCmd.RunE() expected error, got nil")
	}

	dec := json.NewDecoder(bytes.NewReader(captured))
	var resp output.Response
	if err := dec.Decode(&resp); err != nil {
		t.Fatalf("decoding JSON output error = %v; output = %s", err, captured)
	}
	if resp.Success {
		t.Errorf("response success = true, want false")
	}
	if resp.Code != output.ErrPolicy {
		t.Errorf("response code = %q, want %q", resp.Code, output.ErrPolicy)
	}
}

// TestCompleteUnderPolicyWithCommitHEADSucceeds verifies that --commit HEAD
// resolves to the repository's current commit SHA and records it under the
// configured commit field in one write.
func TestCompleteUnderPolicyWithCommitHEADSucceeds(t *testing.T) {
	b := setupCompleteTestWithCommitPolicy(t)
	resetCompleteFlags(t)

	repoDir := t.TempDir()
	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = repoDir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	runGit("init", "-q")
	runGit("commit", "-q", "--allow-empty", "-m", "seed")

	headOut, err := exec.Command("git", "-C", repoDir, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatalf("git rev-parse HEAD: %v", err)
	}
	wantSHA := strings.TrimSpace(string(headOut))

	t.Chdir(repoDir)

	completeCommit = "HEAD"
	if err := completeCmd.RunE(completeCmd, []string{b.ID}); err != nil {
		t.Fatalf("completeCmd.RunE() error = %v", err)
	}

	got, err := core.Get(b.ID)
	if err != nil {
		t.Fatalf("core.Get() error = %v", err)
	}
	sha, _ := got.Extra[cfg.GetCommitField()].(string)
	if sha != wantSHA {
		t.Errorf("Extra[%q] = %q, want %q", cfg.GetCommitField(), sha, wantSHA)
	}
	if len(sha) != 40 {
		t.Errorf("commit sha length = %d, want 40", len(sha))
	}
}

// TestCompleteWithUnresolvableCommitFails verifies a --commit ref that does
// not resolve to a commit is rejected rather than silently stored.
func TestCompleteWithUnresolvableCommitFails(t *testing.T) {
	b := setupCompleteTestWithCommitPolicy(t)
	resetCompleteFlags(t)

	repoDir := t.TempDir()
	cmd := exec.Command("git", "init", "-q")
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	t.Chdir(repoDir)

	completeCommit = "deadbeefdeadbeef"
	err := completeCmd.RunE(completeCmd, []string{b.ID})
	if err == nil {
		t.Fatal("completeCmd.RunE() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "does not resolve to a commit") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "does not resolve to a commit")
	}
}

// captureCompleteStdout redirects os.Stdout for the duration of fn and
// returns everything written to it, mirroring captureNextStdout.
func captureCompleteStdout(t *testing.T, fn func()) []byte {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
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

// mkCompleteBean adds another bean to the store set up by setupCompleteTest.
func mkCompleteBean(t *testing.T, id, title string) *bean.Bean {
	t.Helper()
	b := &bean.Bean{
		ID:     id,
		Slug:   bean.Slugify(title),
		Title:  title,
		Status: "todo",
		Type:   "task",
	}
	if err := core.Create(b); err != nil {
		t.Fatalf("core.Create(%s) error = %v", id, err)
	}
	return b
}

func statusOf(t *testing.T, id string) string {
	t.Helper()
	got, err := core.Get(id)
	if err != nil {
		t.Fatalf("core.Get(%s) error = %v", id, err)
	}
	return got.Status
}

// TestCompleteTakesMultipleIDs verifies the batch signature completes every
// named bean and applies the shared --summary to each of them.
func TestCompleteTakesMultipleIDs(t *testing.T) {
	first := setupCompleteTest(t)
	resetCompleteFlags(t)
	second := mkCompleteBean(t, "beans-btc2", "Second bean")
	third := mkCompleteBean(t, "beans-btc3", "Third bean")

	completeSummary = "Shipped together"
	if err := completeCmd.RunE(completeCmd, []string{first.ID, second.ID, third.ID}); err != nil {
		t.Fatalf("completeCmd.RunE() error = %v", err)
	}

	for _, id := range []string{first.ID, second.ID, third.ID} {
		if got := statusOf(t, id); got != "completed" {
			t.Errorf("bean %s status = %q, want %q", id, got, "completed")
		}
		got, _ := core.Get(id)
		if !strings.Contains(got.Body, "Shipped together") {
			t.Errorf("bean %s body does not carry the shared summary, body = %q", id, got.Body)
		}
	}
}

// TestCompletePreflightRejectsUnknownID is the core of the batch promise: an
// unresolvable ID anywhere in the call means nothing is written at all.
func TestCompletePreflightRejectsUnknownID(t *testing.T) {
	first := setupCompleteTest(t)
	resetCompleteFlags(t)
	second := mkCompleteBean(t, "beans-brc2", "Second bean")

	err := completeCmd.RunE(completeCmd, []string{first.ID, "beans-nope", second.ID})
	if err == nil {
		t.Fatal("completeCmd.RunE() error = nil, want an error for the unknown ID")
	}

	for _, id := range []string{first.ID, second.ID} {
		if got := statusOf(t, id); got != "todo" {
			t.Errorf("bean %s status = %q, want it untouched at %q", id, got, "todo")
		}
	}
}

// TestCompleteRejectsDuplicateIDs guards against completing the same bean
// twice in one call, which would append the summary section twice.
func TestCompleteRejectsDuplicateIDs(t *testing.T) {
	first := setupCompleteTest(t)
	resetCompleteFlags(t)

	if err := completeCmd.RunE(completeCmd, []string{first.ID, first.ID}); err == nil {
		t.Fatal("completeCmd.RunE() error = nil, want an error for the repeated ID")
	}
	if got := statusOf(t, first.ID); got != "todo" {
		t.Errorf("bean status = %q, want it untouched at %q", got, "todo")
	}
}

// TestCompletePreflightCatchesPolicyViolation verifies that a required-fields
// policy is evaluated before the first write, not on the way into it.
func TestCompletePreflightCatchesPolicyViolation(t *testing.T) {
	first := setupCompleteTest(t)
	resetCompleteFlags(t)
	second := mkCompleteBean(t, "beans-bpc2", "Second bean")

	cfg.Beans.RequireFieldsOn = map[string][]string{"completed": {"reviewer"}}
	// The first bean satisfies the policy, the second does not — so a
	// per-bean write loop would complete the first before failing.
	first.Extra = map[string]any{"reviewer": "erik"}
	if err := core.Update(first, nil); err != nil {
		t.Fatalf("core.Update(first) error = %v", err)
	}

	if err := completeCmd.RunE(completeCmd, []string{first.ID, second.ID}); err == nil {
		t.Fatal("completeCmd.RunE() error = nil, want a policy violation")
	}

	for _, id := range []string{first.ID, second.ID} {
		if got := statusOf(t, id); got != "todo" {
			t.Errorf("bean %s status = %q, want it untouched at %q", id, got, "todo")
		}
	}
}

// TestCompletePolicySatisfiedByCommitFlag verifies the preflight sees the
// fields --commit and --set contribute, rather than judging the stored bean.
func TestCompletePolicySatisfiedByCommitFlag(t *testing.T) {
	first := setupCompleteTest(t)
	resetCompleteFlags(t)
	second := mkCompleteBean(t, "beans-bsc2", "Second bean")

	cfg.Beans.RequireFieldsOn = map[string][]string{"completed": {"reviewer"}}
	completeSet = []string{"reviewer=erik"}

	if err := completeCmd.RunE(completeCmd, []string{first.ID, second.ID}); err != nil {
		t.Fatalf("completeCmd.RunE() error = %v, want the --set value to satisfy the policy", err)
	}
	for _, id := range []string{first.ID, second.ID} {
		if got := statusOf(t, id); got != "completed" {
			t.Errorf("bean %s status = %q, want %q", id, got, "completed")
		}
	}
}

// TestCompleteJSONShape pins E5 down: one ID keeps the shape earlier releases
// emitted, several IDs give a bare array.
func TestCompleteJSONShape(t *testing.T) {
	t.Run("single stays an envelope", func(t *testing.T) {
		b := setupCompleteTest(t)
		resetCompleteFlags(t)
		completeJSON = true

		out := captureCompleteStdout(t, func() {
			if err := completeCmd.RunE(completeCmd, []string{b.ID}); err != nil {
				t.Fatalf("completeCmd.RunE() error = %v", err)
			}
		})

		var resp struct {
			Success bool `json:"success"`
			Bean    struct {
				ID string `json:"id"`
			} `json:"bean"`
		}
		if err := json.Unmarshal(out, &resp); err != nil {
			t.Fatalf("decoding JSON: %v; output = %s", err, out)
		}
		if !resp.Success || resp.Bean.ID != b.ID {
			t.Errorf("single-ID JSON = %s, want the unchanged envelope for %s", out, b.ID)
		}
	})

	t.Run("multiple gives a bare array", func(t *testing.T) {
		first := setupCompleteTest(t)
		resetCompleteFlags(t)
		second := mkCompleteBean(t, "beans-bjc2", "Second bean")
		completeJSON = true

		out := captureCompleteStdout(t, func() {
			if err := completeCmd.RunE(completeCmd, []string{first.ID, second.ID}); err != nil {
				t.Fatalf("completeCmd.RunE() error = %v", err)
			}
		})

		var got []struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(out, &got); err != nil {
			t.Fatalf("decoding JSON as a bare array: %v; output = %s", err, out)
		}
		if len(got) != 2 || got[0].ID != first.ID || got[1].ID != second.ID {
			t.Errorf("array = %s, want [%s %s]", out, first.ID, second.ID)
		}
	})
}
