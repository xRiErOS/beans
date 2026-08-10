package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hmans/beans/pkg/beancore"
	"github.com/hmans/beans/pkg/config"
)

// setupCreateTest installs a throwaway core and default config into the
// package globals createCmd.RunE reads, and restores both afterwards.
func setupCreateTest(t *testing.T) {
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

// resetCreateFlags clears every create* package global the tests below touch
// and restores the pre-test values afterwards, so tests stay isolated despite
// sharing the createCmd package-level flag vars.
func resetCreateFlags(t *testing.T) {
	t.Helper()
	oldType, oldSet, oldUnset := createType, createSet, createUnset
	createType, createSet, createUnset = "", nil, nil
	t.Cleanup(func() {
		createType, createSet, createUnset = oldType, oldSet, oldUnset
	})
}

// AC1/AC2/SC-01: --set writes extra front matter keys, and repeats accumulate.
func TestCreateCmdSetWritesExtraKeys(t *testing.T) {
	setupCreateTest(t)
	resetCreateFlags(t)

	createType = "task"
	createSet = []string{"release=0-4-1", "klasse=bugfix"}

	if err := createCmd.RunE(createCmd, []string{"x"}); err != nil {
		t.Fatalf("createCmd.RunE() error = %v", err)
	}

	list := core.All()
	if len(list) != 1 {
		t.Fatalf("expected 1 bean, got %d", len(list))
	}
	b := list[0]
	if b.Extra["release"] != "0-4-1" {
		t.Errorf("Extra[release] = %v, want %q", b.Extra["release"], "0-4-1")
	}
	if b.Extra["klasse"] != "bugfix" {
		t.Errorf("Extra[klasse] = %v, want %q", b.Extra["klasse"], "bugfix")
	}
}

// AC4/SC-01: --set on a reserved key fails naming the native flag, and no
// bean is created.
func TestCreateCmdSetReservedKeyFails(t *testing.T) {
	setupCreateTest(t)
	resetCreateFlags(t)

	createType = "task"
	createSet = []string{"status=done"}

	err := createCmd.RunE(createCmd, []string{"x"})
	if err == nil {
		t.Fatal("expected error for reserved key")
	}
	if !contains(err.Error(), "--status") {
		t.Errorf("expected error to name --status, got %q", err.Error())
	}

	list := core.All()
	if len(list) != 0 {
		t.Errorf("expected no bean to be created, got %d", len(list))
	}
}

// AC6: --set without "=" is a usage error.
func TestCreateCmdSetWithoutEqualsFails(t *testing.T) {
	setupCreateTest(t)
	resetCreateFlags(t)

	createType = "task"
	createSet = []string{"release"}

	if err := createCmd.RunE(createCmd, []string{"x"}); err == nil {
		t.Fatal("expected error for --set without '='")
	}
}

// Conflict resolution: a key named by both --set and --unset in the same
// invocation ends up unset (see applyExtraOps).
func TestCreateCmdSetAndUnsetSameKey(t *testing.T) {
	setupCreateTest(t)
	resetCreateFlags(t)

	createType = "task"
	createSet = []string{"release=0-4-1"}
	createUnset = []string{"release"}

	if err := createCmd.RunE(createCmd, []string{"x"}); err != nil {
		t.Fatalf("createCmd.RunE() error = %v", err)
	}

	list := core.All()
	if len(list) != 1 {
		t.Fatalf("expected 1 bean, got %d", len(list))
	}
	if _, ok := list[0].Extra["release"]; ok {
		t.Errorf("expected release to be unset, Extra = %#v", list[0].Extra)
	}
}
