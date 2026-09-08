package commands

import "github.com/spf13/cobra"

// completionUnbounded is the ValidArgsFunction for verbs whose cobra.Args
// arity has no ceiling (cobra.MinimumNArgs(1): complete, delete, scrap,
// show, start, tag, each declaring "Use: <verb> <id> [id...]"). Every
// position, however many the user has already typed, still names a bean, so
// candidates never stop (R-04 AC-04).
func completionUnbounded(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return beanIDCandidates(), cobra.ShellCompDirectiveNoFileComp
}

// completionUpTo returns a ValidArgsFunction that offers bean-ID candidates
// only while fewer than n positions are already filled, then stops (R-04
// AC-05). n is the number of positions that actually name an existing bean,
// which is not always the verb's cobra.Args ceiling: rename declares
// cobra.MaximumNArgs(2) but n is 1 there, because its second position is a
// brand-new identifier the caller is choosing, never one looked up in the
// store (R-04 AC-04's rename exclusion).
func completionUpTo(n int) func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) >= n {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return beanIDCandidates(), cobra.ShellCompDirectiveNoFileComp
	}
}

// beanIDCandidates lists every bean's ID as a shell-completion candidate,
// each annotated with type/title/status for the shell's description column.
// It walks only core.All(), the in-memory slice PersistentPreRunE already
// populated via Core.Load (pkg/beancore) before any ValidArgsFunction runs --
// it reads Bean.ID/.Type/.Title/.Status and never a bean's Body, never
// pkg/search, and triggers no further disk read (R-04 SC-01, SC-03).
func beanIDCandidates() []string {
	if core == nil {
		return nil
	}
	all := core.All()
	candidates := make([]string, 0, len(all))
	for _, b := range all {
		candidates = append(candidates, b.ID+"\t"+b.Type+" "+b.Title+" ("+b.Status+")")
	}
	return candidates
}
