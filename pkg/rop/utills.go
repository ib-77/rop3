package rop

import (
	"context"
	"errors"
	"reflect"
)

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

func IsCancellationError(err error) bool {
	return errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled)
}
