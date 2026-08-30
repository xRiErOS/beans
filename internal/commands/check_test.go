package commands

import (
	"testing"

	"github.com/hmans/beans/pkg/bean"
	"github.com/hmans/beans/pkg/config"
)

func TestUnknownTypeBeansFindsBeansTheConfigDoesNotCover(t *testing.T) {
	prev := cfg
	cfg = &config.Config{}
	defer func() { cfg = prev }()

	beans := []*bean.Bean{
		{ID: "a1", Type: "task"},
		{ID: "a2", Type: "chore"},
		{ID: "a3", Type: "bug"},
		{ID: "a4", Type: ""},
	}

	got := unknownTypeBeans(beans)

	if len(got) != 1 {
		t.Fatalf("got %d beans, want 1", len(got))
	}
	if got[0].ID != "a2" {
		t.Errorf("flagged %q, want \"a2\"", got[0].ID)
	}
}

func TestUnknownTypeBeansAcceptsAConfiguredType(t *testing.T) {
	prev := cfg
	cfg = &config.Config{Types: []config.TypeOverride{{Name: "chore"}}}
	defer func() { cfg = prev }()

	if got := unknownTypeBeans([]*bean.Bean{{ID: "a2", Type: "chore"}}); len(got) != 0 {
		t.Errorf("got %d beans, want 0 — chore is configured", len(got))
	}
}
