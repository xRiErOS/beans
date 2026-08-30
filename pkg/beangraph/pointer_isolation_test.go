package beangraph

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/hmans/beans/pkg/bean"
	"github.com/hmans/beans/pkg/beancore"
	"github.com/hmans/beans/pkg/beangraph/model"
	"github.com/hmans/beans/pkg/config"
)

// resolverWithBeans builds a resolver over a throwaway store with a small
// fixture: p (feature) with children c1, c2; x blocked_by y. beangraph
// cannot reach c.beans directly (unlike pkg/beancore's isolation tests), so
// these tests assert the behavioural property that matters: mutating a
// resolver result must never be observable through a fresh read.
func resolverWithBeans(t *testing.T) *CoreResolver {
	t.Helper()
	beansDir := filepath.Join(t.TempDir(), ".beans")
	if err := os.MkdirAll(beansDir, 0755); err != nil {
		t.Fatalf("creating test .beans dir: %v", err)
	}
	core := beancore.New(beansDir, config.Default())
	if err := core.Load(); err != nil {
		t.Fatalf("loading core: %v", err)
	}

	must := func(b *bean.Bean) {
		t.Helper()
		if err := core.Create(b); err != nil {
			t.Fatalf("creating fixture bean %s: %v", b.ID, err)
		}
	}

	must(&bean.Bean{ID: "beans-p", Slug: "p", Title: "p", Status: "todo", Type: "feature"})
	must(&bean.Bean{ID: "beans-c1", Slug: "c1", Title: "c1", Status: "todo", Type: "task", Parent: "beans-p"})
	must(&bean.Bean{ID: "beans-c2", Slug: "c2", Title: "c2", Status: "todo", Type: "task", Parent: "beans-p"})
	must(&bean.Bean{ID: "beans-y", Slug: "y", Title: "y", Status: "todo", Type: "task", Blocking: []string{"beans-x"}})
	must(&bean.Bean{ID: "beans-x", Slug: "x", Title: "x", Status: "todo", Type: "task", BlockedBy: []string{"beans-y"}})

	return &CoreResolver{Core: core}
}

func strPtr(s string) *string { return &s }

// TestBeanQueryResultIsDetached covers the single-bean query resolver.
func TestBeanQueryResultIsDetached(t *testing.T) {
	r := resolverWithBeans(t)
	ctx := context.Background()

	got, err := r.Bean(ctx, "beans-p")
	if err != nil {
		t.Fatalf("Bean() error = %v", err)
	}
	if got == nil {
		t.Fatal("Bean() returned nil")
	}
	got.Title = "mutated"

	fresh, err := r.Core.Get("beans-p")
	if err != nil {
		t.Fatalf("Core.Get() error = %v", err)
	}
	if fresh.Title == "mutated" {
		t.Error("mutating the Bean() result reached the store (observed via Get)")
	}
	for _, b := range r.Core.All() {
		if b.ID == "beans-p" && b.Title == "mutated" {
			t.Error("mutating the Bean() result reached the store (observed via All)")
		}
	}
}

// TestFieldResolverResultsAreDetached covers BeanBlockedBy, BeanBlocking,
// BeanParent and BeanChildren. BeanChildren and BeanBlockedBy reach the
// store through FindIncomingLinks, which beans-x4ex's site list missed.
func TestFieldResolverResultsAreDetached(t *testing.T) {
	r := resolverWithBeans(t)
	ctx := context.Background()

	p, err := r.Core.Get("beans-p")
	if err != nil {
		t.Fatalf("Core.Get(p) error = %v", err)
	}
	x, err := r.Core.Get("beans-x")
	if err != nil {
		t.Fatalf("Core.Get(x) error = %v", err)
	}
	y, err := r.Core.Get("beans-y")
	if err != nil {
		t.Fatalf("Core.Get(y) error = %v", err)
	}
	c1, err := r.Core.Get("beans-c1")
	if err != nil {
		t.Fatalf("Core.Get(c1) error = %v", err)
	}

	children, err := r.BeanChildren(ctx, p, nil)
	if err != nil {
		t.Fatalf("BeanChildren() error = %v", err)
	}
	if len(children) == 0 {
		t.Fatal("BeanChildren() returned no children")
	}
	for _, c := range children {
		c.Title = "mutated"
	}
	for _, b := range r.Core.All() {
		if (b.ID == "beans-c1" || b.ID == "beans-c2") && b.Title == "mutated" {
			t.Errorf("mutating a BeanChildren() result reached the store: %s", b.ID)
		}
	}

	blockedBy, err := r.BeanBlockedBy(ctx, x, nil)
	if err != nil {
		t.Fatalf("BeanBlockedBy() error = %v", err)
	}
	if len(blockedBy) == 0 {
		t.Fatal("BeanBlockedBy() returned no results")
	}
	for _, b := range blockedBy {
		b.Title = "mutated"
	}
	freshY, err := r.Core.Get("beans-y")
	if err != nil {
		t.Fatalf("Core.Get(y) error = %v", err)
	}
	if freshY.Title == "mutated" {
		t.Error("mutating a BeanBlockedBy() result reached the store")
	}

	blocking, err := r.BeanBlocking(ctx, y, nil)
	if err != nil {
		t.Fatalf("BeanBlocking() error = %v", err)
	}
	if len(blocking) == 0 {
		t.Fatal("BeanBlocking() returned no results")
	}
	for _, b := range blocking {
		b.Title = "mutated"
	}
	freshX, err := r.Core.Get("beans-x")
	if err != nil {
		t.Fatalf("Core.Get(x) error = %v", err)
	}
	if freshX.Title == "mutated" {
		t.Error("mutating a BeanBlocking() result reached the store")
	}

	parentOfC1, err := r.BeanParent(ctx, c1)
	if err != nil {
		t.Fatalf("BeanParent(c1) error = %v", err)
	}
	if parentOfC1 == nil {
		t.Fatal("BeanParent(c1) returned nil, want beans-p")
	}
	parentOfC1.Title = "mutated"
	freshP, err := r.Core.Get("beans-p")
	if err != nil {
		t.Fatalf("Core.Get(p) error = %v", err)
	}
	if freshP.Title == "mutated" {
		t.Error("mutating a BeanParent() result reached the store")
	}
}

