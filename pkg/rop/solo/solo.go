package solo

import (
	"context"
	"errors"

	"github.com/ib-77/rop3/pkg/rop"
)

func Succeed[T any](input T) rop.Result[T] {
	return rop.Success(input)
}

func Fail[T any](err error) rop.Result[T] {
	return rop.Fail[T](err)
}

func Cancel[T any](err error) rop.Result[T] {
	return rop.Cancel[T](err)
}

func Validate[T any](ctx context.Context, input T,
	validate func(ctx context.Context, in T) (isValid bool, errMsg string)) rop.Result[T] {
	return AndValidate(ctx, Succeed(input), validate)
}

func AndValidate[T any](ctx context.Context, input rop.Result[T],
	validate func(ctx context.Context, in T) (valid bool, errMsg string)) rop.Result[T] {

	if input.IsSuccess() {

		if isValid, errMsg := validate(ctx, input.Result()); isValid {
			return rop.Success(input.Result())
		} else {
			return rop.Fail[T](errors.New(errMsg))
		}
	}
	return input
}

func ValidateMany[T any](ctx context.Context, input T,
	validateF func(ctx context.Context, in T) (valid bool, errMsg string)) rop.Result[T] {

	if valid, errMsg := validateF(ctx, input); valid {
		return rop.Success(input)
	} else {
		return rop.Fail[T](errors.New(errMsg))
	}
}

func Switch[In any, Out any](ctx context.Context,
	input rop.Result[In],
	onSuccess func(ctx context.Context, r In) rop.Result[Out]) rop.Result[Out] {

	if input.IsSuccess() {
		return onSuccess(ctx, input.Result())
	} else {
		if input.IsCancel() {
			return rop.Cancel[Out](input.Err())
		} else {
			return rop.Fail[Out](input.Err())
		}
	}
}

func Map[In any, Out any](ctx context.Context,
	input rop.Result[In],
	onSuccess func(ctx context.Context, r In) Out) rop.Result[Out] {

	if input.IsSuccess() {
		return rop.Success(onSuccess(ctx, input.Result()))
	} else {
		if input.IsCancel() {
			return rop.Cancel[Out](input.Err())
		} else {
			return rop.Fail[Out](input.Err())
		}
	}
}

func Tee[T any](ctx context.Context,
	input rop.Result[T],
	onSuccess func(ctx context.Context, r rop.Result[T])) rop.Result[T] {

	if input.IsSuccess() {
		onSuccess(ctx, input)
	}

	return input
}

func TeeIf[T any](ctx context.Context,
	input rop.Result[T],
	condition func(ctx context.Context, r rop.Result[T]) bool,
	onSuccessAndCondition func(ctx context.Context, r rop.Result[T])) rop.Result[T] {

	if input.IsSuccess() {
		if condition(ctx, input) {
			onSuccessAndCondition(ctx, input)
		}
	}

	return input
}

func DoubleTee[T any](ctx context.Context, input rop.Result[T],
	onSuccess func(ctx context.Context, r T),
	onError func(ctx context.Context, err error),
	onCancel func(ctx context.Context, err error)) rop.Result[T] {

	if input.IsSuccess() {
		onSuccess(ctx, input.Result())
	} else {
		if input.IsCancel() {
			onCancel(ctx, input.Err())
		} else {
			onError(ctx, input.Err())
		}
	}

	return input
}

func DoubleMap[In any, Out any](ctx context.Context, input rop.Result[In],
	onSuccess func(ctx context.Context, r In) Out,
	onError func(ctx context.Context, err error) Out,
	onCancel func(ctx context.Context, err error) Out) rop.Result[Out] {

	if input.IsSuccess() {
		return rop.Success(onSuccess(ctx, input.Result()))
	}

	if input.IsCancel() {
		onCancel(ctx, input.Err())
	} else {
		onError(ctx, input.Err())
	}

	if input.IsCancel() {
		return rop.Cancel[Out](input.Err())
	} else {
		return rop.Fail[Out](input.Err())
	}
}

func Try[In any, Out any](ctx context.Context, input rop.Result[In],
	onTryExecute func(ctx context.Context, r In) (Out, error)) rop.Result[Out] {

	if input.IsSuccess() {

		out, err := onTryExecute(ctx, input.Result())
		if err != nil {
			return rop.Fail[Out](err)
		}

		return rop.Success(out)
	}

	if input.IsCancel() {
		return rop.Cancel[Out](input.Err())
	} else {
		return rop.Fail[Out](input.Err())
	}
}

func FailOnError[T any](ctx context.Context, input rop.Result[T],
	maybeErr func(ctx context.Context, in T) error) rop.Result[T] {
	if input.IsSuccess() {
		err := maybeErr(ctx, input.Result())
		if err != nil {
			return rop.Fail[T](err)
		} else {
			return input
		}
	}
	return input
}

func Finally[In, Out any](ctx context.Context, input rop.Result[In],
	onSuccess func(ctx context.Context, r In) Out,
	onError func(ctx context.Context, err error) Out,
	onCancel func(ctx context.Context, err error) Out) Out {

	if input.IsSuccess() {
		return onSuccess(ctx, input.Result())
	} else if input.IsCancel() {
		return onCancel(ctx, input.Err())
	} else {
		return onError(ctx, input.Err())
	}
}
