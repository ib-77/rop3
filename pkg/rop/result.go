package rop

import (
	"time"

	"github.com/google/uuid"
)

type Result[T any] struct {
	id        uuid.UUID
	createdAt time.Time
	result    T
	err       error
	isSuccess bool
	isCancel  bool
	hasResult bool
}

func Success[T any](r T) Result[T] {
	return Result[T]{
		result:    r,
		err:       nil,
		isSuccess: true,
		isCancel:  false,
		createdAt: time.Now().UTC(),
		hasResult: true,
		id:        uuid.New(),
	}
}

func Fail[T any](err error) Result[T] {
	return Result[T]{
		err:       err,
		isSuccess: false,
		isCancel:  false,
		createdAt: time.Now().UTC(),
		hasResult: false,
		id:        uuid.New(),
	}
}

func Cancel[T any](err error) Result[T] {
	return Result[T]{
		err:       err,
		isSuccess: false,
		isCancel:  true,
		createdAt: time.Now().UTC(),
		hasResult: false,
		id:        uuid.New(),
	}
}

//func CancelWithResult[T any](err error, res T) Result[T] {
//	return Result[T]{
//		err:       err,
//		isSuccess: false,
//		isCancel:  true,
//		createdAt: time.Now().UTC(),
//		result:    res,
//		hasResult: true,
//		id:        uuid.New(),
//	}
//}

func CancelFrom[In, Out any](from Result[In]) Result[Out] {
	return Result[Out]{
		err:       from.err,
		isSuccess: from.isSuccess,
		isCancel:  from.isCancel,
		createdAt: from.createdAt,
		hasResult: from.hasResult,
		id:        from.id,
	}
}

func (r Result[T]) Result() T {
	return r.result
}

func (r Result[T]) Err() error {
	return r.err
}

func (r Result[T]) IsSuccess() bool {
	return r.isSuccess
}

func (r Result[T]) IsCancel() bool {
	return r.isCancel
}

func (r Result[T]) IsCancelWithResult() bool {
	return r.isCancel && r.hasResult
}

func (r Result[T]) HasResult() bool {
	return r.hasResult
}

func (r Result[T]) CreatedAt() time.Time {
	return r.createdAt
}

func (r Result[T]) IsEmpty() bool {
	return r.err == nil && !r.isCancel && !r.isSuccess
}

func (r Result[T]) Id() uuid.UUID {
	return r.id
}
