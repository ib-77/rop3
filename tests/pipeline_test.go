package tests

import (
	"context"
	"fmt"
	"github.com/ib-77/rop3/pkg/rop"
	"github.com/ib-77/rop3/pkg/rop/core"
	"github.com/ib-77/rop3/pkg/rop/lite"
	"github.com/ib-77/rop3/pkg/rop/mass"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestURLProcessingDirectly tests the URL processing logic directly without HTTP requests
func TestURLProcessingDirectly(t *testing.T) {
	// Prepare test URLs - using a smaller set for testing
	urls := []string{
		// Valid URLs by structure (we won't actually fetch them)
		"https://www.example.com",
		"https://www.test.org",
		"https://www.google.com",
		"https://www.microsoft.com",
		"https://www.micros---oft.com",
		"https://www.mic--ros---oft.com",

		// Invalid URLs by structure
		"invalid-url",
		"ftp://invalid-protocol.com",
	}

	// Process URLs directly
	results := processRequest(urls)

	// Print results for inspection
	fmt.Println("Test Results:")
	for i, res := range results {
		fmt.Printf("%d. %s - %s\n", i+1, urls[i], res)
	}

	// Count valid and invalid results
	validCount := 0
	invalidCount := 0
	for _, res := range results {
		if res == "invalid" {
			invalidCount++
		} else {
			validCount++
		}
	}

	fmt.Printf("\nSummary: %d valid results, %d invalid results\n", validCount, invalidCount)

	// Verify we have results for all URLs
	assert.Equal(t, len(urls), len(results))

	// Verify we have the expected number of invalid results
	assert.Equal(t, 2, invalidCount)
}

func processRequest(urls []string) []string {
	ctx := context.Background()

	finallyHandlers := mass.FinallyHandlers[int, string]{
		OnSuccess: func(ctx context.Context, r int) string {
			return fmt.Sprintf("title length: %d", r)
		},
		OnError: func(ctx context.Context, err error) string {
			return "invalid"
		},
		OnCancel: func(ctx context.Context, err error) string {
			return "invalid"
		},
	}

	return core.FromChanMany(ctx,
		lite.Finally(ctx,
			lite.Turnout(ctx,
				lite.Turnout(ctx,
					lite.Run(ctx,
						core.ToChanManyResults(ctx, urls),
						lite.Validate(validateURLTest), 2),
					lite.Try(mockFetchTitle), 2),
				lite.Switch(calculateTitleLength), 2),
			finallyHandlers,
		),
	)
}

// mockFetchTitle simulates fetching a title without making HTTP requests
func mockFetchTitle(ctx context.Context, url string) (string, error) {
	// For testing, we'll return a mock title for valid URLs
	valid, _ := validateURLTest(ctx, url)
	if valid {

		return "Mock Page Title for " + url, nil
	}
	return "", fmt.Errorf("invalid URL")
}

// validateURLTest is a test version of validateURL
func validateURLTest(_ context.Context, url string) (bool, string) {
	// Simple validation: check if URL starts with http:// or https://
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return false, "URL must start with http:// or https://"
	}
	return true, ""
}

func calculateTitleLength(_ context.Context, title string) rop.Result[int] {
	return rop.Success(len(title))
}
