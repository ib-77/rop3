package core

import (
	"context"
	"fmt"
	"rop2/pkg/rop"
	"rop2/pkg/rop/solo"
	"sync"
)

type ToChanHandlers[T any] struct {
	OnStartFail func(ctx context.Context, input []T)
	OnSuccess   func(ctx context.Context, input T)
	OnBreak     func(ctx context.Context, rest []T)
}

func ToChanFromArgs[T any](ctx context.Context, values ...T) <-chan T {
	in := make(chan T)

	go func() {
		defer close(in)

		if ctx.Err() != nil {
			fmt.Println("in: ctx.err 1") // TODO remove!
			return
		}

		for _, v := range values {

			if ctx.Err() != nil {
				fmt.Println("in: ctx.err 2") // TODO remove!
				return
			}

			select {
			case in <- v:
				fmt.Println("in: ", v) // TODO remove!
			case <-ctx.Done():
				fmt.Println("in: done") // TODO remove!
				return
			}
		}
	}()

	return in
}

func ToChanFromArgsResults[T any](ctx context.Context, handlers ToChanHandlers[T], values ...T) <-chan rop.Result[T] {
	in := make(chan rop.Result[T])

	go func() {
		defer close(in)

		if ctx.Err() != nil {
			if handlers.OnStartFail != nil {
				handlers.OnStartFail(ctx, values)
			}
			return
		}

		for i, v := range values {
			select {
			case in <- solo.Succeed(v):
				if handlers.OnSuccess != nil {
					handlers.OnSuccess(ctx, v)
				}
			case <-ctx.Done():
				if handlers.OnBreak != nil {
					restCount := len(values) - i
					rest := make([]T, restCount)
					if restCount > 0 {
						rest = values[i:]
					}
					handlers.OnBreak(ctx, rest)
				}
				return
			}
		}
	}()

	return in
}

func ToChan[T any](ctx context.Context, value T) <-chan T {
	return ToChanFromArgs[T](ctx, value)
}

func FromChanFirstOrDefault[T any](ctx context.Context, out <-chan T, defaultV T) T {
	res := defaultV
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()

		select {
		case v, ok := <-out:
			if !ok {
				return
			}
			res = v
			return
		case <-ctx.Done():
			return
		}
	}()
	wg.Wait()
	return res
}

func ToChanMany[T any](ctx context.Context, values []T) <-chan T {
	return ToChanFromArgs[T](ctx, values...)
}

func ToChanManyResultsWithHandlers[T any](ctx context.Context, handlers ToChanHandlers[T], values []T) <-chan rop.Result[T] {
	return ToChanFromArgsResults[T](ctx, handlers, values...)
}

func ToChanManyResults[T any](ctx context.Context, values []T) <-chan rop.Result[T] {
	return ToChanFromArgsResults[T](ctx, ToChanHandlers[T]{}, values...)
}

func FromChanMany[T any](ctx context.Context, out <-chan T) []T {
	res := make([]T, 0)
	wg := &sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()
		for {
			select {
			case v, ok := <-out:
				if !ok {
					return
				}
				res = append(res, v)
			case <-ctx.Done():
				return
			}
		}
	}()

	wg.Wait()
	return res
}
