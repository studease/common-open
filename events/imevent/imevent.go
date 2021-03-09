package imevent

import (
	"fmt"

	Event "github.com/studease/common/events/event"
	"github.com/studease/common/im/message"
)

// IMEvent types.
const (
	MESSAGE = "message"
)

// IMEvent dispatched when an im message received.
type IMEvent struct {
	Event.Event
	Message *message.Message
}

// Init this class
func (me *IMEvent) Init(typ string, target interface{}, message *message.Message) *IMEvent {
	me.Event.Init(typ, target)
	me.Message = message
	return me
}

// Clone an instance of an IMEvent subclass.
func (me *IMEvent) Clone() *IMEvent {
	return New(me.Type, me.Target, me.Message)
}

// String returns a string containing all the properties of the IMEvent object.
func (me *IMEvent) String() string {
	return fmt.Sprintf("[IMEvent type=%s message=%v]", me.Type, me.Message)
}

// New creates a new IMEvent object.
func New(typ string, target interface{}, message *message.Message) *IMEvent {
	return new(IMEvent).Init(typ, target, message)
}
