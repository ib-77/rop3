package custom

import (
	"context"
	"sync"

	"github.com/ib-77/rop3/pkg/rop"
	"github.com/ib-77/rop3/pkg/rop/core"
	"github.com/ib-77/rop3/pkg/rop/mass"
)

func Run[T any](ctx context.Context, inputCh <-chan rop.Result[T],
	engine func(ctx context.Context, input rop.Result[T]) <-chan rop.Result[T],
	handlers core.CancellationHandlers[T, T],
	onSuccess func(ctx context.Context, in rop.Result[T]), lines int) <-chan rop.Result[T] {

	out := make(chan rop.Result[T])
	wg := &sync.WaitGroup{}

	for range lines {
		wg.Add(1)
		go core.Locomotive(ctx, inputCh, out, engine, handlers, onSuccess, wg)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

func Turnout[In, Out any](ctx context.Context, inputCh <-chan rop.Result[In],
	engine func(ctx context.Context, input rop.Result[In]) <-chan rop.Result[Out],
	handlers core.CancellationHandlers[In, Out],
	onSuccess func(ctx context.Context, in rop.Result[Out]), lines int) <-chan rop.Result[Out] {

	out := make(chan rop.Result[Out])
	wg := &sync.WaitGroup{}

	for range lines {
		wg.Add(1)
		go core.Locomotive(ctx, inputCh, out, engine, handlers, onSuccess, wg)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

func RunSingle[T any](ctx context.Context, inputCh <-chan rop.Result[T],
	engine func(ctx context.Context, input rop.Result[T]) <-chan rop.Result[T],
	handlers core.CancellationHandlers[T, T],
	onSuccess func(ctx context.Context, in rop.Result[T])) <-chan rop.Result[T] {
	return Run[T](ctx, inputCh, engine, handlers, onSuccess, 1)
}

func Validate[T any](validate func(ctx context.Context, in T) (valid bool, errorMessage string),
	onCancel func(ctx context.Context, in rop.Result[T])) func(ctx context.Context,
	input rop.Result[T]) <-chan rop.Result[T] {
	return func(ctx context.Context, input rop.Result[T]) <-chan rop.Result[T] {
		return mass.Validating(ctx, input, validate, onCancel)
	}
}

func Switch[In, Out any](switchOnSuccess func(ctx context.Context, r In) rop.Result[Out],
	onCancel func(ctx context.Context, in rop.Result[In])) func(ctx context.Context,
	input rop.Result[In]) <-chan rop.Result[Out] {
	return func(ctx context.Context, input rop.Result[In]) <-chan rop.Result[Out] {
		return mass.Switching(ctx, input, switchOnSuccess, onCancel)
	}
}

func Map[In, Out any](mapOnSuccess func(ctx context.Context, r In) Out,
	onCancel func(ctx context.Context, in rop.Result[In])) func(ctx context.Context,
	input rop.Result[In]) <-chan rop.Result[Out] {
	return func(ctx context.Context, input rop.Result[In]) <-chan rop.Result[Out] {
		return mass.Mapping(ctx, input, mapOnSuccess, onCancel)
	}
}

func DoubleMap[In, Out any](
	mapOnSuccess func(ctx context.Context, r In) Out,
	mapOnError func(ctx context.Context, err error) Out,
	mapOnCancel func(ctx context.Context, err error) Out,
	onCancel func(ctx context.Context, in rop.Result[In])) func(ctx context.Context,
	input rop.Result[In]) <-chan rop.Result[Out] {
	return func(ctx context.Context, input rop.Result[In]) <-chan rop.Result[Out] {
		return mass.DoubleMapping(ctx, input, mapOnSuccess, mapOnError, mapOnCancel, onCancel)
	}
}

func Tee[T any](sideEffect func(ctx context.Context, r rop.Result[T]),
	onCancel func(ctx context.Context, in rop.Result[T])) func(ctx context.Context,
	input rop.Result[T]) <-chan rop.Result[T] {
	return func(ctx context.Context, input rop.Result[T]) <-chan rop.Result[T] {
		return mass.Teeing(ctx, input, sideEffect, onCancel)
	}
}

func DoubleTee[T any](sideEffect func(ctx context.Context, r T),
	sideEffectOnError func(ctx context.Context, err error),
	sideEffectOnCancel func(ctx context.Context, err error),
	onCancel func(ctx context.Context, in rop.Result[T])) func(ctx context.Context,
	input rop.Result[T]) <-chan rop.Result[T] {
	return func(ctx context.Context, input rop.Result[T]) <-chan rop.Result[T] {
		return mass.DoubleTeeing(ctx, input, sideEffect, sideEffectOnError, sideEffectOnCancel, onCancel)
	}
}

func Try[In, Out any](
	onTryExecute func(ctx context.Context, r In) (Out, error),
	onCancel func(ctx context.Context, in rop.Result[In])) func(ctx context.Context,
	input rop.Result[In]) <-chan rop.Result[Out] {
	return func(ctx context.Context, input rop.Result[In]) <-chan rop.Result[Out] {
		return mass.Trying(ctx, input, onTryExecute, onCancel)
	}
}

func Finally[In, Out any](ctx context.Context, input <-chan rop.Result[In],
	handlers mass.FinallyHandlers[In, Out],
	cancelHandlers mass.FinallyCancelHandlers[In, Out],
	onSuccessResult func(ctx context.Context, out Out)) <-chan Out {
	return mass.Finalizing(ctx, input, handlers, cancelHandlers, onSuccessResult)
}
