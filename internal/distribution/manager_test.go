package distribution

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockPublisher is a mock implementation of Publisher
type MockPublisher struct {
	mock.Mock
	name   string
	status PublishStatus
}

func (m *MockPublisher) Publish(ctx context.Context, release Release) error {
	args := m.Called(ctx, release)
	return args.Error(0)
}

func (m *MockPublisher) Validate(ctx context.Context, release Release) error {
	args := m.Called(ctx, release)
	return args.Error(0)
}

func (m *MockPublisher) GetStatus() PublishStatus {
	return m.status
}

func (m *MockPublisher) GetName() string {
	return m.name
}

// MockValidator is a mock implementation of Validator
type MockValidator struct {
	mock.Mock
	name string
}

func (m *MockValidator) Validate(ctx context.Context, release Release) error {
	args := m.Called(ctx, release)
	return args.Error(0)
}

func (m *MockValidator) GetName() string {
	return m.name
}

// MockNotificationService is a mock implementation of NotificationService
type MockNotificationService struct {
	mock.Mock
}

func (m *MockNotificationService) NotifySuccess(publisher string, release Release) error {
	args := m.Called(publisher, release)
	return args.Error(0)
}

func (m *MockNotificationService) NotifyFailure(publisher string, release Release, err error) error {
	args := m.Called(publisher, release, err)
	return args.Error(0)
}

func (m *MockNotificationService) NotifyProgress(publisher string, release Release, progress float64) error {
	args := m.Called(publisher, release, progress)
	return args.Error(0)
}

func TestNewDistributionCoordinator(t *testing.T) {
	config := &DistributionConfig{
		Publishers:      make(map[string]PublisherConfig),
		Validators:      make(map[string]ValidatorConfig),
		ConcurrentLimit: 2,
	}
	
	coordinator := NewDistributionCoordinator(config)
	
	assert.NotNil(t, coordinator)
	assert.Equal(t, config, coordinator.config)
	assert.NotNil(t, coordinator.publishers)
	assert.NotNil(t, coordinator.validators)
}

