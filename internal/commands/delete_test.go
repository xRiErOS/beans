package commands

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xRiErOS/beans/pkg/bean"
	"github.com/xRiErOS/beans/pkg/beancore"
	"github.com/xRiErOS/beans/pkg/config"
)

// setupDeleteTest installs a throwaway core and default config into the
// package globals deleteCmd.RunE reads. delete had no command-level test at
// all before the batch verbs were unified on one JSON shape.
func setupDeleteTest(t *testing.T) {
	t.Helper()
	beansDir := filepath.Join(t.TempDir(), ".beans")
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

func resetDeleteFlags(t *testing.T) {
	t.Helper()
	oldForce, oldJSON := forceDelete, deleteJSON
	forceDelete, deleteJSON = false, false
	t.Cleanup(func() { forceDelete, deleteJSON = oldForce, oldJSON })
}

func mkDeleteBean(t *testing.T, id, title string) *bean.Bean {
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

func captureDeleteStdout(t *testing.T, fn func()) []byte {
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

// TestDeleteJSONShape pins delete to the same D05 contract as the other
// batch verbs: one ID gives a bare bean document, several IDs give a bare
// array instead of the hand-built response delete used to construct.
func TestDeleteJSONShape(t *testing.T) {
	t.Run("single gives a bare bean", func(t *testing.T) {
		setupDeleteTest(t)
		resetDeleteFlags(t)
		b := mkDeleteBean(t, "beans-del1", "First bean")
		deleteJSON = true

		out := captureDeleteStdout(t, func() {
			if err := deleteCmd.RunE(deleteCmd, []string{b.ID}); err != nil {
				t.Fatalf("deleteCmd.RunE() error = %v", err)
			}
		})

		var got struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(out, &got); err != nil {
			t.Fatalf("decoding JSON: %v; output = %s", err, out)
		}
		if got.ID != b.ID {
			t.Errorf("bare bean = %s, want id %s", out, b.ID)
		}
	})

	t.Run("multiple gives a bare array", func(t *testing.T) {
		setupDeleteTest(t)
		resetDeleteFlags(t)
		first := mkDeleteBean(t, "beans-del2", "First bean")
		second := mkDeleteBean(t, "beans-del3", "Second bean")
		deleteJSON = true

		out := captureDeleteStdout(t, func() {
			if err := deleteCmd.RunE(deleteCmd, []string{first.ID, second.ID}); err != nil {
				t.Fatalf("deleteCmd.RunE() error = %v", err)
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

// TestDeleteRemovesEveryNamedBean guards the behaviour the shape change sits
// on top of.
func TestDeleteRemovesEveryNamedBean(t *testing.T) {
	setupDeleteTest(t)
	resetDeleteFlags(t)
	first := mkDeleteBean(t, "beans-del4", "First bean")
	second := mkDeleteBean(t, "beans-del5", "Second bean")
	forceDelete = true

	captureDeleteStdout(t, func() {
		if err := deleteCmd.RunE(deleteCmd, []string{first.ID, second.ID}); err != nil {
			t.Fatalf("deleteCmd.RunE() error = %v", err)
		}
	})

	for _, id := range []string{first.ID, second.ID} {
		if _, err := core.Get(id); err == nil {
			t.Errorf("bean %s still resolves after delete", id)
		}
	}
}

// answerDeletePrompt runs fn with os.Stdin replaced by answer and returns
// what the prompt wrote.
func answerDeletePrompt(t *testing.T, answer string, fn func()) string {
	t.Helper()
	in, err := os.CreateTemp(t.TempDir(), "stdin")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := in.WriteString(answer); err != nil {
		t.Fatal(err)
	}
	if _, err := in.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	old := os.Stdin
	os.Stdin = in
	t.Cleanup(func() { os.Stdin = old })
	return string(captureDeleteStdout(t, fn))
}

// Delete takes the attachment directory with the bean, so the prompt has to
// say which files that is before the operator consents. Under the current
// sink those are the container's design documents, and a confirmation that
// names only the bean understates what it destroys -- the same reason
// rename's --dry-run names the attachment move.
func TestDeletePromptNamesAttachments(t *testing.T) {
	setupDeleteTest(t)
	resetDeleteFlags(t)
	b := mkDeleteBean(t, "beans-del6", "Host bean")
	dir := filepath.Join(core.Root(), beancore.AttachmentsDir, b.ID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "DESIGN.md"), []byte("x\n"), 0644); err != nil {
		t.Fatal(err)
	}

	out := answerDeletePrompt(t, "n\n", func() {
		if confirmDeleteMultiple([]beanWithLinks{{bean: b, attachments: []string{"DESIGN.md"}}}) {
			t.Fatal("prompt accepted a declined deletion")
		}
	})

	if !strings.Contains(out, "DESIGN.md") {
		t.Fatalf("prompt does not name the attachment:\n%s", out)
	}
}

// The command has to collect the names itself; a prompt that is only fed by
// its caller's test would pass while the real run says nothing.
func TestDeleteRunNamesAttachments(t *testing.T) {
	setupDeleteTest(t)
	resetDeleteFlags(t)
	b := mkDeleteBean(t, "beans-del7", "Host bean")
	dir := filepath.Join(core.Root(), beancore.AttachmentsDir, b.ID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "REQUIREMENTS.md"), []byte("x\n"), 0644); err != nil {
		t.Fatal(err)
	}

	out := answerDeletePrompt(t, "n\n", func() {
		if err := deleteCmd.RunE(deleteCmd, []string{b.ID}); err != nil {
			t.Fatalf("deleteCmd.RunE() error = %v", err)
		}
	})

	if !strings.Contains(out, "REQUIREMENTS.md") {
		t.Fatalf("delete run does not name the attachment:\n%s", out)
	}
	if _, err := core.Get(b.ID); err != nil {
		t.Fatalf("declined deletion still removed the bean: %v", err)
	}
}
