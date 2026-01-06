package tiny

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/ib-77/rop3/pkg/rop"
)

func TestStartAndResult_Success(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	res := rop.Success(5)
	chain := Start(ctx, res)

	out := chain.Result()
	if !out.IsSuccess() || out.Result() != 5 {
		t.Fatalf("expected success with 5, got: success=%v, val=%v, err=%v", out.IsSuccess(), out.Result(), out.Err())
	}
}

func TestFromValue(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	chain := FromValue(ctx, 7)
	out := chain.Result()
	if !out.IsSuccess() || out.Result() != 7 {
		t.Fatalf("expected success with 7, got: success=%v, val=%v, err=%v", out.IsSuccess(), out.Result(), out.Err())
	}
}

func TestThen_ShortCircuitOnFailure(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	err := errors.New("boom")
	chain := Start(ctx, rop.Fail[int](err))

	called := false
	chain = chain.Then(func(ctx context.Context, t int) rop.Result[int] {
		called = true
		return rop.Success(t + 1)
	})

	out := chain.Result()
	if out.IsSuccess() || out.Err() == nil || out.Err().Error() != "boom" {
		t.Fatalf("expected failure 'boom', got: success=%v, err=%v", out.IsSuccess(), out.Err())
	}
	if called {
		t.Fatalf("onSuccess should not be called when initial result is failure")
	}
}

func TestThen_SuccessPath(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	chain := FromValue(ctx, 3).
		Then(func(ctx context.Context, t int) rop.Result[int] { return rop.Success(t * 2) })

	out := chain.Result()
	if !out.IsSuccess() || out.Result() != 6 {
		t.Fatalf("expected success with 6, got: success=%v, val=%v, err=%v", out.IsSuccess(), out.Result(), out.Err())
	}
}

func TestThenTry_ShortCircuitOnFailure(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	err := errors.New("bad")
	chain := Start(ctx, rop.Fail[int](err)).
		ThenTry(func(ctx context.Context, t int) (int, error) { return t + 1, nil })

	out := chain.Result()
	if out.IsSuccess() || out.Err() == nil || out.Err().Error() != "bad" {
		t.Fatalf("expected failure 'bad', got: success=%v, err=%v", out.IsSuccess(), out.Err())
	}
}

func TestThenTry_ErrorPropagation(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	chain := FromValue(ctx, 10).
		ThenTry(func(ctx context.Context, t int) (int, error) {
			return 0, errors.New("try-error")
		})

	out := chain.Result()
	if out.IsSuccess() || out.Err() == nil || out.Err().Error() != "try-error" {
		t.Fatalf("expected failure 'try-error', got: success=%v, err=%v", out.IsSuccess(), out.Err())
	}
}

func TestThenTry_Success(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	chain := FromValue(ctx, 4).
		ThenTry(func(ctx context.Context, t int) (int, error) { return t * t, nil })

	out := chain.Result()
	if !out.IsSuccess() || out.Result() != 16 {
		t.Fatalf("expected success with 16, got: success=%v, val=%v, err=%v", out.IsSuccess(), out.Result(), out.Err())
	}
}

func TestMap_ShortCircuitOnFailure(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	err := errors.New("oops")
	chain := Start(ctx, rop.Fail[int](err)).
		Map(func(ctx context.Context, t int) int { return t + 100 })

	out := chain.Result()
	if out.IsSuccess() || out.Err() == nil || out.Err().Error() != "oops" {
		t.Fatalf("expected failure 'oops', got: success=%v, err=%v", out.IsSuccess(), out.Err())
	}
}

func TestMap_Success(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	chain := FromValue(ctx, 5).
		Map(func(ctx context.Context, t int) int { return t + 3 })

	out := chain.Result()
	if !out.IsSuccess() || out.Result() != 8 {
		t.Fatalf("expected success with 8, got: success=%v, val=%v, err=%v", out.IsSuccess(), out.Result(), out.Err())
	}
}

func TestRepeatUntil_ShortCircuitOnFailure(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	err := errors.New("repeat-fail")
	chain := Start(ctx, rop.Fail[int](err))

	called := false
	chain = chain.RepeatUntil(
		func(ctx context.Context, t int) rop.Result[int] {
			called = true
			return rop.Success(t + 1)
		},
		func(ctx context.Context, t int) bool { return true },
	)

	out := chain.Result()
	if out.IsSuccess() || out.Err() == nil || out.Err().Error() != "repeat-fail" {
		t.Fatalf("expected failure 'repeat-fail', got: success=%v, err=%v", out.IsSuccess(), out.Err())
	}
	if called {
		t.Fatalf("onSuccess must not be called when starting RepeatUntil with failure")
	}
}

