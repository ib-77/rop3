// Package tiny provides a minimal fluent Chain[T] for synchronous
// composition of Result[T] values.
//
// It parallels the chain package but keeps API surface very small:
// - Start/FromValue: create a Chain
// - Then/ThenTry: compose result-returning or error-returning functions
// - Map/To: transform or switch value to a new Result
// - Ensure: trigger side effects on success only
// - Finally: reduce to a concrete value via handlers
//
// Tiny is ideal for small services or tests where lightweight synchronous
// chaining improves readability without introducing channels.
package tiny
