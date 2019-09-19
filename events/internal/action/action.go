package action

// Action types
const (
	ADD = iota
	REMOVE
)

// Action holds an uncompleted action
type Action struct {
	Type     int
	Event    string
	Listener interface{}
}

// Init this class
func (me *Action) Init(typ int, event string, listener interface{}) *Action {
	me.Type = typ
	me.Event = event
	me.Listener = listener
	return me
}

// New creates a new Action
func New(typ int, event string, listener interface{}) *Action {
	return new(Action).Init(typ, event, listener)
}
