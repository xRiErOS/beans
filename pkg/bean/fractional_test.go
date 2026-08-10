package bean

import (
	"testing"
)

func TestOrderBetween_BothEmpty(t *testing.T) {
	result := OrderBetween("", "")
	if result != "V" {
		t.Errorf("OrderBetween(\"\", \"\") = %q, want %q", result, "V")
	}
}

func TestOrderBetween_BeforeKey(t *testing.T) {
	result := OrderBetween("", "V")
	if result >= "V" {
		t.Errorf("OrderBetween(\"\", \"V\") = %q, should be < \"V\"", result)
	}
	if result == "" {
		t.Error("OrderBetween(\"\", \"V\") should not be empty")
	}
}

func TestOrderBetween_AfterKey(t *testing.T) {
	result := OrderBetween("V", "")
	if result <= "V" {
		t.Errorf("OrderBetween(\"V\", \"\") = %q, should be > \"V\"", result)
	}
}

func TestOrderBetween_Between(t *testing.T) {
	tests := []struct {
		a, b string
	}{
		{"A", "Z"},
		{"V", "X"},
		{"a", "z"},
		{"0", "z"},
		{"Va", "Vz"},
		{"V", "W"},
	}

	for _, tt := range tests {
		result := OrderBetween(tt.a, tt.b)
		if result <= tt.a || result >= tt.b {
			t.Errorf("OrderBetween(%q, %q) = %q, should be between", tt.a, tt.b, result)
		}
	}
}

func TestOrderBetween_AdjacentDigits(t *testing.T) {
	// When digits are adjacent (e.g., A and B), we need to go deeper
	result := OrderBetween("A", "B")
	if result <= "A" || result >= "B" {
		t.Errorf("OrderBetween(\"A\", \"B\") = %q, should be between", result)
	}
}

func TestOrderBetween_DeepAdjacentPrefixAtMaxDigit(t *testing.T) {
	// Regression: midpoint("yz", "z") used to return "yz" itself (equal to
	// the lower bound) because the "go deeper" branch didn't check whether
	// a's next digit was already at the maximum (61, 'z').
	result := OrderBetween("yz", "z")
	if result <= "yz" || result >= "z" {
		t.Errorf("OrderBetween(\"yz\", \"z\") = %q, should be strictly between", result)
	}
}

func TestOrderBetween_ManyInsertions(t *testing.T) {
	// Simulate inserting many items at the end
	keys := []string{"V"}
	for i := 0; i < 50; i++ {
		newKey := OrderBetween(keys[len(keys)-1], "")
		if newKey <= keys[len(keys)-1] {
			t.Fatalf("insertion %d: %q should be > %q", i, newKey, keys[len(keys)-1])
		}
		keys = append(keys, newKey)
	}

	// Verify all are in order
	for i := 1; i < len(keys); i++ {
		if keys[i] <= keys[i-1] {
			t.Errorf("keys not in order at %d: %q <= %q", i, keys[i], keys[i-1])
		}
	}
}

func TestOrderBetween_ManyInsertionsAtStart(t *testing.T) {
	keys := []string{"V"}
	for i := 0; i < 50; i++ {
		newKey := OrderBetween("", keys[0])
		if newKey >= keys[0] {
			t.Fatalf("insertion %d: %q should be < %q", i, newKey, keys[0])
		}
		keys = append([]string{newKey}, keys...)
	}

	// Verify all are in order
	for i := 1; i < len(keys); i++ {
		if keys[i] <= keys[i-1] {
			t.Errorf("keys not in order at %d: %q <= %q", i, keys[i], keys[i-1])
		}
	}
}

func TestOrderBetween_ThousandInsertionsStayDistinctAndOrdered(t *testing.T) {
	// SC-01: repeatedly inserting into the same conceptual gap between two
	// fixed neighbours (narrowing toward one side each time) must yield
	// 1000 distinct values, each strictly between its own immediate
	// neighbours under plain lexicographic comparison.
	lo, hi := "A", "z"
	seen := make(map[string]bool, 1000)

	for i := 0; i < 1000; i++ {
		mid := OrderBetween(lo, hi)
		if mid <= lo || mid >= hi {
			t.Fatalf("insertion %d: %q not strictly between %q and %q", i, mid, lo, hi)
		}
		if seen[mid] {
			t.Fatalf("insertion %d: %q was already generated", i, mid)
		}
		seen[mid] = true
		lo = mid // narrow toward hi each time, worst case for key growth
	}

	if len(seen) != 1000 {
		t.Errorf("len(seen) = %d, want 1000", len(seen))
	}
}

func TestOrderBetween_ThousandInsertionsNarrowingTowardLo(t *testing.T) {
	// Symmetric to the above, narrowing toward the lower bound instead.
	lo, hi := "A", "z"
	seen := make(map[string]bool, 1000)

	for i := 0; i < 1000; i++ {
		mid := OrderBetween(lo, hi)
		if mid <= lo || mid >= hi {
			t.Fatalf("insertion %d: %q not strictly between %q and %q", i, mid, lo, hi)
		}
		if seen[mid] {
			t.Fatalf("insertion %d: %q was already generated", i, mid)
		}
		seen[mid] = true
		hi = mid
	}

	if len(seen) != 1000 {
		t.Errorf("len(seen) = %d, want 1000", len(seen))
	}
}

// AC3: IsValidOrderKey rejects anything that isn't a non-empty string built
// entirely from base62Digits characters.
func TestIsValidOrderKey(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		valid bool
	}{
		{"single valid digit", "V", true},
		{"multi-char valid key", "Va1z", true},
		{"empty string is invalid", "", false},
		{"non-base62 character", "V!", false},
		{"space is invalid", "V a", false},
		{"unicode character is invalid", "Vé", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidOrderKey(tt.key)
			if got != tt.valid {
				t.Errorf("IsValidOrderKey(%q) = %v, want %v", tt.key, got, tt.valid)
			}
		})
	}
}

func TestOrderBetween_ManyInsertionsBetween(t *testing.T) {
	// Repeatedly insert between two keys
	a := "A"
	b := "z"
	for i := 0; i < 100; i++ {
		mid := OrderBetween(a, b)
		if mid <= a || mid >= b {
			t.Fatalf("insertion %d: %q should be between %q and %q", i, mid, a, b)
		}
		// Alternate inserting before and after the midpoint
		if i%2 == 0 {
			b = mid
		} else {
			a = mid
		}
	}
}
