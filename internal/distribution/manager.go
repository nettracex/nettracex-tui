package distribution

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// DistributionCoordinator manages cross-platform package publishing
type DistributionCoordinator struct {
	publishers map[string]Publisher
	validators map[string]Validator
	notifier   NotificationService
	config     *DistributionConfig
	mu         sync.RWMutex
}

// Publisher defines the interface for package publishers
type Publisher interface {
	Publish(ctx context.Context, release Release) error
	Validate(ctx context.Context, release Release) error
	GetStatus() PublishStatus
	GetName() string
}

// Validator defines the interface for release validators
type Validator interface {
	Validate(ctx context.Context, release Release) error
	GetName() string
}

// NotificationService handles publishing notifications
type NotificationService interface {
	NotifySuccess(publisher string, release Release) error
	NotifyFailure(publisher string, release Release, err error) error
	NotifyProgress(publisher string, release Release, progress float64) error
}

// DistributionConfig contains configuration for distribution
type DistributionConfig struct {
	Publishers      map[string]PublisherConfig `json:"publishers"`
	Validators      map[string]ValidatorConfig `json:"validators"`
	Notifications   NotificationConfig         `json:"notifications"`
	RetryPolicy     RetryPolicy                `json:"retry_policy"`
	ConcurrentLimit int                        `json:"concurrent_limit"`
}

// PublisherConfig contains publisher-specific configuration
type PublisherConfig struct {
	Enabled    bool                   `json:"enabled"`
	Priority   int                    `json:"priority"`
	Timeout    time.Duration          `json:"timeout"`
	RetryCount int                    `json:"retry_count"`
	Config     map[string]interface{} `json:"config"`
}

// ValidatorConfig contains validator-specific configuration
type ValidatorConfig struct {
	Enabled bool                   `json:"enabled"`
	Config  map[string]interface{} `json:"config"`
}

// NotificationConfig contains notification settings
type NotificationConfig struct {
	Enabled  bool     `json:"enabled"`
	Channels []string `json:"channels"`
	OnError  bool     `json:"on_error"`
	OnSuccess bool    `json:"on_success"`
}

// RetryPolicy defines retry behavior
type RetryPolicy struct {
	MaxRetries int           `json:"max_retries"`
	BaseDelay  time.Duration `json:"base_delay"`
	MaxDelay   time.Duration `json:"max_delay"`
	Multiplier float64       `json:"multiplier"`
}

// NewDistributionCoordinator creates a new distribution coordinator
func NewDistributionCoordinator(config *DistributionConfig) *DistributionCoordinator {
	return &DistributionCoordinator{
		publishers: make(map[string]Publisher),
		validators: make(map[string]Validator),
		config:     config,
	}
}

// RegisterPublisher registers a new publisher
func (dc *DistributionCoordinator) RegisterPublisher(publisher Publisher) error {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	
	name := publisher.GetName()
	if _, exists := dc.publishers[name]; exists {
		return fmt.Errorf("publisher %s already registered", name)
	}
	
	dc.publishers[name] = publisher
	log.Printf("Registered publisher: %s", name)
	return nil
}

// RegisterValidator registers a new validator
func (dc *DistributionCoordinator) RegisterValidator(validator Validator) error {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	
	name := validator.GetName()
	if _, exists := dc.validators[name]; exists {
		return fmt.Errorf("validator %s already registered", name)
	}
	
	dc.validators[name] = validator
	log.Printf("Registered validator: %s", name)
	return nil
}

// SetNotificationService sets the notification service
func (dc *DistributionCoordinator) SetNotificationService(service NotificationService) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	dc.notifier = service
}

// Distribute publishes a release to all configured publishers
func (dc *DistributionCoordinator) Distribute(ctx context.Context, release Release) error {
	// Validate release first
	if err := dc.validateRelease(ctx, release); err != nil {
		return fmt.Errorf("release validation failed: %w", err)
	}
	
	// Get enabled publishers sorted by priority
	publishers := dc.getEnabledPublishers()
	if len(publishers) == 0 {
		return fmt.Errorf("no enabled publishers configured")
	}
	
	// Publish to all publishers with concurrency control
	return dc.publishConcurrently(ctx, release, publishers)
}

