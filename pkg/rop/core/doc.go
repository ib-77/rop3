// Package core contains pipeline plumbing utilities: channel helpers, worker
// configuration via context, and the locomotive that drives stages. It does
// not define business logic; instead it provides the scaffolding for packages
// like lite, mass, and custom to run pipelines with controlled concurrency.
package core