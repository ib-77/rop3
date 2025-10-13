package mass

import (
	"context"
	"rop2/pkg/rop"
	"rop2/pkg/rop/solo"
)

func Validating[T any](ctx context.Context, input rop.Result[T],
	validate func(ctx context.Context, in T) (valid bool, errMsg string),
	onCancel func(ctx context.Context, in rop.Result[T])) <-chan rop.Result[T] {

	ch := make(chan rop.Result[T])
	out := make(chan rop.Result[T])

	go func() {
		defer close(ch)

		if ctx.Err() == nil {

			if !input.HasResult() {
				panic("no results!")
			}
			ch <- solo.Validate[T](ctx, input.Result(), validate)
		}

	}()

	go func() {
		defer close(out)

		select {
		case pr, ok := <-ch:
			if ok {
				out <- pr
			} else {
				if onCancel != nil {
					onCancel(ctx, input)
				}
			}
		case <-ctx.Done():
			if onCancel != nil {
				onCancel(ctx, input)
			}
		}
	}()

	return out
}

func Switching[In, Out any](ctx context.Context, input rop.Result[In],
	switchOnSuccess func(ctx context.Context, r In) rop.Result[Out],
	onCancel func(ctx context.Context, in rop.Result[In])) <-chan rop.Result[Out] {

	ch := make(chan rop.Result[Out])
	out := make(chan rop.Result[Out])

	go func() {
		defer close(ch)

		if ctx.Err() == nil {
			ch <- solo.Switch[In, Out](ctx, input, switchOnSuccess)
		}

	}()

	go func() {
		defer close(out)

		select {
		case pr, ok := <-ch:
			if ok {
				out <- pr
			} else {
				if onCancel != nil {
					onCancel(ctx, input)
				}
			}
		case <-ctx.Done():
			if onCancel != nil {
				onCancel(ctx, input)
			}
		}
	}()

	return out
}

func Mapping[In, Out any](ctx context.Context, input rop.Result[In],
	mapOnSuccess func(ctx context.Context, r In) Out,
	onCancel func(ctx context.Context, in rop.Result[In])) <-chan rop.Result[Out] {

	ch := make(chan rop.Result[Out])
	out := make(chan rop.Result[Out])

	go func() {
		defer close(ch)

		if ctx.Err() == nil {
			ch <- solo.Map[In, Out](ctx, input, mapOnSuccess)
		}

	}()

	go func() {
		defer close(out)

		select {
		case pr, ok := <-ch:
			if ok {
				out <- pr
			} else {
				if onCancel != nil {
					onCancel(ctx, input)
				}
			}
		case <-ctx.Done():
			if onCancel != nil {
				onCancel(ctx, input)
			}
		}
	}()

	return out
}

func DoubleMapping[In, Out any](ctx context.Context, input rop.Result[In],
	mapOnSuccess func(ctx context.Context, r In) Out,
	mapOnError func(ctx context.Context, err error) Out,
	mapOnCancel func(ctx context.Context, err error) Out,
	onCancel func(ctx context.Context, in rop.Result[In])) <-chan rop.Result[Out] {

	ch := make(chan rop.Result[Out])
	out := make(chan rop.Result[Out])

	go func() {
		defer close(ch)

		if ctx.Err() == nil {
			ch <- solo.DoubleMap[In, Out](ctx, input, mapOnSuccess, mapOnError, mapOnCancel)
		}

	}()

	go func() {
		defer close(out)

		select {
		case pr, ok := <-ch:
			if ok {
				out <- pr
			} else {
				if onCancel != nil {
					onCancel(ctx, input)
				}
			}
		case <-ctx.Done():
			if onCancel != nil {
				onCancel(ctx, input)
			}
		}
	}()

	return out
}

func Teeing[T any](ctx context.Context, input rop.Result[T],
	sideEffect func(ctx context.Context, r rop.Result[T]),
	onCancel func(ctx context.Context, in rop.Result[T])) <-chan rop.Result[T] {

	ch := make(chan rop.Result[T])
	out := make(chan rop.Result[T])

	go func() {
		defer close(ch)

		if ctx.Err() == nil {
			ch <- solo.Tee[T](ctx, input, sideEffect)
		}

	}()

	go func() {
		defer close(out)

		select {
		case pr, ok := <-ch:
			if ok {
				out <- pr
			} else {
				if onCancel != nil {
					onCancel(ctx, input)
				}
			}
		case <-ctx.Done():
			if onCancel != nil {
				onCancel(ctx, input)
			}
		}
	}()

	return out
}

