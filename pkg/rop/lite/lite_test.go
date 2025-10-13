package lite

import (
	"context"
	"errors"
	"fmt"
	"rop2/pkg/rop"
	"rop2/pkg/rop/core"
	"rop2/pkg/rop/mass"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func Test_Parallel(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	source := []int{10, 5, 1, 20, 2}

	start := time.Now()
	ctx = core.WithProcessOptions(core.WithWorkerOptions(ctx, 2), true)

	workers := core.GetWorkerMaxCount(ctx, 5)

	finallyHandlers := mass.FinallyHandlers[int, int]{
		OnSuccess: func(ctx context.Context, in int) int {
			return in
		},
		OnError: func(ctx context.Context, err error) int {
			//time.Sleep(20 * time.Second)
			return -1
		},
		OnCancel: func(ctx context.Context, err error) int {
			return -2
		},
	}

	ch := Finally(
		ctx,
		Turnout(
			ctx,
			Run(
				ctx,
				core.ToChanManyResults[int](
					ctx,
					source),
				Validate(
					func(ctx context.Context, in int) (valid bool, errMsg string) {
						//time.Sleep(1 * time.Second)
						if in != 1 {
							return true, ""
						}
						return false, "value should not be 1"
					}),
				workers),
			Switch(
				func(ctx context.Context, r int) rop.Result[int] {
					return rop.Success[int](r + 1000)
				},
			), 2),
		finallyHandlers)

	//for _, v := range pool.FromChanMany(ctx, ch) {
	//	fmt.Println(">>>> Finally: ", v)
	//}

	for v := range ch {
		fmt.Println(">>>> Finally: ", v)
	}

	fmt.Println("Duration: ", time.Since(start))
}

// Test Run function with single worker
func TestProcess_SingleWorker(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Create test data
	input := []int{1, 2, 3, 4, 5}
	expected := []int{2, 4, 6, 8, 10}

	// Create processor that doubles the input
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

	// Run with single worker
	resultCh := Run(ctx, core.ToChanManyResults(ctx, input), processor, 1)

	// Collect results
	var results []int
	for result := range resultCh {
		if result.IsSuccess() {
			results = append(results, result.Result())
		} else {
			t.Errorf("Unexpected error: %v", result.Err())
		}
	}

	// Verify results
	if len(results) != len(expected) {
		t.Errorf("Expected %d results, got %d", len(expected), len(results))
	}

	// Results might not be in order due to concurrency
	resultMap := make(map[int]bool)
	for _, r := range results {
		resultMap[r] = true
	}

	for _, exp := range expected {
		if !resultMap[exp] {
			t.Errorf("Expected result %d not found", exp)
		}
	}
}

// Test Run function with multiple workers
func TestProcess_MultipleWorkers(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	input := make([]int, 100)
	for i := range input {
		input[i] = i + 1
	}

	// Processor with slight delay to test concurrency
	processor := func(ctx context.Context, input rop.Result[int]) <-chan rop.Result[int] {
		output := make(chan rop.Result[int], 1)
		go func() {
			defer close(output)
			time.Sleep(10 * time.Millisecond) // Small delay
			if input.IsSuccess() {
				output <- rop.Success(input.Result() * 2)
			} else {
				output <- input
			}
		}()
		return output
	}

	start := time.Now()
	resultCh := Run(ctx, core.ToChanManyResults(ctx, input), processor, 5)

	var results []int
	for result := range resultCh {
		if result.IsSuccess() {
			results = append(results, result.Result())
		}
	}

	duration := time.Since(start)
	if len(results) != len(input) {
		t.Errorf("Expected %d results, got %d", len(input), len(results))
	}

	// With 5 workers, should be faster than single worker
	if duration > 1*time.Second {
		t.Errorf("Processing took too long: %v", duration)
	}
}

// Test Run with context cancellation
func TestProcess_ContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())

	input := make([]int, 10)
	for i := range input {
		input[i] = i + 1
	}

	processedCount := 0
	var mu sync.Mutex

	processor := func(ctx context.Context, input rop.Result[int]) <-chan rop.Result[int] {
		output := make(chan rop.Result[int], 1)
		go func() {
			defer close(output)
			time.Sleep(100 * time.Millisecond) // Delay to allow cancellation
			select {
			case <-ctx.Done():
				return
			default:
				mu.Lock()
				processedCount++
				mu.Unlock()
				if input.IsSuccess() {
					output <- rop.Success(input.Result())
				}
			}
		}()
		return output
	}

	resultCh := Run(ctx, core.ToChanManyResults(ctx, input), processor, 3)

	// Cancel after short delay
	go func() {
		time.Sleep(150 * time.Millisecond)
		cancel()
	}()

	var results []int
	for result := range resultCh {
		if result.IsSuccess() {
			results = append(results, result.Result())
		}
	}

	// Should have processed fewer items due to cancellation
	if len(results) >= len(input) {
		t.Errorf("Expected cancellation to stop processing, but got %d results", len(results))
	}
}

