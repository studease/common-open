package netstatusevent

import (
	"fmt"

	"github.com/studease/common/av/utils/amf"
	Event "github.com/studease/common/events/event"
)

// NetStatusEvent types
const (
	NET_STATUS = "onStatus"
)

// NetStatusEvent dispatched when a net status event occurred
type NetStatusEvent struct {
	Event.Event
	Info *amf.Value
}

// Init this class
func (me *NetStatusEvent) Init(typ string, target interface{}, info *amf.Value) *NetStatusEvent {
	me.Event.Init(typ, target)
	me.Info = info
	return me
}

// Clone an instance of an NetStatusEvent subclass
func (me *NetStatusEvent) Clone() *NetStatusEvent {
	return New(me.Type, me.Target, me.Info)
}

// String returns a string containing all the properties of the NetStatusEvent object
func (me *NetStatusEvent) String() string {
	return fmt.Sprintf("[NetStatusEvent type=%s info=%v]", me.Type, me.Info)
}

// New creates a new NetStatusEvent object
func New(typ string, target interface{}, info *amf.Value) *NetStatusEvent {
	return new(NetStatusEvent).Init(typ, target, info)
}
