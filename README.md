## Railway Oriented Programming (ROP) for Go

Lightweight, composable primitives for building robust synchronous and concurrent processing pipelines in Go using the Railway Oriented Programming pattern.

## Overview

Railway Oriented Programming is a functional programming pattern for handling errors in a clean and composable way. This Go library implements ROP principles using generics, allowing you to build robust error handling pipelines.

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

Railway Oriented Programming visualizes program flow as a railway track:

- Success Track: When operations succeed, they continue along the main track.
- Failure Track: When operations fail, they switch to a parallel error track.

Diagram (simplified):

![ROP diagram](assets/diagram.svg)

This repository centers around a typed `Result[T]` that carries either a value, an error (fail), or a cancel signal. The package split is roughly:

- `solo`: single-value, synchronous composition helpers
- `mass`: channel-based building blocks that lift `solo` operations over channels
- `lite`: ergonomic wrappers around `mass` to simplify common patterns and set up multi-worker stages
- `core`: utilities for channel IO and worker orchestration

Common primitives and what they do

- Result[T]: a typed container that can be success, failure, or cancel.
  - `rop.Success(value)` / `rop.Fail[T](err)` / `rop.Cancel[T](err)`
  - Inspect with `IsSuccess()`, `IsCancel()`, `Err()`, `Result()`

Operations (with `lite`-style examples)

- Validate (validate input data): run a predicate that returns `(bool, string)`. On false the result becomes a failure with the provided message.

```go
// simple validator for non-empty strings
func notEmpty(_ context.Context, s string) (bool, string) {
    if s == "" { return false, "empty" }
    return true, ""
}

out := core.FromChanMany(context.Background(),
    lite.Run(context.Background(), core.ToChanManyResults(context.Background(), []string{"a", "", "c"}),
        lite.Validate(notEmpty), 2),
)
// out -> ["a", <fail:"empty">, "c"]
```

Try it locally

The `examples/lite_examples` folder includes a runnable Go example that demonstrates Validate, Try, Turnout and Finally. From the repository root run:

```powershell
go run ./examples/lite_examples
```

The example prints final values produced by the pipeline.

- Switch (convert a successful input result into another Result, possibly a failure): run a function that returns `rop.Result[Out]`.

```go
func toNumber(_ context.Context, s string) rop.Result[int] {
    if s == "bad" { return rop.Fail[int](fmt.Errorf("bad input")) }
    n, _ := strconv.Atoi(s)
    return rop.Success(n)
}

out := core.FromChanMany(context.Background(),
    lite.Run(context.Background(), core.ToChanManyResults(context.Background(), []string{"1","bad","3"}),
        lite.Switch(toNumber), 2),
)
// bad -> fail, others -> success
```

- Map (convert a successful result value to another success value).

```go
out := core.FromChanMany(context.Background(),
    lite.Run(context.Background(), core.ToChanManyResults(context.Background(), []int{1,2,3}),
        lite.Map(func(_ context.Context, n int) int { return n * 2 }), 2),
)
// -> [2,4,6]
```

- DoubleMap (convert both success and failure results to another type; also handles cancel).

```go
handlers := mass.FinallyHandlers[int, string]{
    OnSuccess: func(_ context.Context, v int) string { return fmt.Sprintf("ok:%d", v) },
    OnError: func(_ context.Context, err error) string { return "err:" + err.Error() },
    OnCancel: func(_ context.Context, err error) string { return "cancel" },
}

out := core.FromChanMany(context.Background(),
    lite.Finally(context.Background(),
        lite.Run(context.Background(), core.ToChanManyResults(context.Background(), []int{1,2}), lite.DoubleMap(func(_ context.Context, r rop.Result[int]) string {
            if r.IsSuccess() { return fmt.Sprintf("ok:%d", r.Result()) }
            return "err" // simplified
        }), 2),
    , handlers),
)
```

- Tee (side-effect on success): run a function for its side-effects but continue passing the original success value.

```go
out := core.FromChanMany(context.Background(),
    lite.Run(context.Background(), core.ToChanManyResults(context.Background(), []int{1,2}),
        lite.Tee(func(_ context.Context, n int) { fmt.Println("seen", n) }), 2),
)
```

- DoubleTee (side-effect for both success and failure, also receives cancel): similar to `Tee` but gets the full `rop.Result[T]` for inspection.

```go
out := core.FromChanMany(context.Background(),
    lite.Run(context.Background(), core.ToChanManyResults(context.Background(), []int{1,2}),
        lite.DoubleTee(func(_ context.Context, r rop.Result[int]) { if r.IsSuccess() { fmt.Println("ok", r.Result()) } else { fmt.Println("err/cancel") } }), 2),
)
```

- Try (execute a function that returns a value and error): on error, the stage emits a failure result; on success, a success result.

```go
func maybeFail(_ context.Context, n int) (int, error) {
    if n%2 == 0 { return 0, fmt.Errorf("even") }
    return n, nil
}

out := core.FromChanMany(context.Background(),
    lite.Run(context.Background(), core.ToChanManyResults(context.Background(), []int{1,2,3}),
        lite.Try(maybeFail), 2),
)
// -> success for 1 and 3, fail for 2
```

- Finally (prepare the output values of the pipeline by reducing Result[T] into final values)

```go
handlers := mass.FinallyHandlers[int, string]{
    OnSuccess: func(_ context.Context, v int) string { return fmt.Sprintf("val:%d", v) },
    OnError: func(_ context.Context, err error) string { return "bad" },
    OnCancel: func(_ context.Context, err error) string { return "cancelled" },
}

out := core.FromChanMany(context.Background(),
    lite.Finally(context.Background(),
        lite.Run(context.Background(), core.ToChanManyResults(context.Background(), []int{1,2}), lite.Map(func(_ context.Context, n int) int { return n + 1 }), 2),
    , handlers),
)
// out -> ["val:2","val:3"]
```

Workers and execution patterns

The `lite` package provides small helpers to run stages with worker pools. Here are the common worker patterns:

- Run: execute a stage over an input channel of `Result[T]` without changing the result type. Use multiple workers to parallelize processing.

```go
out := core.FromChanMany(context.Background(),
    lite.Run(context.Background(), core.ToChanManyResults(context.Background(), []int{1,2,3}),
        lite.Tee(func(_ context.Context, n int) { fmt.Println("processing", n) }), 3), // 3 workers
)
```

- Turnout: execute a stage that changes the underlying value type (e.g., from string to int). This is used when a stage needs to change the pipeline's payload type.

```go
out := core.FromChanMany(context.Background(),
    lite.Turnout(context.Background(),
        lite.Run(context.Background(), core.ToChanManyResults(context.Background(), []string{"1","2"}),
            lite.Switch(toNumber), 2), // Turnout wraps a Run that changes type
    , 2),
)
```

- Single-worker execution: run the pipeline with one worker using `lite.Run(..., 1)`. This is useful for deterministic, ordered processing or when parallelism is not desired.

```go
out := core.FromChanMany(context.Background(),
    lite.Run(context.Background(), core.ToChanManyResults(context.Background(), []int{1,2,3}),
        lite.Map(func(_ context.Context, n int) int { return n * 10 }), 1), // one worker
)
```



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

    "github.com/ib-77/rop3/pkg/rop"
    "github.com/ib-77/rop3/pkg/rop/core"
    "github.com/ib-77/rop3/pkg/rop/lite"
    "github.com/ib-77/rop3/pkg/rop/mass"
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


## License

MIT


