package events

// IEvent defines basic event methods
type IEvent interface {
	Clone() IEvent
	StopPropagation()
	String() string
}

// IEventDispatcher defines methods for adding or removing event listeners, checks whether specific types of event listeners are registered, and dispatches events
type IEventDispatcher interface {
	AddEventListener(event string, listener *EventListener)
	RemoveEventListener(event string, listener *EventListener)
	HasEventListener(event string) bool
	DispatchEvent(event interface{})
}
