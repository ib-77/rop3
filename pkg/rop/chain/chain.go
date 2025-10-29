package chain

import (
	"context"
	"errors"

	"github.com/ib-77/rop3/pkg/rop"
)

func ValidateAll[T any](
	ctx context.Context,
	input rop.Result[T],
	breakOnError bool, // exit on first error
	inputsF ...func(ctx context.Context, in rop.Result[T]) rop.Result[T]) rop.Result[T] {

	var err error
	return Join(
		ctx,
		input,
		breakOnError,
		func(ctx context.Context, current rop.Result[T]) rop.Result[T] {

			if current.IsFailure() {
				e := rop.GetErrors(err)
				e = append(e, current.Err())
				err = errors.Join(e...)
			}

			if rop.IsNil(err) {
				return current
			}

			return rop.Fail[T](err)
		},
		inputsF...,
	)
}

func Join[T any](ctx context.Context,
	input rop.Result[T],
	breakOnError bool, // exit on first error
	concat func(ctx context.Context, current rop.Result[T]) rop.Result[T],
	inputsF ...func(ctx context.Context, in rop.Result[T]) rop.Result[T]) rop.Result[T] {

	if len(inputsF) == 0 || concat == nil || !rop.IsNil(ctx.Err()) {
		return input
	}

	finalResult := concat(ctx, inputsF[0](ctx, input))

	if !rop.IsNil(ctx.Err()) {
		return finalResult
	}

	if finalResult.IsSuccess() || !breakOnError {
		for _, in := range inputsF[1:] {
			if !rop.IsNil(ctx.Err()) {
				return finalResult
			}

			nextRes := concat(ctx, in(ctx, finalResult))
			if nextRes.IsFailure() && breakOnError {
				return nextRes
			} else {
				finalResult = nextRes
			}
		}
	}
	return finalResult
}
