package commands

import (
	"bytes"
	"strings"
	"testing"

	"github.com/hmans/beans/pkg/beancore"
	"github.com/hmans/beans/pkg/config"
)

// The store path is resolved in exactly one place (resolveBeansPath). Scripts
// that need it must be able to ask for it rather than re-deriving it, so the
// command prints the bare path and nothing else.
func TestPathCmdPrintsResolvedStore(t *testing.T) {
	dir := t.TempDir()
	core = beancore.New(dir, config.Default())

	var out bytes.Buffer
	pathCmd.SetOut(&out)
	pathCmd.Run(pathCmd, nil)

	got := out.String()
	if strings.TrimSpace(got) != dir {
		t.Errorf("expected bare store path %q, got %q", dir, got)
	}
	if strings.Count(got, "\n") != 1 {
		t.Errorf("expected a single line, got %q", got)
	}
}

// Scripts call `beans path`, so the command has to be wired into the CLI, not
// merely defined.
func TestPathCmdIsRegistered(t *testing.T) {
	root := NewRootCmd()
	RegisterCoreCommands(root)

	for _, c := range root.Commands() {
		if c.Name() == "path" {
			return
		}
	}
	t.Fatal("expected a registered \"path\" command, found none")
}
