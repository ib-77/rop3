package tiny

import (
    "context"
    "errors"
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

func TestTo_ShortCircuitOnFailure(t *testing.T) {
    t.Parallel()
    ctx := context.Background()
    err := errors.New("fail")
    chain := Start(ctx, rop.Fail[int](err)).
        To(func(ctx context.Context, t int) rop.Result[int] { return rop.Success(t + 1) })

    out := chain.Result()
    if out.IsSuccess() || out.Err() == nil || out.Err().Error() != "fail" {
        t.Fatalf("expected failure 'fail', got: success=%v, err=%v", out.IsSuccess(), out.Err())
    }
}

func TestTo_Success(t *testing.T) {
    t.Parallel()
    ctx := context.Background()
    chain := FromValue(ctx, 2).
        To(func(ctx context.Context, t int) rop.Result[int] { return rop.Success(t * 3) })

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
    out1 := FromValue(ctx, 11).
        Ensure(func(ctx context.Context, v int) { sCalled = true }, func(ctx context.Context, err error) { fCalled = true }).
        Result()
    if !out1.IsSuccess() || out1.Result() != 11 {
        t.Fatalf("expected success with 11, got: %v, %v", out1.IsSuccess(), out1.Err())
    }
    if !sCalled || fCalled {
        t.Fatalf("expected success side-effect only; sCalled=%v, fCalled=%v", sCalled, fCalled)
    }

    // failure path (including cancel treated as failure in Ensure)
    sCalled = false
    fCalled = false
    out2 := Start(ctx, rop.Fail[int](errors.New("bad"))).
        Ensure(func(ctx context.Context, v int) { sCalled = true }, func(ctx context.Context, err error) { fCalled = true }).
        Result()
    if out2.IsSuccess() || out2.Err() == nil || out2.Err().Error() != "bad" {
        t.Fatalf("expected failure 'bad', got: success=%v, err=%v", out2.IsSuccess(), out2.Err())
    }
    if sCalled || !fCalled {
        t.Fatalf("expected failure side-effect only; sCalled=%v, fCalled=%v", sCalled, fCalled)
    }

    // nil callbacks should be safe
    out3 := FromValue(ctx, 1).Ensure(nil, nil).Result()
    if !out3.IsSuccess() || out3.Result() != 1 {
        t.Fatalf("expected unchanged success result, got: %v, %v", out3.IsSuccess(), out3.Err())
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