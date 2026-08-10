package commands

import (
	"github.com/hmans/beans/pkg/bean"
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
// from both completed and total.
func descendantProgress(id string, idx map[string][]*bean.Bean) (completed, total int) {
	for _, d := range descendants(id, idx) {
		if d.Status == "scrapped" {
			continue
		}
		total++
		if d.Status == "completed" {
			completed++
		}
	}
	return completed, total
}
