package statusevent

import (
	"fmt"

	Event "github.com/studease/common/events/event"
)

// StatusEvent types.
const (
	STATUS = "status"
)

// StatusEvent dispatched when a net status event occurred.
type StatusEvent struct {
	Event.Event
	Level       string
	Code        string
	Description string
}

// Init this class.
func (me *StatusEvent) Init(typ string, target interface{}, level, code, description string) *StatusEvent {
	me.Event.Init(typ, target)
	me.Level = level
	me.Code = code
	me.Description = description
	return me
}

// Clone an instance of an StatusEvent subclass.
func (me *StatusEvent) Clone() *StatusEvent {
	return New(me.Type, me.Target, me.Level, me.Code, me.Description)
}

// String returns a string containing all the properties of the StatusEvent object.
func (me *StatusEvent) String() string {
	return fmt.Sprintf("[StatusEvent type=%s level=%s code=%s description=%s]", me.Type, me.Level, me.Code, me.Description)
}

// New creates a new StatusEvent object.
func New(typ string, target interface{}, level, code, description string) *StatusEvent {
	return new(StatusEvent).Init(typ, target, level, code, description)
}