func TestRepeatUntil_ShortCircuitOnProcessed(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	base := rop.SetProcessed(rop.Success(1))
	chain := Start(ctx, base)

	called := false
	chain = chain.RepeatUntil(
		func(ctx context.Context, t int) rop.Result[int] {
			called = true
			return rop.Success(t + 1)
		},
		func(ctx context.Context, t int) bool { return true },
	)

	out := chain.Result()
	if !out.IsSuccess() || out.Result() != 1 || !out.IsProcessed() {
		t.Fatalf("expected unchanged processed success, got: success=%v, val=%v, processed=%v",
			out.IsSuccess(), out.Result(), out.IsProcessed())
	}
	if called {
		t.Fatalf("onSuccess must not be called when starting RepeatUntil with processed result")
	}
}

func TestRepeatUntil_ExecutesOnceWhenUntilFalse(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	chain := FromValue(ctx, 1)

	calls := 0
	chain = chain.RepeatUntil(
		func(ctx context.Context, t int) rop.Result[int] {
			calls++
			return rop.Success(t + 1)
		},
		func(ctx context.Context, t int) bool { return false },
	)

	out := chain.Result()
	if !out.IsSuccess() {
		t.Fatalf("expected success result, got: success=%v, err=%v", out.IsSuccess(), out.Err())
	}
	if calls != 1 {
		t.Fatalf("expected RepeatUntil body to execute once, got %d calls", calls)
	}
}

func TestRepeatUntil_ExecutesTenTimes(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	chain := FromValue(ctx, 1)

	calls := 0
	chain = chain.RepeatUntil(
		func(ctx context.Context, v int) rop.Result[int] {
			calls++
			if calls > 20 {
				t.Fatalf("infinity loop")
			}
			return rop.Success(v + 1)
		},
		func(ctx context.Context, v int) bool { return v <= 10 },
	)

	out := chain.Result()
	if !out.IsSuccess() {
		t.Fatalf("expected success result, got: success=%v, err=%v", out.IsSuccess(), out.Err())
	}
	if calls != 10 {
		t.Fatalf("expected RepeatUntil body to execute once, got %d calls", calls)
	}
}

func TestWhile_ExecutesTenTimes(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	chain := FromValue(ctx, 1)

	calls := 0
	chain = chain.While(
		func(ctx context.Context, v int) rop.Result[int] {
			calls++
			if calls > 20 {
				t.Fatalf("infinity loop")
			}
			return rop.Success(v + 1)
		},
		func(ctx context.Context, v int) bool { return v <= 10 },
	)

	out := chain.Result()
	if !out.IsSuccess() {
		t.Fatalf("expected success result, got: success=%v, err=%v", out.IsSuccess(), out.Err())
	}
	if calls != 10 {
		t.Fatalf("expected RepeatUntil body to execute once, got %d calls", calls)
	}
}

func TestWhile_ShortCircuitOnFailure(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	err := errors.New("while-fail")
	chain := Start(ctx, rop.Fail[int](err))

	called := false
	chain = chain.While(
		func(ctx context.Context, t int) rop.Result[int] {
			called = true
			return rop.Success(t + 1)
		},
		func(ctx context.Context, t int) bool { return true },
	)

	out := chain.Result()
	if out.IsSuccess() || out.Err() == nil || out.Err().Error() != "while-fail" {
		t.Fatalf("expected failure 'while-fail', got: success=%v, err=%v", out.IsSuccess(), out.Err())
	}
	if called {
		t.Fatalf("onSuccess must not be called when starting While with failure")
	}
}

func TestWhile_ShortCircuitOnProcessed(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	base := rop.SetProcessed(rop.Success(2))
	chain := Start(ctx, base)

	called := false
	chain = chain.While(
		func(ctx context.Context, t int) rop.Result[int] {
			called = true
			return rop.Success(t + 1)
		},
		func(ctx context.Context, t int) bool { return true },
	)

	out := chain.Result()
	if !out.IsSuccess() || out.Result() != 2 || !out.IsProcessed() {
		t.Fatalf("expected unchanged processed success, got: success=%v, val=%v, processed=%v",
			out.IsSuccess(), out.Result(), out.IsProcessed())
	}
	if called {
		t.Fatalf("onSuccess must not be called when starting While with processed result")
	}
}

