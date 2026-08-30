package commands

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
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

// TestDeleteJSONShape pins delete to the same contract as the other batch
// verbs: one ID keeps the envelope earlier releases emitted, several IDs give
// a bare array instead of the hand-built response delete used to construct.
func TestDeleteJSONShape(t *testing.T) {
	t.Run("single stays an envelope", func(t *testing.T) {
		setupDeleteTest(t)
		resetDeleteFlags(t)
		b := mkDeleteBean(t, "beans-del1", "First bean")
		deleteJSON = true

		out := captureDeleteStdout(t, func() {
			if err := deleteCmd.RunE(deleteCmd, []string{b.ID}); err != nil {
				t.Fatalf("deleteCmd.RunE() error = %v", err)
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