// TestMutationResolverResultIsDetached covers UpdateBean, SetParent,
// AddBlocking and AddBlockedBy — the sites listed in beans-8p3h.
func TestMutationResolverResultIsDetached(t *testing.T) {
	r := resolverWithBeans(t)
	ctx := context.Background()

	t.Run("UpdateBean", func(t *testing.T) {
		got, err := r.UpdateBean(ctx, "beans-c1", model.UpdateBeanInput{Title: strPtr("first")})
		if err != nil {
			t.Fatalf("UpdateBean() error = %v", err)
		}
		got.Title = "second"
		fresh, err := r.Core.Get("beans-c1")
		if err != nil {
			t.Fatalf("Core.Get() error = %v", err)
		}
		if fresh.Title != "first" {
			t.Errorf("store Title = %q, want %q", fresh.Title, "first")
		}
	})

	t.Run("SetParent", func(t *testing.T) {
		got, err := r.SetParent(ctx, "beans-c2", nil, nil)
		if err != nil {
			t.Fatalf("SetParent() error = %v", err)
		}
		got.Parent = "mutated"
		fresh, err := r.Core.Get("beans-c2")
		if err != nil {
			t.Fatalf("Core.Get() error = %v", err)
		}
		if fresh.Parent != "" {
			t.Errorf("store Parent = %q, want empty", fresh.Parent)
		}
	})

	t.Run("AddBlocking", func(t *testing.T) {
		got, err := r.AddBlocking(ctx, "beans-c1", "beans-c2", nil)
		if err != nil {
			t.Fatalf("AddBlocking() error = %v", err)
		}
		got.Blocking[0] = "mutated"
		fresh, err := r.Core.Get("beans-c1")
		if err != nil {
			t.Fatalf("Core.Get() error = %v", err)
		}
		found := false
		for _, id := range fresh.Blocking {
			if id == "beans-c2" {
				found = true
			}
		}
		if !found {
			t.Errorf("store Blocking = %v, want to contain %q", fresh.Blocking, "beans-c2")
		}
	})

	t.Run("AddBlockedBy", func(t *testing.T) {
		got, err := r.AddBlockedBy(ctx, "beans-c1", "beans-y", nil)
		if err != nil {
			t.Fatalf("AddBlockedBy() error = %v", err)
		}
		got.BlockedBy[0] = "mutated"
		fresh, err := r.Core.Get("beans-c1")
		if err != nil {
			t.Fatalf("Core.Get() error = %v", err)
		}
		found := false
		for _, id := range fresh.BlockedBy {
			if id == "beans-y" {
				found = true
			}
		}
		if !found {
			t.Errorf("store BlockedBy = %v, want to contain %q", fresh.BlockedBy, "beans-y")
		}
	})
}

// TestMutationIsRaceFreeAgainstAll reports a real DATA RACE today (before
// step 2), because UpdateBean mutates the stored struct with no lock held
// while All() copies it. Meaningful under -race (`just test-race`).
func TestMutationIsRaceFreeAgainstAll(t *testing.T) {
	r := resolverWithBeans(t)
	ctx := context.Background()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := range 200 {
			title := "even"
			if i%2 == 1 {
				title = "odd"
			}
			if _, err := r.UpdateBean(ctx, "beans-c1", model.UpdateBeanInput{Title: strPtr(title)}); err != nil {
				return
			}
		}
	}()

	go func() {
		defer wg.Done()
		for range 200 {
			for _, b := range r.Core.All() {
				_ = fmt.Sprintf("%s %s %s %v %v %v", b.ID, b.Title, b.Status, b.Tags, b.Blocking, b.Extra)
			}
		}
	}()

	wg.Wait()
}
