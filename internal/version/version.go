package version

import "fmt"

// Set via ldflags at build time.
var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
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
}

// JSON returns the version information as a struct ready for JSON encoding.
func JSON() Info {
	return Info{
		Version:           Version,
		Commit:            Commit,
		Date:              Date,
		CustomFrontMatter: CustomFrontMatter,
	}
}

// String returns a formatted version string, including whether the binary
// preserves custom front matter keys.
func String() string {
	status := "not preserved"
	if CustomFrontMatter {
		status = "preserved"
	}
	return fmt.Sprintf("beans %s (%s) built %s\ncustom front matter: %s", Version, Commit, Date, status)
}
