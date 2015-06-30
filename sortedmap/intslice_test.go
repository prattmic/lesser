package sortedmap

import (
	"testing"
)

func TestIntSliceInsert(t *testing.T) {
	cases := []struct {
		insert   []int
		expected []int
	}{
		{insert: []int{1, 2, 3}, expected: []int{1, 2, 3}},
		{insert: []int{3, 2, 1}, expected: []int{1, 2, 3}},
		{insert: []int{-4, 20, -10, 5}, expected: []int{-10, -4, 5, 20}},
	}

	for _, c := range cases {
		l := make(intSlice, 0)

		for _, v := range c.insert {
			l.Insert(v)
		}

		if len(l) != len(c.expected) {
			t.Errorf("Bad length, got %d, expected %d. %v vs %v", len(l), len(c.expected), l, c.expected)
		}

		for i, e := range c.expected {
			if l[i] != e {
				t.Errorf("Got %v, expected %v", l, c.expected)
				break
			}
		}
	}
}

func TestIntSliceDelete(t *testing.T) {
	cases := []struct {
		slice    intSlice
		del      []int
		expected []int
	}{
		{slice: intSlice{1, 2, 3}, del: []int{2}, expected: []int{1, 3}},
		{slice: intSlice{1, 2, 3}, del: []int{3, 2, 1}, expected: []int{}},
		{slice: intSlice{-10, -4, 5, 20}, del: []int{-4}, expected: []int{-10, 5, 20}},
	}

	for _, c := range cases {
		for _, v := range c.del {
			c.slice.Delete(v)
		}

		if len(c.slice) != len(c.expected) {
			t.Errorf("Bad length, got %d, expected %d. %v vs %v", len(c.slice), len(c.expected), c.slice, c.expected)
		}

		for i, e := range c.expected {
			if c.slice[i] != e {
				t.Errorf("Got %v, expected %v", c.slice, c.expected)
				break
			}
		}
	}
}
