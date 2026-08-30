package beangraph

import (
	"errors"
	"testing"

	"github.com/xRiErOS/beans/pkg/bean"
	"github.com/xRiErOS/beans/pkg/beancore"
	"github.com/xRiErOS/beans/pkg/config"
)

// TestValidateETagReturnsCoreErrorTypes pins the ETag error types to the domain
// layer. beangraph used to declare its own copies, so a caller matching on
// beancore's types silently missed the resolver's errors (and vice versa).
func TestValidateETagReturnsCoreErrorTypes(t *testing.T) {
	r := newTestResolver(t)

	b := &bean.Bean{
		ID:     "beans-etag1",
		Slug:   bean.Slugify("Etag types"),
		Title:  "Etag types",
		Status: "todo",
		Type:   "task",
	}
	if err := r.Core.Create(b); err != nil {
		t.Fatalf("core.Create: %v", err)
	}

	t.Run("mismatch", func(t *testing.T) {
		wrong := "not-the-current-etag"
		err := r.validateETag(b, &wrong)

		var mismatch *beancore.ETagMismatchError
		if !errors.As(err, &mismatch) {
			t.Fatalf("validateETag() = %T (%v), want *beancore.ETagMismatchError", err, err)
		}
		if mismatch.Provided != wrong {
			t.Errorf("Provided = %q, want %q", mismatch.Provided, wrong)
		}
		if mismatch.Current != b.ETag() {
			t.Errorf("Current = %q, want %q", mismatch.Current, b.ETag())
		}
	})

	t.Run("required", func(t *testing.T) {
		cfg := config.Default()
		cfg.Beans.RequireIfMatch = true
		strict := &CoreResolver{Core: beancore.New(t.TempDir(), cfg)}

		err := strict.validateETag(b, nil)

		var required *beancore.ETagRequiredError
		if !errors.As(err, &required) {
			t.Fatalf("validateETag() = %T (%v), want *beancore.ETagRequiredError", err, err)
		}
	})

	t.Run("match", func(t *testing.T) {
		current := b.ETag()
		if err := r.validateETag(b, &current); err != nil {
			t.Fatalf("validateETag() with current etag = %v, want nil", err)
		}
	})
}
