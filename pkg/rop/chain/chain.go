package chain

import (
	"context"

	"github.com/ib-77/rop3/pkg/rop"
	"github.com/ib-77/rop3/pkg/rop/solo"
)

// Chain wraps a rop.Result with context to enable fluent chaining
type Chain[T any] struct {
	ctx    context.Context
	result rop.Result[T]
}

// Start creates a new chain from a rop.Result
func Start[T any](ctx context.Context, result rop.Result[T]) *Chain[T] {
	return &Chain[T]{
		ctx:    ctx,
		result: result,
	}
}

// FromValue creates a new chain from a successful value
func FromValue[T any](ctx context.Context, value T) *Chain[T] {
	return &Chain[T]{
		ctx:    ctx,
		result: rop.Success(value),
	}
}

// Result returns the underlying rop.Result
func (c *Chain[T]) Result() rop.Result[T] {
	return c.result
}

// Then chains a function that returns rop.Result[U]
func Then[T, U any](c *Chain[T], onSuccess func(context.Context, T) rop.Result[U]) *Chain[U] {
	return &Chain[U]{
		ctx:    c.ctx,
		result: solo.Switch[T, U](c.ctx, c.result, onSuccess),
	}
}

// ThenTry chains a function that returns (U, error)
func ThenTry[T, U any](c *Chain[T], tryOnSuccess func(context.Context, T) (U, error)) *Chain[U] {
	return &Chain[U]{
		ctx:    c.ctx,
		result: solo.Try[T, U](c.ctx, c.result, tryOnSuccess),
	}
}

// Map chains a pure transformation function
func Map[T, U any](c *Chain[T], onSuccess func(context.Context, T) U) *Chain[U] {
	return &Chain[U]{
		ctx:    c.ctx,
		result: solo.Map[T, U](c.ctx, c.result, onSuccess),
	}
}

// Ensure performs a side effect without changing the result
func (c *Chain[T]) Ensure(onSuccess func(context.Context, T)) *Chain[T] {
	return &Chain[T]{
		ctx: c.ctx,
		result: solo.Tee[T](c.ctx, c.result,
			func(ctx context.Context, result rop.Result[T]) {
				if result.IsSuccess() {
					onSuccess(ctx, result.Result())
				}
			}),
	}
}

// Finally collapses the chain into a final result using solo.Finally
func Finally[T, U any](c *Chain[T], onSuccess func(context.Context, T) U, onFailure func(context.Context, error) U, onCancel func(context.Context, error) U) U {
	return solo.Finally[T, U](c.ctx, c.result, onSuccess, onFailure, onCancel)
}
