package libra

import (
	"fmt"
	"sort"
	"sync"

	"github.com/pxLi-io/libra/set"

	"github.com/cespare/xxhash"
)

const (
	keyReplicaFormat = "%s-%d"
)

// Consistent simple consistent hashing
// https://en.wikipedia.org/wiki/Consistent_hashing
type Consistent struct {
	Nodes   *set.Set
	replica int

	ring map[uint64]string // consistent hash ring
	hash []uint64          // sorted hash slice

	sync.RWMutex
}

// NewConsistent ...
func NewConsistent(nodes []string, replica int) *Consistent {
	if replica < 1 {
		replica = 1
	}
	c := &Consistent{
		Nodes:   set.New(nodes),
		replica: replica,
		ring:    make(map[uint64]string),
	}

	c.Add(c.Nodes.List()...)
	return c
}

func (c *Consistent) Add(nodes ...string) {
	c.Lock()
	defer c.Unlock()

	exist := 0
	for _, node := range nodes {
		if c.Nodes.Has(node) {
			exist++
			continue
		}
		c.Nodes.Add(node)
		for r := 0; r < c.replica; r++ {
			h := hash(fmt.Sprintf(keyReplicaFormat, node, r))
			c.hash = append(c.hash, h)
			c.ring[h] = node
		}
	}
	if exist == len(nodes) {
		return
	}

	sort.Slice(c.hash, func(i, j int) bool {
		return c.hash[i] < c.hash[j]
	})
}

func (c *Consistent) Del(nodes ...string) {
	c.Lock()
	defer c.Unlock()

	for _, node := range nodes {
		if !c.Nodes.Has(node) {
			continue
		}
		c.Nodes.Del(node)
		for r := 0; r < c.replica; r++ {
			h := hash(fmt.Sprintf(keyReplicaFormat, node, r))
			delete(c.ring, h)
			c.delKeys(h)
		}
	}
}

func (c *Consistent) delKeys(h uint64) {
	//i, l, r := -1, 0, len(c.hash)-1
	//for l <= r {
	//	m := int(uint(l+r) >> 1)
	//	if c.hash[m] == h {
	//		i = m
	//		break
	//	} else if c.hash[m] < h {
	//		l = m + 1
	//	} else if c.hash[m] > h {
	//		l = m - 1
	//	}
	//}
	i := sort.Search(len(c.hash), func(i int) bool {
		return c.hash[i] == h
	})
	if c.hash[i] == h {
		c.hash = append(c.hash[:i], c.hash[i+1:]...)
	}
}

func (c *Consistent) Get(keys ...string) map[string][]string {
	c.RLock()
	defer c.RUnlock()

	m := make(map[string][]string)
	if len(c.hash) == 0 {
		return m
	}

	for i := range keys {
		h := hash(keys[i])
		i := c.search(h)
		if list, ok := m[c.ring[c.hash[i]]]; ok {
			list = append(list, keys[i])
			continue
		}
		m[c.ring[c.hash[i]]] = []string{keys[i]}
	}
	return m
}

func (c *Consistent) search(h uint64) int {
	i := sort.Search(len(c.hash), func(i int) bool {
		return c.hash[i] >= h
	})
	if i >= len(c.hash) {
		return 0
	}
	return i
}

func hash(k string) uint64 {
	return xxhash.Sum64String(k)
}
