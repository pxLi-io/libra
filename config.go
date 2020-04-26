package libra

import (
	"github.com/hashicorp/memberlist"
)

type Config struct {
	*memberlist.Config

	Seeds          []string
	LoadMultiplier int
}