// Test Turnout function with type conversion
func TestTransform_TypeConversion(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	input := []int{1, 2, 3, 4, 5}

	// Convert int to string
	processor := func(ctx context.Context, input rop.Result[int]) <-chan rop.Result[string] {
		output := make(chan rop.Result[string], 1)
		go func() {
			defer close(output)
			if input.IsSuccess() {
				output <- rop.Success(fmt.Sprintf("num_%d", input.Result()))
			} else {
				output <- rop.Fail[string](input.Err())
			}
		}()
		return output
	}

	resultCh := Turnout(ctx, core.ToChanManyResults(ctx, input), processor, 2)

	var results []string
	for result := range resultCh {
		if result.IsSuccess() {
			results = append(results, result.Result())
		} else {
			t.Errorf("Unexpected error: %v", result.Err())
		}
	}

	if len(results) != len(input) {
		t.Errorf("Expected %d results, got %d", len(input), len(results))
	}

	// Check that all results are properly formatted
	for _, result := range results {
		if len(result) < 5 || result[:4] != "num_" {
			t.Errorf("Invalid result format: %s", result)
		}
	}
}

// Test Validate function with valid inputs
func TestValidate_ValidInputs(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	validator := Validate(func(ctx context.Context, in int) (bool, string) {
		return in > 0, "value must be positive"
	})

	input := rop.Success(5)
	resultCh := validator(ctx, input)

	select {
	case result := <-resultCh:
		if !result.IsSuccess() {
			t.Errorf("Expected success, got error: %v", result.Err())
		}
		if result.Result() != 5 {
			t.Errorf("Expected 5, got %d", result.Result())
		}
	case <-ctx.Done():
		t.Error("Test timed out")
	}
}

// Test Validate function with invalid inputs
func TestValidate_InvalidInputs(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	validator := Validate(func(ctx context.Context, in int) (bool, string) {
		return in > 0, "value must be positive"
	})

	input := rop.Success(-5)
	resultCh := validator(ctx, input)

	select {
	case result := <-resultCh:
		if result.IsSuccess() {
			t.Error("Expected validation to fail for negative value")
		}
		if result.Err() == nil {
			t.Error("Expected error message")
		}
	case <-ctx.Done():
		t.Error("Test timed out")
	}
}

// Test Switch function
func TestSwitch_Success(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	switcher := Switch(func(ctx context.Context, r int) rop.Result[string] {
		if r%2 == 0 {
			return rop.Success("even")
		}
		return rop.Success("odd")
	})

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
}

