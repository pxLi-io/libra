package set

import (
	"sync"
)

var pool = sync.Pool{
	New: func() interface{} {
		return &Set{items: make(map[string]bool)}
	},
}

// Set simple string set based on map, not thread-safe
type Set struct {
	items map[string]bool
}

func New(items []string) *Set {
	s := pool.Get().(*Set)
	for _, item := range items {
		s.items[item] = true
	}
	return s
}

func (s *Set) Add(item string) {
	s.items[item] = true
}

func (s *Set) Del(item string) {
	delete(s.items, item)
}

func (s *Set) Has(item string) bool {
	return s.items[item]
}

func (s *Set) List() []string {
	var list []string
	for item := range s.items {
		list = append(list, item)
	}
	return list
}

func (s *Set) Len() int {
	return len(s.items)
}

func (s *Set) Diff(o *Set) (add, del []string) {
	// add
	for item := range o.items {
		if !s.Has(item) {
			add = append(add, item)
		}
	}
	// del
	for item := range s.items {
		if !o.Has(item) {
			del = append(del, item)
		}
	}
	return
}

// Collect cleans all items and put the set back to sync.Pool
func (s *Set) Collect() {
	for k := range s.items {
		delete(s.items, k)
	}
	pool.Put(s)
}
