package errorevent

import (
	"fmt"

	Event "github.com/studease/common/events/event"
)

// ErrorEvent types.
const (
	ERROR = "error"
)

// ErrorEvent dispatched when an error causes an asynchronous operation to fail.
type ErrorEvent struct {
	Event.Event
	Name    string
	Message error
}

// Init this class
func (me *ErrorEvent) Init(typ string, target interface{}, name string, message error) *ErrorEvent {
	me.Event.Init(typ, target)
	me.Name = name
	me.Message = message
	return me
}

// Clone an instance of an ErrorEvent subclass.
func (me *ErrorEvent) Clone() *ErrorEvent {
	return New(me.Type, me.Target, me.Name, me.Message)
}

// String returns a string containing all the properties of the ErrorEvent object.
func (me *ErrorEvent) String() string {
	return fmt.Sprintf("[ErrorEvent type=%s name=%s message=%v]", me.Type, me.Name, me.Message)
}

// New creates a new ErrorEvent object.
func New(typ string, target interface{}, name string, message error) *ErrorEvent {
	return new(ErrorEvent).Init(typ, target, name, message)
}