func TestWhile_NoExecutionWhenConditionFalse(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	chain := FromValue(ctx, 5)

	calls := 0
	chain = chain.While(
		func(ctx context.Context, t int) rop.Result[int] {
			calls++
			return rop.Success(t + 1)
		},
		func(ctx context.Context, t int) bool { return false },
	)

	out := chain.Result()
	if !out.IsSuccess() || out.Result() != 5 {
		t.Fatalf("expected unchanged success result 5, got: success=%v, val=%v, err=%v",
			out.IsSuccess(), out.Result(), out.Err())
	}
	if calls != 0 {
		t.Fatalf("expected While body to never execute, got %d calls", calls)
	}
}

func TestWhileChain_OuterStateAndInnerIterations(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	type state struct {
		outer int
		inner int
	}

	outerStart := state{outer: 0, inner: 0}

	innerCalls := 0
	chain := FromValue(ctx, outerStart).
		WhileChain(
			func(ctx context.Context, s state) Chain[state] {
				innerCalls++

				current := FromValue(ctx, s).
					Then(func(ctx context.Context, in state) rop.Result[state] {
						in.inner++
						return rop.Success(in)
					}).
					Then(func(ctx context.Context, in state) rop.Result[state] {
						in.outer++
						return rop.Success(in)
					})

				return current
			},
			func(ctx context.Context, s state) bool {
				return s.outer < 3
			},
		)

	out := chain.Result()
	if !out.IsSuccess() {
		t.Fatalf("expected success, got: success=%v, err=%v", out.IsSuccess(), out.Err())
	}

	res := out.Result()
	if res.outer != 3 {
		t.Fatalf("expected outer to be 3, got %d", res.outer)
	}
	if innerCalls != 3 {
		t.Fatalf("expected inner chain to run 3 times, got %d", innerCalls)
	}
	if res.inner != 3 {
		t.Fatalf("expected inner to be 3, got %d", res.inner)
	}
}

func TestWhileChain_ShortCircuitOnFailure(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	err := errors.New("whilechain-fail")
	chain := Start(ctx, rop.Fail[int](err)).
		WhileChain(
			func(ctx context.Context, v int) Chain[int] {
				t.Fatalf("inner chain must not be executed on failure start")
				return FromValue(ctx, v+1)
			},
			func(ctx context.Context, v int) bool { return true },
		)

	out := chain.Result()
	if out.IsSuccess() || out.Err() == nil || out.Err().Error() != "whilechain-fail" {
		t.Fatalf("expected failure 'whilechain-fail', got: success=%v, err=%v", out.IsSuccess(), out.Err())
	}
}

func TestWhileChain_ShortCircuitOnProcessed(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	base := rop.SetProcessed(rop.Success(1))
	chain := Start(ctx, base).
		WhileChain(
			func(ctx context.Context, v int) Chain[int] {
				t.Fatalf("inner chain must not be executed on processed start")
				return FromValue(ctx, v+1)
			},
			func(ctx context.Context, v int) bool { return true },
		)

	out := chain.Result()
	if !out.IsSuccess() || out.Result() != 1 || !out.IsProcessed() {
		t.Fatalf("expected unchanged processed success, got: success=%v, val=%v, processed=%v",
			out.IsSuccess(), out.Result(), out.IsProcessed())
	}
}

func TestWhileChain_NoExecutionWhenConditionFalse(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	chain := FromValue(ctx, 5).
		WhileChain(
			func(ctx context.Context, v int) Chain[int] {
				t.Fatalf("inner chain must not be executed when predicate is false")
				return FromValue(ctx, v+1)
			},
			func(ctx context.Context, v int) bool { return false },
		)

	out := chain.Result()
	if !out.IsSuccess() || out.Result() != 5 {
		t.Fatalf("expected unchanged success result 5, got: success=%v, val=%v, err=%v",
			out.IsSuccess(), out.Result(), out.Err())
	}
}

func TestWhileChain_InnerFailureBreaksLoop(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	calls := 0
	chain := FromValue(ctx, 1).
		WhileChain(
			func(ctx context.Context, v int) Chain[int] {
				calls++
				return Start(ctx, rop.Fail[int](errors.New("inner-fail")))
			},
			func(ctx context.Context, v int) bool { return true },
		)

	out := chain.Result()
	if out.IsSuccess() || out.Err() == nil || out.Err().Error() != "inner-fail" {
		t.Fatalf("expected failure 'inner-fail', got: success=%v, err=%v", out.IsSuccess(), out.Err())
	}
	if calls != 1 {
		t.Fatalf("expected inner chain to execute once before failure, got %d", calls)
	}
}

