package events

import (
	"container/list"
	"reflect"
	"sync"
	"sync/atomic"

	"github.com/studease/common/events/internal/action"
	"github.com/studease/common/log"
	"github.com/studease/common/utils"
)

const (
	// MaxRecursion of event flows
	MaxRecursion int32 = 8
)

// EventDispatcher is the base class for all classes that dispatch events
type EventDispatcher struct {
	mtx       sync.RWMutex
	logger    log.ILogger
	listeners map[string]*list.List
	recursion int32
	goid      int64
	manager   action.Manager
}

// Init this class
func (me *EventDispatcher) Init(logger log.ILogger) *EventDispatcher {
	me.logger = logger
	me.listeners = make(map[string]*list.List)
	me.recursion = 0
	me.goid = 0
	me.manager.Init()
	return me
}

// AddEventListener registers an event listener object with an EventDispatcher object so that the listener receives notification of an event
func (me *EventDispatcher) AddEventListener(event string, listener *EventListener) {
	me.logger.Debugf(1, "Adding event listener: type=%s, listener=%v", event, listener)

	if event == "" || listener == nil {
		me.logger.Debugf(1, "Event type or listener not found: type=%s, listener=%v", event, listener)
		return
	}

	if atomic.LoadInt32(&me.recursion) > 0 {
		me.manager.Append(action.New(action.ADD, event, listener))
		return
	}

	me.mtx.Lock()
	defer me.mtx.Unlock()

	me.addEventListener(event, listener)
}

func (me *EventDispatcher) addEventListener(event string, listener *EventListener) {
	l := me.listeners[event]
	if l == nil {
		l = list.New()
		me.listeners[event] = l
	}

	l.PushBack(listener)
}

// RemoveEventListener removes an event listener from the EventDispatcher object
func (me *EventDispatcher) RemoveEventListener(event string, listener *EventListener) {
	me.logger.Debugf(1, "Removing event listener: type=%s, listener=%v", event, listener)

	if event == "" || listener == nil {
		me.logger.Debugf(1, "Event type or listener not found: type=%s, listener=%v", event, listener)
		return
	}

	if atomic.LoadInt32(&me.recursion) > 0 {
		me.manager.Append(action.New(action.REMOVE, event, listener))
		return
	}

	me.mtx.Lock()
	defer me.mtx.Unlock()

	me.removeEventListener(event, listener)
}

func (me *EventDispatcher) removeEventListener(event string, listener *EventListener) {
	l := me.listeners[event]
	if l == nil {
		me.logger.Debugf(1, "Listeners not found: type=%s", event)
		return
	}

	for e := l.Front(); e != nil; e = e.Next() {
		if e.Value.(*EventListener) == listener {
			l.Remove(e)
			break
		}
	}
}

// HasEventListener checks whether an event listener is registered with this EventDispatcher object for the specified event type
func (me *EventDispatcher) HasEventListener(event string) bool {
	curr := utils.GoID()
	goid := atomic.LoadInt64(&me.goid)

	if atomic.CompareAndSwapInt64(&me.goid, 0, curr) || goid != curr {
		me.mtx.RLock()
		defer me.mtx.RUnlock()
	}

	l := me.listeners[event]
	return l != nil && l.Len() != 0
}

// DispatchEvent dispatches an event into the event flow
func (me *EventDispatcher) DispatchEvent(e interface{}) {
	defer func() {
		if err := recover(); err != nil {
			me.logger.Debugf(1, "Failed to reflect event: %v", err)
		}
	}()

	value := reflect.ValueOf(e)
	event := value.Elem().FieldByName("Type").String()

	me.logger.Debugf(0, "Dispatching event: %s", event)
	me.dispatchEvent(me.listeners[event], event, value)
}

func (me *EventDispatcher) dispatchEvent(l *list.List, event string, value reflect.Value) {
	if l == nil {
		return
	}

	defer func() {
		if err := recover(); err != nil {
			me.logger.Debugf(1, "Failed to handle event: type=%s, %v", event, err)
		}
	}()

	curr := utils.GoID()
	goid := atomic.LoadInt64(&me.goid)

	if atomic.CompareAndSwapInt64(&me.goid, 0, curr) || goid != curr {
		me.mtx.Lock()
		defer func() {
			atomic.StoreInt64(&me.goid, 0)
			me.mtx.Unlock()
		}()
	}

	recursion := atomic.AddInt32(&me.recursion, 1)
	defer func() {
		atomic.AddInt32(&me.recursion, -1)
	}()

	if recursion > MaxRecursion {
		panic("max recursion reached")
	}

	for e := l.Front(); e != nil; e = e.Next() {
		listener := e.Value.(*EventListener)
		reflect.ValueOf(listener.handler).Call([]reflect.Value{value})

		if listener.count > 0 {
			listener.count--
			if listener.count == 0 {
				me.RemoveEventListener(event, listener)
			}
		}

		stopPropagation := value.Elem().FieldByName("stopPropagation").Bool()
		if stopPropagation {
			me.logger.Debugf(1, "Stop propagation event: type=%s", event)
			break
		}
	}

	me.manager.ForEach(func(a *action.Action) {
		switch a.Type {
		case action.ADD:
			me.addEventListener(a.Event, a.Listener.(*EventListener))
		case action.REMOVE:
			me.removeEventListener(a.Event, a.Listener.(*EventListener))
		}
	})
}
