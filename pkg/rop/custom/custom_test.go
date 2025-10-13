package custom

import (
	"context"
	"errors"
	"fmt"
	"github.com/ib-77/rop3/pkg/rop"
	"github.com/ib-77/rop3/pkg/rop/core"
	"github.com/ib-77/rop3/pkg/rop/mass"
	"sync"
	"testing"
	"time"
)

// Test Run function with single worker and custom handlers
func TestProcess_WithHandlers(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	input := []int{1, 2, 3, 4, 5}
	var successCount int
	var mu sync.Mutex

	// Custom cancellation handlers
	handlers := core.CancellationHandlers[int, int]{
		OnCancel: func(ctx context.Context, inputCh <-chan rop.Result[int], outCh chan<- rop.Result[int]) {
			// Drain remaining inputs and send cancellation
			for range inputCh {
				outCh <- rop.Cancel[int](ErrCancelled)
			}
		},
		OnCancelUnprocessed: func(ctx context.Context, unprocessed rop.Result[int], outCh chan<- rop.Result[int]) {
			outCh <- rop.Cancel[int](ErrCancelled)
		},
		OnCancelProcessed: func(ctx context.Context, in rop.Result[int], processed rop.Result[int], outCh chan<- rop.Result[int]) {
			outCh <- processed // Allow processed results through
		},
	}

	// Success callback
	onSuccess := func(ctx context.Context, in rop.Result[int]) {
		if in.IsSuccess() {
			mu.Lock()
			successCount++
			mu.Unlock()
		}
	}

	// Processor that doubles the input
	processor := func(ctx context.Context, input rop.Result[int]) <-chan rop.Result[int] {
		output := make(chan rop.Result[int], 1)
		go func() {
			defer close(output)
			if input.IsSuccess() {
				output <- rop.Success(input.Result() * 2)
			} else {
				output <- input
			}
		}()
		return output
	}

	resultCh := Run(ctx, core.ToChanManyResults(ctx, input), processor, handlers, onSuccess, 2)

	var results []int
	for result := range resultCh {
		if result.IsSuccess() {
			results = append(results, result.Result())
		}
	}

	// Verify all inputs were processed
	if len(results) != len(input) {
		t.Errorf("Expected %d results, got %d", len(input), len(results))
	}

	// Verify success callback was called
	mu.Lock()
	if successCount != len(input) {
		t.Errorf("Expected %d success callbacks, got %d", len(input), successCount)
	}
	mu.Unlock()

	// Verify results are doubled
	resultMap := make(map[int]bool)
	for _, r := range results {
		resultMap[r] = true
	}

	expected := []int{2, 4, 6, 8, 10}
	for _, exp := range expected {
		if !resultMap[exp] {
			t.Errorf("Expected result %d not found", exp)
		}
	}
}

// Test Turnout function with type conversion and custom handlers
func TestTransform_WithHandlers(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	input := []int{1, 2, 3}
	var successCount int
	var mu sync.Mutex

	// Custom cancellation handlers
	handlers := core.CancellationHandlers[int, string]{
		OnCancel: func(ctx context.Context, inputCh <-chan rop.Result[int], outCh chan<- rop.Result[string]) {
			for range inputCh {
				outCh <- rop.Cancel[string](ErrCancelled)
			}
		},
	}

	// Success callback
	onSuccess := func(ctx context.Context, in rop.Result[string]) {
		if in.IsSuccess() {
			mu.Lock()
			successCount++
			mu.Unlock()
		}
	}

	// Type conversion processor (int to string)
	processor := func(ctx context.Context, input rop.Result[int]) <-chan rop.Result[string] {
		output := make(chan rop.Result[string], 1)
		go func() {
			defer close(output)
			if input.IsSuccess() {
				output <- rop.Success(fmt.Sprintf("str_%d", input.Result()))
			} else {
				output <- rop.Fail[string](input.Err())
			}
		}()
		return output
	}

	resultCh := Turnout(ctx, core.ToChanManyResults(ctx, input), processor, handlers, onSuccess, 1)

	var results []string
	for result := range resultCh {
		if result.IsSuccess() {
			results = append(results, result.Result())
		}
	}

	if len(results) != len(input) {
		t.Errorf("Expected %d results, got %d", len(input), len(results))
	}

	mu.Lock()
	if successCount != len(input) {
		t.Errorf("Expected %d success callbacks, got %d", len(input), successCount)
	}
	mu.Unlock()

	// Verify string conversion
	expected := []string{"str_1", "str_2", "str_3"}
	resultMap := make(map[string]bool)
	for _, r := range results {
		resultMap[r] = true
	}

	for _, exp := range expected {
		if !resultMap[exp] {
			t.Errorf("Expected result %s not found", exp)
		}
	}
}

