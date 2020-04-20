package libra

import (
	"flag"
	"github.com/hashicorp/memberlist"
)

var (
	Name = flag.String("libra.name", "localhost", "node name for libra member")
	Port = flag.Int("libra.port", 7946, "the port used for both UDP and TCP gossip")
)

type Config struct {
	*memberlist.Config

	Seeds []string
}
