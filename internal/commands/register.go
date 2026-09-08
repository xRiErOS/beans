package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// audienceAnnotationKey carries the R-07 audience marker in a command's
// Annotations map: whether a verb is meant for direct human/interactive use
// (audienceUserFacing) or is plumbing consumed by scripts, shell completion,
// or the CLI's own bootstrap (audiencePlumbing). It is the single source of
// truth an interactive caller (e.g. `beans pick`) queries through
// IsUserFacing — no second, driftable verb-name list may exist anywhere
// else in the codebase (SC-03).
const (
	audienceAnnotationKey = "beans.audience"
	audienceUserFacing    = "user-facing"
	audiencePlumbing      = "plumbing"
)

func markPlumbing(cmd *cobra.Command) {
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	cmd.Annotations[audienceAnnotationKey] = audiencePlumbing
}

func markUserFacing(cmd *cobra.Command) {
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	cmd.Annotations[audienceAnnotationKey] = audienceUserFacing
}

// IsUserFacing reports whether cmd was classified as a verb meant for a
// human to type directly, as opposed to plumbing (AC1/AC4). Every command
// under the root carries the marker once RegisterCoreCommands returns, so a
// caller needs no verb-name literal of its own to filter by audience.
func IsUserFacing(cmd *cobra.Command) bool {
	return cmd.Annotations[audienceAnnotationKey] == audienceUserFacing
}

// RegisterCoreCommands adds all core CLI commands to the root command.
func RegisterCoreCommands(root *cobra.Command) {
	RegisterArchiveCmd(root)
	RegisterCheckCmd(root)
	RegisterCompleteCmd(root)
	RegisterCreateCmd(root)
	RegisterDeleteCmd(root)
	RegisterGraphCmd(root)
	RegisterGraphqlCmd(root)
	RegisterInitCmd(root)
	markPlumbing(initCmd)
	RegisterListCmd(root)
	RegisterMilestonesCmd(root)
	RegisterNextCmd(root)
	RegisterOrderCmd(root)
	RegisterPathCmd(root)
	markPlumbing(pathCmd)
	RegisterPrimeCmd(root)
	RegisterProgressCmd(root)
	RegisterPromoteCmd(root)
	RegisterRenameCmd(root)
	RegisterRoadmapCmd(root)
	RegisterScrapCmd(root)
	RegisterShowCmd(root)
	RegisterStartCmd(root)
	RegisterTagCmd(root)
	RegisterUpdateCmd(root)
	RegisterVersionCmd(root)
	markPlumbing(versionCmd)

	// Deprecated placeholders for commands that moved to separate binaries
	registerDeprecatedCmd(root, "serve", "beans-serve")
	registerDeprecatedCmd(root, "tui", "beans-tui")

	// Cobra's own help and completion commands are plumbing exactly like
	// init/path/version (AC2). Materialise them now — idempotent, and the
	// standard way a cobra app gets a handle on its own built-ins before
	// Execute() — so they carry the marker before any caller inspects the
	// tree, the same as every verb this package registers explicitly.
	root.InitDefaultHelpCmd()
	root.InitDefaultCompletionCmd()
	for _, cmd := range root.Commands() {
		if cmd.Name() == "help" || cmd.Name() == "completion" {
			markPlumbing(cmd)
		}
	}

	// Every verb this loop has not already marked plumbing above —
	// including the 22 real commands and the serve/tui stubs — is
	// user-facing per AC2's "every other registered verb" default.
	for _, cmd := range root.Commands() {
		if cmd.Annotations[audienceAnnotationKey] == "" {
			markUserFacing(cmd)
		}
	}
}

func registerDeprecatedCmd(root *cobra.Command, name, binary string) {
	root.AddCommand(&cobra.Command{
		Use:    name,
		Args:   cobra.NoArgs,
		Short:  fmt.Sprintf("(moved to %s)", binary),
		Hidden: true,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(os.Stderr, "The %q command has moved to a separate binary: %s\n", name, binary)
			fmt.Fprintf(os.Stderr, "Please install and use %q instead.\n", binary)
			os.Exit(1)
		},
	})
}