// Test Single function
func TestSequential_OrderPreservation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	input := []int{1, 2, 3, 4, 5}
	var processOrder []int
	var mu sync.Mutex

	handlers := core.CancellationHandlers[int, int]{}

	onSuccess := func(ctx context.Context, in rop.Result[int]) {
		if in.IsSuccess() {
			mu.Lock()
			processOrder = append(processOrder, in.Result())
			mu.Unlock()
		}
	}

	// Processor with delays to test ordering
	processor := func(ctx context.Context, input rop.Result[int]) <-chan rop.Result[int] {
		output := make(chan rop.Result[int], 1)
		go func() {
			defer close(output)
			// Add delay proportional to value to test sequential processing
			time.Sleep(time.Duration(input.Result()) * 10 * time.Millisecond)
			if input.IsSuccess() {
				output <- rop.Success(input.Result())
			} else {
				output <- input
			}
		}()
		return output
	}

	resultCh := Single(ctx, core.ToChanManyResults(ctx, input), processor, handlers, onSuccess)

	var results []int
	for result := range resultCh {
		if result.IsSuccess() {
			results = append(results, result.Result())
		}
	}

	if len(results) != len(input) {
		t.Errorf("Expected %d results, got %d", len(input), len(results))
	}

	// Single should maintain order even with different processing times
	mu.Lock()
	for i, expected := range input {
		if i < len(processOrder) && processOrder[i] != expected {
			t.Errorf("Expected sequential order %v, got %v", input, processOrder)
			break
		}
	}
	mu.Unlock()
}

// Test Validate function with custom cancellation handler
func TestValidate_WithCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	var cancelCalled bool
	var mu sync.Mutex

	onCancel := func(ctx context.Context, in rop.Result[int]) {
		mu.Lock()
		cancelCalled = true
		mu.Unlock()
	}

	validator := Validate(func(ctx context.Context, in int) (bool, string) {
		return in > 0, "value must be positive"
	}, onCancel)

	// Test with valid input
	input := rop.Success(5)
	resultCh := validator(ctx, input)

	select {
	case result := <-resultCh:
		if !result.IsSuccess() {
			t.Errorf("Expected success for valid input, got error: %v", result.Err())
		}
		if result.Result() != 5 {
			t.Errorf("Expected 5, got %d", result.Result())
		}
	case <-ctx.Done():
		t.Error("Test timed out")
	}

	// Test with invalid input
	input = rop.Success(-1)
	resultCh = validator(ctx, input)

	select {
	case result := <-resultCh:
		if result.IsSuccess() {
			t.Error("Expected validation to fail for negative value")
		}
	case <-ctx.Done():
		t.Error("Test timed out")
	}

	// Cancellation handler should not be called in normal operation
	mu.Lock()
	if cancelCalled {
		t.Error("Cancel handler should not be called in normal operation")
	}
	mu.Unlock()
}

// Test Switch function with custom cancellation
func TestSwitch_WithCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	var cancelCalled bool
	var mu sync.Mutex

	onCancel := func(ctx context.Context, in rop.Result[int]) {
		mu.Lock()
		cancelCalled = true
		mu.Unlock()
	}

	switcher := Switch(func(ctx context.Context, r int) rop.Result[string] {
		if r < 0 {
			return rop.Fail[string](errors.New("negative not allowed"))
		}
		if r%2 == 0 {
			return rop.Success("even")
		}
		return rop.Success("odd")
	}, onCancel)

	// Test with success case
	input := rop.Success(4)
	resultCh := switcher(ctx, input)

	select {
	case result := <-resultCh:
		if !result.IsSuccess() {
			t.Errorf("Expected success, got error: %v", result.Err())
		}
		if result.Result() != "even" {
			t.Errorf("Expected 'even', got %s", result.Result())
		}
	case <-ctx.Done():
		t.Error("Test timed out")
	}

	// Test with error case
	input = rop.Success(-1)
	resultCh = switcher(ctx, input)

	select {
	case result := <-resultCh:
		if result.IsSuccess() {
			t.Error("Expected error for negative input")
		}
		if result.Err().Error() != "negative not allowed" {
			t.Errorf("Expected 'negative not allowed', got: %v", result.Err())
		}
	case <-ctx.Done():
		t.Error("Test timed out")
	}

	mu.Lock()
	if cancelCalled {
		t.Error("Cancel handler should not be called in normal operation")
	}
	mu.Unlock()
}

