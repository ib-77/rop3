package tiny

import (
	"context"

	"github.com/ib-77/rop3/pkg/rop"
	"github.com/ib-77/rop3/pkg/rop/solo"
)

type Chain[T any] struct {
	ctx context.Context
	res rop.Result[T]
}

func Start[T any](ctx context.Context, r rop.Result[T]) Chain[T] {
	return Chain[T]{ctx: ctx, res: r}
}

func FromValue[T any](ctx context.Context, v T) Chain[T] {
	return Start(ctx, rop.Success(v))
}

func (c Chain[T]) Result() rop.Result[T] {
	return c.res
}

// Then composes functions that already return rop.Result[T]
func (c Chain[T]) Then(onSuccess func(ctx context.Context, t T) rop.Result[T]) Chain[T] {
	if c.res.IsFailure() {
		return Chain[T]{ctx: c.ctx, res: rop.Fail[T](c.res.Err())}
	}
	return Chain[T]{ctx: c.ctx, res: onSuccess(c.ctx, c.res.Result())}
}

// ThenTry composes functions that return (U, error) — like repo calls
func (c Chain[T]) ThenTry(f func(ctx context.Context, t T) (T, error)) Chain[T] {
	if c.res.IsFailure() {
		return Chain[T]{ctx: c.ctx, res: rop.Fail[T](c.res.Err())}
	}
	u, err := f(c.ctx, c.res.Result())
	if err != nil {
		return Chain[T]{ctx: c.ctx, res: rop.Fail[T](err)}
	}
	return Chain[T]{ctx: c.ctx, res: rop.Success(u)}
}

// Map transforms the successful value to a new value
func (c Chain[T]) Map(onSuccess func(ctx context.Context, t T) T) Chain[T] {
	if c.res.IsFailure() {
		return Chain[T]{ctx: c.ctx, res: rop.Fail[T](c.res.Err())}
	}
	return Chain[T]{ctx: c.ctx, res: rop.Success(onSuccess(c.ctx, c.res.Result()))}
}

// To switch the successful value to a new result
func (c Chain[T]) To(onSuccess func(ctx context.Context, t T) rop.Result[T]) Chain[T] {
	if c.res.IsFailure() {
		return Chain[T]{ctx: c.ctx, res: rop.Fail[T](c.res.Err())}
	}
	return Chain[T]{ctx: c.ctx, res: onSuccess(c.ctx, c.res.Result())}
}

// Ensure triggers side effects for success/failure without changing the result
func (c Chain[T]) Ensure(onSuccess func(context.Context, T), onFailure func(context.Context, error)) Chain[T] {
	if c.res.IsFailure() {
		if onFailure != nil {
			onFailure(c.ctx, c.res.Err())
		}
		return c
	}
	if onSuccess != nil {
		onSuccess(c.ctx, c.res.Result())
	}
	return c
}

// Finally collapses the chain to a final value, delegating to solo.Finally
func (c Chain[T]) Finally(
	onSuccess func(context.Context, T) T,
	onFailure func(context.Context, error) T,
	onCancel func(context.Context, error) T,
) T {
	return solo.Finally(c.ctx, c.res, onSuccess, onFailure, onCancel)
}
