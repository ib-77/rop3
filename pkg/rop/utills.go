package rop

import "reflect"

func IsNil(i interface{}) bool {
	if i == nil || (reflect.ValueOf(i).Kind() == reflect.Ptr && reflect.ValueOf(i).IsNil()) {
		return true
	}
	return false
}

func GetErrors(err error) []error {
	if IsNil(err) {
		return []error{}
	}

	e, ok := err.(interface{ Unwrap() []error })
	if ok {
		return e.Unwrap()
	}

	return []error{err}
}