// validateRelease runs all enabled validators
func (dc *DistributionCoordinator) validateRelease(ctx context.Context, release Release) error {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	
	for name, validator := range dc.validators {
		if config, exists := dc.config.Validators[name]; exists && config.Enabled {
			if err := validator.Validate(ctx, release); err != nil {
				return fmt.Errorf("validator %s failed: %w", name, err)
			}
		}
	}
	
	return nil
}

// getEnabledPublishers returns enabled publishers sorted by priority
func (dc *DistributionCoordinator) getEnabledPublishers() []Publisher {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	
	var publishers []Publisher
	for name, publisher := range dc.publishers {
		if config, exists := dc.config.Publishers[name]; exists && config.Enabled {
			publishers = append(publishers, publisher)
		}
	}
	
	// Sort by priority (higher priority first)
	// Implementation would sort based on config.Priority
	
	return publishers
}

// publishConcurrently publishes to multiple publishers with concurrency control
func (dc *DistributionCoordinator) publishConcurrently(ctx context.Context, release Release, publishers []Publisher) error {
	concurrentLimit := dc.config.ConcurrentLimit
	if concurrentLimit <= 0 {
		concurrentLimit = len(publishers)
	}
	
	semaphore := make(chan struct{}, concurrentLimit)
	errChan := make(chan error, len(publishers))
	var wg sync.WaitGroup
	
	for _, publisher := range publishers {
		wg.Add(1)
		go func(pub Publisher) {
			defer wg.Done()
			
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			
			if err := dc.publishWithRetry(ctx, pub, release); err != nil {
				errChan <- fmt.Errorf("publisher %s failed: %w", pub.GetName(), err)
				if dc.notifier != nil {
					dc.notifier.NotifyFailure(pub.GetName(), release, err)
				}
			} else {
				if dc.notifier != nil {
					dc.notifier.NotifySuccess(pub.GetName(), release)
				}
			}
		}(publisher)
	}
	
	wg.Wait()
	close(errChan)
	
	// Collect all errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("publishing failed for %d publishers: %v", len(errors), errors)
	}
	
	return nil
}

// publishWithRetry publishes with retry logic
func (dc *DistributionCoordinator) publishWithRetry(ctx context.Context, publisher Publisher, release Release) error {
	policy := dc.config.RetryPolicy
	
	for attempt := 0; attempt <= policy.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := dc.calculateBackoffDelay(attempt, policy)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}
		
		if err := publisher.Publish(ctx, release); err != nil {
			if attempt == policy.MaxRetries {
				return fmt.Errorf("failed after %d attempts: %w", policy.MaxRetries+1, err)
			}
			log.Printf("Publisher %s attempt %d failed: %v", publisher.GetName(), attempt+1, err)
			continue
		}
		
		return nil
	}
	
	return fmt.Errorf("unexpected retry loop exit")
}

// calculateBackoffDelay calculates exponential backoff delay
func (dc *DistributionCoordinator) calculateBackoffDelay(attempt int, policy RetryPolicy) time.Duration {
	delay := float64(policy.BaseDelay) * pow(policy.Multiplier, float64(attempt-1))
	if delay > float64(policy.MaxDelay) {
		delay = float64(policy.MaxDelay)
	}
	return time.Duration(delay)
}

// pow is a simple power function for exponential backoff
func pow(base float64, exp float64) float64 {
	if exp == 0 {
		return 1
	}
	result := base
	for i := 1; i < int(exp); i++ {
		result *= base
	}
	return result
}

// GetPublisherStatus returns the status of all publishers
func (dc *DistributionCoordinator) GetPublisherStatus() map[string]PublishStatus {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	
	status := make(map[string]PublishStatus)
	for name, publisher := range dc.publishers {
		status[name] = publisher.GetStatus()
	}
	
	return status
}