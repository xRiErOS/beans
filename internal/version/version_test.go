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

// AC1 (beans-yl2q): WHEN Tree is set (local/dev build) THE CLI SHALL report
// the tree path and whether it is the main worktree.
func TestStringReportsTreeWhenSet(t *testing.T) {
	origTree, origKind := Tree, TreeKind
	defer func() { Tree, TreeKind = origTree, origKind }()

	Tree = "/Users/erik/dev/example/repo"
	TreeKind = "main"

	got := String()
	if !strings.Contains(got, "tree: /Users/erik/dev/example/repo (main)") {
		t.Errorf("String() = %q, want it to report the tree and kind", got)
	}
}

// AC1 (beans-yl2q): WHEN Tree is empty (release build, or ldflags never set
// it) THE CLI SHALL NOT print a tree line, so a published binary never
// carries a host path (rule 4).
func TestStringOmitsTreeWhenUnset(t *testing.T) {
	origTree, origKind := Tree, TreeKind
	defer func() { Tree, TreeKind = origTree, origKind }()

	Tree = ""
	TreeKind = ""

	got := String()
	if strings.Contains(got, "tree:") {
		t.Errorf("String() = %q, want no tree line when Tree is unset", got)
	}
}

// AC2 (beans-yl2q): WHEN Tree is set THE CLI SHALL report the same fact in
// the machine-readable field, and SHALL omit it entirely when unset.
func TestJSONReportsTree(t *testing.T) {
	origTree, origKind := Tree, TreeKind
	defer func() { Tree, TreeKind = origTree, origKind }()

	Tree = "/Users/erik/dev/example/repo"
	TreeKind = "worktree"

	info := JSON()
	if info.Tree != Tree || info.TreeKind != TreeKind {
		t.Errorf("JSON() = %+v, want Tree=%q TreeKind=%q", info, Tree, TreeKind)
	}

	b, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if !strings.Contains(string(b), `"tree":"/Users/erik/dev/example/repo"`) {
		t.Errorf("marshaled JSON = %s, want a tree field", b)
	}

	Tree, TreeKind = "", ""
	b, err = json.Marshal(JSON())
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if strings.Contains(string(b), `"tree"`) {
		t.Errorf("marshaled JSON = %s, want tree field omitted when unset", b)
	}
}