// Test Map function with custom cancellation
func TestMap_WithCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	var cancelCalled bool
	var mu sync.Mutex

	onCancel := func(ctx context.Context, in rop.Result[int]) {
		mu.Lock()
		cancelCalled = true
		mu.Unlock()
	}

	mapper := Map(func(ctx context.Context, r int) string {
		return fmt.Sprintf("mapped_%d", r*2)
	}, onCancel)

	input := rop.Success(3)
	resultCh := mapper(ctx, input)

	select {
	case result := <-resultCh:
		if !result.IsSuccess() {
			t.Errorf("Expected success, got error: %v", result.Err())
		}
		expected := "mapped_6"
		if result.Result() != expected {
			t.Errorf("Expected %s, got %s", expected, result.Result())
		}
	case <-ctx.Done():
		t.Error("Test timed out")
	}

	mu.Lock()
	if cancelCalled {
		t.Error("Cancel handler should not be called in normal operation")
	}
	mu.Unlock()
}

// Test Tee function with side effects and cancellation
func TestTee_WithCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	var sideEffectValue int
	var cancelCalled bool
	var mu sync.Mutex

	onCancel := func(ctx context.Context, in rop.Result[int]) {
		mu.Lock()
		cancelCalled = true
		mu.Unlock()
	}

	tee := Tee(func(ctx context.Context, r rop.Result[int]) {
		if r.IsSuccess() {
			mu.Lock()
			sideEffectValue = r.Result() * 10
			mu.Unlock()
		}
	}, onCancel)

	input := rop.Success(7)
	resultCh := tee(ctx, input)

	select {
	case result := <-resultCh:
		if !result.IsSuccess() {
			t.Errorf("Expected success, got error: %v", result.Err())
		}
		if result.Result() != 7 {
			t.Errorf("Expected input value unchanged: %d, got %d", 7, result.Result())
		}
		// Check side effect
		mu.Lock()
		if sideEffectValue != 70 {
			t.Errorf("Expected side effect value 70, got %d", sideEffectValue)
		}
		if cancelCalled {
			t.Error("Cancel handler should not be called in normal operation")
		}
		mu.Unlock()
	case <-ctx.Done():
		t.Error("Test timed out")
	}
}

// Test DoubleMap function with custom cancellation (mirrors Map test)
func TestDoubleMap_WithCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	var cancelCalled bool
	var mu sync.Mutex

	onCancel := func(ctx context.Context, in rop.Result[int]) {
		mu.Lock()
		cancelCalled = true
		mu.Unlock()
	}

	mapper := DoubleMap[int, string](
		func(ctx context.Context, r int) string { return fmt.Sprintf("mapped_%d", r*2) },
		func(ctx context.Context, err error) string { return "err:" + err.Error() },
		func(ctx context.Context, err error) string { return "cancel:" + err.Error() },
		onCancel,
	)

	// success case
	input := rop.Success(3)
	resultCh := mapper(ctx, input)
	select {
	case result := <-resultCh:
		if !result.IsSuccess() {
			t.Errorf("Expected success, got error: %v", result.Err())
		}
		expected := "mapped_6"
		if result.Result() != expected {
			t.Errorf("Expected %s, got %s", expected, result.Result())
		}
	case <-ctx.Done():
		t.Error("Test timed out")
	}

	// error case
	errInput := rop.Fail[int](errors.New("boom"))
	resultCh = mapper(ctx, errInput)
	select {
	case result := <-resultCh:
		if result.IsSuccess() {
			t.Errorf("Expected error mapping to pass through error, got success: %v", result.Result())
		}
		if result.Err().Error() != "boom" {
			t.Errorf("Expected error 'boom', got: %v", result.Err())
		}
	case <-ctx.Done():
		t.Error("Test timed out")
	}

	// cancel should not be called in normal success/error mapping
	mu.Lock()
	if cancelCalled {
		t.Error("Cancel handler should not be called in normal operation")
	}
	mu.Unlock()
}

