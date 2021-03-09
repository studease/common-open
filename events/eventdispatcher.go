package events

import (
	"reflect"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/studease/common/log"
	"github.com/studease/common/utils"
)

// Static constants.
const (
	MaxRecursion int32 = 8
)

// EventDispatcher is the base class for all classes that dispatch events.
// It uses array instead of list, which causes the add/remove method expensive.
// However, it is possible to clone the listener group fast while triggering an event.
// And, the frequency of triggering event is much higher than that of add/remove.
type EventDispatcher struct {
	logger    log.ILogger
	mtx       sync.Mutex
	listeners map[string]map[uintptr]*EventListener
	goid      int64
	recursion int32
}

// Init this class.
func (me *EventDispatcher) Init(logger log.ILogger) *EventDispatcher {
	me.logger = logger
	me.listeners = make(map[string]map[uintptr]*EventListener)
	me.goid = 0
	me.recursion = 0
	return me
}

// AddEventListener registers an event listener object with an EventDispatcher object so that the listener receives notification of an event.
func (me *EventDispatcher) AddEventListener(event string, listener *EventListener) {
	if event == "" || listener == nil {
		me.logger.Debugf(1, "Event type or listener not present: type=%s, listener=%v", event, listener)
		return
	}

	self := utils.GoID()
	if atomic.LoadInt64(&me.goid) != self {
		me.mtx.Lock()
		atomic.StoreInt64(&me.goid, self)
		defer func() {
			atomic.StoreInt64(&me.goid, 0)
			me.mtx.Unlock()
		}()
	}

	me.addEventListener(event, listener)
}

func (me *EventDispatcher) addEventListener(event string, listener *EventListener) {
	evts := me.listeners[event]
	if evts == nil {
		evts = make(map[uintptr]*EventListener, 0)
		me.listeners[event] = evts
	}
	me.logger.Debugf(1, "Adding event listener: type=%s, listener=%v", event, listener)
	evts[uintptr(unsafe.Pointer(listener))] = listener
}

// RemoveEventListener removes an event listener from the EventDispatcher object.
func (me *EventDispatcher) RemoveEventListener(event string, listener *EventListener) {
	if event == "" || listener == nil {
		me.logger.Debugf(1, "Event type or listener not present: type=%s, listener=%v", event, listener)
		return
	}

	self := utils.GoID()
	if atomic.LoadInt64(&me.goid) != self {
		me.mtx.Lock()
		atomic.StoreInt64(&me.goid, self)
		defer func() {
			atomic.StoreInt64(&me.goid, 0)
			me.mtx.Unlock()
		}()
	}

	me.removeEventListener(event, listener)
}

func (me *EventDispatcher) removeEventListener(event string, listener *EventListener) {
	evts := me.listeners[event]
	if evts == nil {
		me.logger.Debugf(1, "Listeners not found: type=%s", event)
		return
	}
	me.logger.Debugf(1, "Removing event listener: type=%s, listener=%v", event, listener)
	delete(evts, uintptr(unsafe.Pointer(listener)))
}

// HasEventListener checks whether an event listener is registered with this EventDispatcher object for the specified event type.
func (me *EventDispatcher) HasEventListener(event string) bool {
	self := utils.GoID()
	if atomic.LoadInt64(&me.goid) != self {
		me.mtx.Lock()
		atomic.StoreInt64(&me.goid, self)
		defer func() {
			atomic.StoreInt64(&me.goid, 0)
			me.mtx.Unlock()
		}()
	}

	evts := me.listeners[event]
	return evts != nil && len(evts) != 0
}

// DispatchEvent dispatches an event into the event flow.
func (me *EventDispatcher) DispatchEvent(evt interface{}) {
	defer func() {
		if err := recover(); err != nil {
			me.logger.Debugf(1, "Failed to reflect event: %v", err)
		}
	}()

	value := reflect.ValueOf(evt)
	event := value.Elem().FieldByName("Type").String()

	defer func() {
		if err := recover(); err != nil {
			me.logger.Debugf(1, "Failed to handle event: type=%s, %v", event, err)
		}
	}()

	// It is not recommended which multi-goroutines call the same target,
	// especially in high-frequency. Better to run as self-driven.
	self := utils.GoID()
	if atomic.LoadInt64(&me.goid) != self {
		me.mtx.Lock()
		atomic.StoreInt64(&me.goid, self)
		defer func() {
			atomic.StoreInt64(&me.goid, 0)
			me.mtx.Unlock()
		}()
	}

	me.logger.Debugf(0, "Dispatching event: %s", event)
	recursion := atomic.AddInt32(&me.recursion, 1)
	defer func() {
		atomic.AddInt32(&me.recursion, -1)
	}()
	if recursion > MaxRecursion {
		panic("max recursion reached")
	}

	// Make a shallow copy of the listener map, so that it triggers the event to the exact listeners.
	evts := me.listeners[event]
	cloned := make(map[uintptr]*EventListener, len(evts))
	for k, v := range evts {
		cloned[k] = v
	}

	for _, listener := range cloned {
		reflect.ValueOf(listener.handler).Call([]reflect.Value{value})

		if listener.count > 0 {
			if listener.count--; listener.count == 0 {
				me.removeEventListener(event, listener)
			}
		}
		if stopPropagation := value.Elem().FieldByName("StopPropagation").Bool(); stopPropagation {
			me.logger.Debugf(1, "Stop propagation event: type=%s", event)
			break
		}
	}
}
