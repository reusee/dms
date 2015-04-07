package dms

import "reflect"

type ErrNotProvided struct {
	What string
}

type ErrUnknownCastType struct {
	What reflect.Type
}

type ErrBadCastFunc struct {
	What interface{}
}
