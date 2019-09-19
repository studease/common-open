package action

import (
	"container/list"
	"sync"
)

type Manager struct {
	mtx     sync.RWMutex
	actions list.List
}

// Init this class
func (me *Manager) Init() *Manager {
	me.actions.Init()
	return me
}

// Append action to the list
func (me *Manager) Append(a *Action) {
	me.mtx.Lock()
	defer me.mtx.Unlock()

	me.actions.PushBack(a)
}

// ForEach calls the func for each of action in the list
func (me *Manager) ForEach(cb func(*Action)) {
	me.mtx.RLock()
	defer me.mtx.RUnlock()

	for e := me.actions.Front(); e != nil; e = me.actions.Front() {
		a := e.Value.(*Action)
		cb(a)

		me.actions.Remove(e)
	}
}