func TestRegisterPublisher(t *testing.T) {
	coordinator := NewDistributionCoordinator(&DistributionConfig{})
	
	publisher := &MockPublisher{name: "test-publisher"}
	
	err := coordinator.RegisterPublisher(publisher)
	assert.NoError(t, err)
	
	// Test duplicate registration
	err = coordinator.RegisterPublisher(publisher)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestRegisterValidator(t *testing.T) {
	coordinator := NewDistributionCoordinator(&DistributionConfig{})
	
	validator := &MockValidator{name: "test-validator"}
	
	err := coordinator.RegisterValidator(validator)
	assert.NoError(t, err)
	
	// Test duplicate registration
	err = coordinator.RegisterValidator(validator)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestDistribute_ValidationFailure(t *testing.T) {
	config := &DistributionConfig{
		Validators: map[string]ValidatorConfig{
			"test-validator": {Enabled: true},
		},
	}
	
	coordinator := NewDistributionCoordinator(config)
	
	validator := &MockValidator{name: "test-validator"}
	validator.On("Validate", mock.Anything, mock.Anything).Return(errors.New("validation failed"))
	
	coordinator.RegisterValidator(validator)
	
	release := Release{Version: "v1.0.0"}
	
	err := coordinator.Distribute(context.Background(), release)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")
	
	validator.AssertExpectations(t)
}

func TestDistribute_NoPublishers(t *testing.T) {
	coordinator := NewDistributionCoordinator(&DistributionConfig{})
	
	release := Release{Version: "v1.0.0"}
	
	err := coordinator.Distribute(context.Background(), release)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no enabled publishers")
}

func TestDistribute_Success(t *testing.T) {
	config := &DistributionConfig{
		Publishers: map[string]PublisherConfig{
			"test-publisher": {Enabled: true},
		},
		Validators: map[string]ValidatorConfig{
			"test-validator": {Enabled: true},
		},
		RetryPolicy: RetryPolicy{
			MaxRetries: 1,
			BaseDelay:  time.Millisecond,
			MaxDelay:   time.Second,
			Multiplier: 2.0,
		},
		ConcurrentLimit: 1,
	}
	
	coordinator := NewDistributionCoordinator(config)
	
	validator := &MockValidator{name: "test-validator"}
	validator.On("Validate", mock.Anything, mock.Anything).Return(nil)
	
	publisher := &MockPublisher{name: "test-publisher"}
	publisher.On("Publish", mock.Anything, mock.Anything).Return(nil)
	
	notifier := &MockNotificationService{}
	notifier.On("NotifySuccess", "test-publisher", mock.Anything).Return(nil)
	
	coordinator.RegisterValidator(validator)
	coordinator.RegisterPublisher(publisher)
	coordinator.SetNotificationService(notifier)
	
	release := Release{Version: "v1.0.0"}
	
	err := coordinator.Distribute(context.Background(), release)
	assert.NoError(t, err)
	
	validator.AssertExpectations(t)
	publisher.AssertExpectations(t)
	notifier.AssertExpectations(t)
}

func TestDistribute_PublisherFailure(t *testing.T) {
	config := &DistributionConfig{
		Publishers: map[string]PublisherConfig{
			"test-publisher": {Enabled: true},
		},
		RetryPolicy: RetryPolicy{
			MaxRetries: 1,
			BaseDelay:  time.Millisecond,
			MaxDelay:   time.Second,
			Multiplier: 2.0,
		},
		ConcurrentLimit: 1,
	}
	
	coordinator := NewDistributionCoordinator(config)
	
	publisher := &MockPublisher{name: "test-publisher"}
	publisher.On("Publish", mock.Anything, mock.Anything).Return(errors.New("publish failed"))
	
	notifier := &MockNotificationService{}
	notifier.On("NotifyFailure", "test-publisher", mock.Anything, mock.Anything).Return(nil)
	
	coordinator.RegisterPublisher(publisher)
	coordinator.SetNotificationService(notifier)
	
	release := Release{Version: "v1.0.0"}
	
	err := coordinator.Distribute(context.Background(), release)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "publishing failed")
	
	publisher.AssertExpectations(t)
	notifier.AssertExpectations(t)
}

func TestDistribute_ConcurrentPublishing(t *testing.T) {
	config := &DistributionConfig{
		Publishers: map[string]PublisherConfig{
			"publisher-1": {Enabled: true},
			"publisher-2": {Enabled: true},
		},
		RetryPolicy: RetryPolicy{
			MaxRetries: 0,
			BaseDelay:  time.Millisecond,
			MaxDelay:   time.Second,
			Multiplier: 2.0,
		},
		ConcurrentLimit: 2,
	}
	
	coordinator := NewDistributionCoordinator(config)
	
	publisher1 := &MockPublisher{name: "publisher-1"}
	publisher1.On("Publish", mock.Anything, mock.Anything).Return(nil)
	
	publisher2 := &MockPublisher{name: "publisher-2"}
	publisher2.On("Publish", mock.Anything, mock.Anything).Return(nil)
	
	notifier := &MockNotificationService{}
	notifier.On("NotifySuccess", mock.Anything, mock.Anything).Return(nil)
	
	coordinator.RegisterPublisher(publisher1)
	coordinator.RegisterPublisher(publisher2)
	coordinator.SetNotificationService(notifier)
	
	release := Release{Version: "v1.0.0"}
	
	err := coordinator.Distribute(context.Background(), release)
	assert.NoError(t, err)
	
	publisher1.AssertExpectations(t)
	publisher2.AssertExpectations(t)
	notifier.AssertExpectations(t)
}

func TestCalculateBackoffDelay(t *testing.T) {
	coordinator := NewDistributionCoordinator(&DistributionConfig{})
	
	policy := RetryPolicy{
		BaseDelay:  100 * time.Millisecond,
		MaxDelay:   time.Second,
		Multiplier: 2.0,
	}
	
	// Test first retry
	delay := coordinator.calculateBackoffDelay(1, policy)
	assert.Equal(t, 100*time.Millisecond, delay)
	
	// Test second retry
	delay = coordinator.calculateBackoffDelay(2, policy)
	assert.Equal(t, 200*time.Millisecond, delay)
	
	// Test max delay cap
	delay = coordinator.calculateBackoffDelay(10, policy)
	assert.Equal(t, time.Second, delay)
}

func TestGetPublisherStatus(t *testing.T) {
	coordinator := NewDistributionCoordinator(&DistributionConfig{})
	
	publisher := &MockPublisher{
		name: "test-publisher",
		status: PublishStatus{
			Name:   "test-publisher",
			Status: StatusSuccess,
		},
	}
	
	coordinator.RegisterPublisher(publisher)
	
	statuses := coordinator.GetPublisherStatus()
	assert.Len(t, statuses, 1)
	assert.Equal(t, StatusSuccess, statuses["test-publisher"].Status)
}

func TestPowFunction(t *testing.T) {
	tests := []struct {
		base     float64
		exp      float64
		expected float64
	}{
		{2.0, 0, 1.0},
		{2.0, 1, 2.0},
		{2.0, 2, 4.0},
		{2.0, 3, 8.0},
		{3.0, 2, 9.0},
	}
	
	for _, test := range tests {
		result := pow(test.base, test.exp)
		assert.Equal(t, test.expected, result)
	}
}