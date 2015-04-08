package dms

import (
	"fmt"
	"reflect"
)

type ErrNotProvided struct {
	What string
}

type ErrTypeMismatch struct {
	Provided reflect.Type
	Required reflect.Type
}

func (e ErrTypeMismatch) Error() string { //NOCOVER
	return fmt.Sprintf("provided %v, required %v", e.Provided, e.Required)
}

type ErrUnknownCastType struct {
	What reflect.Type
}

type ErrBadCastFunc struct {
	What interface{}
}

type ErrStarvation struct {
}
