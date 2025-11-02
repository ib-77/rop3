package c2

import (
	"context"
	"errors"
	"testing"

	"github.com/ib-77/rop3/pkg/rop"
)

func TestStart_Then_Success(t *testing.T) {
	ctx := context.Background()
	ch := Start[int, int](ctx, rop.Success(5)).
		Then(func(ctx context.Context, n int) rop.Result[int] { return rop.Success(n * 2) })

	res := ch.Result()
	if !res.IsSuccess() {
		t.Fatalf("expected success, got error: %v", res.Err())
	}
	if got := res.Result(); got != 10 {
		t.Fatalf("expected 10, got %d", got)
	}
	// Input remains the original value
	if in := ch.Input(); !in.IsSuccess() || in.Result() != 5 {
		t.Fatalf("expected input success=5, got: success=%v value=%v err=%v", in.IsSuccess(), in.Result(), in.Err())
	}
}

func TestStart_Then_Failure(t *testing.T) {
	ctx := context.Background()
	expectedErr := errors.New("boom")
	ch := Start[int, int](ctx, rop.Success(5)).
		Then(func(ctx context.Context, n int) rop.Result[int] { return rop.Fail[int](expectedErr) })

	res := ch.Result()
	if res.IsSuccess() {
		t.Fatalf("expected failure, got success: %v", res.Result())
	}
	if res.Err() == nil || res.Err().Error() != expectedErr.Error() {
		t.Fatalf("expected error %q, got %v", expectedErr, res.Err())
	}
}

func TestThenTry_SuccessAndError(t *testing.T) {
	ctx := context.Background()
	// Success path
	ch1 := Start[int, int](ctx, rop.Success(3)).
		ThenTry(func(ctx context.Context, n int) (int, error) { return n + 7, nil })
	res1 := ch1.Result()
	if !res1.IsSuccess() || res1.Result() != 10 {
		t.Fatalf("ThenTry success: expected 10, got success=%v value=%v err=%v", res1.IsSuccess(), res1.Result(), res1.Err())
	}

	// Error path
	expectedErr := errors.New("bad input")
	ch2 := Start[int, int](ctx, rop.Success(3)).
		ThenTry(func(ctx context.Context, n int) (int, error) { return 0, expectedErr })
	res2 := ch2.Result()
	if res2.IsSuccess() || res2.Err() == nil || res2.Err().Error() != expectedErr.Error() {
		t.Fatalf("ThenTry error: expected %q, got success=%v err=%v", expectedErr, res2.IsSuccess(), res2.Err())
	}
}

func TestMap_PassesFailureThrough(t *testing.T) {
	ctx := context.Background()
	expectedErr := errors.New("fail")
	ch := Start[int, int](ctx, rop.Fail[int](expectedErr)).
		Map(func(ctx context.Context, n int) int { return n * 100 })

	res := ch.Result()
	if res.IsSuccess() {
		t.Fatalf("expected failure to pass through, got success: %v", res.Result())
	}
	if res.Err() == nil || res.Err().Error() != expectedErr.Error() {
		t.Fatalf("expected error %q, got %v", expectedErr, res.Err())
	}
}

func TestEnsure_SideEffectOnlyOnSuccess(t *testing.T) {
	ctx := context.Background()
	called := 0

	// Success triggers side effect
	ch1 := Start[int, int](ctx, rop.Success(2)).
		Ensure(func(ctx context.Context, n int) { called++ })
	if !ch1.Result().IsSuccess() {
		t.Fatalf("expected success, got: %v", ch1.Result().Err())
	}
	if called != 1 {
		t.Fatalf("expected side effect to be called once, got %d", called)
	}

	// Failure does not trigger side effect
	ch2 := Start[int, int](ctx, rop.Fail[int](errors.New("x"))).
		Ensure(func(ctx context.Context, n int) { called++ })
	if ch2.Result().IsSuccess() {
		t.Fatalf("expected failure, got success")
	}
	if called != 1 {
		t.Fatalf("expected side effect count to remain 1, got %d", called)
	}
}

func TestFinally_UsesInputNotTransformedResult(t *testing.T) {
	ctx := context.Background()
	// Transform to a failure but Finally reduces based on input
	ch := Start[int, int](ctx, rop.Success(3)).
		ThenTry(func(ctx context.Context, n int) (int, error) { return 0, errors.New("ignored failure") })

	// Even though the chain's result is a failure, Finally consults the original input.
	out := ch.Finally(
		func(ctx context.Context, n int) int { return n + 100 },
		func(ctx context.Context, err error) int { return -1 },
		func(ctx context.Context, err error) int { return -2 },
	)

	if out != 103 {
		t.Fatalf("expected Finally to use input (103), got %d", out)
	}
}

func TestFromValue_BasicFlow(t *testing.T) {
	ctx := context.Background()
	ch := FromValue[int](ctx, 7).
		Then(func(ctx context.Context, n int) rop.Result[int] { return rop.Success(n + 1) })

	res := ch.Result()
	if !res.IsSuccess() || res.Result() != 8 {
		t.Fatalf("FromValue: expected 8, got success=%v value=%v err=%v", res.IsSuccess(), res.Result(), res.Err())
	}

	out := ch.Finally(
		func(ctx context.Context, n int) int { return n * 10 },
		func(ctx context.Context, err error) int { return -1 },
		func(ctx context.Context, err error) int { return -2 },
	)
	// Finally uses input value 7, not transformed result 8
	if out != 70 {
		t.Fatalf("FromValue Finally: expected 70 (7*10), got %d", out)
	}
}
