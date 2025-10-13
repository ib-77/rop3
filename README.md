## Railway Oriented Programming (ROP) for Go

Lightweight, composable primitives for building robust synchronous and concurrent processing pipelines in Go using the Railway Oriented Programming pattern.

This library centers around a typed `Result[T]` that carries either a value or an error (and supports cancellations), plus small, testable building blocks to compose steps in single-value flows (`solo`), channel-based/concurrent flows (`mass`), and ergonomic helpers (`lite`).

### Install

```bash
go get github.com/ib-77/rop3
```

Import paths follow the module name from `go.mod`:

```go
import (
    "github.com/ib-77/rop3/pkg/rop"
    "github.com/ib-77/rop3/pkg/rop/solo"
    "github.com/ib-77/rop3/pkg/rop/mass"
    "github.com/ib-77/rop3/pkg/rop/lite"
    "github.com/ib-77/rop3/pkg/rop/core"
)
```

---

## Core concepts

- **Result[T]**: a typed container that can be success, failure, or cancel.
  - `rop.Success(value)` / `rop.Fail[T](err)` / `rop.Cancel[T](err)`
  - Inspect with `IsSuccess()`, `IsCancel()`, `Err()`, `Result()`

- **solo**: single-value, synchronous composition helpers.
  - `solo.Validate`, `solo.Map`, `solo.Switch`, `solo.Try`, `solo.Finally`, `solo.Tee`, `solo.DoubleMap`, `solo.DoubleTee`.

- **mass**: channel-based building blocks that lift `solo` operations over channels for pipeline stages.
  - `mass.Validating`, `mass.Mapping`, `mass.Switching`, `mass.Trying`, `mass.Teeing`, `mass.DoubleMapping`, `mass.DoubleTeeing`, `mass.Finalizing`.

- **lite**: ergonomic wrappers around `mass` to simplify common patterns and set up multi-worker stages.
  - `lite.Run`, `lite.Turnout`, `lite.Validate`, `lite.Map`, `lite.Switch`, `lite.Try`, `lite.Finally`, `lite.Tee`, `lite.DoubleMap`, `lite.DoubleTee`.

- **core**: utilities for channel IO and worker orchestration.
  - `core.ToChanMany`, `core.ToChanManyResults`, `core.FromChanMany`
  - `core.Locomotive` powers the multi-worker execution under the hood
  - `core.WithWorkerOptions`, `core.WithProcessOptions` for optional runtime tuning

---

## Example: URL processing pipeline (from tests/processRequest)

The example below mirrors `tests/pipeline_test.go` and demonstrates building a concurrent pipeline that:
- Validates input URLs
- Tries to fetch a page title (mocked in tests)
- Switches to a title-length calculation
- Finally maps each result to a printable string, returning "invalid" on errors/cancels

```go
package main

import (
    "context"
    "fmt"
    "strings"

    "rop2/pkg/rop"
    "rop2/pkg/rop/core"
    "rop2/pkg/rop/lite"
    "rop2/pkg/rop/mass"
)

func processRequest(urls []string) []string {
    ctx := context.Background()

    finallyHandlers := mass.FinallyHandlers[int, string]{
        OnSuccess: func(ctx context.Context, r int) string {
            return fmt.Sprintf("title length: %d", r)
        },
        OnError: func(ctx context.Context, err error) string {
            return "invalid"
        },
        OnCancel: func(ctx context.Context, err error) string {
            return "invalid"
        },
    }

    return core.FromChanMany(ctx,
        lite.Finally(ctx,
            lite.Turnout(ctx,
                lite.Turnout(ctx,
                    lite.Run(ctx,
                        core.ToChanManyResults(ctx, urls),
                        lite.Validate(validateURLTest), 2),
                    lite.Try(mockFetchTitle), 2),
                lite.Switch(calculateTitleLength), 2),
            finallyHandlers,
        ),
    )
}

// validateURLTest checks structure and scheme in this test/demo version
func validateURLTest(_ context.Context, url string) (bool, string) {
    if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
        return false, "URL must start with http:// or https://"
    }
    return true, ""
}

// mockFetchTitle simulates a network call based on validation (no HTTP made here)
func mockFetchTitle(ctx context.Context, url string) (string, error) {
    valid, _ := validateURLTest(ctx, url)
    if valid {
        return "Mock Page Title for " + url, nil
    }
    return "", fmt.Errorf("invalid URL")
}

func calculateTitleLength(_ context.Context, title string) rop.Result[int] {
    return rop.Success(len(title))
}
```

Notes:
- `lite.Run` lifts an input channel of `Result[T]` into a worker pool that applies a stage function; `2` creates two workers.
- `lite.Turnout` changes the output type between stages while preserving `Result` semantics.
- `lite.Finally` reduces a channel of `Result[In]` to a channel of final values using the provided handlers.
- `core.FromChanMany` drains the output channel into a slice.

---

## Why ROP here?

- **Explicit success/error/cancel flow**: compile-time assurance and predictable control flow.
- **Composable**: chain simple steps into rich pipelines without hidden control paths.
- **Concurrent by construction**: easily choose worker counts per stage.
- **Testable**: each function is a small, regular Go function; orchestration is separated from business logic.

---

## Minimal solo example

```go
input := rop.Success(41)
out := solo.Map(context.Background(), input, func(_ context.Context, n int) int { return n + 1 })
if out.IsSuccess() { fmt.Println(out.Result()) } // 42
```

---

## License

MIT


