package components

import "testing"

// Pager window + summary math — the visible behaviour phases rely on.

func TestPageCount(t *testing.T) {
	cases := []struct{ total, size, want int }{
		{0, 25, 0},
		{1, 25, 1},
		{25, 25, 1},
		{26, 25, 2},
		{217, 25, 9},
	}
	for _, c := range cases {
		if got := pageCount(c.total, c.size); got != c.want {
			t.Errorf("pageCount(%d, %d) = %d, want %d", c.total, c.size, got, c.want)
		}
	}
}

func TestPagerRangeText(t *testing.T) {
	cases := []struct {
		page, size, total int
		want              string
	}{
		{1, 25, 217, "1–25 of 217"},
		{2, 25, 217, "26–50 of 217"},
		{9, 25, 217, "201–217 of 217"},
		{0, 25, 0, "0 of 0"},
		{1, 10, 10, "1–10 of 10"},
	}
	for _, c := range cases {
		if got := pagerRangeText(c.page, c.size, c.total); got != c.want {
			t.Errorf("pagerRangeText(%d,%d,%d) = %q, want %q", c.page, c.size, c.total, got, c.want)
		}
	}
}

func numbers(items []pageItem) []int {
	var out []int
	for _, it := range items {
		if !it.gap {
			out = append(out, it.number)
		}
	}
	return out
}

func gaps(items []pageItem) int {
	n := 0
	for _, it := range items {
		if it.gap {
			n++
		}
	}
	return n
}

func TestPagerItemsWindowed(t *testing.T) {
	// Single page.
	if got := pagerItems(1, 1); len(got) != 1 || got[0].number != 1 {
		t.Errorf("single page = %+v", got)
	}
	// Head of a long list: near pages, ellipsis, last.
	items := pagerItems(1, 100)
	if got := numbers(items); !eq(got, 1, 2, 3, 100) {
		t.Errorf("head window = %v", got)
	}
	if gaps(items) != 1 {
		t.Errorf("head gaps = %d", gaps(items))
	}
	// Middle: window without an ellipsis inside it.
	items = pagerItems(50, 100)
	if got := numbers(items); !eq(got, 1, 48, 49, 50, 51, 52, 100) {
		t.Errorf("mid window = %v", got)
	}
	if gaps(items) != 2 {
		t.Errorf("mid gaps = %d", gaps(items))
	}
	// Tail mirrors the head.
	items = pagerItems(100, 100)
	if got := numbers(items); !eq(got, 1, 98, 99, 100) {
		t.Errorf("tail window = %v", got)
	}
	// Near the start clamps the window (no leading ellipsis).
	items = pagerItems(3, 100)
	if got := numbers(items); !eq(got, 1, 2, 3, 4, 5, 100) {
		t.Errorf("clamped head = %v", got)
	}
	// Small lists show every page, no ellipsis.
	items = pagerItems(3, 5)
	if got := numbers(items); !eq(got, 1, 2, 3, 4, 5) {
		t.Errorf("short list = %v", got)
	}
	if gaps(items) != 0 {
		t.Errorf("short list gaps = %d", gaps(items))
	}
}

func TestPagerItemsBounds(t *testing.T) {
	for _, cur := range []int{-5, 0, 1} {
		for _, it := range pagerItems(cur, 50) {
			if !it.gap && (it.number < 1 || it.number > 50) {
				t.Errorf("page out of range for current %d: %+v", cur, it)
			}
		}
	}
	for _, it := range pagerItems(99, 50) { // over-count clamps
		if !it.gap && it.number > 50 {
			t.Errorf("over-count clamp failed: %+v", it)
		}
	}
	if pagerItems(1, 0) != nil {
		t.Error("zero pages should render nothing")
	}
}

func eq(a []int, b ...int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
