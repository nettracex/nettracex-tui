// Package network provides retry logic for network operations
package network

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/nettracex/nettracex-tui/internal/domain"
)

// RetryManager handles retry logic for network operations
type RetryManager struct {
	maxAttempts int
	baseDelay   time.Duration
}

// NewRetryManager creates a new retry manager with the specified configuration
func NewRetryManager(maxAttempts int, baseDelay time.Duration) *RetryManager {
	if maxAttempts <= 0 {
		maxAttempts = 3 // Default to 3 attempts
	}
	if baseDelay <= 0 {
		baseDelay = time.Second // Default to 1 second
	}

	return &RetryManager{
		maxAttempts: maxAttempts,
		baseDelay:   baseDelay,
	}
}

// RetryableFunc represents a function that can be retried
type RetryableFunc func() (interface{}, error)

// ShouldRetryFunc determines if an error should trigger a retry
type ShouldRetryFunc func(error) bool

// ExecuteWithRetry executes a function with exponential backoff retry logic
func (rm *RetryManager) ExecuteWithRetry(ctx context.Context, fn RetryableFunc, shouldRetry ShouldRetryFunc) (interface{}, error) {
	var lastErr error

	for attempt := 1; attempt <= rm.maxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return nil, &domain.NetTraceError{
				Type:      domain.ErrorTypeNetwork,
				Message:   "operation cancelled during retry",
				Cause:     ctx.Err(),
				Context:   map[string]interface{}{"attempt": attempt, "max_attempts": rm.maxAttempts},
				Timestamp: time.Now(),
				Code:      "RETRY_CANCELLED",
			}
		default:
		}

		result, err := fn()
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Don't retry if this is the last attempt or if the error is not retryable
		if attempt == rm.maxAttempts || !shouldRetry(err) {
			break
		}

		// Calculate delay with exponential backoff
		delay := rm.calculateDelay(attempt)
		
		select {
		case <-ctx.Done():
			return nil, &domain.NetTraceError{
				Type:      domain.ErrorTypeNetwork,
				Message:   "operation cancelled during retry delay",
				Cause:     ctx.Err(),
				Context:   map[string]interface{}{"attempt": attempt, "delay": delay},
				Timestamp: time.Now(),
				Code:      "RETRY_DELAY_CANCELLED",
			}
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	// All attempts failed
	return nil, &domain.NetTraceError{
		Type:      domain.ErrorTypeNetwork,
		Message:   fmt.Sprintf("operation failed after %d attempts", rm.maxAttempts),
		Cause:     lastErr,
		Context:   map[string]interface{}{"max_attempts": rm.maxAttempts, "base_delay": rm.baseDelay},
		Timestamp: time.Now(),
		Code:      "RETRY_EXHAUSTED",
	}
}

// calculateDelay calculates the delay for the given attempt using exponential backoff
func (rm *RetryManager) calculateDelay(attempt int) time.Duration {
	// Exponential backoff: baseDelay * 2^(attempt-1)
	// Cap at 30 seconds to prevent excessive delays
	delay := time.Duration(float64(rm.baseDelay) * math.Pow(2, float64(attempt-1)))
	maxDelay := 30 * time.Second
	
	if delay > maxDelay {
		delay = maxDelay
	}
	
	return delay
}

