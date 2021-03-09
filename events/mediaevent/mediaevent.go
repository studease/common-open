package mediaevent

import (
	"fmt"

	"github.com/studease/common/av"
	Event "github.com/studease/common/events/event"
)

// MediaEvent types.
const (
	PACKET = "packet"
)

// MediaEvent dispatched when a media packet received.
type MediaEvent struct {
	Event.Event
	Packet *av.Packet
}

// Init this class.
func (me *MediaEvent) Init(typ string, target interface{}, pkt *av.Packet) *MediaEvent {
	me.Event.Init(typ, target)
	me.Packet = pkt
	return me
}

// Clone an instance of an MediaEvent subclass.
func (me *MediaEvent) Clone() *MediaEvent {
	return New(me.Type, me.Target, me.Packet)
}

// String returns a string containing all the properties of the MediaEvent object.
func (me *MediaEvent) String() string {
	return fmt.Sprintf("[MediaEvent type=%s codec=%02X size=%d]", me.Type, me.Packet.Codec, me.Packet.Length)
}

// New creates a new MediaEvent object.
func New(typ string, target interface{}, pkt *av.Packet) *MediaEvent {
	return new(MediaEvent).Init(typ, target, pkt)
}
