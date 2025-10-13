package custom

import (
	"context"
	"errors"
	"rop2/pkg/rop"
	"rop2/pkg/rop/core"
)

var ErrCancelled = errors.New("operation cancelled")

func CancelRemainingResults[In, Out any](ctx context.Context,
	inputCh <-chan rop.Result[In], outCh chan<- rop.Result[Out]) {

	required := core.IsProcessRemainingEnabled(ctx, true)

	if required {
		for in := range inputCh {

			if in.IsCancel() {
				outCh <- rop.CancelFrom[In, Out](in)
			} else {
				outCh <- rop.Cancel[Out](ErrCancelled)
			}
		}
	}
}

func CancelRemainingResult[In, Out any](ctx context.Context, in rop.Result[In],
	outCh chan<- rop.Result[Out]) {

	required := core.IsProcessRemainingEnabled(ctx, true)

	if required {

		if in.IsCancel() {
			outCh <- rop.CancelFrom[In, Out](in)
		} else {
			outCh <- rop.Cancel[Out](ErrCancelled)
		}
	}
}

func CancelRemainingValue[In, Out any](ctx context.Context, in rop.Result[In],
	brokenF func(ctx context.Context, in rop.Result[In]) Out, outCh chan<- Out) {

	required := core.IsProcessRemainingEnabled(ctx, true)

	if required {
		//fmt.Println("1 --------->")

		outCh <- brokenF(ctx, in)
	}
}

func CancelResult[T any](ctx context.Context, out T, outCh chan<- T) {
	required := core.IsProcessRemainingEnabled(ctx, true)

	if required {
		outCh <- out
	}
}

func CancelResults[T any](ctx context.Context, inputCh <-chan T, outCh chan<- T) {
	required := core.IsProcessRemainingEnabled(ctx, true)

	if required {
		for in := range inputCh {
			outCh <- in
		}
	}
}

func CancelRemainingValues[In, Out any](ctx context.Context, inputCh <-chan rop.Result[In],
	brokenF func(ctx context.Context, in rop.Result[In]) Out, outCh chan<- Out) {

	required := core.IsProcessRemainingEnabled(ctx, true)

	if required {
		for in := range inputCh {
			outCh <- brokenF(ctx, in)
		}
	}
}