func TestWhileChain_InnerProcessedBreaksLoop(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	calls := 0
	chain := FromValue(ctx, 1).
		WhileChain(
			func(ctx context.Context, v int) Chain[int] {
				calls++
				return Start(ctx, rop.SetProcessed(rop.Success(v+1)))
			},
			func(ctx context.Context, v int) bool { return true },
		)

	out := chain.Result()
	if !out.IsSuccess() || !out.IsProcessed() || out.Result() != 2 {
		t.Fatalf("expected processed success with value 2, got: success=%v, processed=%v, val=%v, err=%v",
			out.IsSuccess(), out.IsProcessed(), out.Result(), out.Err())
	}
	if calls != 1 {
		t.Fatalf("expected inner chain to execute once before processed stop, got %d", calls)
	}
}

func TestRepeatChainUntil_OuterStateAndInnerIterations(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	type state struct {
		outer int
		inner int
	}

	outerStart := state{outer: 0, inner: 0}

	innerCalls := 0
	chain := FromValue(ctx, outerStart).
		RepeatChainUntil(
			func(ctx context.Context, s state) Chain[state] {
				innerCalls++

				current := FromValue(ctx, s).
					Then(func(ctx context.Context, in state) rop.Result[state] {
						in.inner++
						return rop.Success(in)
					}).
					Then(func(ctx context.Context, in state) rop.Result[state] {
						in.outer++
						return rop.Success(in)
					})

				return current
			},
			func(ctx context.Context, s state) bool {
				return s.outer < 3
			},
		)

	out := chain.Result()
	if !out.IsSuccess() {
		t.Fatalf("expected success, got: success=%v, err=%v", out.IsSuccess(), out.Err())
	}

	res := out.Result()
	if res.outer != 3 {
		t.Fatalf("expected outer to be 3, got %d", res.outer)
	}
	if innerCalls != 3 {
		t.Fatalf("expected inner chain to run 3 times, got %d", innerCalls)
	}
	if res.inner != 3 {
		t.Fatalf("expected inner to be 3, got %d", res.inner)
	}
}

func TestRepeatChainUntil_ShortCircuitOnFailure(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	err := errors.New("repeatchain-fail")
	chain := Start(ctx, rop.Fail[int](err)).
		RepeatChainUntil(
			func(ctx context.Context, v int) Chain[int] {
				t.Fatalf("inner chain must not be executed on failure start")
				return FromValue(ctx, v+1)
			},
			func(ctx context.Context, v int) bool { return true },
		)

	out := chain.Result()
	if out.IsSuccess() || out.Err() == nil || out.Err().Error() != "repeatchain-fail" {
		t.Fatalf("expected failure 'repeatchain-fail', got: success=%v, err=%v", out.IsSuccess(), out.Err())
	}
}

func TestRepeatChainUntil_ShortCircuitOnProcessed(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	base := rop.SetProcessed(rop.Success(1))
	chain := Start(ctx, base).
		RepeatChainUntil(
			func(ctx context.Context, v int) Chain[int] {
				t.Fatalf("inner chain must not be executed on processed start")
				return FromValue(ctx, v+1)
			},
			func(ctx context.Context, v int) bool { return true },
		)

	out := chain.Result()
	if !out.IsSuccess() || out.Result() != 1 || !out.IsProcessed() {
		t.Fatalf("expected unchanged processed success, got: success=%v, val=%v, processed=%v",
			out.IsSuccess(), out.Result(), out.IsProcessed())
	}
}

func TestRepeatChainUntil_ExecutesOnceWhenUntilFalse(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	calls := 0
	chain := FromValue(ctx, 1).
		RepeatChainUntil(
			func(ctx context.Context, v int) Chain[int] {
				calls++
				return FromValue(ctx, v+1)
			},
			func(ctx context.Context, v int) bool { return false },
		)

	out := chain.Result()
	if !out.IsSuccess() {
		t.Fatalf("expected success result, got: success=%v, err=%v", out.IsSuccess(), out.Err())
	}
	if calls != 1 {
		t.Fatalf("expected inner chain to execute once, got %d calls", calls)
	}
}

