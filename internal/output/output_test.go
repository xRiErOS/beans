package output

import (
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/hmans/beans/pkg/bean"
)

// captureStdout redirects os.Stdout for the duration of fn and returns
// everything written to it. SuccessSingle/SuccessMultiple encode directly
// to os.Stdout, so this is the only way to observe their output.
func captureStdout(t *testing.T, fn func()) []byte {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}

	orig := os.Stdout
	os.Stdout = w
	fn()
	os.Stdout = orig
	w.Close()

	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("reading captured stdout: %v", err)
	}
	return data
}

func TestSuccessSingleAndMultipleAgreeOnExtraField(t *testing.T) {
	// list --json uses SuccessMultiple, show --json uses SuccessSingle.
	// Both must expose extra front matter keys under the same field name.
	withExtra := &bean.Bean{
		ID:     "test-1",
		Title:  "With Extra",
		Status: "todo",
		Extra:  map[string]any{"release": "0-4-1"},
	}
	withoutExtra := &bean.Bean{
		ID:     "test-2",
		Title:  "Without Extra",
		Status: "todo",
	}

	singleOut := captureStdout(t, func() {
		if err := SuccessSingle(withExtra); err != nil {
			t.Fatalf("SuccessSingle: %v", err)
		}
	})
	multipleOut := captureStdout(t, func() {
		if err := SuccessMultiple([]*bean.Bean{withExtra, withoutExtra}); err != nil {
			t.Fatalf("SuccessMultiple: %v", err)
		}
	})

	var single map[string]interface{}
	if err := json.Unmarshal(singleOut, &single); err != nil {
		t.Fatalf("unmarshal SuccessSingle output: %v", err)
	}
	singleExtra, ok := single["extra"].(map[string]interface{})
	if !ok || singleExtra["release"] != "0-4-1" {
		t.Errorf("SuccessSingle output missing extra.release, got: %s", singleOut)
	}

	var multiple []map[string]interface{}
	if err := json.Unmarshal(multipleOut, &multiple); err != nil {
		t.Fatalf("unmarshal SuccessMultiple output: %v", err)
	}
	if len(multiple) != 2 {
		t.Fatalf("len(multiple) = %d, want 2", len(multiple))
	}
	multiExtra, ok := multiple[0]["extra"].(map[string]interface{})
	if !ok || multiExtra["release"] != "0-4-1" {
		t.Errorf("SuccessMultiple output missing extra.release for first bean, got: %s", multipleOut)
	}
	if _, ok := multiple[1]["extra"]; ok {
		t.Errorf("SuccessMultiple output should omit extra for bean without extra keys, got: %s", multipleOut)
	}
}
