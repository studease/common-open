package events

// EventListener holds the event handler
type EventListener struct {
	handler interface{}
	count   int
}

// Init this class
// @count: if -1, no limit
func (me *EventListener) Init(handler interface{}, count int) *EventListener {
	me.handler = handler
	me.count = count
	return me
}

// NewListener returns new EventListener
func NewListener(handler interface{}, count int) *EventListener {
	return new(EventListener).Init(handler, count)
}
