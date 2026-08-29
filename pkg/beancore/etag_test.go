package beancore

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/hmans/beans/pkg/bean"
	"github.com/hmans/beans/pkg/config"
)

// beans-xqsq AC1/AC2: the etag `list`, `show` and the write path each expose
// for a bean must be the same value, and that value must be the one
// core.Update's own optimistic-concurrency check accepts. Each of the three
// "commands" below is its own freshly reloaded Core, exactly as `beans list`,
// `beans show` and `beans update` are three separate OS processes against the
// same on-disk store -- the scenario the measured SPF-34aa/SPF-ukwd
// divergence came from, not the same in-memory Core serving all three, which
// would hide the defaulting-on-load discrepancy entirely.
func TestETagConsistentAcrossListShowAndWrite(t *testing.T) {
	tmpDir := t.TempDir()
	beansDir := filepath.Join(tmpDir, BeansDir)
	if err := os.MkdirAll(beansDir, 0755); err != nil {
		t.Fatalf("failed to create test .beans dir: %v", err)
	}
	cfg := config.Default()

	// "beans create": no -p/-t, so priority/type are absent on disk -- the
	// exact shape that gets defaulted in memory on every subsequent load.
	writer := New(beansDir, cfg)
	writer.SetWarnWriter(nil)
	if err := writer.Load(); err != nil {
		t.Fatalf("writer.Load() error = %v", err)
	}
	b := &bean.Bean{ID: "beans-etagx", Slug: bean.Slugify("Etag test"), Title: "Etag test", Status: "todo"}
	if err := writer.Create(b); err != nil {
		t.Fatalf("writer.Create() error = %v", err)
	}

	// "beans list": a fresh process, reads the whole store.
	listCore := New(beansDir, cfg)
	listCore.SetWarnWriter(nil)
	if err := listCore.Load(); err != nil {
		t.Fatalf("listCore.Load() error = %v", err)
	}
	listBean, err := listCore.Get(b.ID)
	if err != nil {
		t.Fatalf("listCore.Get() error = %v", err)
	}
	listETag := listBean.ETag()

	// "beans show": another fresh process, reads one bean.
	showCore := New(beansDir, cfg)
	showCore.SetWarnWriter(nil)
	if err := showCore.Load(); err != nil {
		t.Fatalf("showCore.Load() error = %v", err)
	}
	showBean, err := showCore.Get(b.ID)
	if err != nil {
		t.Fatalf("showCore.Get() error = %v", err)
	}
	showETag := showBean.ETag()

	if listETag != showETag {
		t.Fatalf("list etag %q != show etag %q -- same on-disk bean must report the same etag from every read path", listETag, showETag)
	}

	// "beans update ... --if-match <list's etag>": a third fresh process.
	// The write path must accept the exact token the read paths handed out.
	updateCore := New(beansDir, cfg)
	updateCore.SetWarnWriter(nil)
	if err := updateCore.Load(); err != nil {
		t.Fatalf("updateCore.Load() error = %v", err)
	}
	updateBean, err := updateCore.Get(b.ID)
	if err != nil {
		t.Fatalf("updateCore.Get() error = %v", err)
	}
	updateBean.Priority = "high"
	if err := updateCore.Update(updateBean, &listETag); err != nil {
		t.Fatalf("Update() with the list/show etag as --if-match was rejected: %v (an etag a caller can only read after a write has already failed is not a usable lock)", err)
	}
}

// beans-xqsq AC3: the fix must not turn --if-match into a no-op. A caller
// that reads an etag, then loses a race to a genuinely concurrent write
// before its own write lands, must have that write rejected.
func TestETagIfMatchRejectsGenuineConcurrentModification(t *testing.T) {
	tmpDir := t.TempDir()
	beansDir := filepath.Join(tmpDir, BeansDir)
	if err := os.MkdirAll(beansDir, 0755); err != nil {
		t.Fatalf("failed to create test .beans dir: %v", err)
	}
	cfg := config.Default()

	writer := New(beansDir, cfg)
	writer.SetWarnWriter(nil)
	if err := writer.Load(); err != nil {
		t.Fatalf("writer.Load() error = %v", err)
	}
	b := &bean.Bean{ID: "beans-etagy", Slug: bean.Slugify("Etag race"), Title: "Etag race", Status: "todo"}
	if err := writer.Create(b); err != nil {
		t.Fatalf("writer.Create() error = %v", err)
	}

	// Caller A reads the bean and its etag (its own fresh process).
	callerA := New(beansDir, cfg)
	callerA.SetWarnWriter(nil)
	if err := callerA.Load(); err != nil {
		t.Fatalf("callerA.Load() error = %v", err)
	}
	beanA, err := callerA.Get(b.ID)
	if err != nil {
		t.Fatalf("callerA.Get() error = %v", err)
	}
	staleETag := beanA.ETag()

	// Caller B (a genuinely concurrent process) writes first.
	callerB := New(beansDir, cfg)
	callerB.SetWarnWriter(nil)
	if err := callerB.Load(); err != nil {
		t.Fatalf("callerB.Load() error = %v", err)
	}
	beanB, err := callerB.Get(b.ID)
	if err != nil {
		t.Fatalf("callerB.Get() error = %v", err)
	}
	beanB.Priority = "critical"
	if err := callerB.Update(beanB, nil); err != nil {
		t.Fatalf("callerB.Update() error = %v", err)
	}

	// Caller A's write, using the etag it read before B's write landed,
	// must now be rejected -- not silently accepted.
	beanA.Priority = "low"
	err = callerA.Update(beanA, &staleETag)
	if err == nil {
		t.Fatal("expected an etag-mismatch error for a write racing a genuine concurrent modification, got nil")
	}
	var mismatchErr *ETagMismatchError
	if !errors.As(err, &mismatchErr) {
		t.Fatalf("expected *ETagMismatchError, got %T: %v", err, err)
	}
}

// TestUpdateWithIfMatchAndStatusPolicy covers the two on-disk checks Update
// runs together: the etag it validates If-Match against, and the previous
// status the field policy is measured from. Both read the same file, and they
// have to keep agreeing after that read was consolidated.
func TestUpdateWithIfMatchAndStatusPolicy(t *testing.T) {
	core, _ := setupTestCoreWithRequireFieldsOn(t, map[string][]string{"completed": {"commit"}})
	b := createTestBean(t, core, "test-etpol", "Gated bean", "todo")
	etag := b.ETag()

	t.Run("policy blocks the transition even with a matching etag", func(t *testing.T) {
		b.Status = "completed"
		err := core.Update(b, &etag)

		var polErr *PolicyViolationError
		if !errors.As(err, &polErr) {
			t.Fatalf("Update() error = %v, want *PolicyViolationError", err)
		}
	})

	t.Run("a stale etag is rejected before the policy runs", func(t *testing.T) {
		stale := "0123456789abcdef"
		b.Status = "completed"
		err := core.Update(b, &stale)

		var mismatch *ETagMismatchError
		if !errors.As(err, &mismatch) {
			t.Fatalf("Update() error = %v, want *ETagMismatchError", err)
		}
	})

	t.Run("the transition goes through once the field is there", func(t *testing.T) {
		b.Status = "completed"
		b.Extra = map[string]any{"commit": "abc1234"}
		if err := core.Update(b, &etag); err != nil {
			t.Fatalf("Update() error = %v, want nil", err)
		}

		got, err := core.Get("test-etpol")
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if got.Status != "completed" {
			t.Errorf("Status = %q, want %q", got.Status, "completed")
		}
	})
}
