package main

import (
	"context"
	"fmt"
	"strconv"

	"github.com/ib-77/rop3/pkg/rop/core"
	"github.com/ib-77/rop3/pkg/rop/lite"
	"github.com/ib-77/rop3/pkg/rop/mass"
)

func main() {
	ctx := context.Background()

	// simple pipeline: strings -> validate non-empty -> try to parse int -> map to doubled value -> finalize
	inputs := []string{"1", "2", "bad", "", "5"}

	out := core.FromChanMany(ctx,
		lite.Finally(ctx,
			lite.Turnout(ctx,
				lite.Run(ctx,
					core.ToChanManyResults(ctx, inputs),
					lite.Validate(func(_ context.Context, s string) (bool, string) {
						if s == "" {
							return false, "empty"
						}
						return true, ""
					}),
					2),
				lite.Try(func(_ context.Context, s string) (int, error) {
					if s == "bad" {
						return 0, fmt.Errorf("bad")
					}
					n, err := strconv.Atoi(s)
					return n, err
				}),
				2),
			mass.FinallyHandlers[int, string]{
				OnSuccess: func(_ context.Context, v int) string { return fmt.Sprintf("val:%d", v) },
				OnError:   func(_ context.Context, err error) string { return "err" },
				OnCancel:  func(_ context.Context, err error) string { return "cancel" },
			},
		),
	)

	fmt.Println("Output:")
	for _, v := range out {
		fmt.Println(v)
	}
}
