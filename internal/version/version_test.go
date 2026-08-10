package version

import (
	"encoding/json"
	"strings"
	"testing"
)

// AC1: WHEN beans version runs THE CLI SHALL report whether the binary
// preserves custom front matter keys.
func TestStringReportsCustomFrontMatterCapability(t *testing.T) {
	got := String()
	if !strings.Contains(got, "custom front matter") {
		t.Errorf("String() = %q, want it to mention the custom front matter capability", got)
	}
	if !strings.Contains(got, "preserved") {
		t.Errorf("String() = %q, want it to report the capability as preserved", got)
	}
}

// AC2: WHEN beans version --json runs THE CLI SHALL report the same fact in a
// machine-readable field.
func TestJSONReportsCustomFrontMatterCapability(t *testing.T) {
	info := JSON()
	if !info.CustomFrontMatter {
		t.Errorf("JSON().CustomFrontMatter = false, want true")
	}

	b, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	v, ok := decoded["custom_front_matter"]
	if !ok {
		t.Fatalf("decoded JSON missing custom_front_matter field: %v", decoded)
	}
	if v != true {
		t.Errorf("custom_front_matter = %v, want true", v)
	}
}
