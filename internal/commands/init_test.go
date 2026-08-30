package commands

import (
	"path/filepath"
	"testing"

	"github.com/hmans/beans/pkg/config"
)

func TestInitWithAProfileWritesTheExpandedTypeList(t *testing.T) {
	dir := t.TempDir()

	cfg := config.DefaultWithPrefix("demo-")
	types, ok := config.ProfileTypes("simple")
	if !ok {
		t.Fatal("simple profile missing")
	}
	cfg.Types = types
	cfg.SetConfigDir(dir)
	if err := cfg.Save(dir); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := config.Load(filepath.Join(dir, config.ConfigFileName))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := loaded.RankOf("bucket"); got != 1 {
		t.Errorf("RankOf(\"bucket\") after round-trip = %d, want 1", got)
	}
	if loaded.IsRoadmapType("bucket") {
		t.Error("bucket must stay hidden after a round-trip through the config file")
	}
}
