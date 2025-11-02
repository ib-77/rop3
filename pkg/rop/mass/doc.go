// Package mass implements channel-based building blocks that lift solo
// primitives and provide orchestration (validation, mapping, try, finalizing)
// with more control over cancellation and worker behaviors.
//
// It is typically used by higher-level packages (lite/custom) to compose
// concurrent pipelines, integrating cancellation handlers and select loops.
package mass