// Test Switch function with error
func TestSwitch_WithError(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	switcher := Switch(func(ctx context.Context, r int) rop.Result[string] {
		if r < 0 {
			return rop.Fail[string](errors.New("negative number"))
		}
		return rop.Success("positive")
	})

	input := rop.Success(-1)
	resultCh := switcher(ctx, input)

	select {
	case result := <-resultCh:
		if result.IsSuccess() {
			t.Error("Expected error for negative input")
		}
		if result.Err().Error() != "negative number" {
			t.Errorf("Expected 'negative number' error, got: %v", result.Err())
		}
	case <-ctx.Done():
		t.Error("Test timed out")
	}
}

// Test Map function
func TestMap_SimpleTransformation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	mapper := Map(func(ctx context.Context, r int) string {
		return fmt.Sprintf("mapped_%d", r*2)
	})

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
}

// Test DoubleMap function
func TestDoubleMap_AllTransformations(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Create a DoubleMap that handles success, error, and cancellation
	doubleMapper := DoubleMap(
		func(ctx context.Context, r int) string {
			return fmt.Sprintf("success_%d", r*2)
		},
		func(ctx context.Context, err error) string {
			return fmt.Sprintf("error_%s", err.Error())
		},
		func(ctx context.Context, err error) string {
			return fmt.Sprintf("cancel_%s", err.Error())
		},
	)

	// Test success case
	t.Run("Success case", func(t *testing.T) {
		input := rop.Success(5)
		resultCh := doubleMapper(ctx, input)

		select {
		case result := <-resultCh:
			if !result.IsSuccess() {
				t.Errorf("Expected success, got error: %v", result.Err())
			}
			expected := "success_10"
			if result.Result() != expected {
				t.Errorf("Expected %s, got %s", expected, result.Result())
			}
		case <-ctx.Done():
			t.Error("Test timed out")
		}
	})

	// Test error case
	t.Run("Error case", func(t *testing.T) {
		testErr := errors.New("test error")
		input := rop.Fail[int](testErr)
		resultCh := doubleMapper(ctx, input)

		select {
		case result := <-resultCh:
			if result.IsSuccess() {
				t.Errorf("Expected error, got success: %v", result.Result())
			}
			if result.Err().Error() != testErr.Error() {
				t.Errorf("Expected error %v, got %v", testErr, result.Err())
			}
		case <-ctx.Done():
			t.Error("Test timed out")
		}
	})

	// Test cancel case
	t.Run("Cancel case", func(t *testing.T) {
		cancelErr := errors.New("operation cancelled")
		input := rop.Cancel[int](cancelErr)
		resultCh := doubleMapper(ctx, input)

		select {
		case result := <-resultCh:
			if !result.IsCancel() {
				t.Errorf("Expected cancel, got: %v", result)
			}
			if result.Err().Error() != cancelErr.Error() {
				t.Errorf("Expected error %v, got %v", cancelErr, result.Err())
			}
		case <-ctx.Done():
			t.Error("Test timed out")
		}
	})
}

// Test Tee function (side effects)
func TestTee_SideEffect(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	var sideEffectValue int
	var mu sync.Mutex

	tee := Tee(func(ctx context.Context, r rop.Result[int]) {
		if r.IsSuccess() {
			mu.Lock()
			sideEffectValue = r.Result() * 10
			mu.Unlock()
		}
	})

	input := rop.Success(5)
	resultCh := tee(ctx, input)

	select {
	case result := <-resultCh:
		if !result.IsSuccess() {
			t.Errorf("Expected success, got error: %v", result.Err())
		}
		if result.Result() != 5 {
			t.Errorf("Expected input value unchanged: %d, got %d", 5, result.Result())
		}
		// Check side effect
		mu.Lock()
		if sideEffectValue != 50 {
			t.Errorf("Expected side effect value 50, got %d", sideEffectValue)
		}
		mu.Unlock()
	case <-ctx.Done():
		t.Error("Test timed out")
	}
}