// Test DoubleTee function with side effects and cancellation (mirrors Tee test)
func TestDoubleTee_WithCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	var sideEffectValue int
	var sideEffectErr string
	var cancelCalled bool
	var mu sync.Mutex

	onCancel := func(ctx context.Context, in rop.Result[string]) {
		mu.Lock()
		cancelCalled = true
		mu.Unlock()
	}

	tee := DoubleTee[string](
		func(ctx context.Context, r string) {
			mu.Lock()
			sideEffectValue = len(r)
			mu.Unlock()
		},
		func(ctx context.Context, err error) {
			mu.Lock()
			sideEffectErr = "error:" + err.Error()
			mu.Unlock()
		},
		func(ctx context.Context, err error) {
			// no-op for cancel path in this test
		},
		onCancel,
	)

	// success case
	input := rop.Success("abc")
	resultCh := tee(ctx, input)
	select {
	case result := <-resultCh:
		if !result.IsSuccess() {
			t.Errorf("Expected success, got error: %v", result.Err())
		}
		if result.Result() != "abc" {
			t.Errorf("Expected input value unchanged: %s, got %s", "abc", result.Result())
		}
		mu.Lock()
		if sideEffectValue != 3 {
			t.Errorf("Expected side effect length 3, got %d", sideEffectValue)
		}
		mu.Unlock()
	case <-ctx.Done():
		t.Error("Test timed out")
	}

	// error case
	errInput := rop.Fail[string](errors.New("tee_err"))
	resultCh = tee(ctx, errInput)
	<-resultCh // consume
	mu.Lock()
	if sideEffectErr != "error:tee_err" {
		t.Errorf("Expected error side effect 'error:tee_err', got '%s'", sideEffectErr)
	}
	mu.Unlock()

	// cancel handler should not be called in normal operation
	mu.Lock()
	if cancelCalled {
		t.Error("Cancel handler should not be called in normal operation")
	}
	mu.Unlock()
}

// Test Try function with custom cancellation
func TestTry_WithCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	var cancelCalled bool
	var mu sync.Mutex

	onCancel := func(ctx context.Context, in rop.Result[int]) {
		mu.Lock()
		cancelCalled = true
		mu.Unlock()
	}

	tryFn := Try[int, string](func(ctx context.Context, r int) (string, error) {
		if r > 0 {
			return fmt.Sprintf("processed_%d", r), nil
		}
		return "", errors.New("invalid")
	}, onCancel)

	// success case
	input := rop.Success(2)
	resultCh := tryFn(ctx, input)
	select {
	case result := <-resultCh:
		if !result.IsSuccess() {
			t.Errorf("Expected success, got error: %v", result.Err())
		}
		if result.Result() != "processed_2" {
			t.Errorf("Unexpected result: %s", result.Result())
		}
	case <-ctx.Done():
		t.Error("Test timed out")
	}

	// error case
	errCh := tryFn(ctx, rop.Success(0))
	res := <-errCh
	if res.IsSuccess() {
		t.Errorf("Expected failure result")
	}

	mu.Lock()
	if cancelCalled {
		t.Error("Cancel handler should not be called in normal operation")
	}
	mu.Unlock()
}

// Test Finally function with comprehensive handlers (replace Finally2)
func TestFinally_WithCancellationHandlers(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var successResults []string
	var onSuccessCount int
	var mu sync.Mutex

	handlers := mass.FinallyHandlers[int, string]{
		OnSuccess: func(ctx context.Context, in int) string {
			return fmt.Sprintf("success_%d", in)
		},
		OnError: func(ctx context.Context, err error) string {
			return fmt.Sprintf("error_%s", err.Error())
		},
		OnCancel: func(ctx context.Context, err error) string {
			return "cancelled"
		},
	}

	cancelHandlers := mass.FinallyCancelHandlers[int, string]{
		OnBreak: func(ctx context.Context, in rop.Result[int]) string {
			return "broken"
		},
		OnCancelValue: func(ctx context.Context, in rop.Result[int], brokenF func(ctx context.Context, in rop.Result[int]) string, outCh chan<- string) {
			outCh <- brokenF(ctx, in)
		},
	}

	onSuccessResult := func(ctx context.Context, out string) {
		mu.Lock()
		onSuccessCount++
		successResults = append(successResults, out)
		mu.Unlock()
	}

	// Create input channel with mixed results
	inputCh := make(chan rop.Result[int], 4)
	inputCh <- rop.Success(1)
	inputCh <- rop.Success(2)
	inputCh <- rop.Fail[int](errors.New("test_error"))
	inputCh <- rop.Cancel[int](errors.New("cancelled"))
	close(inputCh)

	resultCh := Finally(ctx, inputCh, handlers, cancelHandlers, onSuccessResult)

	var results []string
	for result := range resultCh {
		results = append(results, result)
	}

	if len(results) != 4 {
		t.Errorf("Expected 4 results, got %d", len(results))
	}

	// Check that we have expected result types
	foundSuccess := 0
	foundError := false
	foundCancel := false

	for _, result := range results {
		switch {
		case result == "success_1" || result == "success_2":
			foundSuccess++
		case result == "error_test_error":
			foundError = true
		case result == "cancelled":
			foundCancel = true
		}
	}

	if foundSuccess != 2 {
		t.Errorf("Expected 2 success results, got %d", foundSuccess)
	}
	if !foundError {
		t.Error("Expected to find error result")
	}
	if !foundCancel {
		t.Error("Expected to find cancel result")
	}

	// Check success callback was called
	mu.Lock()
	if onSuccessCount != 4 {
		t.Errorf("Expected 4 success callbacks, got %d", onSuccessCount)
	}
	mu.Unlock()
}

