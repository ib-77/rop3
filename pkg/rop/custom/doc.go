// Package custom exposes higher-level concurrent helpers that build on mass
// to add cancellation strategies and multi-worker orchestration. It is suited
// for pipelines that need explicit handling of unprocessed, processed, and
// remaining values on cancel.
//
// Key constructs:
// - Run/RunSingle: orchestrate engines with handlers and success callbacks
// - Validate, Switch, Map, DoubleMap, Try: channel-lifted operations
// - CancelRemaining* utilities: define how remaining items are canceled
package custom