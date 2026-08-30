package bean

import (
	"testing"
	"time"
)

// TestCloneIsIndependent covers the copy primitive beans-sy1v needs: every
// field the original carries survives the copy, and nothing the caller can
// write to is still shared with the original.
func TestCloneIsIndependent(t *testing.T) {
	created := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	updated := created.Add(time.Hour)
	original := &Bean{
		ID:        "beans-abc1",
		Slug:      "original",
		Path:      "beans-abc1--original.md",
		Title:     "original",
		Status:    "todo",
		Type:      "task",
		Priority:  "normal",
		Tags:      []string{"alpha", "beta"},
		CreatedAt: &created,
		UpdatedAt: &updated,
		Order:     "V",
		Body:      "body",
		Parent:    "beans-par1",
		Blocking:  []string{"beans-blk1"},
		BlockedBy: []string{"beans-blk2"},
		Extra:     map[string]any{"release": "0-4-1"},
	}
	original.SetContentETag([]byte("content"))

	clone := original.Clone()

	if clone == original {
		t.Fatal("Clone() returned the same pointer")
	}
	if *clone.CreatedAt != created || *clone.UpdatedAt != updated {
		t.Errorf("Clone() lost timestamps: created=%v updated=%v", clone.CreatedAt, clone.UpdatedAt)
	}
	if clone.ETag() != original.ETag() {
		t.Errorf("Clone() ETag = %q, want %q", clone.ETag(), original.ETag())
	}
	if clone.ID != original.ID || clone.Slug != original.Slug || clone.Path != original.Path ||
		clone.Title != original.Title || clone.Status != original.Status || clone.Type != original.Type ||
		clone.Priority != original.Priority || clone.Order != original.Order ||
		clone.Body != original.Body || clone.Parent != original.Parent {
		t.Errorf("Clone() lost a scalar field: %+v", clone)
	}

	clone.Tags[0] = "mutated"
	clone.Blocking[0] = "mutated"
	clone.BlockedBy[0] = "mutated"
	clone.Extra["release"] = "mutated"
	*clone.CreatedAt = created.Add(24 * time.Hour)
	clone.Title = "mutated"

	if original.Tags[0] != "alpha" || original.Blocking[0] != "beans-blk1" || original.BlockedBy[0] != "beans-blk2" {
		t.Errorf("mutating the clone's slices reached the original: %+v", original)
	}
	if original.Extra["release"] != "0-4-1" {
		t.Errorf("mutating the clone's Extra reached the original: %v", original.Extra)
	}
	if *original.CreatedAt != created {
		t.Errorf("mutating the clone's CreatedAt reached the original: %v", original.CreatedAt)
	}
	if original.Title != "original" {
		t.Errorf("mutating the clone's Title reached the original: %q", original.Title)
	}

	// Appending to a clone's slice must not write into the original's
	// backing array either.
	shared := &Bean{ID: "beans-abc2", Tags: make([]string, 1, 4)}
	shared.Tags[0] = "keep"
	c2 := shared.Clone()
	c2.Tags = append(c2.Tags, "added")
	if len(shared.Tags) != 1 {
		t.Errorf("clone append changed the original's length: %v", shared.Tags)
	}
	shared.Tags = shared.Tags[:cap(shared.Tags)]
	if shared.Tags[1] != "" {
		t.Errorf("clone append wrote into the original's backing array: %v", shared.Tags)
	}
}

func TestCloneNil(t *testing.T) {
	var b *Bean
	if b.Clone() != nil {
		t.Error("Clone() on a nil bean should return nil")
	}
}
