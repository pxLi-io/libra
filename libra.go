package libra

import (
	"crypto/md5"
	"encoding/binary"
	"golang.org/x/sys/cpu"
	"sync"
	"sync/atomic"
	"time"

	ml "github.com/hashicorp/memberlist"
	"go.uber.org/zap"
)

// Star node unit
type Star struct {
	ID     string
	Weight int
}

func NodeToStar(nodes []*ml.Node) []*Star {
	stars := make([]*Star, len(nodes))
	for i := range nodes {
		stars[i] = nodeToStar(nodes[i])
	}
	return stars
}

func nodeToStar(node *ml.Node) *Star {
	return &Star{
		ID:     node.Name,
		Weight: CalWeight(int(binary.LittleEndian.Uint16(node.Meta))),
	}
}

const (
	noCD = iota
	inCD
)

// Libra ...
type Libra struct {
	conf   *Config
	logger *zap.SugaredLogger

	c *Consistent
	// event channel
	eventCh chan ml.NodeEvent
	// gossip based membership and failure detection
	list *ml.Memberlist

	updateLoadInCD uint64

	close chan struct{}

	sync.RWMutex
}

func New() (*Libra, error) {
	mlConf := ml.DefaultLANConfig()
	mlConf.Name = nodeName()
	mlConf.BindPort = *Port
	mlConf.AdvertisePort = *Port
	mlConf.DisableTcpPings = true

	sum := md5.Sum([]byte("service-1"))
	println(len(sum))
	secret := sum[:]
	mlConf.SecretKey = secret
	println(mlConf.SecretKey)
	//mlConf.LogOutput, _ = os.Open(os.DevNull)

	conf := &Config{Seeds: []string{
		"10.252.2.231:7946",
		"10.252.2.231:7947",
		"10.252.2.231:7948",
		//"10.5.24.19:7946",
		//"10.5.24.19:7947",
		//"10.252.2.231:7948",
	}, Config: mlConf}

	return newLibra(conf)
}

func newLibra(conf *Config) (*Libra, error) {
	list, err := ml.Create(conf.Config)
	if err != nil {
		return nil, err
	}
	list.LocalNode().Meta = make([]byte, 2) // load multiplier
	binary.LittleEndian.PutUint16(list.LocalNode().Meta, uint16(2))

	conf.Delegate = &Delegate{Memberlist: list}
	ch := make(chan ml.NodeEvent, len(conf.Seeds))
	conf.Events = &ml.ChannelEventDelegate{
		Ch: ch,
	}

	mul := conf.LoadMultiplier
	if mul == 0 {
		mul = *LoadMultiplier
	}

	return &Libra{
		conf:    conf,
		c:       NewConsistent(nil),
		logger:  NewSugar(),
		list:    list,
		eventCh: ch,
	}, nil
}

func (l *Libra) Join() {
	l.Lock()
	defer l.Unlock()

	l.logger.Info("joining cluster...")
	for {
		_, err := l.list.Join(l.conf.Seeds)
		l.logger.Info("try joining cluster...")
		if err != nil {
			l.logger.Errorw("failed to join cluster. will retry in 10 seconds", "error", err)
			time.Sleep(10 * time.Second)
			continue
		}
		//if !l.quorum() {
		//	l.logger.Errorw("failed to join cluster. will retry in 10 seconds", "error", ErrQuorumNotMet)
		//	time.Sleep(10 * time.Second)
		//	continue
		//}
		l.logger.Info("joined cluster")
		break
	}
	l.c.Add(NodeToStar(l.list.Members())...)
	l.logger.Info("initiated consistent hash")
}

func (l *Libra) Leave() error {
	l.Lock()
	defer l.Unlock()

	return l.list.Leave(10 * time.Second)
}

