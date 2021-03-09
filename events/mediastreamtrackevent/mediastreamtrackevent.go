package mediastreamtrackevent

import (
	"fmt"

	"github.com/studease/common/av"
	Event "github.com/studease/common/events/event"
)

// MediaStreamTrackEvent types
const (
	ADDTRACK    = "addtrack"
	REMOVETRACK = "removetrack"
)

// MediaStreamTrackEvent represents an event announcing that a IMediaStreamTrack has been added to or removed from a IMediaStream.
type MediaStreamTrackEvent struct {
	Event.Event
	Track av.IMediaStreamTrack
}

// Init this class.
func (me *MediaStreamTrackEvent) Init(typ string, target interface{}, track av.IMediaStreamTrack) *MediaStreamTrackEvent {
	me.Event.Init(typ, target)
	me.Track = track
	return me
}

// Clone an instance of an MediaStreamTrackEvent subclass.
func (me *MediaStreamTrackEvent) Clone() *MediaStreamTrackEvent {
	return New(me.Type, me.Target, me.Track)
}

// String returns a string containing all the properties of the MediaStreamTrackEvent object
func (me *MediaStreamTrackEvent) String() string {
	return fmt.Sprintf("[MediaStreamTrackEvent type=%s kind=%s]", me.Type, me.Track.Kind())
}

// New creates a new MediaStreamTrackEvent object
func New(typ string, target interface{}, track av.IMediaStreamTrack) *MediaStreamTrackEvent {
	return new(MediaStreamTrackEvent).Init(typ, target, track)
}
