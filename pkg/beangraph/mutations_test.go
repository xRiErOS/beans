package beangraph

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/xRiErOS/beans/pkg/bean"
	"github.com/xRiErOS/beans/pkg/beancore"
	"github.com/xRiErOS/beans/pkg/beangraph/model"
	"github.com/xRiErOS/beans/pkg/config"
)

// newTestResolver builds a resolver over a throwaway store.
func newTestResolver(t *testing.T) *CoreResolver {
	t.Helper()
	beansDir := filepath.Join(t.TempDir(), ".beans")
	if err := os.MkdirAll(beansDir, 0755); err != nil {
		t.Fatalf("creating test .beans dir: %v", err)
	}
	c := beancore.New(beansDir, config.Default())
	if err := c.Load(); err != nil {
		t.Fatalf("loading core: %v", err)
	}
	return &CoreResolver{Core: c}
}

// TestUpdateBeanWritesTagsInDeterministicOrder pins the tag order down. The
// merge builds its result from a Go map, whose iteration order is randomized
// per run, so without an explicit sort the same input can produce different
// files on consecutive runs.
func TestUpdateBeanWritesTagsInDeterministicOrder(t *testing.T) {
	r := newTestResolver(t)
	ctx := context.Background()

	b := &bean.Bean{
		ID:     "beans-tag1",
		Slug:   bean.Slugify("Tag ordering"),
		Title:  "Tag ordering",
		Status: "todo",
		Type:   "task",
		Tags:   []string{"zeta", "mu"},
	}
	if err := r.Core.Create(b); err != nil {
		t.Fatalf("core.Create: %v", err)
	}

	got, err := r.UpdateBean(ctx, b.ID, model.UpdateBeanInput{
		AddTags:    []string{"alpha", "omega"},
		RemoveTags: []string{"mu"},
	})
	if err != nil {
		t.Fatalf("UpdateBean: %v", err)
	}

	want := []string{"alpha", "omega", "zeta"}
	if len(got.Tags) != len(want) {
		t.Fatalf("tags = %v, want %v", got.Tags, want)
	}
	for i := range want {
		if got.Tags[i] != want[i] {
			t.Fatalf("tags = %v, want %v", got.Tags, want)
		}
	}
}
