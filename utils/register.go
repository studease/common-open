package utils

import (
	"reflect"
)

// Register provides methods to store and create objects
type Register struct {
	types map[string]reflect.Type
}

// Init this class
func (me *Register) Init() *Register {
	me.types = make(map[string]reflect.Type)
	return me
}

// Add registers an object with the given name
func (me *Register) Add(name string, obj interface{}) {
	me.types[name] = reflect.ValueOf(obj).Type()
}

// New creates an registered object by the name
func (me *Register) New(name string) interface{} {
	if t := me.types[name]; t != nil {
		return reflect.New(t).Interface()
	}

	return nil
}

// NewRegister creates a Register
func NewRegister() *Register {
	return new(Register).Init()
}
