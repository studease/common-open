package mediaevent

import (
	"fmt"

	Event "github.com/studease/common/events/event"
)

// MediaRecorderEvent types.
const (
	START  = "start"
	PAUSE  = "pause"
	RESUME = "resume"
	STOP   = "stop"
)

// MediaRecorderEvent dispatched when the state of MediaRecorder has changed.
type MediaRecorderEvent struct {
	Event.Event
}

// Init this class.
func (me *MediaRecorderEvent) Init(typ string, target interface{}) *MediaRecorderEvent {
	me.Event.Init(typ, target)
	return me
}

// Clone an instance of an MediaRecorderEvent subclass.
func (me *MediaRecorderEvent) Clone() *MediaRecorderEvent {
	return New(me.Type, me.Target)
}

// String returns a string containing all the properties of the MediaRecorderEvent object.
func (me *MediaRecorderEvent) String() string {
	return fmt.Sprintf("[MediaRecorderEvent type=%s]", me.Type)
}

// New creates a new MediaRecorderEvent object.
func New(typ string, target interface{}) *MediaRecorderEvent {
	return new(MediaRecorderEvent).Init(typ, target)
}
