package rop

import "time"

type ResultProvider[T any] interface {
	// Result returns the successful result value
	Result() T
	// CreatedAt time creation (UTC)
	CreatedAt() time.Time
}

// WithError defines an interface for types that can return a result or an error
type WithError[T any] interface {
	ResultProvider[T]
	// Err returns the error if operation failed
	Err() error
	// IsSuccess returns true if the operation was successful
	IsSuccess() bool
}

// WithCancel extends WithError with cancellation support
type WithCancel[T any] interface {
	WithError[T]
	// IsCancel returns true if the operation was cancelled
	IsCancel() bool
}

//type WithCancelAndResult[T any] interface {
//	WithCancel[T]
//	IsCancelWithResult() bool
//}
//
//type WithEmpty[T any] interface {
//	WithCancelAndResult[T]
//	IsEmpty() bool
//}
