package libra

import (
	"fmt"
	//"runtime"
	"sort"
	"sync"

	ml "github.com/hashicorp/memberlist"

	"github.com/cespare/xxhash"
	"golang.org/x/sys/cpu"
)

const (
	distributionFormat = "%s-%d"

	baseWeight    = 2
	MinMultiplier = 1
	MaxMultiplier = 1 << 4 // max weight = 32
)

// Consistent simple consistent hashing
// https://en.wikipedia.org/wiki/Consistent_hashing
type Consistent struct {
	_     cpu.CacheLinePad
	Stars *Map

	ring map[uint64]string // consistent hash ring
	//ring sync.Map
	hash []uint64          // sorted hash slice

	sync.RWMutex
	_ cpu.CacheLinePad
}

// NewConsistent ...
func NewConsistent(nodes []*ml.Node) *Consistent {
	c := &Consistent{
		Stars: NewMap(nodes),
		ring:  make(map[uint64]string),
		//ring:  sync.Map{},
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
		for r := 0; r < star.Weight; r++ {
			h := hash(fmt.Sprintf(distributionFormat, star.ID, r))
			c.hash = append(c.hash, h)
			c.ring[h] = star.ID
			//c.ring.Store(h, star.ID)
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
		for r := 0; r < star.Weight; r++ {
			h := hash(fmt.Sprintf(distributionFormat, star.ID, r))
			delete(c.ring, h)
			//c.ring.Delete(h)
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

	//type Item struct {
	//	idx int
	//	k   string
	//}

	//for i := range keys {
	//	//go func() {
	//	//	h := hash(keys[i])
	//	//	idx := c.search(h)
	//	//	lock.Lock()
	//	//	if list, ok := a.m[c.ring[c.hash[idx]]]; ok {
	//	//		a.m[c.ring[c.hash[idx]]] = append(list, keys[i])
	//	//		lock.Unlock()
	//	//		return
	//	//	}
	//	//	newList := akPool.Get().([]string)
	//	//	a.m[c.ring[c.hash[idx]]] = append(newList, keys[i])
	//	//	lock.Unlock()
	//	//}()
	//	h := hash(keys[i])
	//	idx := c.search(h)
	//	if list, ok := a.m[c.ring[c.hash[idx]]]; ok {
	//		a.m[c.ring[c.hash[idx]]] = append(list, keys[i])
	//		continue
	//	}
	//	newList := akPool.Get().([]string)
	//	a.m[c.ring[c.hash[idx]]] = append(newList, keys[i])
	//}

	var wg sync.WaitGroup
	//chunkSize := (len(keys) + runtime.NumCPU() - 1) / runtime.NumCPU()
	chunkSize := (len(keys) + 4 - 1) / 4

	var divided [][]string
	for i := 0; i < len(keys); i += chunkSize {
		end := i + chunkSize

		if end > len(keys) {
			end = len(keys)
		}

		divided = append(divided, keys[i:end])
	}
	wg.Add(len(divided))

	var arr []map[string][]string
	for _, d := range divided {
		d := d
		arr = append(arr, make(map[string][]string))
		go func() {
			m := mPool.Get().(map[string][]string)
			//m := make(map[string][]string)
			for _, k := range d {
				h := hash(k)
				idx := c.search(h)
				//star, _ := c.ring.Load(c.hash[idx])
				//s := star.(string)
				s, _ := c.ring[c.hash[idx]]
				if list, ok := m[s]; ok {
					m[s] = append(list, k)
					continue
				}
				newList := akPool.Get().([]string)
				m[s] = append(newList, k)
			}
			wg.Done()

			for k, v := range m {
				for i := range v {
					v[i] = ""
				}
				v = v[:0]
				akPool.Put(v)
				delete(m, k)
			}
			mPool.Put(m)
		}()
	}
	wg.Wait()
	for _, m := range arr {
		for k,v :=range m {
			a.m[k] = v
		}
	}

	//go func() {
		//for item := range resCh {
		//	if list, ok := a.m[c.ring[c.hash[item.idx]]]; ok {
		//		a.m[c.ring[c.hash[item.idx]]] = append(list, item.k)
		//		wg.Done()
		//		continue
		//	}
		//	newList := akPool.Get().([]string)
		//	a.m[c.ring[c.hash[item.idx]]] = append(newList, item.k)
		//	wg.Done()
		//}
	//
	//	for {
	//		select {
	//		case _, ok := <-resCh:
	//			if ok {
	//			wg.Done()
	//
	//			}
	//		}
	//	}
	//}()

	//wg.Wait()
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

	if star.Weight != c.Stars.Get(star.ID).Weight {
		c.Stars.Add(star) // overwrite
		c.reBalance()
	}

	return nil
}

func (c *Consistent) reBalance() {
	c.hash = c.hash[:0]
	for _, star := range c.Stars.ListStar() {
		for r := 0; r < star.Weight; r++ {
			h := hash(fmt.Sprintf(distributionFormat, star.ID, r))
			c.hash = append(c.hash, h)
			c.ring[h] = star.ID
			//c.ring.Store(h, star.ID)
		}
	}
	sort.Slice(c.hash, func(i, j int) bool {
		return c.hash[i] < c.hash[j]
	})
}

func CalWeight(mul int) int {
	if mul < MinMultiplier {
		mul = MinMultiplier
	} else if mul > MaxMultiplier {
		mul = MaxMultiplier
	}
	return baseWeight * mul
}