func (l *Libra) Serve() {
	l.Join()
	for {
		select {
		case <-time.After(5 * time.Second):
			// quorum check
			if !l.quorum() {
				l.logger.Errorw("quorum is not met. try rejoining...", "alive_node_num", l.list.NumMembers())

				//l.Join()
			}

			l.Lock()
			nodes := l.list.Members()
			//l.logger.Infow("syncing node list", "nodes", nodes)
			add, del := l.diff(nodes)
			if len(add) > 0 {
				l.c.Add(add...)
				l.logger.Infow("added new nodes", "nodes", add)
			}
			if len(del) > 0 {
				l.c.Del(del...)
				l.logger.Infow("deleted off-duty nodes", "nodes", del)
			}
			l.Unlock()
		case e := <-l.eventCh:
			switch e.Event {
			case ml.NodeJoin:
				l.c.Add(nodeToStar(e.Node))
				l.logger.Infow("added new node", "node", e.Node)
			case ml.NodeLeave:
				l.c.Del(nodeToStar(e.Node))
				l.logger.Infow("deleted off-duty node", "node", e.Node)
			case ml.NodeUpdate:
				err := l.c.Update(nodeToStar(e.Node))
				if err != nil {
					l.logger.Errorw("failed to update node", "node", e, "error", err)
					continue
				}
			default:

			}
			//l.logger.Infow("updated node", "len", len(l.c.hash), "node", l.c.hash)
		}
	}
}

func (l *Libra) diff(newNodes []*ml.Node) (add, del []*Star) {
	o := NewMap(newNodes)
	defer o.Collect()
	return l.c.Stars.Diff(o)
}

func (l *Libra) Address() string {
	return l.list.LocalNode().Address()
}

func (l *Libra) UpdateWeight(mul int) error {
	if atomic.LoadUint64(&l.updateLoadInCD) == inCD {
		return ErrUpdateLoadInCD
	}

	l.Lock()
	defer l.Unlock()

	binary.LittleEndian.PutUint16(l.list.LocalNode().Meta, uint16(mul))
	err := l.list.UpdateNode(*UpdateTimeout)
	if err != nil {
		return err
	}
	err = l.c.Update(nodeToStar(l.list.LocalNode()))
	if err != nil {
		l.logger.Errorw("failed to update node", "node", "localhost", "error", err)
		return err
	}

	atomic.CompareAndSwapUint64(&l.updateLoadInCD, noCD, inCD)
	go l.updateLoadCoolDown()
	//l.logger.Infow("updated node", "len", len(l.c.hash), "node", l.c.hash, "node", "localhost")
	return nil
}

func (l *Libra) updateLoadCoolDown() {
	time.Sleep(60 * time.Second)
	atomic.CompareAndSwapUint64(&l.updateLoadInCD, inCD, noCD)
}

var (
	countM = 0
	countK = 0
	countO = 0
)

var aPool = sync.Pool{New: func() interface{} {
	countM++
	println("alloc new atlas", countM)
	return &Atlas{m: make(map[string][]string)}
}}

var akPool = sync.Pool{New: func() interface{} {
	countK++
	println("alloc new", countK)
	return []string{}
}}

var mPool = sync.Pool{New: func() interface{} {
	countO++
	println("alloc new k", countK)
	return make(map[string][]string)
}}

type Atlas struct {
	_ cpu.CacheLinePad
	m map[string][]string
	_ cpu.CacheLinePad
}

func (a *Atlas) Get(id string) []string {
	return a.m[id]
}

func (a *Atlas) GetStar(s *Star) []string {
	return a.Get(s.ID)
}

func (a *Atlas) Free() {
	for k, v := range a.m {
		for i := range v {
			v[i] = ""
		}
		v = v[:0]
		akPool.Put(v)
		delete(a.m, k)
	}
	aPool.Put(a)
}

func (l *Libra) Get(keys ...string) *Atlas {
	l.RLock()
	defer l.RUnlock()

	if l.quorum() {
		return l.c.Get(keys...)
	}
	l.logger.Errorw("quorum is not met", "alive_node_num", l.list.NumMembers())
	return nil
}

func (l *Libra) LocalID() string {
	return l.list.LocalNode().Name
}

func (l *Libra) quorum() bool {
	q := len(l.conf.Seeds) >> 1
	return l.list.NumMembers() > q && l.c.Len() > q
}

func (l *Libra) Shutdown() {
	_ = l.logger.Sync()
}