func DoubleTeeing[T any](ctx context.Context, input rop.Result[T],
	sideEffect func(ctx context.Context, r T),
	sideEffectOnError func(ctx context.Context, err error),
	sideEffectOnCancel func(ctx context.Context, err error),
	onCancel func(ctx context.Context, in rop.Result[T])) <-chan rop.Result[T] {

	ch := make(chan rop.Result[T])
	out := make(chan rop.Result[T])

	go func() {
		defer close(ch)

		if ctx.Err() == nil {
			ch <- solo.DoubleTee[T](ctx, input, sideEffect, sideEffectOnError, sideEffectOnCancel)
		}

	}()

	go func() {
		defer close(out)

		select {
		case pr, ok := <-ch:
			if ok {
				out <- pr
			} else {
				if onCancel != nil {
					onCancel(ctx, input)
				}
			}
		case <-ctx.Done():
			if onCancel != nil {
				onCancel(ctx, input)
			}
		}
	}()

	return out
}

func Trying[In, Out any](ctx context.Context, input rop.Result[In],
	onTryExecute func(ctx context.Context, r In) (Out, error),
	onCancel func(ctx context.Context, in rop.Result[In])) <-chan rop.Result[Out] {

	ch := make(chan rop.Result[Out])
	out := make(chan rop.Result[Out])

	go func() {
		defer close(ch)

		if ctx.Err() == nil {
			ch <- solo.Try[In, Out](ctx, input, onTryExecute)
		}

	}()

	go func() {
		defer close(out)

		select {
		case pr, ok := <-ch:
			if ok {
				out <- pr
			} else {
				if onCancel != nil {
					onCancel(ctx, input)
				}
			}
		case <-ctx.Done():
			if onCancel != nil {
				onCancel(ctx, input)
			}
		}
	}()

	return out
}

type FinallyHandlers[In, Out any] struct {
	OnSuccess func(ctx context.Context, r In) Out
	OnError   func(ctx context.Context, err error) Out
	OnCancel  func(ctx context.Context, err error) Out
}

type FinallyCancelHandlers[In, Out any] struct {
	OnBreak       func(ctx context.Context, in rop.Result[In]) Out
	OnCancelValue func(ctx context.Context, in rop.Result[In],
		brokenF func(ctx context.Context, in rop.Result[In]) Out, outCh chan<- Out)
	OnCancelValues func(ctx context.Context, inputCh <-chan rop.Result[In],
		brokenF func(ctx context.Context, in rop.Result[In]) Out, outCh chan<- Out)
	OnCancelResult  func(ctx context.Context, out Out, outCh chan<- Out)
	OnCancelResults func(ctx context.Context, inputCh <-chan Out, outCh chan<- Out)
}

func Finalizing[In, Out any](ctx context.Context, inputCh <-chan rop.Result[In],
	handlers FinallyHandlers[In, Out],
	cancelHandlers FinallyCancelHandlers[In, Out],
	onSuccessResult func(ctx context.Context, out Out)) <-chan Out {

	ch := make(chan Out)
	out := make(chan Out)

	go func() {
		defer close(ch)

		if ctx.Err() != nil {
			if cancelHandlers.OnCancelValues != nil {
				cancelHandlers.OnCancelValues(ctx, inputCh, cancelHandlers.OnBreak, ch)
			}
			return
		}

		for {
			select {
			case <-ctx.Done():
				if cancelHandlers.OnCancelValues != nil {
					cancelHandlers.OnCancelValues(ctx, inputCh, cancelHandlers.OnBreak, ch)
				}
				return
			case in, ok := <-inputCh:
				if !ok {
					return
				}

				res := solo.Finally[In, Out](ctx, in, handlers.OnSuccess, handlers.OnError, handlers.OnCancel)
				if ctx.Err() != nil {
					if cancelHandlers.OnCancelValue != nil {
						cancelHandlers.OnCancelValue(ctx, in, cancelHandlers.OnBreak, ch)
					}
					if cancelHandlers.OnCancelValues != nil {
						cancelHandlers.OnCancelValues(ctx, inputCh, cancelHandlers.OnBreak, ch)
					}
					return
				}

				select {
				case <-ctx.Done():
					if cancelHandlers.OnCancelValue != nil {
						cancelHandlers.OnCancelValue(ctx, in, cancelHandlers.OnBreak, ch)
					}
					if cancelHandlers.OnCancelValues != nil {
						cancelHandlers.OnCancelValues(ctx, inputCh, cancelHandlers.OnBreak, ch)
					}
					return
				case ch <- res:
				}
			}
		}
	}()

	go func() {
		defer close(out)

		for {
			select {
			case <-ctx.Done():
				if cancelHandlers.OnCancelResults != nil {
					cancelHandlers.OnCancelResults(ctx, ch, out)
				}
				return
			case finalized, ok := <-ch:
				if !ok {
					return
				}

				select {
				case <-ctx.Done():
					if cancelHandlers.OnCancelResult != nil {
						cancelHandlers.OnCancelResult(ctx, finalized, out)
					}
					return
				case out <- finalized:
					if onSuccessResult != nil {
						onSuccessResult(ctx, finalized)
					}
				}
			}
		}
	}()

	return out
}
