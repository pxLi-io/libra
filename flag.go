package libra

import (
	"flag"
	"time"
)

var (
	Port           = flag.Int("libra.port", 7946, "the port used for both UDP and TCP gossip")
	LoadMultiplier = flag.Int("libra.load_multiplier", 1, "load multiplier for current node")
	UpdateTimeout  = flag.Duration("libra.update_timeout", 10*time.Second, "timeout for update node")
)
