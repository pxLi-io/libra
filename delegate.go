package libra

import ml "github.com/hashicorp/memberlist"

type Delegate struct {
}

func (d *Delegate) NodeMeta(limit int) []byte {
	return nil
}

func (d *Delegate) NotifyMsg([]byte) {

}

func (d *Delegate) GetBroadcasts(overhead, limit int) [][]byte {
	return nil
}

func (d *Delegate) LocalState(join bool) []byte {
	return nil
}

func (d *Delegate) MergeRemoteState(buf []byte, join bool) {

}

type EventDelegate struct {
}

func (e *EventDelegate) NotifyJoin(n *ml.Node) {

}

func (e *EventDelegate) NotifyLeave(n *ml.Node) {

}

func (e *EventDelegate) NotifyUpdate(n *ml.Node) {

}
