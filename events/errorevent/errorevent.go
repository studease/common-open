package errorevent

import (
	"fmt"

	Event "github.com/studease/common/events/event"
)

// ErrorEvent types
const (
	ERROR = "error"
)

// ErrorEvent dispatched when an error causes an asynchronous operation to fail
type ErrorEvent struct {
	Event.Event
	Error error
}

// Init this class
func (me *ErrorEvent) Init(typ string, target interface{}, err error) *ErrorEvent {
	me.Event.Init(typ, target)
	me.Error = err
	return me
}

// Clone an instance of an ErrorEvent subclass
func (me *ErrorEvent) Clone() *ErrorEvent {
	return New(me.Type, me.Target, me.Error)
}

// String returns a string containing all the properties of the ErrorEvent object
func (me *ErrorEvent) String() string {
	return fmt.Sprintf("[ErrorEvent type=%s error=%v]", me.Type, me.Error)
}

// New creates a new ErrorEvent object
func New(typ string, target interface{}, err error) *ErrorEvent {
	return new(ErrorEvent).Init(typ, target, err)
}