// ExecuteWithLinearRetry executes a function with linear backoff retry logic
func (rm *RetryManager) ExecuteWithLinearRetry(ctx context.Context, fn RetryableFunc, shouldRetry ShouldRetryFunc) (interface{}, error) {
	var lastErr error

	for attempt := 1; attempt <= rm.maxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return nil, &domain.NetTraceError{
				Type:      domain.ErrorTypeNetwork,
				Message:   "operation cancelled during linear retry",
				Cause:     ctx.Err(),
				Context:   map[string]interface{}{"attempt": attempt, "max_attempts": rm.maxAttempts},
				Timestamp: time.Now(),
				Code:      "LINEAR_RETRY_CANCELLED",
			}
		default:
		}

		result, err := fn()
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Don't retry if this is the last attempt or if the error is not retryable
		if attempt == rm.maxAttempts || !shouldRetry(err) {
			break
		}

		// Linear backoff: baseDelay * attempt
		delay := time.Duration(int64(rm.baseDelay) * int64(attempt))
		
		select {
		case <-ctx.Done():
			return nil, &domain.NetTraceError{
				Type:      domain.ErrorTypeNetwork,
				Message:   "operation cancelled during linear retry delay",
				Cause:     ctx.Err(),
				Context:   map[string]interface{}{"attempt": attempt, "delay": delay},
				Timestamp: time.Now(),
				Code:      "LINEAR_RETRY_DELAY_CANCELLED",
			}
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	// All attempts failed
	return nil, &domain.NetTraceError{
		Type:      domain.ErrorTypeNetwork,
		Message:   fmt.Sprintf("linear retry operation failed after %d attempts", rm.maxAttempts),
		Cause:     lastErr,
		Context:   map[string]interface{}{"max_attempts": rm.maxAttempts, "base_delay": rm.baseDelay},
		Timestamp: time.Now(),
		Code:      "LINEAR_RETRY_EXHAUSTED",
	}
}

// ExecuteWithCustomRetry executes a function with custom retry logic
func (rm *RetryManager) ExecuteWithCustomRetry(ctx context.Context, fn RetryableFunc, shouldRetry ShouldRetryFunc, delayFunc func(int) time.Duration) (interface{}, error) {
	var lastErr error

	for attempt := 1; attempt <= rm.maxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return nil, &domain.NetTraceError{
				Type:      domain.ErrorTypeNetwork,
				Message:   "operation cancelled during custom retry",
				Cause:     ctx.Err(),
				Context:   map[string]interface{}{"attempt": attempt, "max_attempts": rm.maxAttempts},
				Timestamp: time.Now(),
				Code:      "CUSTOM_RETRY_CANCELLED",
			}
		default:
		}

		result, err := fn()
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Don't retry if this is the last attempt or if the error is not retryable
		if attempt == rm.maxAttempts || !shouldRetry(err) {
			break
		}

		// Use custom delay function
		delay := delayFunc(attempt)
		
		select {
		case <-ctx.Done():
			return nil, &domain.NetTraceError{
				Type:      domain.ErrorTypeNetwork,
				Message:   "operation cancelled during custom retry delay",
				Cause:     ctx.Err(),
				Context:   map[string]interface{}{"attempt": attempt, "delay": delay},
				Timestamp: time.Now(),
				Code:      "CUSTOM_RETRY_DELAY_CANCELLED",
			}
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	// All attempts failed
	return nil, &domain.NetTraceError{
		Type:      domain.ErrorTypeNetwork,
		Message:   fmt.Sprintf("custom retry operation failed after %d attempts", rm.maxAttempts),
		Cause:     lastErr,
		Context:   map[string]interface{}{"max_attempts": rm.maxAttempts},
		Timestamp: time.Now(),
		Code:      "CUSTOM_RETRY_EXHAUSTED",
	}
}

// GetMaxAttempts returns the maximum number of retry attempts
func (rm *RetryManager) GetMaxAttempts() int {
	return rm.maxAttempts
}

// GetBaseDelay returns the base delay for retry operations
func (rm *RetryManager) GetBaseDelay() time.Duration {
	return rm.baseDelay
}

// SetMaxAttempts updates the maximum number of retry attempts
func (rm *RetryManager) SetMaxAttempts(maxAttempts int) {
	if maxAttempts > 0 {
		rm.maxAttempts = maxAttempts
	}
}

// SetBaseDelay updates the base delay for retry operations
func (rm *RetryManager) SetBaseDelay(baseDelay time.Duration) {
	if baseDelay > 0 {
		rm.baseDelay = baseDelay
	}
}