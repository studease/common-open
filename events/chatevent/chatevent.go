package mediaevent

import (
	"fmt"

	"github.com/studease/common/chat/message"
	Event "github.com/studease/common/events/event"
)

// ChatEvent types
const (
	MESSAGE = "chat-message"
)

// ChatEvent dispatched when a chat message received
type ChatEvent struct {
	Event.Event
	Message *message.Message
}

// Init this class
func (me *ChatEvent) Init(typ string, target interface{}, m *message.Message) *ChatEvent {
	me.Event.Init(typ, target)
	me.Message = m
	return me
}

// Clone an instance of an ChatEvent subclass
func (me *ChatEvent) Clone() *ChatEvent {
	return New(me.Type, me.Target, me.Message)
}

// String returns a string containing all the properties of the ChatEvent object
func (me *ChatEvent) String() string {
	return fmt.Sprintf("[ChatEvent type=%s cmd=%s data=%s mode=%d sn=%d tar=%s]",
		me.Type, me.Message.Command, me.Message.Data, me.Message.Mode, me.Message.SN, me.Message.Target)
}

// New creates a new ChatEvent object
func New(typ string, target interface{}, m *message.Message) *ChatEvent {
	return new(ChatEvent).Init(typ, target, m)
}
