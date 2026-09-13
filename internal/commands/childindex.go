package commands

import (
	"github.com/xRiErOS/beans/pkg/bean"
)

// buildChildrenIndex maps each bean ID to its direct children.
func buildChildrenIndex(all []*bean.Bean) map[string][]*bean.Bean {
	children := make(map[string][]*bean.Bean)
	for _, b := range all {
		if b.Parent != "" {
			children[b.Parent] = append(children[b.Parent], b)
		}
	}
	return children
}

// descendants returns every bean transitively parented under id (not including id itself).
func descendants(id string, idx map[string][]*bean.Bean) []*bean.Bean {
	var out []*bean.Bean
	queue := idx[id]
	for len(queue) > 0 {
		b := queue[0]
		queue = queue[1:]
		out = append(out, b)
		queue = append(queue, idx[b.ID]...)
	}
	return out
}

// descendantProgress returns (completed, total) descendants per the
// project's percent-complete convention: scrapped beans are excluded
// from both completed and total. "completed" is checked via
// cfg.IsArchiveStatus rather than a literal so a project-defined second
// "done" status (any status marked archive: true besides "scrapped")
// counts too. "scrapped" itself stays a literal: Archive alone cannot
// distinguish "counts as done" from "excluded from the denominator", and
// scrapped is the one status this convention gives the latter meaning.
func descendantProgress(id string, idx map[string][]*bean.Bean) (completed, total int) {
	for _, d := range descendants(id, idx) {
		if d.Status == "scrapped" {
			continue
		}
		total++
		if cfg.IsArchiveStatus(d.Status) {
			completed++
		}
	}
	return completed, total
}
