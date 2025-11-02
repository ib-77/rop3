package chain

import (
    "context"
    "errors"
    "testing"

    "github.com/ib-77/rop3/pkg/rop"
)

func TestStart_Result_Success(t *testing.T) {
    t.Parallel()
    ctx := context.Background()
    base := rop.Success(10)
    c := Start(ctx, base)
    out := c.Result()
    if !out.IsSuccess() || out.Result() != 10 {
        t.Fatalf("expected success with 10, got success=%v, val=%v, err=%v", out.IsSuccess(), out.Result(), out.Err())
    }
}

func TestFromValue_Success(t *testing.T) {
    t.Parallel()
    ctx := context.Background()
    c := FromValue(ctx, 7)
    out := c.Result()
    if !out.IsSuccess() || out.Result() != 7 {
        t.Fatalf("expected success with 7, got success=%v, val=%v, err=%v", out.IsSuccess(), out.Result(), out.Err())
    }
}

func TestThen_ShortCircuitOnFailure(t *testing.T) {
    t.Parallel()
    ctx := context.Background()
    err := errors.New("boom")
    c := Start(ctx, rop.Fail[int](err))
    called := false
    c2 := Then(c, func(ctx context.Context, v int) rop.Result[string] {
        called = true
        return rop.Success("ok")
    })
    out := c2.Result()
    if out.IsSuccess() || out.Err() == nil || out.Err().Error() != "boom" {
        t.Fatalf("expected failure 'boom', got success=%v err=%v", out.IsSuccess(), out.Err())
    }
    if called {
        t.Fatalf("Then onSuccess must not be called on failure input")
    }
}

func TestThen_PropagateCancel(t *testing.T) {
    t.Parallel()
    ctx := context.Background()
    c := Start(ctx, rop.Cancel[int](errors.New("cancel")))
    called := false
    c2 := Then(c, func(ctx context.Context, v int) rop.Result[string] {
        called = true
        return rop.Success("x")
    })
    out := c2.Result()
    if !out.IsCancel() || out.Err() == nil || out.Err().Error() != "cancel" {
        t.Fatalf("expected cancel 'cancel', got cancel=%v err=%v", out.IsCancel(), out.Err())
    }
    if called {
        t.Fatalf("Then onSuccess must not be called on cancel input")
    }
}

func TestThenTry_SuccessAndError(t *testing.T) {
    t.Parallel()
    ctx := context.Background()

    // success path
    c := FromValue(ctx, 3)
    c2 := ThenTry(c, func(ctx context.Context, v int) (string, error) {
        return "val_3", nil
    })
    out := c2.Result()
    if !out.IsSuccess() || out.Result() != "val_3" {
        t.Fatalf("expected success 'val_3', got success=%v val=%v err=%v", out.IsSuccess(), out.Result(), out.Err())
    }

    // error path
    c3 := FromValue(ctx, 9)
    c4 := ThenTry(c3, func(ctx context.Context, v int) (string, error) {
        return "", errors.New("try-error")
    })
    out2 := c4.Result()
    if out2.IsSuccess() || out2.Err() == nil || out2.Err().Error() != "try-error" {
        t.Fatalf("expected failure 'try-error', got success=%v err=%v", out2.IsSuccess(), out2.Err())
    }

    // short-circuit on failure input
    c5 := Start(ctx, rop.Fail[int](errors.New("bad")))
    c6 := ThenTry(c5, func(ctx context.Context, v int) (string, error) { return "ignored", nil })
    out3 := c6.Result()
    if out3.IsSuccess() || out3.Err() == nil || out3.Err().Error() != "bad" {
        t.Fatalf("expected failure 'bad', got success=%v err=%v", out3.IsSuccess(), out3.Err())
    }
}

func TestMap_SuccessAndFailure(t *testing.T) {
    t.Parallel()
    ctx := context.Background()

    // success path
    c := FromValue(ctx, 5)
    c2 := Map(c, func(ctx context.Context, v int) string { return "n:" + string(rune('0'+v)) })
    out := c2.Result()
    if !out.IsSuccess() || out.Result() != "n:5" {
        t.Fatalf("expected success 'n:5', got success=%v val=%v err=%v", out.IsSuccess(), out.Result(), out.Err())
    }

    // failure short-circuit
    c3 := Start(ctx, rop.Fail[int](errors.New("oops")))
    c4 := Map(c3, func(ctx context.Context, v int) string { return "ignored" })
    out2 := c4.Result()
    if out2.IsSuccess() || out2.Err() == nil || out2.Err().Error() != "oops" {
        t.Fatalf("expected failure 'oops', got success=%v err=%v", out2.IsSuccess(), out2.Err())
    }
}

func TestEnsure_SideEffectCalledOnSuccess(t *testing.T) {
    t.Parallel()
    ctx := context.Background()
    called := false
    c := FromValue(ctx, 11).Ensure(func(ctx context.Context, v int) { called = true })
    out := c.Result()
    if !out.IsSuccess() || out.Result() != 11 {
        t.Fatalf("expected success with 11, got success=%v val=%v err=%v", out.IsSuccess(), out.Result(), out.Err())
    }
    if !called {
        t.Fatalf("expected Ensure to invoke onSuccess for success result")
    }

    // failure path should not call onSuccess
    called = false
    c2 := Start(ctx, rop.Fail[int](errors.New("x"))).Ensure(func(ctx context.Context, v int) { called = true })
    out2 := c2.Result()
    if out2.IsSuccess() || out2.Err() == nil || out2.Err().Error() != "x" {
        t.Fatalf("expected failure 'x', got success=%v err=%v", out2.IsSuccess(), out2.Err())
    }
    if called {
        t.Fatalf("Ensure onSuccess must not be called for failure result")
    }
}

func TestFinally_SuccessFailureCancel(t *testing.T) {
    t.Parallel()
    ctx := context.Background()

    // success
    s := Finally(FromValue(ctx, 2),
        func(ctx context.Context, v int) string { return "ok" },
        func(ctx context.Context, err error) string { return "fail" },
        func(ctx context.Context, err error) string { return "cancel" },
    )
    if s != "ok" {
        t.Fatalf("expected 'ok', got %q", s)
    }

    // failure
    f := Finally(Start(ctx, rop.Fail[int](errors.New("e"))),
        func(ctx context.Context, v int) string { return "ok" },
        func(ctx context.Context, err error) string { return "fail" },
        func(ctx context.Context, err error) string { return "cancel" },
    )
    if f != "fail" {
        t.Fatalf("expected 'fail', got %q", f)
    }

    // cancel
    c := Finally(Start(ctx, rop.Cancel[int](errors.New("c"))),
        func(ctx context.Context, v int) string { return "ok" },
        func(ctx context.Context, err error) string { return "fail" },
        func(ctx context.Context, err error) string { return "cancel" },
    )
    if c != "cancel" {
        t.Fatalf("expected 'cancel', got %q", c)
    }
}