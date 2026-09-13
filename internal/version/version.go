package version

import "fmt"

// Set via ldflags at build time.
var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"

	// Tree is the absolute path of the git worktree this binary was built
	// from. Set only by `mise run build` (local/dev builds); left empty by
	// .goreleaser.yaml so a published release binary never carries a host
	// path (rule 4). TreeKind is "main" or "worktree" and is set alongside
	// it (beans-yl2q).
	Tree     = ""
	TreeKind = ""
)

// CustomFrontMatter reports whether this binary preserves unknown ("custom")
// front matter keys (pkg/bean.Bean.Extra) across a parse/render round-trip
// instead of silently dropping them. It is always true for this and every
// following version -- an older binary (predating this field) simply lacks
// it, which is what makes the two tell apart in `beans version` output.
const CustomFrontMatter = true

// Info is the machine-readable shape of `beans version --json`.
type Info struct {
	Version           string `json:"version"`
	Commit            string `json:"commit"`
	Date              string `json:"date"`
	CustomFrontMatter bool   `json:"custom_front_matter"`
	Tree              string `json:"tree,omitempty"`
	TreeKind          string `json:"tree_kind,omitempty"`
}

// JSON returns the version information as a struct ready for JSON encoding.
func JSON() Info {
	return Info{
		Version:           Version,
		Commit:            Commit,
		Date:              Date,
		CustomFrontMatter: CustomFrontMatter,
		Tree:              Tree,
		TreeKind:          TreeKind,
	}
}

// String returns a formatted version string, including whether the binary
// preserves custom front matter keys and, for local builds, which tree it
// was built from.
func String() string {
	status := "not preserved"
	if CustomFrontMatter {
		status = "preserved"
	}
	s := fmt.Sprintf("beans %s (%s) built %s\ncustom front matter: %s", Version, Commit, Date, status)
	if Tree != "" {
		kind := TreeKind
		if kind == "" {
			kind = "unknown"
		}
		s += fmt.Sprintf("\ntree: %s (%s)", Tree, kind)
	}
	return s
}
