package amf

import (
	"container/list"
	"fmt"
	"time"
)

// Value defines different AMF types
type Value struct {
	Type   byte
	Key    string
	value  interface{}
	table  map[string]*Value
	offset uint16
	Ended  bool
}

// Init this class
func (me *Value) Init(typ byte) *Value {
	me.Type = typ
	me.Key = ""
	me.value = nil
	me.table = nil
	me.offset = 0
	me.Ended = false
	return me
}

// Set typed value on the key
func (me *Value) Set(key string, value interface{}, offset ...uint16) *Value {
	switch me.Type {
	case DOUBLE:
		if _, ok := value.(float64); !ok {
			panic("value should be float64")
		}

	case BOOLEAN:
		if _, ok := value.(bool); !ok {
			panic("value should be bool")
		}

	case STRING:
		fallthrough
	case LONG_STRING:
		if _, ok := value.(string); !ok {
			panic("value should be string")
		}

	case OBJECT:
		fallthrough
	case ECMA_ARRAY:
		l, ok := value.(*list.List)
		if !ok {
			panic("value should be List")
		}

		if me.table == nil {
			me.table = make(map[string]*Value)
		}

		for e := l.Front(); e != nil; e = e.Next() {
			v := e.Value.(*Value)
			me.table[v.Key] = v
		}

	case STRICT_ARRAY:
		if _, ok := value.(*list.List); !ok {
			panic("value should be List")
		}

	case DATE:
		if _, ok := value.(float64); !ok {
			panic("value should be float64")
		}

		if len(offset) == 0 {
			panic("offset not presented")
		}

		me.offset = offset[0]

	case NULL:
		fallthrough
	case UNDEFINED:
		if value != nil {
			panic("value should be nil")
		}

	default:
		panic(fmt.Errorf("unrecognized AMF type %02X", me.Type))
	}

	me.Key = key
	me.value = value

	return me
}

// Get value by the key if type equals to OBJECT/ECMA_ARRAY, otherwise panic
func (me *Value) Get(key string) *Value {
	switch me.Type {
	case OBJECT:
		fallthrough
	case ECMA_ARRAY:
		return me.table[key]

	default:
		panic(fmt.Errorf("operation not allowed on this type %02X", me.Type))
	}
}

// Add a key-value pair into the value, panic if not OBJECT/ECMA_ARRAY/STRICT_ARRAY
func (me *Value) Add(v *Value) {
	switch me.Type {
	case OBJECT:
		fallthrough
	case ECMA_ARRAY:
		if me.table == nil {
			me.table = make(map[string]*Value)
		}

		me.table[v.Key] = v
		fallthrough

	case STRICT_ARRAY:
		l, ok := me.value.(*list.List)
		if !ok {
			l = list.New()
			me.value = l
		}

		l.PushBack(v)

	default:
		panic("type not befitting to the operation")
	}
}

// Del the value related to the key, panic if not OBJECT/ECMA_ARRAY
func (me *Value) Del(key string) *Value {
	switch me.Type {
	case OBJECT:
		fallthrough
	case ECMA_ARRAY:
		if me.table == nil {
			return nil
		}

		delete(me.table, key)

		l, ok := me.value.(*list.List)
		if ok {
			for e := l.Front(); e != nil; e = e.Next() {
				v := e.Value.(*Value)
				if v.Key == key {
					l.Remove(e)
					return v
				}
			}
		}

	default:
		panic("type not befitting to the operation")
	}

	return nil
}

// Double returns value as float64
func (me *Value) Double() float64 {
	return me.value.(float64)
}

// Bool returns value as bool
func (me *Value) Bool() bool {
	return me.value.(bool)
}

// String returns value as string
func (me *Value) String() string {
	return me.value.(string)
}

// Raw returns raw value
func (me *Value) Raw() interface{} {
	return me.value
}

// Time returns value as Time
func (me *Value) Time() time.Time {
	ms := me.value.(float64) + float64(me.offset)*60*1000
	return time.Unix(int64(ms/1000), int64(ms)%1000)
}

// NewValue creates an Value
func NewValue(typ byte) *Value {
	return new(Value).Init(typ)
}
