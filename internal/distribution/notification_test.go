package distribution

import (
	"context"
	"errors"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewDefaultNotificationService(t *testing.T) {
	config := NotificationServiceConfig{
		Enabled: true,
		RetryPolicy: NotificationRetryPolicy{
			MaxRetries: 3,
			BaseDelay:  time.Millisecond,
			MaxDelay:   time.Second,
		},
	}
	
	service := NewDefaultNotificationService(config)
	
	assert.NotNil(t, service)
	assert.Equal(t, config, service.config)
	assert.Len(t, service.channels, 2) // log and console channels
}

func TestDefaultNotificationService_RegisterChannel(t *testing.T) {
	service := NewDefaultNotificationService(NotificationServiceConfig{})
	
	channel := &MockNotificationChannel{name: "test-channel", enabled: true}
	service.RegisterChannel(channel)
	
	assert.Contains(t, service.channels, "test-channel")
	assert.Equal(t, channel, service.channels["test-channel"])
}

func TestDefaultNotificationService_NotifySuccess(t *testing.T) {
	config := NotificationServiceConfig{
		Enabled: true,
		RetryPolicy: NotificationRetryPolicy{
			MaxRetries: 0,
			BaseDelay:  time.Millisecond,
		},
	}
	
	service := NewDefaultNotificationService(config)
	
	// Replace channels with mock
	mockChannel := &MockNotificationChannel{name: "mock", enabled: true}
	service.channels = map[string]NotificationChannel{
		"mock": mockChannel,
	}
	
	release := Release{
		Version: "v1.0.0",
		Tag:     "v1.0.0",
	}
	
	err := service.NotifySuccess("test-publisher", release)
	assert.NoError(t, err)
	
	assert.Len(t, mockChannel.notifications, 1)
	notification := mockChannel.notifications[0]
	assert.Equal(t, NotificationTypeSuccess, notification.Type)
	assert.Equal(t, "test-publisher", notification.Publisher)
	assert.Contains(t, notification.Message, "v1.0.0")
}

func TestDefaultNotificationService_NotifyFailure(t *testing.T) {
	config := NotificationServiceConfig{
		Enabled: true,
		RetryPolicy: NotificationRetryPolicy{
			MaxRetries: 0,
			BaseDelay:  time.Millisecond,
		},
	}
	
	service := NewDefaultNotificationService(config)
	
	// Replace channels with mock
	mockChannel := &MockNotificationChannel{name: "mock", enabled: true}
	service.channels = map[string]NotificationChannel{
		"mock": mockChannel,
	}
	
	release := Release{
		Version: "v1.0.0",
		Tag:     "v1.0.0",
	}
	
	testError := errors.New("test error")
	err := service.NotifyFailure("test-publisher", release, testError)
	assert.NoError(t, err)
	
	assert.Len(t, mockChannel.notifications, 1)
	notification := mockChannel.notifications[0]
	assert.Equal(t, NotificationTypeFailure, notification.Type)
	assert.Equal(t, "test-publisher", notification.Publisher)
	assert.Equal(t, testError, notification.Error)
	assert.Contains(t, notification.Message, "Failed to publish")
}

func TestDefaultNotificationService_NotifyProgress(t *testing.T) {
	config := NotificationServiceConfig{
		Enabled: true,
		RetryPolicy: NotificationRetryPolicy{
			MaxRetries: 0,
			BaseDelay:  time.Millisecond,
		},
	}
	
	service := NewDefaultNotificationService(config)
	
	// Replace channels with mock
	mockChannel := &MockNotificationChannel{name: "mock", enabled: true}
	service.channels = map[string]NotificationChannel{
		"mock": mockChannel,
	}
	
	release := Release{
		Version: "v1.0.0",
		Tag:     "v1.0.0",
	}
	
	err := service.NotifyProgress("test-publisher", release, 0.75)
	assert.NoError(t, err)
	
	assert.Len(t, mockChannel.notifications, 1)
	notification := mockChannel.notifications[0]
	assert.Equal(t, NotificationTypeProgress, notification.Type)
	assert.Equal(t, "test-publisher", notification.Publisher)
	assert.Contains(t, notification.Message, "75.0%")
}

func TestDefaultNotificationService_DisabledService(t *testing.T) {
	config := NotificationServiceConfig{
		Enabled: false,
	}
	
	service := NewDefaultNotificationService(config)
	
	release := Release{Version: "v1.0.0"}
	
	err := service.NotifySuccess("test-publisher", release)
	assert.NoError(t, err)
	
	err = service.NotifyFailure("test-publisher", release, errors.New("test"))
	assert.NoError(t, err)
	
	err = service.NotifyProgress("test-publisher", release, 0.5)
	assert.NoError(t, err)
}

func TestDefaultNotificationService_SendWithRetry(t *testing.T) {
	config := NotificationServiceConfig{
		Enabled: true,
		RetryPolicy: NotificationRetryPolicy{
			MaxRetries: 2,
			BaseDelay:  time.Millisecond,
			MaxDelay:   10 * time.Millisecond,
		},
	}
	
	service := NewDefaultNotificationService(config)
	
	// Create a channel that fails first two attempts
	failingChannel := &FailingNotificationChannel{
		name:         "failing",
		enabled:      true,
		failAttempts: 2,
	}
	
	notification := Notification{
		Type:    NotificationTypeInfo,
		Message: "test message",
	}
	
	err := service.sendWithRetry(context.Background(), failingChannel, notification)
	assert.NoError(t, err)
	assert.Equal(t, 3, failingChannel.attempts) // 1 initial + 2 retries
}

func TestDefaultNotificationService_CalculateBackoffDelay(t *testing.T) {
	service := NewDefaultNotificationService(NotificationServiceConfig{})
	
	policy := NotificationRetryPolicy{
		BaseDelay: 100 * time.Millisecond,
		MaxDelay:  time.Second,
	}
	
	// Test first retry
	delay := service.calculateBackoffDelay(1, policy)
	assert.Equal(t, 100*time.Millisecond, delay)
	
	// Test second retry
	delay = service.calculateBackoffDelay(2, policy)
	assert.Equal(t, 200*time.Millisecond, delay)
	
	// Test max delay cap
	delay = service.calculateBackoffDelay(10, policy)
	assert.Equal(t, time.Second, delay)
}

func TestLogNotificationChannel_Send(t *testing.T) {
	// Capture log output
	var logOutput strings.Builder
	logger := log.New(&logOutput, "", 0)
	
	channel := &LogNotificationChannel{
		name:    "log",
		enabled: true,
		logger:  logger,
	}
	
	notification := Notification{
		Type:    NotificationTypeSuccess,
		Title:   "Test Title",
		Message: "Test message",
	}
	
	err := channel.Send(context.Background(), notification)
	assert.NoError(t, err)
	
	output := logOutput.String()
	assert.Contains(t, output, "[INFO]")
	assert.Contains(t, output, "Test Title")
	assert.Contains(t, output, "Test message")
}

func TestLogNotificationChannel_SendWithError(t *testing.T) {
	var logOutput strings.Builder
	logger := log.New(&logOutput, "", 0)
	
	channel := &LogNotificationChannel{
		name:    "log",
		enabled: true,
		logger:  logger,
	}
	
	notification := Notification{
		Type:    NotificationTypeFailure,
		Title:   "Error Title",
		Message: "Error message",
		Error:   errors.New("test error"),
	}
	
	err := channel.Send(context.Background(), notification)
	assert.NoError(t, err)
	
	output := logOutput.String()
	assert.Contains(t, output, "[ERROR]")
	assert.Contains(t, output, "Error Title")
	assert.Contains(t, output, "Error message")
	assert.Contains(t, output, "test error")
}

func TestConsoleNotificationChannel_Send(t *testing.T) {
	// Redirect stdout to capture console output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	
	channel := &ConsoleNotificationChannel{
		name:    "console",
		enabled: true,
		colored: false, // Disable colors for easier testing
	}
	
	notification := Notification{
		Type:      NotificationTypeSuccess,
		Message:   "Test console message",
		Timestamp: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
	}
	
	err := channel.Send(context.Background(), notification)
	assert.NoError(t, err)
	
	// Restore stdout and read captured output
	w.Close()
	os.Stdout = oldStdout
	
	output := make([]byte, 1024)
	n, _ := r.Read(output)
	outputStr := string(output[:n])
	
	assert.Contains(t, outputStr, "12:00:00")
	assert.Contains(t, outputStr, "success")
	assert.Contains(t, outputStr, "Test console message")
}

func TestConsoleNotificationChannel_SendWithColors(t *testing.T) {
	channel := &ConsoleNotificationChannel{
		name:    "console",
		enabled: true,
		colored: true,
	}
	
	tests := []struct {
		notificationType NotificationType
		expectedColor    string
	}{
		{NotificationTypeSuccess, "\033[32m"},  // Green
		{NotificationTypeFailure, "\033[31m"},  // Red
		{NotificationTypeWarning, "\033[33m"},  // Yellow
		{NotificationTypeProgress, "\033[34m"}, // Blue
		{NotificationTypeInfo, "\033[37m"},     // White
	}
	
	for _, test := range tests {
		t.Run(string(test.notificationType), func(t *testing.T) {
			// This test would need to capture stdout to verify colors
			// For now, we just ensure no errors occur
			notification := Notification{
				Type:      test.notificationType,
				Message:   "Test message",
				Timestamp: time.Now(),
			}
			
			err := channel.Send(context.Background(), notification)
			assert.NoError(t, err)
		})
	}
}

// Mock implementations for testing

type MockNotificationChannel struct {
	name          string
	enabled       bool
	notifications []Notification
}

func (m *MockNotificationChannel) Send(ctx context.Context, notification Notification) error {
	m.notifications = append(m.notifications, notification)
	return nil
}

func (m *MockNotificationChannel) GetName() string {
	return m.name
}

func (m *MockNotificationChannel) IsEnabled() bool {
	return m.enabled
}

type FailingNotificationChannel struct {
	name         string
	enabled      bool
	failAttempts int
	attempts     int
}

func (f *FailingNotificationChannel) Send(ctx context.Context, notification Notification) error {
	f.attempts++
	if f.attempts <= f.failAttempts {
		return errors.New("simulated failure")
	}
	return nil
}

func (f *FailingNotificationChannel) GetName() string {
	return f.name
}

func (f *FailingNotificationChannel) IsEnabled() bool {
	return f.enabled
}