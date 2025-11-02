package c2

import (
	"context"

	"github.com/ib-77/rop3/pkg/rop"
	"github.com/ib-77/rop3/pkg/rop/solo"
)

// Chain wraps a rop.Result with context to enable fluent chaining
type Chain[T, U any] struct {
	ctx    context.Context
	input  rop.Result[T]
	result rop.Result[U]
}

// Start creates a new chain from a rop.Result
func Start[T, U any](ctx context.Context, result rop.Result[T]) *Chain[T, U] {
	return &Chain[T, U]{
		ctx:   ctx,
		input: result,
		//result: result,
	}
}

// FromValue creates a new chain from a successful value
func FromValue[T any](ctx context.Context, value T) *Chain[T, T] {
	return &Chain[T, T]{
		ctx:    ctx,
		input:  rop.Success(value),
		result: rop.Success(value),
	}
}

// Result returns the underlying rop.Result
func (c *Chain[T, U]) Result() rop.Result[U] {
	return c.result
}

func (c *Chain[T, U]) Input() rop.Result[T] {
	return c.input
}

// Then chains a function that returns rop.Result[U]
func (c *Chain[T, U]) Then(onSuccess func(context.Context, T) rop.Result[U]) *Chain[T, U] {
	return &Chain[T, U]{
		ctx:    c.ctx,
		input:  c.input,
		result: solo.Switch[T, U](c.ctx, c.input, onSuccess),
	}
}

// ThenTry chains a function that returns (U, error)
func (c *Chain[T, U]) ThenTry(tryOnSuccess func(context.Context, T) (U, error)) *Chain[T, U] {
	return &Chain[T, U]{
		ctx:    c.ctx,
		input:  c.input,
		result: solo.Try[T, U](c.ctx, c.input, tryOnSuccess),
	}
}

// Map chains a pure transformation function
func (c *Chain[T, U]) Map(onSuccess func(context.Context, T) U) *Chain[T, U] {
	return &Chain[T, U]{
		ctx:    c.ctx,
		input:  c.input,
		result: solo.Map[T, U](c.ctx, c.input, onSuccess),
	}
}

// Ensure performs a side effect without changing the result
func (c *Chain[T, U]) Ensure(onSuccess func(context.Context, T)) *Chain[T, T] {
	return &Chain[T, T]{
		ctx:   c.ctx,
		input: c.input,
		result: solo.Tee[T](c.ctx, c.input,
			func(ctx context.Context, result rop.Result[T]) {
				if result.IsSuccess() {
					onSuccess(ctx, result.Result())
				}
			}),
	}
}

// Finally collapses the chain into a final result using solo.Finally
func (c *Chain[T, U]) Finally(onSuccess func(context.Context, T) U,
	onFailure func(context.Context, error) U, onCancel func(context.Context, error) U) U {
	return solo.Finally[T, U](c.ctx, c.input, onSuccess, onFailure, onCancel)
}
