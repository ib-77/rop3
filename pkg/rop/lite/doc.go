// Package lite provides lightweight channel-lifted helpers that wrap solo
// primitives for concurrent pipelines. It is designed for simple fan-out/fan-in
// flows without custom cancellation handling.
//
// Common usage:
// - Run: execute an engine over an input channel with a fixed number of lines
// - Validate/Try/Switch/Map/DoubleMap: lift solo operations over channels
// - Turnout: compose stages with configurable parallelism
// - Finally: map Result[In] to Out on completion
//
// For advanced cancellation routing and multi-worker control, see package mass
// and custom.
package lite