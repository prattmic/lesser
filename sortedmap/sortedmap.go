package sortedmap

import (
	"fmt"
	"sort"
	"sync"
)

// intSlice is a sorted slice of unique ints.
type intSlice sort.IntSlice

// Insert inserts value v into the appropriate location in the slice.
func (s *intSlice) Insert(v int) {
	// Search returns the index to insert v if it exists,
	// so check that it doesn't exist before adding it.
	i := sort.IntSlice(*s).Search(v)
	if i < len(*s) && (*s)[i] == v {
		return
	}

	// Grow the slice by one element.
	*s = append(*s, 0)
	// Move the upper part of the slice out of the way and open a hole.
	copy((*s)[i+1:], (*s)[i:])
	// Store the new value.
	(*s)[i] = v
}

// Delete deletes the value v from the slice.
func (s *intSlice) Delete(v int) {
	// Search returns the index to insert v if it doesn't exist,
	// so check that it exists before deleting it.
	i := sort.IntSlice(*s).Search(v)
	if i >= len(*s) || (*s)[i] != v {
		return
	}

	*s = append((*s)[:i], (*s)[i+1:]...)
}

// Search returns the index of v in the slice, if exists is true.
// Otherwise, it is the location v would be inserted.  All indices less
// than i contain values less than v.
func (s *intSlice) Search(v int) (i int, exists bool) {
	i = sort.IntSlice(*s).Search(v)
	// Does v exist, or is this just the location to insert it.
	if i < len(*s) && (*s)[i] == v {
		exists = true
	}

	return
}

// Map is a sorted map[int]int.
type Map struct {
	// m is the underlying map store
	m map[int]int

	// k is the sorted list of keys
	k intSlice

	// mu locks the Map.
	mu sync.Mutex
}

// Insert inserts a key, value pair.
func (m *Map) Insert(k, v int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Delete any duplicate entry.
	m.deleteImpl(k)

	m.m[k] = v
	m.k.Insert(k)
}

// Delete key from map, must be called with mu held.
func (m *Map) deleteImpl(k int) {
	delete(m.m, k)
	m.k.Delete(k)
}

// Delete deletes the value stored at k from the map.
func (m *Map) Delete(k int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.deleteImpl(k)
}

// NearestLessEqual returns the nearest key, value pair that exists in
// the map with a key <= want.
func (m *Map) NearestLessEqual(want int) (key, value int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	i, exists := m.k.Search(want)
	// Key already exists in the map.
	if exists {
		return want, m.m[want], nil
	}

	// i - 1 contains the nearest key less than the desired key.
	if i < 1 {
		return 0, 0, fmt.Errorf("no key less than %d", want)
	}

	key = m.k[i-1]
	value = m.m[key]

	return key, value, nil
}

func NewMap() Map {
	return Map{
		m: make(map[int]int),
		k: make(intSlice, 0),
	}
}
