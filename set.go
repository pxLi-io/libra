package libra

import (
	"encoding/binary"
	ml "github.com/hashicorp/memberlist"
	"sync"
)

var pool = sync.Pool{
	New: func() interface{} {
		return &Map{items: make(map[string]*Star)}
	},
}

// Map simple string set based on map, not thread-safe
type Map struct {
	items map[string]*Star
}

func NewMap(nodes []*ml.Node) *Map {
	m := pool.Get().(*Map)
	for _, node := range nodes {
		m.items[node.Name] = &Star{
			ID:   node.Name,
			Load: CalLoad(int(binary.LittleEndian.Uint16(node.Meta))),
		}
	}
	return m
}

func (m *Map) Add(s *Star) {
	m.items[s.ID] = s
}

func (m *Map) Del(s *Star) {
	delete(m.items, s.ID)
}

func (m *Map) HasStar(s *Star) bool {
	return m.Has(s.ID)
}

func (m *Map) Has(name string) bool {
	if _, ok := m.items[name]; ok {
		return true
	}
	return false
}

func (m *Map) Get(name string) *Star {
	return m.items[name]
}

func (m *Map) ListName() []string {
	var list []string
	for name := range m.items {
		list = append(list, name)
	}
	return list
}

func (m *Map) ListStar() []*Star {
	var list []*Star
	for _, s := range m.items {
		list = append(list, s)
	}
	return list
}

func (m *Map) Len() int {
	return len(m.items)
}

func (m *Map) Diff(o *Map) (add, del []*Star) {
	// add
	for _, star := range o.items {
		if !m.HasStar(star) {
			add = append(add, star)
		}
	}
	// del
	for _, star := range m.items {
		if !o.HasStar(star) {
			del = append(del, star)
		}
	}
	return
}

// Collect cleans all items and put the set back to sync.Pool
func (m *Map) Collect() {
	for k := range m.items {
		delete(m.items, k)
	}
	pool.Put(m)
}
