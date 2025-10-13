package core

import (
	"context"
	"github.com/ib-77/rop3/pkg/rop"
	"sync"
)

type CancellationHandlers[In, Out any] struct {
	OnCancel            func(ctx context.Context, inputCh <-chan rop.Result[In], outCh chan<- rop.Result[Out])
	OnCancelUnprocessed func(ctx context.Context, unprocessed rop.Result[In], outCh chan<- rop.Result[Out])
	OnCancelProcessed   func(ctx context.Context, in rop.Result[In], processed rop.Result[Out], outCh chan<- rop.Result[Out])
}

func Locomotive[In, Out any](ctx context.Context, inputCh <-chan rop.Result[In], outCh chan<- rop.Result[Out],
	engine func(ctx context.Context, input rop.Result[In]) <-chan rop.Result[Out],
	handlers CancellationHandlers[In, Out],
	onSuccess func(ctx context.Context, in rop.Result[Out]), wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			if handlers.OnCancel != nil {
				handlers.OnCancel(ctx, inputCh, outCh)
			}
			return
		case in, ok := <-inputCh:
			if !ok {
				return
			}

			select {
			case <-ctx.Done():
				if handlers.OnCancelUnprocessed != nil {
					handlers.OnCancelUnprocessed(ctx, in, outCh)
				}
				if handlers.OnCancel != nil {
					handlers.OnCancel(ctx, inputCh, outCh)
				}
				return
			case pr, running := <-engine(ctx, in):
				if !running {
					return
				}

				select {
				case <-ctx.Done():
					//outCh <- pr // onCancelProcessed possible duplicate!
					if handlers.OnCancelProcessed != nil {
						handlers.OnCancelProcessed(ctx, in, pr, outCh)
					}
					if handlers.OnCancel != nil {
						handlers.OnCancel(ctx, inputCh, outCh)
					}
					return
				case outCh <- pr:
					if onSuccess != nil {
						onSuccess(ctx, pr)
					}
				}
			}
		}
	}
}
