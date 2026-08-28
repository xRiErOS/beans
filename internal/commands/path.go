package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var pathCmd = &cobra.Command{
	Use:   "path",
	Short: "Print the resolved beans directory",
	Long: `Prints the absolute path of the beans directory this invocation resolved to.

Resolution follows the usual precedence (--beans-path, then the .beans.yml found
upward or named via --config, then BEANS_PATH, then the config directory), so
scripts that need the store can ask for it here instead of re-deriving it and
drifting from the CLI.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintln(cmd.OutOrStdout(), core.Root())
	},
}

func RegisterPathCmd(root *cobra.Command) {
	root.AddCommand(pathCmd)
}