// Test DoubleTee function (side effects for success, error, and cancel)
func TestDoubleTee_AllSideEffects(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	var successValue int
	var errorMsg string
	var cancelMsg string
	var mu sync.Mutex

	doubleTee := DoubleTee(
		func(ctx context.Context, r int) {
			mu.Lock()
			successValue = r * 10
			mu.Unlock()
		},
		func(ctx context.Context, err error) {
			mu.Lock()
			errorMsg = "error:" + err.Error()
			mu.Unlock()
		},
		func(ctx context.Context, err error) {
			mu.Lock()
			cancelMsg = "cancel:" + err.Error()
			mu.Unlock()
		},
	)

	// Test success case
	t.Run("Success case", func(t *testing.T) {
		mu.Lock()
		successValue = 0
		mu.Unlock()

		input := rop.Success(7)
		resultCh := doubleTee(ctx, input)

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
			if successValue != 70 {
				t.Errorf("Expected side effect value 70, got %d", successValue)
			}
			mu.Unlock()
		case <-ctx.Done():
			t.Error("Test timed out")
		}
	})

	// Test error case
	t.Run("Error case", func(t *testing.T) {
		mu.Lock()
		errorMsg = ""
		mu.Unlock()

		testErr := errors.New("test error")
		input := rop.Fail[int](testErr)
		resultCh := doubleTee(ctx, input)

		select {
		case result := <-resultCh:
			if result.IsSuccess() {
				t.Errorf("Expected error, got success: %v", result.Result())
			}
			// Check side effect
			mu.Lock()
			expected := "error:test error"
			if errorMsg != expected {
				t.Errorf("Expected side effect message '%s', got '%s'", expected, errorMsg)
			}
			mu.Unlock()
		case <-ctx.Done():
			t.Error("Test timed out")
		}
	})

	// Test cancel case
	t.Run("Cancel case", func(t *testing.T) {
		mu.Lock()
		cancelMsg = ""
		mu.Unlock()

		cancelErr := errors.New("operation cancelled")
		input := rop.Cancel[int](cancelErr)
		resultCh := doubleTee(ctx, input)

		select {
		case result := <-resultCh:
			if !result.IsCancel() {
				t.Errorf("Expected cancel, got: %v", result)
			}
			// Check side effect
			mu.Lock()
			expected := "cancel:operation cancelled"
			if cancelMsg != expected {
				t.Errorf("Expected side effect message '%s', got '%s'", expected, cancelMsg)
			}
			mu.Unlock()
		case <-ctx.Done():
			t.Error("Test timed out")
		}
	})
}

// Test Try function
func TestTry_SuccessAndError(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Create a Try function that might succeed or fail
	tryFn := Try(func(ctx context.Context, r int) (string, error) {
		if r > 0 {
			return fmt.Sprintf("processed_%d", r), nil
		}
		return "", fmt.Errorf("cannot process non-positive number: %d", r)
	})

	// Test success case
	t.Run("Success case", func(t *testing.T) {
		input := rop.Success(5)
		resultCh := tryFn(ctx, input)

		select {
		case result := <-resultCh:
			if !result.IsSuccess() {
				t.Errorf("Expected success, got error: %v", result.Err())
			}
			expected := "processed_5"
			if result.Result() != expected {
				t.Errorf("Expected %s, got %s", expected, result.Result())
			}
		case <-ctx.Done():
			t.Error("Test timed out")
		}
	})

	// Test error case
	t.Run("Error case", func(t *testing.T) {
		input := rop.Success(-3)
		resultCh := tryFn(ctx, input)

		select {
		case result := <-resultCh:
			if result.IsSuccess() {
				t.Errorf("Expected error, got success: %v", result.Result())
			}
			expectedErrMsg := "cannot process non-positive number: -3"
			if result.Err().Error() != expectedErrMsg {
				t.Errorf("Expected error message '%s', got '%s'", expectedErrMsg, result.Err().Error())
			}
		case <-ctx.Done():
			t.Error("Test timed out")
		}
	})

	// Test with already failed input
	t.Run("Already failed input", func(t *testing.T) {
		originalErr := errors.New("original error")
		input := rop.Fail[int](originalErr)
		resultCh := tryFn(ctx, input)

		select {
		case result := <-resultCh:
			if result.IsSuccess() {
				t.Errorf("Expected error to be passed through, got success: %v", result.Result())
			}
			if result.Err() != originalErr {
				t.Errorf("Expected original error to be passed through, got: %v", result.Err())
			}
		case <-ctx.Done():
			t.Error("Test timed out")
		}
	})
}

