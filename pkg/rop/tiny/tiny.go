package tiny

import (
	"context"
	"errors"

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
	if c.res.IsFailure() || c.res.IsProcessed() {
		return c
	}
	return Chain[T]{ctx: c.ctx, res: onSuccess(c.ctx, c.res.Result())}
}

func (c Chain[T]) RepeatUntil(onSuccess func(ctx context.Context, t T) rop.Result[T],
	until func(ctx context.Context, t T) bool) Chain[T] {

	if c.res.IsFailure() || c.res.IsProcessed() {
		return c
	}

	for {
		c = c.Then(onSuccess)

		if c.res.IsFailure() || c.res.IsProcessed() || !until(c.ctx, c.res.Result()) {
			return c
		}
	}
}

func (c Chain[T]) RepeatChainUntil(inC func(ctx context.Context, t T) Chain[T],
	until func(ctx context.Context, t T) bool) Chain[T] {

	if c.res.IsFailure() || c.res.IsProcessed() {
		return c
	}

	for {
		c = inC(c.ctx, c.res.Result())

		if c.res.IsFailure() || c.res.IsProcessed() || !until(c.ctx, c.res.Result()) {
			return c
		}
	}
}

func (c Chain[T]) While(onSuccess func(ctx context.Context, t T) rop.Result[T],
	while func(ctx context.Context, t T) bool) Chain[T] {

	for !c.res.IsFailure() && !c.res.IsProcessed() && while(c.ctx, c.res.Result()) {
		c = c.Then(onSuccess)
	}
	return c
}

func (c Chain[T]) WhileChain(inC func(ctx context.Context, t T) Chain[T], while func(ctx context.Context, t T) bool) Chain[T] {

	for !c.res.IsFailure() && !c.res.IsProcessed() && while(c.ctx, c.res.Result()) {
		c = inC(c.ctx, c.res.Result())
	}
	return c
}

func (c Chain[T]) Or(alternative Chain[T]) Chain[T] {
	return c.or(alternative)
}

func (c Chain[T]) or(chains ...Chain[T]) Chain[T] {
	candidates := make([]Chain[T], 0, len(chains)+1)
	candidates = append(candidates, c)
	candidates = append(candidates, chains...)

	hasCancel := false
	hasFail := false
	var cancelRes rop.Result[T]
	var failRes rop.Result[T]
	var cancelCtx, failCtx context.Context

	for _, ch := range candidates {
		res := ch.res

		if res.IsSuccess() {
			return Chain[T]{ctx: ch.ctx, res: res}
		}

		if res.IsCancel() {
			if !hasCancel {
				hasCancel = true
				cancelRes = res
				cancelCtx = ch.ctx
			}
		} else if res.IsFailure() {
			if !hasFail {
				hasFail = true
				failRes = res
				failCtx = ch.ctx
			}
		}
	}

	if hasCancel {
		return Chain[T]{ctx: cancelCtx, res: cancelRes}
	}
	if hasFail {
		return Chain[T]{ctx: failCtx, res: failRes}
	}

	return c
}

func (c Chain[T]) And(required Chain[T]) Chain[T] {
	return c.and(required)
}

func (c Chain[T]) and(chains ...Chain[T]) Chain[T] {
	candidates := make([]Chain[T], 0, len(chains)+1)
	candidates = append(candidates, c)
	candidates = append(candidates, chains...)

	var res rop.Result[T]
	for _, ch := range candidates {
		res = ch.res

		if res.IsFailure() {
			return Chain[T]{ctx: ch.ctx, res: res}
		}
	}

	return Chain[T]{ctx: c.ctx, res: res} // what context to return?
}

// ThenTry composes functions that return (U, error) — like repo calls
func (c Chain[T]) ThenTry(try func(ctx context.Context, t T) (T, error)) Chain[T] {
	if c.res.IsFailure() || c.res.IsProcessed() {
		return c
	}
	// try
	u, err := try(c.ctx, c.res.Result())
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return Chain[T]{ctx: c.ctx, res: rop.Cancel[T](err)}
		}
		return Chain[T]{ctx: c.ctx, res: rop.Fail[T](err)}
	}
	return Chain[T]{ctx: c.ctx, res: rop.Success(u)}
}

// Map transforms the successful value to a new value
func (c Chain[T]) Map(onSuccess func(ctx context.Context, t T) T) Chain[T] {
	if c.res.IsFailure() || c.res.IsProcessed() {
		return c
	}

	return Chain[T]{ctx: c.ctx, res: rop.Success(onSuccess(c.ctx, c.res.Result()))}
}

// Ensure triggers side effects for success/failure without changing the result
func (c Chain[T]) Ensure(onSuccess func(context.Context, T), onFailure func(context.Context, error),
	onProcessed func(context.Context, T)) Chain[T] {

	if c.res.IsFailure() {
		if onFailure != nil {
			onFailure(c.ctx, c.res.Err())
		}
		return c
	}

	if c.res.IsProcessed() {
		if onProcessed != nil {
			onProcessed(c.ctx, c.res.Result())
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
