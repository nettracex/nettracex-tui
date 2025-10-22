// Package network provides tests for retry manager
package network

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/nettracex/nettracex-tui/internal/domain"
)

func TestNewRetryManager(t *testing.T) {
	maxAttempts := 5
	baseDelay := 2 * time.Second

	rm := NewRetryManager(maxAttempts, baseDelay)

	if rm == nil {
		t.Fatal("NewRetryManager returned nil")
	}

	if rm.GetMaxAttempts() != maxAttempts {
		t.Errorf("Expected max attempts %d, got %d", maxAttempts, rm.GetMaxAttempts())
	}

	if rm.GetBaseDelay() != baseDelay {
		t.Errorf("Expected base delay %v, got %v", baseDelay, rm.GetBaseDelay())
	}
}

func TestNewRetryManager_DefaultValues(t *testing.T) {
	// Test with invalid values
	rm := NewRetryManager(0, 0)

	if rm.GetMaxAttempts() != 3 {
		t.Errorf("Expected default max attempts 3, got %d", rm.GetMaxAttempts())
	}

	if rm.GetBaseDelay() != time.Second {
		t.Errorf("Expected default base delay 1s, got %v", rm.GetBaseDelay())
	}
}

func TestRetryManager_ExecuteWithRetry_Success(t *testing.T) {
	rm := NewRetryManager(3, 100*time.Millisecond)
	ctx := context.Background()

	callCount := 0
	fn := func() (interface{}, error) {
		callCount++
		return "success", nil
	}

	shouldRetry := func(err error) bool {
		return true
	}

	result, err := rm.ExecuteWithRetry(ctx, fn, shouldRetry)
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}

	if result != "success" {
		t.Errorf("Expected result 'success', got %v", result)
	}

	if callCount != 1 {
		t.Errorf("Expected 1 call, got %d", callCount)
	}
}

func TestRetryManager_ExecuteWithRetry_SuccessAfterRetries(t *testing.T) {
	rm := NewRetryManager(3, 10*time.Millisecond)
	ctx := context.Background()

	callCount := 0
	fn := func() (interface{}, error) {
		callCount++
		if callCount < 3 {
			return nil, &net.OpError{Op: "dial", Net: "tcp", Err: fmt.Errorf("connection refused")}
		}
		return "success", nil
	}

	shouldRetry := func(err error) bool {
		return true
	}

	start := time.Now()
	result, err := rm.ExecuteWithRetry(ctx, fn, shouldRetry)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}

	if result != "success" {
		t.Errorf("Expected result 'success', got %v", result)
	}

	if callCount != 3 {
		t.Errorf("Expected 3 calls, got %d", callCount)
	}

	// Should have some delay due to retries
	expectedMinDelay := 10*time.Millisecond + 20*time.Millisecond // First retry + second retry
	if elapsed < expectedMinDelay {
		t.Errorf("Expected at least %v delay, got %v", expectedMinDelay, elapsed)
	}
}

func TestRetryManager_ExecuteWithRetry_ExhaustRetries(t *testing.T) {
	rm := NewRetryManager(3, 10*time.Millisecond)
	ctx := context.Background()

	callCount := 0
	originalErr := fmt.Errorf("persistent error")
	fn := func() (interface{}, error) {
		callCount++
		return nil, originalErr
	}

	shouldRetry := func(err error) bool {
		return true
	}

	result, err := rm.ExecuteWithRetry(ctx, fn, shouldRetry)
	if err == nil {
		t.Fatal("Expected error after exhausting retries")
	}

	if result != nil {
		t.Errorf("Expected nil result, got %v", result)
	}

	if callCount != 3 {
		t.Errorf("Expected 3 calls, got %d", callCount)
	}

	// Check that it's a NetTraceError
	netErr, ok := err.(*domain.NetTraceError)
	if !ok {
		t.Errorf("Expected NetTraceError, got %T", err)
	}

	if netErr.Code != "RETRY_EXHAUSTED" {
		t.Errorf("Expected error code RETRY_EXHAUSTED, got %s", netErr.Code)
	}

	if netErr.Cause != originalErr {
		t.Errorf("Expected cause to be original error, got %v", netErr.Cause)
	}
}

func TestRetryManager_ExecuteWithRetry_NonRetryableError(t *testing.T) {
	rm := NewRetryManager(3, 10*time.Millisecond)
	ctx := context.Background()

	callCount := 0
	originalErr := fmt.Errorf("non-retryable error")
	fn := func() (interface{}, error) {
		callCount++
		return nil, originalErr
	}

	shouldRetry := func(err error) bool {
		return false // Never retry
	}

	result, err := rm.ExecuteWithRetry(ctx, fn, shouldRetry)
	if err == nil {
		t.Fatal("Expected error for non-retryable error")
	}

	if result != nil {
		t.Errorf("Expected nil result, got %v", result)
	}

	if callCount != 1 {
		t.Errorf("Expected 1 call, got %d", callCount)
	}

	// Should get the original error, not a retry error
	netErr, ok := err.(*domain.NetTraceError)
	if !ok {
		t.Errorf("Expected NetTraceError, got %T", err)
	}

	if netErr.Cause != originalErr {
		t.Errorf("Expected cause to be original error, got %v", netErr.Cause)
	}
}

