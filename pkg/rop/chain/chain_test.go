package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/ib-77/rop3/pkg/rop"
)

// helper validators for int values that ignore prior result and validate captured value
func validateNonNegative(v int) func(ctx context.Context, in rop.Result[int]) rop.Result[int] {
	return func(ctx context.Context, in rop.Result[int]) rop.Result[int] {
		if v < 0 {
			return rop.Fail[int](errors.New("negative"))
		}
		return rop.Success(v)
	}
}

func validateEven(v int) func(ctx context.Context, in rop.Result[int]) rop.Result[int] {
	return func(ctx context.Context, in rop.Result[int]) rop.Result[int] {
		if v%2 != 0 {
			return rop.Fail[int](errors.New("odd"))
		}
		return rop.Success(v)
	}
}

func passThrough[T any]() func(ctx context.Context, in rop.Result[T]) rop.Result[T] {
	return func(ctx context.Context, in rop.Result[T]) rop.Result[T] { return in }
}

func TestValidateAll_AllSuccess(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	v := 10 // non-negative, even
	input := rop.Success(v)

	res := ValidateAll[int](ctx, input, true, validateNonNegative(v), validateEven(v))

	if !res.IsSuccess() {
		t.Fatalf("expected success, got error: %v", res.Err())
	}
	if res.Result() != v {
		t.Fatalf("expected result %d, got %d", v, res.Result())
	}
}

func TestValidateAll_FailBreakOnFirst(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	v := -1 // fails non-negative and odd
	input := rop.Success(v)

	executed := 0
	v1 := func(ctx context.Context, in rop.Result[int]) rop.Result[int] {
		executed++
		return validateNonNegative(v)(ctx, in)
	}

	v2 := func(ctx context.Context, in rop.Result[int]) rop.Result[int] {
		executed++
		return validateEven(v)(ctx, in)
	}

	res := ValidateAll[int](ctx, input, true, v1, v2)

	if res.IsSuccess() {
		t.Fatalf("expected failure, got success: %v", res.Result())
	}
	if executed != 1 {
		t.Fatalf("expected only first validator to execute, got %d", executed)
	}

	// errors.Join(single) should equal the original error
	if res.Err() == nil || res.Err().Error() != "negative" {
		t.Fatalf("expected 'negative' error, got: %v", res.Err())
	}
}

func TestValidateAll_AccumulateErrors_NoBreak(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	v := -3 // negative and odd
	input := rop.Success(v)

	res := ValidateAll[int](ctx, input, false, validateNonNegative(v), validateNonNegative(v), validateEven(v))

	if res.IsSuccess() {
		t.Fatalf("expected failure, got success: %v", res.Result())
	}

	errs := rop.GetErrors(res.Err())
	if len(errs) != 3 {
		t.Fatalf("expected 3 accumulated errors, got %d", len(errs))
	}

	// check messages; order should follow validator sequence
	if errs[0].Error() != "negative" || errs[1].Error() != "negative" || errs[2].Error() != "odd" {
		t.Fatalf("expected errors ['negative', 'negative', 'odd'], got ['%s','%s','%s']",
			errs[0].Error(), errs[1].Error(), errs[2].Error())
	}
}

func TestValidateAll_InitialInputFail(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	initialErr := errors.New("initial")
	input := rop.Fail[int](initialErr)

	res := ValidateAll[int](ctx, input, true, passThrough[int]())

	if res.IsSuccess() {
		t.Fatalf("expected failure, got success")
	}
	if res.Err() == nil || res.Err().Error() != "initial" {
		t.Fatalf("expected initial error to pass through, got: %v", res.Err())
	}
}

func TestValidateAll_ContextCancelled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before running

	input := rop.Success(42)
	res := ValidateAll[int](ctx, input, false, validateNonNegative(42), validateEven(42))

	// When context is canceled, Join should short-circuit and return input unchanged
	if !res.IsSuccess() {
		t.Fatalf("expected success, got error: %v", res.Err())
	}
	if res.Result() != 42 {
		t.Fatalf("expected original value 42, got %d", res.Result())
	}
}

func TestValidateAll_NoValidators(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	input := rop.Success(7)

	res := ValidateAll[int](ctx, input, false /* no validators */)

	if !res.IsSuccess() {
		t.Fatalf("expected success, got error: %v", res.Err())
	}
	if res.Result() != 7 {
		t.Fatalf("expected result 7, got %d", res.Result())
	}
}
