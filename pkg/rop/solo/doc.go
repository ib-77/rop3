// Package solo contains single-value, synchronous ROP primitives that operate
// on Result[T]. These functions form the core building blocks for error-aware
// pipelines without channels.
//
// Highlights:
// - Success/Fail/Cancel: construct Result[T]
// - Validate/AndValidate: apply validation producing failure on invalid input
// - Switch: move from Result[In] to Result[Out]
// - Map/DoubleMap: transform successful values (with optional error/cancel maps)
// - Try: call a function (Out, error) and convert error to failure
// - Tee/TeeIf/DoubleTee: side-effect helpers
// - Finally: reduce to a concrete value via success/error/cancel handlers
package solo