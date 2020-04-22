package libra

import (
	"sync"
	"time"

	"github.com/pxLi-io/libra/set"

	ml "github.com/hashicorp/memberlist"
	"go.uber.org/zap"
)

type Libra struct {
	conf   *Config
	logger *zap.SugaredLogger

	// address set, key format: "host:port"
	c *Consistent
	// gossip based membership and failure detection
	list *ml.Memberlist
	// consistent hashing with bounded loads

	close chan struct{}

	sync.RWMutex
}

func New() (*Libra, error) {
	mlConf := ml.DefaultLANConfig()
	mlConf.Name = *Name
	mlConf.BindPort = *Port
	mlConf.AdvertisePort = *Port
	conf := &Config{Seeds: []string{
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
	return &Libra{conf: conf, c: NewConsistent([]string{}, 1), logger: NewSugar(), list: list}, nil
}

func (l *Libra) Join() error {
	l.Lock()
	defer l.Unlock()

	_, err := l.list.Join(l.conf.Seeds)
	return err
}

func (l *Libra) Leave() error {
	l.Lock()
	defer l.Unlock()

	return l.list.Leave(10 * time.Second)
}

func (l *Libra) Sync() {
	for {
		if err := l.Join(); err != nil {
			l.logger.Errorw("failed to join cluster. will retry in 10 seconds", "error", err)
			time.Sleep(10 * time.Second)
			continue
		}
		l.logger.Info("joined cluster")
		break
	}
	// init
	l.c.Add(toAddresses(l.list.Members())...)
	for {
		select {
		case <-time.After(5 * time.Second):
			l.Lock()
			nodes := toAddresses(l.list.Members())
			l.logger.Infow("syncing node list", "nodes", nodes)
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
		}
	}
}

func toAddresses(nodes []*ml.Node) []string {
	strList := make([]string, len(nodes))
	for i, node := range nodes {
		strList[i] = node.Address()
	}
	return strList
}

func (l *Libra) diff(newAddrs []string) (add, del []string) {
	o := set.New(newAddrs)
	defer o.Collect()
	return l.c.Nodes.Diff(o)
}

func (l *Libra) Address() string {
	return l.list.LocalNode().Address()
}

func (l *Libra) Shutdown() {
	_ = l.logger.Sync()
}
