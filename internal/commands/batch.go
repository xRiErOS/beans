package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/hmans/beans/internal/output"
	"github.com/hmans/beans/pkg/bean"
	"github.com/hmans/beans/pkg/beancore"
	"github.com/hmans/beans/pkg/beangraph"
)

// resolveBatchTargets resolves every ID a batch verb was given before any of
// them is written, following the delete command's preflight: an unresolvable
// ID anywhere in the call means nothing is mutated at all.
//
// IDs are normalized first, so that "abc" and "beans-abc" are recognized as
// the same bean. Naming one bean twice is rejected rather than deduplicated:
// the verbs append body sections, and silently collapsing the repeat would
// hide a typo that changed which beans the caller thought they were touching.
//
// The returned code is the output.Err* constant the caller passes to cmdError.
func resolveBatchTargets(ctx context.Context, resolver *beangraph.CoreResolver, ids []string) ([]*bean.Bean, string, error) {
	targets := make([]*bean.Bean, 0, len(ids))
	seen := make(map[string]string, len(ids))

	for _, id := range ids {
		fullID, _ := core.NormalizeID(id)
		if first, dup := seen[fullID]; dup {
			return nil, output.ErrValidation, fmt.Errorf("bean %s named more than once (as %q and %q)", fullID, first, id)
		}
		seen[fullID] = id

		b, err := resolver.Bean(ctx, id)
		if err != nil {
			return nil, output.ErrNotFound, fmt.Errorf("failed to find bean: %v", err)
		}
		if b == nil {
			return nil, output.ErrNotFound, fmt.Errorf("bean not found: %s", id)
		}
		targets = append(targets, b)
	}

	return targets, "", nil
}

// preflightStatusPolicy answers, without writing, the question the write path
// asks in beancore: would moving these beans into newStatus leave a required
// front matter field unset? It mirrors two conditions of that check — the
// policy only applies on a real status change, and the fields contributed by
// --commit/--set count towards satisfying it.
func preflightStatusPolicy(targets []*bean.Bean, newStatus string, setMap map[string]any) error {
	fields := cfg.RequiredFieldsFor(newStatus)
	if len(fields) == 0 {
		return nil
	}

	for _, b := range targets {
		if b.Status == newStatus {
			continue
		}
		probe := *b
		probe.Extra = make(map[string]any, len(b.Extra)+len(setMap))
		for k, v := range b.Extra {
			probe.Extra[k] = v
		}
		for k, v := range setMap {
			probe.Extra[k] = v
		}
		if missing := beancore.MissingRequiredFields(&probe, fields); len(missing) > 0 {
			return &beancore.PolicyViolationError{BeanID: b.ID, Status: newStatus, Missing: missing}
		}
	}
	return nil
}

// emitBatchSuccess renders the result of a batch mutation. With a single ID
// the caller's established single-bean shape is kept, so an existing
// invocation sees byte-identical output; several IDs give one bare array.
func emitBatchSuccess(jsonMode bool, done []*bean.Bean, single func(*bean.Bean) error, line func(*bean.Bean) string) error {
	if jsonMode {
		if len(done) == 1 {
			return single(done[0])
		}
		return output.SuccessMultiple(done)
	}
	for _, b := range done {
		fmt.Println(line(b))
	}
	return nil
}

// emitBatchFailure reports a failure that struck partway through the write
// loop. The preflight rules out unresolvable IDs and policy violations, so
// what remains here is an etag conflict or an I/O error — and by then some
// beans are already written. Unlike delete, which discards that list on its
// way out, the beans that made it through are named, because an agent's next
// move depends on knowing which half of the batch happened.
func emitBatchFailure(jsonMode bool, done []*bean.Bean, err error) error {
	if jsonMode {
		// A bare array cannot carry the failure, so the error case keeps the
		// envelope — one document, with the beans already written inside it.
		_ = output.JSON(output.Response{
			Success: false,
			Beans:   done,
			Count:   len(done),
			Error:   err.Error(),
			Code:    mutationErrorCode(err),
		})
		return fmt.Errorf("%s", err)
	}

	if len(done) > 0 {
		ids := make([]string, len(done))
		for i, b := range done {
			ids[i] = b.ID
		}
		return fmt.Errorf("%v (already written: %s)", err, strings.Join(ids, ", "))
	}
	return err
}
