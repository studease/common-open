package event

import (
	"fmt"
)

// Event types
const (
	ACTIVATE   = "activate"
	ADDED      = "added"
	CANCEL     = "cancel"
	CHANGE     = "change"
	CLEAR      = "clear"
	CLOSE      = "close"
	COMPLETE   = "complete"
	CONNECT    = "connect"
	DEACTIVATE = "deactivate"
	IDLE       = "idle"
	INIT       = "init"
	OPEN       = "open"
	REMOVED    = "removed"
)

// Event is used as the base class for the creation of Event objects, which are passed as parameters to event listeners when an event occurs
type Event struct {
	Type            string
	Target          interface{}
	stopPropagation bool
}

// Init this class
func (me *Event) Init(typ string, target interface{}) *Event {
	me.Type = typ
	me.Target = target
	me.stopPropagation = false
	return me
}

// Clone an instance of an Event subclass
func (me *Event) Clone() *Event {
	return New(me.Type, me.Target)
}

// StopPropagation prevents processing of any event listeners in nodes subsequent to the current node in the event flow
func (me *Event) StopPropagation() {
	me.stopPropagation = true
}

// String returns a string containing all the properties of the Event object
func (me *Event) String() string {
	return fmt.Sprintf("[Event type=%s]", me.Type)
}

// New creates a new Event object
func New(typ string, target interface{}) *Event {
	return new(Event).Init(typ, target)
}
