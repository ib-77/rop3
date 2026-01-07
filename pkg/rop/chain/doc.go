// Package chain provides a fluent wrapper around Result[T]
// for building synchronous Railway-Oriented chains using solo primitives.
//
// It composes functions like Switch, Map, Try, Tee, and Finally behind a
// convenient Chain[T] type. This enables ergonomic pipelines without
// dealing directly with branching results at each step.
//
// Key operations:
// - Start/FromValue: begin a chain from a Result[T] or value
// - Then: switch to a new Result[U] via a function
// - ThenTry: call a function (U, error) and convert error to failure
// - Map: transform the successful value (T -> U)
// - Ensure: run side effects on success without changing the result
// - Finally: collapse the chain into a final value via handlers
package chain
