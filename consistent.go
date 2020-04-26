package libra

import (
	"fmt"
	"sort"
	"sync"

	"github.com/cespare/xxhash"
	ml "github.com/hashicorp/memberlist"
)

const (
	distributionFormat = "%s-%d"

	baseLoad      = 2
	MinMultiplier = 1
	MaxMultiplier = 1 << 4 // max load = 32
)

// Consistent simple consistent hashing
// https://en.wikipedia.org/wiki/Consistent_hashing
type Consistent struct {
	Stars *Map

	ring map[uint64]string // consistent hash ring
	hash []uint64            // sorted hash slice

	sync.RWMutex
}

// NewConsistent ...
func NewConsistent(nodes []*ml.Node) *Consistent {
	c := &Consistent{
		Stars:   NewMap(nodes),
		ring:    make(map[uint64]string),
	}

	c.Add(NodeToStar(nodes)...)
	return c
}

func (c *Consistent) Len() int {
	c.RLock()
	defer c.RUnlock()

	return c.Stars.Len()
}

func (c *Consistent) Add(stars ...*Star) {
	c.Lock()
	defer c.Unlock()

	exist := 0
	for _, star := range stars {
		if c.Stars.HasStar(star) {
			exist++
			continue
		}

		c.Stars.Add(star)
		for r := 0; r < star.Load; r++ {
			h := hash(fmt.Sprintf(distributionFormat, star.ID, r))
			c.hash = append(c.hash, h)
			c.ring[h] = star.ID
		}
	}
	if exist == len(stars) {
		return
	}

	sort.Slice(c.hash, func(i, j int) bool {
		return c.hash[i] < c.hash[j]
	})
}

func (c *Consistent) Del(stars ...*Star) {
	c.Lock()
	defer c.Unlock()

	for _, star := range stars {
		if !c.Stars.HasStar(star) {
			continue
		}
		for r := 0; r < star.Load; r++ {
			h := hash(fmt.Sprintf(distributionFormat, star.ID, r))
			delete(c.ring, h)
			c.delKeys(h)
		}
		c.Stars.Del(star)
	}
}

func (c *Consistent) delKeys(h uint64) {
	i, l, r := -1, 0, len(c.hash)-1
	for l <= r {
		m := int(uint(l+r) >> 1)
		if c.hash[m] == h {
			i = m
			break
		} else if c.hash[m] < h {
			l = m + 1
		} else if c.hash[m] > h {
			r = m - 1
		}
	}
	if i != -1 {
		c.hash = append(c.hash[:i], c.hash[i+1:]...)
	}
}

func (c *Consistent) Get(keys ...string) *Atlas {
	c.RLock()
	defer c.RUnlock()

	a := aPool.Get().(*Atlas)
	if len(c.hash) == 0 {
		return a
	}

	for i := range keys {
		h := hash(keys[i])
		idx := c.search(h)
		if list, ok := a.m[c.ring[c.hash[idx]]]; ok {
			a.m[c.ring[c.hash[idx]]] = append(list, keys[i])
			continue
		}
		newList := akPool.Get().([]string)
		a.m[c.ring[c.hash[idx]]] = append(newList, keys[i])
	}
	return a
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

func (c *Consistent) Update(star *Star) error {
	c.Lock()
	defer c.Unlock()

	if !c.Stars.HasStar(star) {
		return ErrNodeNotExist
	}

	if star.Load != c.Stars.Get(star.ID).Load {
		c.Stars.Add(star) // overwrite
		c.reBalance()
	}

	return nil
}

func (c *Consistent) reBalance() {
	c.hash = c.hash[:0]
	for _, star := range c.Stars.ListStar() {
		for r := 0; r < star.Load; r++ {
			h := hash(fmt.Sprintf(distributionFormat, star.ID, r))
			c.hash = append(c.hash, h)
			c.ring[h] = star.ID
		}
	}
	sort.Slice(c.hash, func(i, j int) bool {
		return c.hash[i] < c.hash[j]
	})
}

func CalLoad(mul int) int {
	if mul < MinMultiplier {
		mul = MinMultiplier
	} else if mul > MaxMultiplier {
		mul = MaxMultiplier
	}
	return baseLoad * mul
}
