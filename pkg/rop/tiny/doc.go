// Package tiny provides a minimal fluent Chain[T] for synchronous
// composition of Result[T] values.
//
// It parallels the chain package but keeps API surface very small:
//   - Start/FromValue: create a Chain from a Result or value
//   - Then/ThenTry: compose result-returning or error-returning functions
//     (ThenTry converts DeadlineExceeded to a cancel result)
//   - RepeatUntil: loop until a predicate signals stop
//     (composes a nested Chain on each iteration)
//   - While: loop while a predicate holds
//     (rebuilds the Chain from the current value each iteration)
//
// - Map: transform the successful value to a new Result
// - Ensure: trigger side effects for success, failure, or processed results
// - Finally: reduce to a concrete value via handlers
//
// Tiny is ideal for small services or tests where lightweight synchronous
// chaining improves readability without introducing channels.
package tiny