func TestRepeatChainUntil_InnerFailureBreaksLoop(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	calls := 0
	chain := FromValue(ctx, 1).
		RepeatChainUntil(
			func(ctx context.Context, v int) Chain[int] {
				calls++
				return Start(ctx, rop.Fail[int](errors.New("inner-fail")))
			},
			func(ctx context.Context, v int) bool { return true },
		)

	out := chain.Result()
	if out.IsSuccess() || out.Err() == nil || out.Err().Error() != "inner-fail" {
		t.Fatalf("expected failure 'inner-fail', got: success=%v, err=%v", out.IsSuccess(), out.Err())
	}
	if calls != 1 {
		t.Fatalf("expected inner chain to execute once before failure, got %d", calls)
	}
}

func TestRepeatChainUntil_InnerProcessedBreaksLoop(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	calls := 0
	chain := FromValue(ctx, 1).
		RepeatChainUntil(
			func(ctx context.Context, v int) Chain[int] {
				calls++
				return Start(ctx, rop.SetProcessed(rop.Success(v+1)))
			},
			func(ctx context.Context, v int) bool { return true },
		)

	out := chain.Result()
	if !out.IsSuccess() || !out.IsProcessed() || out.Result() != 2 {
		t.Fatalf("expected processed success with value 2, got: success=%v, processed=%v, val=%v, err=%v",
			out.IsSuccess(), out.IsProcessed(), out.Result(), out.Err())
	}
	if calls != 1 {
		t.Fatalf("expected inner chain to execute once before processed stop, got %d", calls)
	}
}

func TestThen_ShortCircuitOnProcessed(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	base := rop.SetProcessed(rop.Success(10))
	chain := Start(ctx, base)

	called := false
	chain = chain.Then(func(ctx context.Context, t int) rop.Result[int] {
		called = true
		return rop.Success(t + 1)
	})

	out := chain.Result()
	if !out.IsSuccess() || out.Result() != 10 || !out.IsProcessed() {
		t.Fatalf("expected unchanged processed success, got: success=%v, val=%v, processed=%v",
			out.IsSuccess(), out.Result(), out.IsProcessed())
	}
	if called {
		t.Fatalf("Then must not invoke onSuccess when result is already processed")
	}
}

func TestThenTry_DeadlineExceededIsCancel(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	chain := FromValue(ctx, 4).
		ThenTry(func(ctx context.Context, t int) (int, error) {
			return 0, context.DeadlineExceeded
		})

	out := chain.Result()
	if !out.IsCancel() {
		t.Fatalf("expected cancel result for DeadlineExceeded, got: success=%v, cancel=%v, err=%v",
			out.IsSuccess(), out.IsCancel(), out.Err())
	}
	if !errors.Is(out.Err(), context.DeadlineExceeded) {
		t.Fatalf("expected DeadlineExceeded error, got: %v", out.Err())
	}
}

func TestTo_ShortCircuitOnFailure(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	err := errors.New("fail")
	chain := Start(ctx, rop.Fail[int](err)).
		Then(func(ctx context.Context, t int) rop.Result[int] { return rop.Success(t + 1) })

	out := chain.Result()
	if out.IsSuccess() || out.Err() == nil || out.Err().Error() != "fail" {
		t.Fatalf("expected failure 'fail', got: success=%v, err=%v", out.IsSuccess(), out.Err())
	}
}

func TestTo_Success(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	chain := FromValue(ctx, 2).
		Then(func(ctx context.Context, t int) rop.Result[int] { return rop.Success(t * 3) })

	out := chain.Result()
	if !out.IsSuccess() || out.Result() != 6 {
		t.Fatalf("expected success with 6, got: success=%v, val=%v, err=%v", out.IsSuccess(), out.Result(), out.Err())
	}
}

