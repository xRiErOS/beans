package output

import (
	"errors"
	"fmt"
	"testing"
)

// TestIsSilentDistinguishesErrorKinds pins the Silent/IsSilent contract
// (beans-lk8t): a plain error and an Emitted one are both NOT silent, and
// only an error built by Silent is.
func TestIsSilentDistinguishesErrorKinds(t *testing.T) {
	plain := errors.New("boom")
	if IsSilent(plain) {
		t.Errorf("IsSilent(plain error) = true, want false")
	}

	emitted := &emittedError{message: "already printed"}
	if IsSilent(emitted) {
		t.Errorf("IsSilent(emittedError) = true, want false")
	}

	silent := Silent("nothing to see here")
	if !IsSilent(silent) {
		t.Errorf("IsSilent(Silent(...)) = false, want true")
	}
}

// TestIsSilentUnwraps pins that a silent error wrapped by fmt.Errorf's %w
// is still recognized -- IsSilent uses errors.As, not a direct type
// assertion.
func TestIsSilentUnwraps(t *testing.T) {
	wrapped := fmt.Errorf("context: %w", Silent("inner"))
	if !IsSilent(wrapped) {
		t.Errorf("IsSilent(wrapped Silent error) = false, want true")
	}
}
