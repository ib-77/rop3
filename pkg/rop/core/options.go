package core

import "context"

type OptionKey string

const (
	ProcessOptionKey OptionKey = "process_options"
	WorkerOptionKey  OptionKey = "worker_options"
)

type MaxLimitOption struct {
	Value int
}
type WorkerOptions struct {
	MaxCount MaxLimitOption
}

type ProcessOptions struct {
	ProcessRemaining bool
}

func WithProcessOptions(ctx context.Context, processRemaining bool) context.Context {
	return context.WithValue(ctx, ProcessOptionKey, ProcessOptions{ProcessRemaining: processRemaining})
}

func WithWorkerOptions(ctx context.Context, maxWorkers int) context.Context {
	return context.WithValue(ctx, WorkerOptionKey, WorkerOptions{MaxLimitOption{Value: maxWorkers}})
}

func GetWorkerMaxCount(ctx context.Context, defaultMaxWorkers int) int {
	options, ok := ctx.Value(WorkerOptionKey).(WorkerOptions)
	if ok {
		return options.MaxCount.Value
	}
	return defaultMaxWorkers
}

func IsProcessRemainingEnabled(ctx context.Context, defaultProcessRemaining bool) bool {
	options, ok := ctx.Value(ProcessOptionKey).(ProcessOptions)
	if ok {
		return options.ProcessRemaining
	}
	return defaultProcessRemaining
}