func TestEnsure_SideEffects(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// success path
	sCalled := false
	fCalled := false
	pCalled := false
	out1 := FromValue(ctx, 11).
		Ensure(func(ctx context.Context, v int) { sCalled = true },
			func(ctx context.Context, err error) { fCalled = true },
			func(ctx context.Context, v int) { pCalled = true }).
		Result()
	if !out1.IsSuccess() || out1.Result() != 11 {
		t.Fatalf("expected success with 11, got: %v, %v", out1.IsSuccess(), out1.Err())
	}

	if !sCalled || fCalled || pCalled {
		t.Fatalf("expected success side-effect only; sCalled=%v, fCalled=%v, pCalled=%v",
			sCalled, fCalled, pCalled)
	}

	// failure path (including cancel treated as failure in Ensure)
	sCalled = false
	fCalled = false
	pCalled = false
	out2 := Start(ctx, rop.Fail[int](errors.New("bad"))).
		Ensure(func(ctx context.Context, v int) { sCalled = true },
			func(ctx context.Context, err error) { fCalled = true },
			func(ctx context.Context, p int) { pCalled = true }).
		Result()
	if out2.IsSuccess() || out2.Err() == nil || out2.Err().Error() != "bad" {
		t.Fatalf("expected failure 'bad', got: success=%v, err=%v", out2.IsSuccess(), out2.Err())
	}

	if sCalled || !fCalled || pCalled {
		t.Fatalf("expected failure side-effect only; sCalled=%v, fCalled=%v, pCalled=%v", sCalled, fCalled, pCalled)
	}

	// success path processed*
	sCalled = false
	fCalled = false
	pCalled = false
	out3 := Start(ctx, rop.SetProcessed(rop.Success(11))).
		Ensure(func(ctx context.Context, v int) { sCalled = true },
			func(ctx context.Context, err error) { fCalled = true },
			func(ctx context.Context, v int) { pCalled = true }).
		Result()

	if !out3.IsSuccess() || out3.Result() != 11 {
		t.Fatalf("expected success with 11, got: %v, %v", out3.IsSuccess(), out3.Err())
	}

	if sCalled || fCalled || !pCalled {
		t.Fatalf("expected success side-effect only; sCalled=%v, fCalled=%v, pCalled=%v",
			sCalled, fCalled, pCalled)
	}

	// failure path processed* (including cancel treated as failure in Ensure)
	sCalled = false
	fCalled = false
	pCalled = false
	out4 := Start(ctx, rop.SetProcessed(rop.Fail[int](errors.New("bad")))).
		Ensure(func(ctx context.Context, v int) { sCalled = true },
			func(ctx context.Context, err error) { fCalled = true },
			func(ctx context.Context, p int) { pCalled = true }).
		Result()

	if out4.IsSuccess() || out4.Err() == nil || out4.Err().Error() != "bad" {
		t.Fatalf("expected failure 'bad', got: success=%v, err=%v", out4.IsSuccess(), out4.Err())
	}

	if sCalled || !fCalled || pCalled {
		t.Fatalf("expected failure side-effect only; sCalled=%v, fCalled=%v, pCalled=%v",
			sCalled, fCalled, pCalled)
	}

	// nil callbacks should be safe
	out5 := FromValue(ctx, 1).Ensure(nil, nil, nil).Result()
	if !out5.IsSuccess() || out5.Result() != 1 {
		t.Fatalf("expected unchanged success result, got: %v, %v", out5.IsSuccess(), out5.Err())
	}

	// unchanged success processed
	out6 := Start(ctx, rop.SetProcessed(rop.Success(1))).
		Ensure(nil, nil, nil).Result()
	if !out6.IsSuccess() || out6.Result() != 1 || !out6.IsProcessed() {
		t.Fatalf("expected unchanged processed success result, got: %v, %v, %v",
			out6.IsSuccess(), out6.Err(), out6.IsProcessed())
	}

	// unchanged fail processed
	out7 := Start(ctx, rop.SetProcessed(rop.Fail[int](fmt.Errorf("bad processed")))).
		Ensure(nil, nil, nil).Result()
	if out7.IsSuccess() || !out7.IsFailure() || out7.Err() == nil || !out7.IsProcessed() {
		t.Fatalf("expected unchanged processed fail result, got: %v, %v, %v",
			out7.IsSuccess(), out7.Err(), out7.IsProcessed())
	}
}

func TestFinally_SuccessFailureCancel(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// success
	s := FromValue(ctx, 3).Finally(
		func(ctx context.Context, v int) int { return v + 100 },
		func(ctx context.Context, err error) int { return -1 },
		func(ctx context.Context, err error) int { return -2 },
	)
	if s != 103 {
		t.Fatalf("expected 103, got %d", s)
	}

	// failure
	f := Start(ctx, rop.Fail[int](errors.New("x"))).Finally(
		func(ctx context.Context, v int) int { return v },
		func(ctx context.Context, err error) int { return -1 },
		func(ctx context.Context, err error) int { return -2 },
	)
	if f != -1 {
		t.Fatalf("expected -1 for failure, got %d", f)
	}

	// cancel
	c := Start(ctx, rop.Cancel[int](errors.New("c"))).Finally(
		func(ctx context.Context, v int) int { return v },
		func(ctx context.Context, err error) int { return -1 },
		func(ctx context.Context, err error) int { return -2 },
	)
	if c != -2 {
		t.Fatalf("expected -2 for cancel, got %d", c)
	}
}
