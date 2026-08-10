package bean

import "testing"

// SortByOrder sorts a slice of siblings purely by their fractional Order key:
// beans with a set Order come first (lexicographic comparison of the key),
// beans without an Order sort after all of those, and any remaining tie
// (two empty Orders, or — in principle — two identical Order values) breaks
// on Title for determinism. It is deliberately independent from
// SortByStatusPriorityAndType so the `order` command and `list --sort order`
// (beans-uo43) can both use it as the single source of "sibling order".

func TestSortByOrder_LexicographicOnSetOrders(t *testing.T) {
	beans := []*Bean{
		{ID: "1", Title: "Z", Order: "C"},
		{ID: "2", Title: "Y", Order: "A"},
		{ID: "3", Title: "X", Order: "B"},
	}

	SortByOrder(beans)

	want := []string{"2", "3", "1"}
	for i, id := range want {
		if beans[i].ID != id {
			t.Errorf("beans[%d].ID = %q, want %q", i, beans[i].ID, id)
		}
	}
}

func TestSortByOrder_EmptyOrderSortsAfterSetOrders(t *testing.T) {
	beans := []*Bean{
		{ID: "1", Title: "No order", Order: ""},
		{ID: "2", Title: "Has order", Order: "A"},
	}

	SortByOrder(beans)

	if beans[0].ID != "2" || beans[1].ID != "1" {
		t.Errorf("got order %q, %q; want bean with Order before bean without", beans[0].ID, beans[1].ID)
	}
}

func TestSortByOrder_TiesBreakOnTitle(t *testing.T) {
	beans := []*Bean{
		{ID: "1", Title: "Zebra", Order: ""},
		{ID: "2", Title: "Apple", Order: ""},
		{ID: "3", Title: "Mango", Order: ""},
	}

	SortByOrder(beans)

	want := []string{"Apple", "Mango", "Zebra"}
	for i, title := range want {
		if beans[i].Title != title {
			t.Errorf("beans[%d].Title = %q, want %q", i, beans[i].Title, title)
		}
	}
}
