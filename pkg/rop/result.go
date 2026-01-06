package rop

import (
	"time"

	"github.com/google/uuid"
)

type Result[T any] struct {
	id          uuid.UUID
	createdAt   time.Time
	result      T
	err         error
	isSuccess   bool
	isCancel    bool
	hasResult   bool
	isProcessed bool // WARNING: tiny package implements ONLY this
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

// SetProcessed mark result as processed (pipeline should not do anything on this result)
// This applies to successful results (in case of failure, processing stops as intended by the design).
// WARNING: tiny package implements ONLY this
func SetProcessed[T any](r Result[T]) Result[T] {
	return Result[T]{
		isProcessed: true,
		result:      r.result,
		err:         r.err,
		isSuccess:   r.isSuccess,
		isCancel:    r.isCancel,
		createdAt:   r.createdAt,
		hasResult:   r.hasResult,
		id:          r.id,
	}
}

func SuccessAndProcessed[T any](r T) Result[T] {
	return SetProcessed(Success(r))
}

func FailAndProcessed[T any](er error) Result[T] {
	return SetProcessed(Fail[T](er))
}

func CancelAndProcessed[T any](er error) Result[T] {
	return SetProcessed(Cancel[T](er))
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

func (r Result[T]) IsFailure() bool {
	return !r.IsSuccess()
}

func (r Result[T]) IsProcessed() bool {
	return r.isProcessed
}
