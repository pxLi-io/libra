package set

import (
	"sync"
)

var (
	pool = sync.Pool{
		New: func() interface{} {
			return &Set{items: make(map[string]bool)}
		},
	}
)

type Set struct {
	sync.RWMutex

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
	s.Lock()
	defer s.Unlock()

	s.items[item] = true
}

func (s *Set) Del(item string) {
	s.Lock()
	defer s.Unlock()

	delete(s.items, item)
}

func (s *Set) Has(item string) bool {
	s.RLock()
	defer s.RUnlock()

	return s.items[item]
}

func (s *Set) List() []string {
	s.RLock()
	defer s.RUnlock()

	var list []string
	for item := range s.items {
		list = append(list, item)
	}
	return list
}

func (s *Set) Len() int {
	s.RLock()
	defer s.RUnlock()

	return len(s.items)
}

func (s *Set) Diff(o *Set) (add, del []string) {
	s.RLock()
	defer s.RUnlock()

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
	s.Lock()
	defer s.Unlock()

	for k := range s.items {
		delete(s.items, k)
	}
	pool.Put(s)
}