// Test Finally function
func TestFinally_DirectUsage(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Create input channel with mixed results
	inputCh := make(chan rop.Result[int], 3)
	inputCh <- rop.Success(10)
	inputCh <- rop.Fail[int](errors.New("test error"))
	inputCh <- rop.Cancel[int](errors.New("test cancel"))
	close(inputCh)

	// Define handlers
	handlers := mass.FinallyHandlers[int, string]{
		OnSuccess: func(ctx context.Context, in int) string {
			return fmt.Sprintf("success:%d", in)
		},
		OnError: func(ctx context.Context, err error) string {
			return fmt.Sprintf("error:%s", err.Error())
		},
		OnCancel: func(ctx context.Context, err error) string {
			return fmt.Sprintf("cancel:%s", err.Error())
		},
	}

	// Use Finally directly
	resultCh := Finally(ctx, inputCh, handlers)

	// Collect results
	var results []string
	for result := range resultCh {
		results = append(results, result)
	}

	// Verify results
	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	// Check that we have the expected results
	expectedResults := map[string]bool{
		"success:10":         false,
		"error:test error":   false,
		"cancel:test cancel": false,
	}

	for _, result := range results {
		if _, exists := expectedResults[result]; !exists {
			t.Errorf("Unexpected result: %s", result)
		} else {
			expectedResults[result] = true
		}
	}

	// Verify all expected results were found
	for result, found := range expectedResults {
		if !found {
			t.Errorf("Expected result not found: %s", result)
		}
	}
}