// Test cancellation functions from cancels.go

// Test CancelRemainingResults function
func TestCancelRemainingResults(t *testing.T) {
	t.Parallel()

	ctx := core.WithProcessOptions(context.Background(), true)

	// Create input channel with mixed results
	inputCh := make(chan rop.Result[int], 3)
	inputCh <- rop.Success(1)
	inputCh <- rop.Fail[int](errors.New("test_error"))
	inputCh <- rop.Cancel[int](errors.New("existing_cancel"))
	close(inputCh)

	outputCh := make(chan rop.Result[string], 3)

	go func() {
		defer close(outputCh)
		CancelRemainingResults[int, string](ctx, inputCh, outputCh)
	}()

	var results []rop.Result[string]
	for result := range outputCh {
		results = append(results, result)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	// All results should be cancelled
	for i, result := range results {
		if !result.IsCancel() {
			t.Errorf("Result %d should be cancelled", i)
		}
	}

	// Check that existing cancel is preserved
	if results[2].Err().Error() != "existing_cancel" {
		t.Errorf("Expected existing cancel error to be preserved, got: %v", results[2].Err())
	}
}

// Test remaining cancellation functions and integration test
func TestCancelRemainingResults_Disabled(t *testing.T) {
	t.Parallel()

	ctx := core.WithProcessOptions(context.Background(), false)

	inputCh := make(chan rop.Result[int], 2)
	inputCh <- rop.Success(1)
	inputCh <- rop.Success(2)
	close(inputCh)

	outputCh := make(chan rop.Result[string], 2)

	go func() {
		defer close(outputCh)
		CancelRemainingResults[int, string](ctx, inputCh, outputCh)
	}()

	var results []rop.Result[string]
	for result := range outputCh {
		results = append(results, result)
	}

	// Should be empty when ProcessRemaining is disabled
	if len(results) != 0 {
		t.Errorf("Expected no results when ProcessRemaining is disabled, got %d", len(results))
	}
}

func TestCancelRemainingResult(t *testing.T) {
	t.Parallel()

	ctx := core.WithProcessOptions(context.Background(), true)
	outputCh := make(chan rop.Result[string], 1)

	// Test with success result
	input := rop.Success(42)
	go func() {
		defer close(outputCh)
		CancelRemainingResult[int, string](ctx, input, outputCh)
	}()

	select {
	case result := <-outputCh:
		if !result.IsCancel() {
			t.Error("Expected result to be cancelled")
		}
		if result.Err() != ErrCancelled {
			t.Errorf("Expected ErrCancelled, got: %v", result.Err())
		}
	case <-time.After(1 * time.Second):
		t.Error("Test timed out")
	}
}

func TestErrCancelled(t *testing.T) {
	t.Parallel()

	if ErrCancelled == nil {
		t.Error("ErrCancelled should not be nil")
	}

	if ErrCancelled.Error() != "operation cancelled" {
		t.Errorf("Expected 'operation cancelled', got: %s", ErrCancelled.Error())
	}
}

// Integration test combining multiple custom features
func TestCustom_Integration(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	ctx = core.WithProcessOptions(core.WithWorkerOptions(ctx, 2), true)
	workers := core.GetWorkerMaxCount(ctx, 5)

	source := []int{10, 5, 1, 20, 2}
	var successCount int
	var mu sync.Mutex

	// Custom cancellation handlers
	handlers := core.CancellationHandlers[int, int]{
		OnCancel:            CancelRemainingResults[int, int],
		OnCancelUnprocessed: CancelRemainingResult[int, int],
		OnCancelProcessed: func(ctx context.Context, in rop.Result[int], processed rop.Result[int], outCh chan<- rop.Result[int]) {
			outCh <- processed
		},
	}

	onSuccess := func(ctx context.Context, in rop.Result[int]) {
		if in.IsSuccess() {
			mu.Lock()
			successCount++
			mu.Unlock()
		}
	}

	finallyHandlers := mass.FinallyHandlers[int, int]{
		OnSuccess: func(ctx context.Context, in int) int {
			return in
		},
		OnError: func(ctx context.Context, err error) int {
			return -1
		},
		OnCancel: func(ctx context.Context, err error) int {
			return -2
		},
	}

	finallyCancelHandlers := mass.FinallyCancelHandlers[int, int]{
		OnBreak: func(ctx context.Context, in rop.Result[int]) int {
			return -5
		},
		OnCancelValue:   CancelRemainingValue[int, int],
		OnCancelValues:  CancelRemainingValues[int, int],
		OnCancelResult:  CancelResult[int],
		OnCancelResults: CancelResults[int],
	}

	resultCh := Finally(
		ctx,
		Turnout[int, int](
			ctx,
			Run(
				ctx,
				core.ToChanManyResults(ctx, source),
				Validate(
					func(ctx context.Context, in int) (bool, string) {
						if in != 1 {
							return true, ""
						}
						return false, "value should not be 1"
					},
					func(ctx context.Context, in rop.Result[int]) {
						// validation cancel handler
					}),
				handlers,
				onSuccess,
				workers),
			Switch[int, int](
				func(ctx context.Context, r int) rop.Result[int] {
					return rop.Success[int](r + 1000)
				},
				func(ctx context.Context, in rop.Result[int]) {
					// switch cancel handler
				}),
			handlers,
			onSuccess,
			2),
		finallyHandlers,
		finallyCancelHandlers,
		func(ctx context.Context, out int) {
			// onSuccessResult callback
		},
	)

	var results []int
	for result := range resultCh {
		results = append(results, result)
	}

	// Should have 4 successful results (excluding the value 1 which fails validation)
	successResults := 0
	for _, result := range results {
		if result > 1000 {
			successResults++
		}
	}

	if successResults != 4 {
		t.Errorf("Expected 4 successful results, got %d", successResults)
	}

	// Check that validation failure is handled (should get -1 for fail)
	foundFailure := false
	for _, result := range results {
		if result == -1 {
			foundFailure = true
			break
		}
	}

	if !foundFailure {
		t.Error("Expected to find validation failure result (-1)")
	}
}

// Benchmark tests for custom functions
func BenchmarkProcess_WithHandlers(b *testing.B) {
	ctx := context.Background()
	input := make([]int, 1000)
	for i := range input {
		input[i] = i
	}

	handlers := core.CancellationHandlers[int, int]{}
	onSuccess := func(ctx context.Context, in rop.Result[int]) {}

	processor := func(ctx context.Context, input rop.Result[int]) <-chan rop.Result[int] {
		output := make(chan rop.Result[int], 1)
		go func() {
			defer close(output)
			if input.IsSuccess() {
				output <- rop.Success(input.Result() * 2)
			}
		}()
		return output
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resultCh := Run(ctx, core.ToChanManyResults(ctx, input), processor, handlers, onSuccess, 4)
		for range resultCh {
			// Consume all results
		}
	}
}

// Additional tests for missing cancellation functions
func TestCancelRemainingValue(t *testing.T) {
	t.Parallel()

	ctx := core.WithProcessOptions(context.Background(), true)
	outputCh := make(chan string, 1)

	brokenF := func(ctx context.Context, in rop.Result[int]) string {
		if in.IsSuccess() {
			return fmt.Sprintf("broken_%d", in.Result())
		}
		return "broken_error"
	}

	input := rop.Success(10)
	go func() {
		defer close(outputCh)
		CancelRemainingValue[int, string](ctx, input, brokenF, outputCh)
	}()

	select {
	case result := <-outputCh:
		expected := "broken_10"
		if result != expected {
			t.Errorf("Expected %s, got %s", expected, result)
		}
	case <-time.After(1 * time.Second):
		t.Error("Test timed out")
	}
}

func TestCancelResult(t *testing.T) {
	t.Parallel()

	ctx := core.WithProcessOptions(context.Background(), true)
	outputCh := make(chan int, 1)

	go func() {
		defer close(outputCh)
		CancelResult(ctx, 100, outputCh)
	}()

	select {
	case result := <-outputCh:
		if result != 100 {
			t.Errorf("Expected 100, got %d", result)
		}
	case <-time.After(1 * time.Second):
		t.Error("Test timed out")
	}
}

func TestCancelResults(t *testing.T) {
	t.Parallel()

	ctx := core.WithProcessOptions(context.Background(), true)

	inputCh := make(chan int, 3)
	inputCh <- 1
	inputCh <- 2
	inputCh <- 3
	close(inputCh)

	outputCh := make(chan int, 3)

	go func() {
		defer close(outputCh)
		CancelResults(ctx, inputCh, outputCh)
	}()

	var results []int
	for result := range outputCh {
		results = append(results, result)
	}

	expected := []int{1, 2, 3}
	if len(results) != len(expected) {
		t.Errorf("Expected %d results, got %d", len(expected), len(results))
	}

	for i, exp := range expected {
		if i < len(results) && results[i] != exp {
			t.Errorf("Expected %d at position %d, got %d", exp, i, results[i])
		}
	}
}

func TestCancelRemainingValues(t *testing.T) {
	t.Parallel()

	ctx := core.WithProcessOptions(context.Background(), true)

	inputCh := make(chan rop.Result[int], 3)
	inputCh <- rop.Success(1)
	inputCh <- rop.Fail[int](errors.New("error"))
	inputCh <- rop.Cancel[int](errors.New("cancel"))
	close(inputCh)

	outputCh := make(chan string, 3)

	brokenF := func(ctx context.Context, in rop.Result[int]) string {
		if in.IsSuccess() {
			return fmt.Sprintf("value_%d", in.Result())
		} else if in.IsCancel() {
			return "cancelled"
		} else {
			return "failed"
		}
	}

	go func() {
		defer close(outputCh)
		CancelRemainingValues[int, string](ctx, inputCh, brokenF, outputCh)
	}()

	var results []string
	for result := range outputCh {
		results = append(results, result)
	}

	expected := []string{"value_1", "failed", "cancelled"}
	if len(results) != len(expected) {
		t.Errorf("Expected %d results, got %d", len(expected), len(results))
	}

	for i, exp := range expected {
		if i < len(results) && results[i] != exp {
			t.Errorf("Expected %s at position %d, got %s", exp, i, results[i])
		}
	}
}

// Test CancelRemainingResult with cancel input
func TestCancelRemainingResult_WithCancelInput(t *testing.T) {
	t.Parallel()

	ctx := core.WithProcessOptions(context.Background(), true)
	outputCh := make(chan rop.Result[string], 1)

	// Test with cancel result that should be preserved
	originalErr := errors.New("original_cancel")
	input := rop.Cancel[int](originalErr)
	go func() {
		defer close(outputCh)
		CancelRemainingResult[int, string](ctx, input, outputCh)
	}()

	select {
	case result := <-outputCh:
		if !result.IsCancel() {
			t.Error("Expected result to be cancelled")
		}
		if result.Err() != originalErr {
			t.Errorf("Expected original error to be preserved, got: %v", result.Err())
		}
	case <-time.After(1 * time.Second):
		t.Error("Test timed out")
	}
}

func BenchmarkTransform_WithHandlers(b *testing.B) {
	ctx := context.Background()
	input := make([]int, 1000)
	for i := range input {
		input[i] = i
	}

	handlers := core.CancellationHandlers[int, string]{}
	onSuccess := func(ctx context.Context, in rop.Result[string]) {}

	processor := func(ctx context.Context, input rop.Result[int]) <-chan rop.Result[string] {
		output := make(chan rop.Result[string], 1)
		go func() {
			defer close(output)
			if input.IsSuccess() {
				output <- rop.Success(fmt.Sprintf("str_%d", input.Result()))
			}
		}()
		return output
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resultCh := Turnout(ctx, core.ToChanManyResults(ctx, input), processor, handlers, onSuccess, 4)
		for range resultCh {
			// Consume all results
		}
	}
}
