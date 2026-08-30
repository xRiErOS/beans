package version

import (
	"runtime/debug"
	"strings"
	"testing"
)

func resetVersionGlobals(t *testing.T) {
	t.Helper()
	origVersion, origCommit, origDate := Version, Commit, Date
	Version, Commit, Date = "dev", "unknown", "unknown"
	t.Cleanup(func() {
		Version, Commit, Date = origVersion, origCommit, origDate
	})
}

func TestApplyBuildInfoFillsInDefaultsFromVCSSettings(t *testing.T) {
	resetVersionGlobals(t)

	info := &debug.BuildInfo{
		Main: debug.Module{Version: "v0.7.0"},
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "abcdef0123456789abcdef0123456789abcdef01"},
			{Key: "vcs.time", Value: "2026-08-30T18:00:00Z"},
		},
	}
	applyBuildInfo(info)

	if Version != "v0.7.0" {
		t.Errorf("Version = %q, want %q", Version, "v0.7.0")
	}
	if Commit != "abcdef0" {
		t.Errorf("Commit = %q, want %q", Commit, "abcdef0")
	}
	if Date != "2026-08-30T18:00:00Z" {
		t.Errorf("Date = %q, want %q", Date, "2026-08-30T18:00:00Z")
	}
}

func TestApplyBuildInfoNeverOverridesLdflagsValues(t *testing.T) {
	resetVersionGlobals(t)
	Version, Commit, Date = "v9.9.9", "deadbee", "2020-01-01T00:00:00Z"

	info := &debug.BuildInfo{
		Main: debug.Module{Version: "v0.7.0"},
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "1111111111111111111111111111111111111111"},
			{Key: "vcs.time", Value: "2026-08-30T18:00:00Z"},
			{Key: "vcs.modified", Value: "true"},
		},
	}
	applyBuildInfo(info)

	if Version != "v9.9.9" {
		t.Errorf("Version = %q, want unchanged %q", Version, "v9.9.9")
	}
	if Commit != "deadbee" {
		t.Errorf("Commit = %q, want unchanged %q", Commit, "deadbee")
	}
	if Date != "2020-01-01T00:00:00Z" {
		t.Errorf("Date = %q, want unchanged %q", Date, "2020-01-01T00:00:00Z")
	}
}

func TestApplyBuildInfoMarksDirtyCommit(t *testing.T) {
	resetVersionGlobals(t)

	info := &debug.BuildInfo{
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "abcdef0123456789abcdef0123456789abcdef01"},
			{Key: "vcs.modified", Value: "true"},
		},
	}
	applyBuildInfo(info)

	if !strings.HasSuffix(Commit, "-dirty") {
		t.Errorf("Commit = %q, want suffix -dirty", Commit)
	}
}

func TestApplyBuildInfoHandlesNilInfo(t *testing.T) {
	resetVersionGlobals(t)
	applyBuildInfo(nil)
	if Version != "dev" || Commit != "unknown" || Date != "unknown" {
		t.Errorf("applyBuildInfo(nil) mutated defaults: %q %q %q", Version, Commit, Date)
	}
}
