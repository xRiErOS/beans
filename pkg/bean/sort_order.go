package bean

import (
	"sort"
	"strings"
)

// SortByOrder sorts beans in place by their fractional-index Order key alone.
//
// This is the shared "sibling order" used by the `order` command (placement)
// and `list --sort order` (display): beans with a set Order come first,
// compared lexicographically by the Order string; beans without an Order
// sort after all of those. Any remaining tie (most commonly two beans that
// both have no Order) breaks on Title, case-insensitively, for determinism.
func SortByOrder(beans []*Bean) {
	sort.SliceStable(beans, func(i, j int) bool {
		oi, oj := beans[i].Order, beans[j].Order
		iHas, jHas := oi != "", oj != ""

		if iHas != jHas {
			// Beans with an explicit order come before those without.
			return iHas
		}
		if iHas && jHas && oi != oj {
			return oi < oj
		}

		return strings.ToLower(beans[i].Title) < strings.ToLower(beans[j].Title)
	})
}
