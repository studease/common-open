package commandevent

import (
	"fmt"

	Event "github.com/studease/common/events/event"
	"github.com/studease/common/rtmp/message"
)

// CommandEvent types
const (
	CONNECT       = "connect"
	CLOSE         = "close"
	CREATE_STREAM = "createStream"
	RESULT        = "_result"
	ERROR         = "_error"

	PLAY          = "play"
	PLAY2         = "play2"
	DELETE_STREAM = "deleteStream"
	CLOSE_STREAM  = "closeStream"
	RECEIVE_AUDIO = "receiveAudio"
	RECEIVE_VIDEO = "receiveVideo"
	PUBLISH       = "publish"
	FC_UNPUBLISH  = "FCUnpublish"
	SEEK          = "seek"
	PAUSE         = "pause"

	CHECK_BANDWIDTH = "checkBandwidth"
	GET_STATS       = "getStats"
)

// CommandEvent dispatched when a RTMP command received
type CommandEvent struct {
	Event.Event
	Message *message.CommandMessage
}

// Init this class
func (me *CommandEvent) Init(typ string, target interface{}, m *message.CommandMessage) *CommandEvent {
	me.Event.Init(typ, target)
	me.Message = m
	return me
}

// Clone an instance of an CommandEvent subclass
func (me *CommandEvent) Clone() *CommandEvent {
	return New(me.Type, me.Target, me.Message)
}

// String returns a string containing all the properties of the CommandEvent object
func (me *CommandEvent) String() string {
	return fmt.Sprintf("[CommandEvent type=%s message=%s]", me.Type, me.Message.CommandName)
}

// New creates a new CommandEvent object
func New(typ string, target interface{}, m *message.CommandMessage) *CommandEvent {
	return new(CommandEvent).Init(typ, target, m)
}
