package version

import (
	"runtime/debug"
	"strings"
)

func init() {
	if info, ok := debug.ReadBuildInfo(); ok {
		applyBuildInfo(info)
	}
}

// applyBuildInfo fills in version metadata that ldflags did not provide,
// using the module and VCS data the Go toolchain stamps into the binary
// (e.g. for a plain `go install`). Values injected via ldflags always win.
func applyBuildInfo(info *debug.BuildInfo) {
	if info == nil {
		return
	}
	if Version == "dev" && info.Main.Version != "" && info.Main.Version != "(devel)" {
		Version = info.Main.Version
	}

	commitFromBuildInfo := Commit == "unknown"
	var modified bool
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			if Commit == "unknown" && s.Value != "" {
				Commit = shortRevision(s.Value)
			}
		case "vcs.time":
			if Date == "unknown" && s.Value != "" {
				Date = s.Value
			}
		case "vcs.modified":
			modified = s.Value == "true"
		}
	}
	if commitFromBuildInfo && modified && Commit != "unknown" && !strings.HasSuffix(Commit, "-dirty") {
		Commit += "-dirty"
	}
}

func shortRevision(rev string) string {
	if len(rev) > 7 {
		return rev[:7]
	}
	return rev
}