// Test Finally function with mixed results
func TestFinally_WithErrors(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

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

	// Create input channel with mixed results
	inputCh := make(chan rop.Result[int], 3)
	inputCh <- rop.Success(1)
	inputCh <- rop.Fail[int](errors.New("test_error"))
	inputCh <- rop.Cancel[int](errors.New("cancelled"))
	close(inputCh)

	resultCh := Finally(ctx, inputCh, handlers)

	var results []string
	for result := range resultCh {
		results = append(results, result)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	// Check expected result types
	foundSuccess := false
	foundError := false
	foundCancel := false

	for _, result := range results {
		switch {
		case result == "success_1":
			foundSuccess = true
		case result == "error_test_error":
			foundError = true
		case result == "cancelled":
			foundCancel = true
		}
	}

	if !foundSuccess {
		t.Error("Expected to find success result")
	}
	if !foundError {
		t.Error("Expected to find error result")
	}
	if !foundCancel {
		t.Error("Expected to find cancel result")
	}
}

// Test edge case: empty input
func TestProcess_EmptyInput(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	processor := func(ctx context.Context, input rop.Result[int]) <-chan rop.Result[int] {
		output := make(chan rop.Result[int], 1)
		go func() {
			defer close(output)
			output <- input
		}()
		return output
	}

	// Empty input
	emptyInput := make(chan rop.Result[int])
	close(emptyInput)

	resultCh := Run(ctx, emptyInput, processor, 2)

	var results []int
	for result := range resultCh {
		results = append(results, result.Result())
	}

	if len(results) != 0 {
		t.Errorf("Expected no results for empty input, got %d", len(results))
	}
}

// Comprehensive integration test using all ROP methods in a single pipeline
func Test_CompletePipeline(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Source data
	source := []int{10, 5, 1, 20, 2, -3, 0}

	// Configure context with process options
	ctx = core.WithProcessOptions(core.WithWorkerOptions(ctx, 3), true)
	workers := core.GetWorkerMaxCount(ctx, 5)

	// Track side effects
	var sideEffectCount int
	var processedValues []int
	var mu sync.Mutex

	// Finally handlers to collect final results
	finallyHandlers := mass.FinallyHandlers[string, string]{
		OnSuccess: func(ctx context.Context, in string) string {
			return fmt.Sprintf("success:%s", in)
		},
		OnError: func(ctx context.Context, err error) string {
			return fmt.Sprintf("fail:%s", err.Error())
		},
		OnCancel: func(ctx context.Context, err error) string {
			return fmt.Sprintf("cancel:%s", err.Error())
		},
	}

	// Build the complete pipeline using all ROP methods in proper sequence
	// Stage 1: Run with validation
	stage1 := Run(ctx,
		core.ToChanManyResults[int](ctx, source),
		Validate(func(ctx context.Context, in int) (bool, string) {
			// Validate that input is positive
			if in <= 0 {
				return false, fmt.Sprintf("value %d is not positive", in)
			}
			return true, ""
		}),
		workers)

	// Stage 2: Turnout with Switch (int -> int)
	stage2 := Turnout[int, int](ctx,
		stage1,
		Switch[int, int](func(ctx context.Context, r int) rop.Result[int] {
			// Switch based on value
			if r%2 == 0 {
				return rop.Success(r * 2) // Double even numbers
			} else {
				return rop.Success(r * 3) // Triple odd numbers
			}
		}),
		2)

	// Stage 3: Turnout with DoubleMap (int -> string)
	stage3 := Turnout[int, string](ctx,
		stage2,
		DoubleMap[int, string](
			func(ctx context.Context, r int) string {
				// Map success values
				return fmt.Sprintf("mapped:%d", r)
			},
			func(ctx context.Context, err error) string {
				// Map error values (won't be used in this test)
				return fmt.Sprintf("error-mapped:%s", err.Error())
			},
			func(ctx context.Context, err error) string {
				// Map cancel values (won't be used in this test)
				return fmt.Sprintf("cancel-mapped:%s", err.Error())
			},
		),
		2)

	// Stage 4: Run with DoubleTee (side effects)
	stage4 := Run(ctx,
		stage3,
		DoubleTee[string](
			func(ctx context.Context, r string) {
				// Side effect for success
				mu.Lock()
				sideEffectCount++
				processedValues = append(processedValues, len(r))
				mu.Unlock()
			},
			func(ctx context.Context, err error) {
				// Side effect for error
				mu.Lock()
				processedValues = append(processedValues, -1)
				mu.Unlock()
			},
			func(ctx context.Context, err error) {
				// Side effect for cancel
				mu.Lock()
				processedValues = append(processedValues, -2)
				mu.Unlock()
			},
		),
		workers)

	// Stage 5: Run with Try (string -> string)
	stage5 := Run(ctx,
		stage4,
		Try[string, string](func(ctx context.Context, r string) (string, error) {
			// Try to process the string
			if strings.Contains(r, "mapped:") {
				return "tried:" + r, nil
			}
			return "", errors.New("invalid format")
		}),
		workers)

	// Stage 6: Finally to collect results
	resultCh := Finally(ctx, stage5, finallyHandlers)

	// Collect all final results
	var finalResults []string
	for result := range resultCh {
		finalResults = append(finalResults, result)
		fmt.Printf("Final result: %s\n", result)
	}

	// Verify we got expected number of results (7 inputs, 2 should fail validation)
	if len(finalResults) < 5 {
		t.Errorf("Expected at least 5 results, got %d", len(finalResults))
	}

	// Check that we have success and fail results
	var successCount, failCount int
	for _, result := range finalResults {
		if strings.HasPrefix(result, "success:") {
			successCount++
		} else if strings.HasPrefix(result, "fail:") {
			failCount++
		}
	}

	// Should have at least 5 successes (positive values) and 2 failures (non-positive values)
	if successCount < 5 {
		t.Errorf("Expected at least 5 success results, got %d", successCount)
	}
	if failCount < 2 {
		t.Errorf("Expected at least 2 fail results, got %d", failCount)
	}

	// Verify side effects were executed
	mu.Lock()
	if sideEffectCount < 5 {
		t.Errorf("Expected at least 5 side effects, got %d", sideEffectCount)
	}
	if len(processedValues) < 5 {
		t.Errorf("Expected at least 5 processed values, got %d", len(processedValues))
	}
	mu.Unlock()

	fmt.Printf("Pipeline completed with %d final results\n", len(finalResults))
}

// Stress test for the complete pipeline under heavy load
func Test_CompletePipeline_Stress(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create large dataset for stress testing
	source := make([]int, 10000)
	for i := range source {
		source[i] = i - 5000 // Mix of negative, zero, and positive values
	}

	// Configure context with process options for stress testing
	ctx = core.WithProcessOptions(core.WithWorkerOptions(ctx, 10), true)
	workers := core.GetWorkerMaxCount(ctx, 20)

	// Track metrics
	var sideEffectCount int64

	// Finally handlers to collect final results
	finallyHandlers := mass.FinallyHandlers[string, string]{
		OnSuccess: func(ctx context.Context, in string) string {
			return fmt.Sprintf("success:%s", in)
		},
		OnError: func(ctx context.Context, err error) string {
			return fmt.Sprintf("fail:%s", err.Error())
		},
		OnCancel: func(ctx context.Context, err error) string {
			return fmt.Sprintf("cancel:%s", err.Error())
		},
	}

	startTime := time.Now()

	// Build the complete pipeline using all ROP methods in proper sequence
	// Stage 1: Run with validation
	stage1 := Run(ctx,
		core.ToChanManyResults[int](ctx, source),
		Validate(func(ctx context.Context, in int) (bool, string) {
			// Validate that input is positive (stress test with simple validation)
			if in <= 0 {
				return false, fmt.Sprintf("value %d is not positive", in)
			}
			return true, ""
		}),
		workers)

	// Stage 2: Turnout with Switch (int -> int)
	stage2 := Turnout[int, int](ctx,
		stage1,
		Switch[int, int](func(ctx context.Context, r int) rop.Result[int] {
			// Switch based on value (stress test with simple computation)
			if r%2 == 0 {
				return rop.Success(r * 2) // Double even numbers
			} else {
				return rop.Success(r * 3) // Triple odd numbers
			}
		}),
		10)

	// Stage 3: Turnout with Map (int -> string)
	stage3 := Turnout[int, string](ctx,
		stage2,
		Map[int, string](func(ctx context.Context, r int) string {
			// Map to formatted string (stress test with string formatting)
			return fmt.Sprintf("mapped:%d", r)
		}),
		10)

	// Stage 4: Run with Tee (side effects)
	stage4 := Run(ctx,
		stage3,
		Tee[string](func(ctx context.Context, r rop.Result[string]) {
			// Side effect: count successful results (stress test with atomic counter)
			if r.IsSuccess() {
				atomic.AddInt64(&sideEffectCount, 1)
			}
		}),
		workers)

	// Stage 5: Finally to collect results
	resultCh := Finally(ctx, stage4, finallyHandlers)

	// Collect all final results
	var successCount, failCount, cancelCount int64
	for result := range resultCh {
		if strings.HasPrefix(result, "success:") {
			successCount++
		} else if strings.HasPrefix(result, "fail:") {
			failCount++
		} else if strings.HasPrefix(result, "cancel:") {
			cancelCount++
		}
	}

	duration := time.Since(startTime)

	// Verify results
	totalResults := successCount + failCount + cancelCount
	if totalResults != 10000 {
		t.Errorf("Expected 10000 total results, got %d", totalResults)
	}

	// With our source data (-5000 to 4999), we expect:
	// - 5000 negative or zero values -> fail count
	// - 4999 positive values -> success count
	// - 0 cancelled values (no cancellation in this test)
	if successCount < 4999 {
		t.Errorf("Expected at least 4999 success results, got %d", successCount)
	}
	if failCount < 5000 {
		t.Errorf("Expected at least 5000 fail results, got %d", failCount)
	}

	// Verify side effects were executed
	sideEffectFinal := atomic.LoadInt64(&sideEffectCount)
	if sideEffectFinal < 4999 {
		t.Errorf("Expected at least 4999 side effects, got %d", sideEffectFinal)
	}

	// Performance metrics
	fmt.Printf("Stress test completed in %v\n", duration)
	fmt.Printf("Processed %d items\n", totalResults)
	fmt.Printf("Success: %d, Fail: %d, Cancel: %d\n", successCount, failCount, cancelCount)
	fmt.Printf("Side effects executed: %d\n", sideEffectFinal)
	fmt.Printf("Processing rate: %.2f items/second\n", float64(totalResults)/duration.Seconds())

	// Performance threshold - should process 10000 items in under 10 seconds
	if duration > 10*time.Second {
		t.Errorf("Stress test took too long: %v", duration)
	}
}

// Benchmark for the complete pipeline
func Benchmark_CompletePipeline(b *testing.B) {
	ctx := context.Background()

	// Create medium dataset for benchmarking
	source := make([]int, 1000)
	for i := range source {
		source[i] = i - 500 // Mix of negative, zero, and positive values
	}

	ctx = core.WithProcessOptions(core.WithWorkerOptions(ctx, 5), true)
	workers := core.GetWorkerMaxCount(ctx, 10)

	// Finally handlers
	finallyHandlers := mass.FinallyHandlers[string, string]{
		OnSuccess: func(ctx context.Context, in string) string {
			return in // Simplified for benchmarking
		},
		OnError: func(ctx context.Context, err error) string {
			return "fail"
		},
		OnCancel: func(ctx context.Context, err error) string {
			return "cancel"
		},
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Build the complete pipeline
			stage1 := Run(ctx,
				core.ToChanManyResults[int](ctx, source),
				Validate(func(ctx context.Context, in int) (bool, string) {
					if in <= 0 {
						return false, "not positive"
					}
					return true, ""
				}),
				workers)

			stage2 := Turnout[int, int](ctx,
				stage1,
				Switch[int, int](func(ctx context.Context, r int) rop.Result[int] {
					if r%2 == 0 {
						return rop.Success(r * 2)
					} else {
						return rop.Success(r * 3)
					}
				}),
				5)

			stage3 := Turnout[int, string](ctx,
				stage2,
				Map[int, string](func(ctx context.Context, r int) string {
					return fmt.Sprintf("mapped:%d", r)
				}),
				5)

			stage4 := Run(ctx,
				stage3,
				Tee[string](func(ctx context.Context, r rop.Result[string]) {
					// Minimal side effect for benchmarking
					_ = r
				}),
				workers)

			resultCh := Finally(ctx, stage4, finallyHandlers)

			// Consume all results
			count := 0
			for range resultCh {
				count++
			}

			if count == 0 {
				b.Error("No results processed")
			}
		}
	})
}

// Benchmark tests
func BenchmarkProcess_SingleWorker(b *testing.B) {
	ctx := context.Background()
	input := make([]int, 1000)
	for i := range input {
		input[i] = i
	}

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
		resultCh := Run(ctx, core.ToChanManyResults(ctx, input), processor, 1)
		for range resultCh {
			// Consume all results
		}
	}
}

func BenchmarkProcess_MultipleWorkers(b *testing.B) {
	ctx := context.Background()
	input := make([]int, 1000)
	for i := range input {
		input[i] = i
	}

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
		resultCh := Run(ctx, core.ToChanManyResults(ctx, input), processor, 4)
		for range resultCh {
			// Consume all results
		}
	}
}
