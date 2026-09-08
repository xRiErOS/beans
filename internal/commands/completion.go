package commands

import (
	"sort"

	"github.com/spf13/cobra"
	"github.com/xRiErOS/beans/pkg/bean"
)

// completionDirective is the ShellCompDirective every ValidArgsFunction in
// this file returns alongside its candidates. ShellCompDirectiveKeepOrder
// tells the shell to preserve beanIDCandidates' actionable-first ordering
// instead of re-sorting candidates alphabetically (beans-sfle AC-01).
const completionDirective = cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder

// completionUnbounded is the ValidArgsFunction for verbs whose cobra.Args
// arity has no ceiling (cobra.MinimumNArgs(1): complete, delete, scrap,
// show, start, tag, each declaring "Use: <verb> <id> [id...]"). Every
// position, however many the user has already typed, still names a bean, so
// candidates never stop (R-04 AC-04).
func completionUnbounded(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return beanIDCandidates(), completionDirective
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
			return nil, completionDirective
		}
		return beanIDCandidates(), completionDirective
	}
}

// completionStatusRank buckets a bean's status into the three-tier
// actionable-first order beans-sfle AC-01 requires: todo/in-progress (0)
// before draft (1), and both before completed/scrapped (2). A status
// outside this vocabulary is not actionable either, so it ranks with draft
// rather than being silently dropped -- every bean stays a candidate
// regardless of status (AC-02).
func completionStatusRank(status string) int {
	switch status {
	case "todo", "in-progress":
		return 0
	case "completed", "scrapped":
		return 2
	default:
		return 1
	}
}

// beanIDCandidates lists every bean's ID as a shell-completion candidate,
// each annotated with type/title/status for the shell's description column,
// ordered actionable-first (beans-sfle AC-01). Sorting never removes a
// candidate: every bean the store holds is still present afterward, just
// reordered, so prefix input keeps reaching any bean-ID (AC-02). It walks
// only core.All(), the in-memory slice PersistentPreRunE already populated
// via Core.Load (pkg/beancore) before any ValidArgsFunction runs -- it
// reads Bean.ID/.Type/.Title/.Status and never a bean's Body, never
// pkg/search, and triggers no further disk read (R-04 SC-01, SC-03).
func beanIDCandidates() []string {
	if core == nil {
		return nil
	}
	all := core.All()
	ordered := make([]*bean.Bean, len(all))
	copy(ordered, all)
	sort.SliceStable(ordered, func(i, j int) bool {
		return completionStatusRank(ordered[i].Status) < completionStatusRank(ordered[j].Status)
	})
	candidates := make([]string, 0, len(ordered))
	for _, b := range ordered {
		candidates = append(candidates, b.ID+"\t"+b.Type+" "+b.Title+" ("+b.Status+")")
	}
	return candidates
}
