package libra

import "errors"

var (
	ErrNodeNotExist   = errors.New("node does not exist")
	ErrQuorumNotMet   = errors.New("quorum is not met")
	ErrUpdateLoadInCD = errors.New("updateLoad is in cooldown. try again later")
)
