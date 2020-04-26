package libra

import ml "github.com/hashicorp/memberlist"

type Delegate struct{
	*ml.Memberlist
}

func (d *Delegate) NodeMeta(limit int) []byte {
	return d.LocalNode().Meta
}

func (d *Delegate) NotifyMsg([]byte) {}

func (d *Delegate) GetBroadcasts(overhead, limit int) [][]byte {
	return nil
}

func (d *Delegate) LocalState(join bool) []byte {
	return nil
}

func (d *Delegate) MergeRemoteState(buf []byte, join bool) {}

type EventDelegate struct {
	UpdateCh chan *ml.Node
}

func (ed *EventDelegate) NotifyJoin(n *ml.Node) {}

func (ed *EventDelegate) NotifyLeave(n *ml.Node) {}

func (ed *EventDelegate) NotifyUpdate(n *ml.Node) {
	ed.UpdateCh <- n
}

type ConflictDelegate struct{}