func TestRetryManager_ExecuteWithRetry_ContextCancellation(t *testing.T) {
	rm := NewRetryManager(5, 100*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())

	callCount := 0
	fn := func() (interface{}, error) {
		callCount++
		if callCount == 2 {
			cancel() // Cancel context on second call
		}
		return nil, fmt.Errorf("error")
	}

	shouldRetry := func(err error) bool {
		return true
	}

	result, err := rm.ExecuteWithRetry(ctx, fn, shouldRetry)
	if err == nil {
		t.Fatal("Expected error due to context cancellation")
	}

	if result != nil {
		t.Errorf("Expected nil result, got %v", result)
	}

	netErr, ok := err.(*domain.NetTraceError)
	if !ok {
		t.Errorf("Expected NetTraceError, got %T", err)
	}

	if netErr.Code != "RETRY_CANCELLED" && netErr.Code != "RETRY_DELAY_CANCELLED" {
		t.Errorf("Expected cancellation error code, got %s", netErr.Code)
	}
}

func TestRetryManager_ExecuteWithLinearRetry(t *testing.T) {
	rm := NewRetryManager(3, 10*time.Millisecond)
	ctx := context.Background()

	callCount := 0
	fn := func() (interface{}, error) {
		callCount++
		if callCount < 3 {
			return nil, fmt.Errorf("error")
		}
		return "success", nil
	}

	shouldRetry := func(err error) bool {
		return true
	}

	start := time.Now()
	result, err := rm.ExecuteWithLinearRetry(ctx, fn, shouldRetry)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}

	if result != "success" {
		t.Errorf("Expected result 'success', got %v", result)
	}

	if callCount != 3 {
		t.Errorf("Expected 3 calls, got %d", callCount)
	}

	// Linear retry: 10ms * 1 + 10ms * 2 = 30ms minimum
	expectedMinDelay := 30 * time.Millisecond
	if elapsed < expectedMinDelay {
		t.Errorf("Expected at least %v delay, got %v", expectedMinDelay, elapsed)
	}
}

func TestRetryManager_ExecuteWithCustomRetry(t *testing.T) {
	rm := NewRetryManager(3, 10*time.Millisecond)
	ctx := context.Background()

	callCount := 0
	fn := func() (interface{}, error) {
		callCount++
		if callCount < 3 {
			return nil, fmt.Errorf("error")
		}
		return "success", nil
	}

	shouldRetry := func(err error) bool {
		return true
	}

	// Custom delay function: constant 5ms delay
	delayFunc := func(attempt int) time.Duration {
		return 5 * time.Millisecond
	}

	start := time.Now()
	result, err := rm.ExecuteWithCustomRetry(ctx, fn, shouldRetry, delayFunc)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}

	if result != "success" {
		t.Errorf("Expected result 'success', got %v", result)
	}

	if callCount != 3 {
		t.Errorf("Expected 3 calls, got %d", callCount)
	}

	// Custom delay: 5ms * 2 attempts = 10ms minimum
	expectedMinDelay := 10 * time.Millisecond
	if elapsed < expectedMinDelay {
		t.Errorf("Expected at least %v delay, got %v", expectedMinDelay, elapsed)
	}
}

func TestRetryManager_CalculateDelay(t *testing.T) {
	rm := NewRetryManager(5, 100*time.Millisecond)

	testCases := []struct {
		attempt      int
		expectedMin  time.Duration
		expectedMax  time.Duration
		description  string
	}{
		{1, 100 * time.Millisecond, 200 * time.Millisecond, "first retry"},
		{2, 200 * time.Millisecond, 400 * time.Millisecond, "second retry"},
		{3, 400 * time.Millisecond, 800 * time.Millisecond, "third retry"},
		{10, 30 * time.Second, 30 * time.Second, "capped at max delay"},
	}

	for _, tc := range testCases {
		delay := rm.calculateDelay(tc.attempt)
		
		if delay < tc.expectedMin {
			t.Errorf("%s: expected delay >= %v, got %v", tc.description, tc.expectedMin, delay)
		}
		
		if delay > tc.expectedMax {
			t.Errorf("%s: expected delay <= %v, got %v", tc.description, tc.expectedMax, delay)
		}
	}
}

func TestRetryManager_SetMaxAttempts(t *testing.T) {
	rm := NewRetryManager(3, time.Second)

	// Test valid value
	rm.SetMaxAttempts(5)
	if rm.GetMaxAttempts() != 5 {
		t.Errorf("Expected max attempts 5, got %d", rm.GetMaxAttempts())
	}

	// Test invalid value (should not change)
	rm.SetMaxAttempts(0)
	if rm.GetMaxAttempts() != 5 {
		t.Errorf("Expected max attempts to remain 5, got %d", rm.GetMaxAttempts())
	}

	rm.SetMaxAttempts(-1)
	if rm.GetMaxAttempts() != 5 {
		t.Errorf("Expected max attempts to remain 5, got %d", rm.GetMaxAttempts())
	}
}

func TestRetryManager_SetBaseDelay(t *testing.T) {
	rm := NewRetryManager(3, time.Second)

	// Test valid value
	newDelay := 2 * time.Second
	rm.SetBaseDelay(newDelay)
	if rm.GetBaseDelay() != newDelay {
		t.Errorf("Expected base delay %v, got %v", newDelay, rm.GetBaseDelay())
	}

	// Test invalid value (should not change)
	rm.SetBaseDelay(0)
	if rm.GetBaseDelay() != newDelay {
		t.Errorf("Expected base delay to remain %v, got %v", newDelay, rm.GetBaseDelay())
	}

	rm.SetBaseDelay(-time.Second)
	if rm.GetBaseDelay() != newDelay {
		t.Errorf("Expected base delay to remain %v, got %v", newDelay, rm.GetBaseDelay())
	}
}