package libra

import (
	"sync"
	"time"

	"github.com/pxLi-io/libra/internal/set"

	ml "github.com/hashicorp/memberlist"
)

type Libra struct {
	conf *Config

	sync.Mutex
	// address set, key format: "host:port"
	addrs *set.Set
	// gossip based membership and failure detection
	list *ml.Memberlist
	// consistent hashing with bounded loads

	close chan struct{}
}

func New() (*Libra, error) {
	mlConf := ml.DefaultLANConfig()
	mlConf.Name = *Name
	mlConf.BindPort = *Port
	mlConf.AdvertisePort = *Port
	conf := &Config{Seeds:[]string{
		"10.252.2.231:7946",
		"10.252.2.231:7947",
		"10.252.2.231:7948",
	}, Config: mlConf}

	return newLibra(conf)
}

func NewWithConf(conf *Config) (*Libra, error) {
	return newLibra(conf)
}

func newLibra(conf *Config) (*Libra, error) {
	list, err := ml.Create(conf.Config)
	if err != nil {
		return nil, err
	}
	return &Libra{conf: conf, list: list}, nil
}

func (l *Libra) Join() error {
	l.Lock()
	defer l.Unlock()

	println(l.conf.Seeds[0])
	_, err := l.list.Join(l.conf.Seeds)
	return err
}

func (l *Libra) Leave() error {
	l.Lock()
	defer l.Unlock()

	return l.list.Leave(10 * time.Second)
}

func (l *Libra) Serve() {
	l.Join()
	l.addrs = nodeToSet(l.list.Members())
	for {
		select {
		case <-time.After(5*time.Second):
			l.Lock()
			println(l.addrs.Len())
			nodes := nodeToSet(l.list.Members())

			//add, del := l.diff(nodes)
			// ring update

			trash := l.addrs
			l.addrs = nodes
			println(l.addrs.Len())
			l.Unlock()

			trash.Collect()
		}
	}
}

func nodeToSet(nodes []*ml.Node) *set.Set {
	strList := make([]string, len(nodes))
	for i, node := range nodes {
		strList[i] = node.Address()
	}
	return set.New(strList)
}

func (l *Libra) diff(newAddrs *set.Set) (add, del []string) {
	l.Lock()
	defer l.Unlock()

	return l.addrs.Diff(newAddrs)
}

func (l *Libra) Address() string {
	return l.list.LocalNode().Address()
}
