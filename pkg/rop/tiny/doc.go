// Package tiny provides a minimal fluent Chain[T] for synchronous
// composition of Result[T] values.
//
// It parallels the chain package but keeps API surface very small:
// - Start/FromValue: create a Chain
// - Then/ThenTry: compose result-returning or error-returning functions
//   (ThenTry converts DeadlineExceeded to a cancel result)
// - RepeatUntil/While: loop while predicates hold, respecting failures and processed results
// - Map: transform the successful value to a new Result
// - Ensure: trigger side effects for success, failure, or processed results
// - Finally: reduce to a concrete value via handlers
//
// Tiny is ideal for small services or tests where lightweight synchronous
// chaining improves readability without introducing channels.
package tiny
