package lite

import (
	"context"
	"github.com/ib-77/rop3/pkg/rop"
	"github.com/ib-77/rop3/pkg/rop/core"
	"github.com/ib-77/rop3/pkg/rop/mass"
	"sync"
)

func Run[T any](ctx context.Context, inputCh <-chan rop.Result[T],
	engine func(ctx context.Context, input rop.Result[T]) <-chan rop.Result[T],
	lines int) <-chan rop.Result[T] {

	out := make(chan rop.Result[T])
	wg := &sync.WaitGroup{}

	for range lines {
		wg.Add(1)
		go core.Locomotive(ctx, inputCh, out, engine, core.CancellationHandlers[T, T]{}, nil, wg)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

func Turnout[In, Out any](ctx context.Context, inputCh <-chan rop.Result[In],
	engine func(ctx context.Context, input rop.Result[In]) <-chan rop.Result[Out],
	lines int) <-chan rop.Result[Out] {

	out := make(chan rop.Result[Out])
	wg := &sync.WaitGroup{}

	for range lines {
		wg.Add(1)
		go core.Locomotive(ctx, inputCh, out, engine, core.CancellationHandlers[In, Out]{}, nil, wg)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

func Validate[T any](validate func(ctx context.Context, in T) (valid bool, errMsg string)) func(ctx context.Context,
	input rop.Result[T]) <-chan rop.Result[T] {
	return func(ctx context.Context, input rop.Result[T]) <-chan rop.Result[T] {
		return mass.Validating(ctx, input, validate, nil)
	}
}

func Switch[In, Out any](switchOnSuccess func(ctx context.Context, r In) rop.Result[Out]) func(ctx context.Context,
	input rop.Result[In]) <-chan rop.Result[Out] {
	return func(ctx context.Context, input rop.Result[In]) <-chan rop.Result[Out] {
		return mass.Switching(ctx, input, switchOnSuccess, nil)
	}
}

func Map[In, Out any](mapOnSuccess func(ctx context.Context, r In) Out) func(ctx context.Context,
	input rop.Result[In]) <-chan rop.Result[Out] {
	return func(ctx context.Context, input rop.Result[In]) <-chan rop.Result[Out] {
		return mass.Mapping(ctx, input, mapOnSuccess, nil)
	}
}

func DoubleMap[In, Out any](
	mapOnSuccess func(ctx context.Context, r In) Out,
	mapOnError func(ctx context.Context, err error) Out,
	mapOnCancel func(ctx context.Context, err error) Out) func(ctx context.Context,
	input rop.Result[In]) <-chan rop.Result[Out] {
	return func(ctx context.Context, input rop.Result[In]) <-chan rop.Result[Out] {
		return mass.DoubleMapping(ctx, input, mapOnSuccess, mapOnError, mapOnCancel, nil)
	}
}

func Tee[T any](sideEffect func(ctx context.Context, r rop.Result[T])) func(ctx context.Context,
	input rop.Result[T]) <-chan rop.Result[T] {
	return func(ctx context.Context, input rop.Result[T]) <-chan rop.Result[T] {
		return mass.Teeing(ctx, input, sideEffect, nil)
	}
}

func DoubleTee[T any](sideEffect func(ctx context.Context, r T),
	sideEffectOnError func(ctx context.Context, err error),
	sideEffectOnCancel func(ctx context.Context, err error)) func(ctx context.Context,
	input rop.Result[T]) <-chan rop.Result[T] {
	return func(ctx context.Context, input rop.Result[T]) <-chan rop.Result[T] {
		return mass.DoubleTeeing(ctx, input, sideEffect, sideEffectOnError, sideEffectOnCancel, nil)
	}
}

func Try[In, Out any](
	onTryExecute func(ctx context.Context, r In) (Out, error)) func(ctx context.Context,
	input rop.Result[In]) <-chan rop.Result[Out] {
	return func(ctx context.Context, input rop.Result[In]) <-chan rop.Result[Out] {
		return mass.Trying(ctx, input, onTryExecute, nil)
	}
}

func Finally[In, Out any](ctx context.Context, input <-chan rop.Result[In],
	handlers mass.FinallyHandlers[In, Out]) <-chan Out {
	return mass.Finalizing(ctx, input, handlers, mass.FinallyCancelHandlers[In, Out]{}, nil)
}
