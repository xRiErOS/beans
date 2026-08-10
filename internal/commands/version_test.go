package commands

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// AC1: WHEN beans version runs THE CLI SHALL report whether the binary
// preserves custom front matter keys.
func TestVersionCmdTextReportsCustomFrontMatter(t *testing.T) {
	oldJSON := versionJSON
	versionJSON = false
	t.Cleanup(func() { versionJSON = oldJSON })

	var buf bytes.Buffer
	versionCmd.SetOut(&buf)
	t.Cleanup(func() { versionCmd.SetOut(nil) })

	if err := versionCmd.RunE(versionCmd, nil); err != nil {
		t.Fatalf("versionCmd.RunE() error = %v", err)
	}

	got := buf.String()
	if !strings.Contains(got, "custom front matter") {
		t.Errorf("version output = %q, want it to mention the custom front matter capability", got)
	}
}

// AC2: WHEN beans version --json runs THE CLI SHALL report the same fact in a
// machine-readable field.
func TestVersionCmdJSONReportsCustomFrontMatter(t *testing.T) {
	oldJSON := versionJSON
	versionJSON = true
	t.Cleanup(func() { versionJSON = oldJSON })

	var buf bytes.Buffer
	versionCmd.SetOut(&buf)
	t.Cleanup(func() { versionCmd.SetOut(nil) })

	if err := versionCmd.RunE(versionCmd, nil); err != nil {
		t.Fatalf("versionCmd.RunE() error = %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v, output = %q", err, buf.String())
	}

	v, ok := decoded["custom_front_matter"]
	if !ok {
		t.Fatalf("decoded JSON missing custom_front_matter field: %v", decoded)
	}
	if v != true {
		t.Errorf("custom_front_matter = %v, want true", v)
	}
}
