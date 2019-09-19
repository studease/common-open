package timerevent

import (
	"fmt"

	Event "github.com/studease/common/events/event"
)

// TimerEvent types
const (
	TIMER    = "timer"
	COMPLETE = "timer-complete"
)

// TimerEvent dispatched whenever the Timer object reaches the interval specified by the Timer.delay property
type TimerEvent struct {
	Event.Event
}

// Init this class
func (me *TimerEvent) Init(typ string, target interface{}) *TimerEvent {
	me.Event.Init(typ, target)
	return me
}

// Clone an instance of an TimerEvent subclass
func (me *TimerEvent) Clone() *TimerEvent {
	return New(me.Type, me.Target)
}

// String returns a string containing all the properties of the TimerEvent object
func (me *TimerEvent) String() string {
	return fmt.Sprintf("[TimerEvent type=%s]", me.Type)
}

// New creates a new TimerEvent object
func New(typ string, target interface{}) *TimerEvent {
	return new(TimerEvent).Init(typ, target)
}
