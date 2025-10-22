package distribution

import (
	"context"
	"fmt"
	"log"
	"time"
)

// DefaultNotificationService provides basic notification functionality
type DefaultNotificationService struct {
	config   NotificationServiceConfig
	channels map[string]NotificationChannel
}

// NotificationServiceConfig contains notification service configuration
type NotificationServiceConfig struct {
	Enabled      bool                              `json:"enabled"`
	Channels     map[string]NotificationChannelConfig `json:"channels"`
	Templates    map[string]string                 `json:"templates"`
	RetryPolicy  NotificationRetryPolicy           `json:"retry_policy"`
}

// NotificationChannelConfig contains channel-specific configuration
type NotificationChannelConfig struct {
	Type     string                 `json:"type"`
	Enabled  bool                   `json:"enabled"`
	Config   map[string]interface{} `json:"config"`
	Priority int                    `json:"priority"`
}

// NotificationRetryPolicy defines retry behavior for notifications
type NotificationRetryPolicy struct {
	MaxRetries int           `json:"max_retries"`
	BaseDelay  time.Duration `json:"base_delay"`
	MaxDelay   time.Duration `json:"max_delay"`
}

// NotificationChannel defines the interface for notification channels
type NotificationChannel interface {
	Send(ctx context.Context, notification Notification) error
	GetName() string
	IsEnabled() bool
}

// Notification represents a notification message
type Notification struct {
	Type      NotificationType      `json:"type"`
	Title     string                `json:"title"`
	Message   string                `json:"message"`
	Publisher string                `json:"publisher"`
	Release   Release               `json:"release"`
	Error     error                 `json:"error,omitempty"`
	Metadata  map[string]interface{} `json:"metadata"`
	Timestamp time.Time             `json:"timestamp"`
}

// NotificationType represents the type of notification
type NotificationType string

const (
	NotificationTypeSuccess  NotificationType = "success"
	NotificationTypeFailure  NotificationType = "failure"
	NotificationTypeProgress NotificationType = "progress"
	NotificationTypeInfo     NotificationType = "info"
	NotificationTypeWarning  NotificationType = "warning"
)

// LogNotificationChannel sends notifications to the log
type LogNotificationChannel struct {
	name    string
	enabled bool
	logger  *log.Logger
}

// ConsoleNotificationChannel sends notifications to console
type ConsoleNotificationChannel struct {
	name    string
	enabled bool
	colored bool
}

// NewDefaultNotificationService creates a new notification service
func NewDefaultNotificationService(config NotificationServiceConfig) *DefaultNotificationService {
	service := &DefaultNotificationService{
		config:   config,
		channels: make(map[string]NotificationChannel),
	}
	
	// Initialize default channels
	service.initializeDefaultChannels()
	
	return service
}

// initializeDefaultChannels initializes default notification channels
func (dns *DefaultNotificationService) initializeDefaultChannels() {
	// Add log channel
	logChannel := &LogNotificationChannel{
		name:    "log",
		enabled: true,
		logger:  log.Default(),
	}
	dns.channels["log"] = logChannel
	
	// Add console channel
	consoleChannel := &ConsoleNotificationChannel{
		name:    "console",
		enabled: true,
		colored: true,
	}
	dns.channels["console"] = consoleChannel
}

// RegisterChannel registers a new notification channel
func (dns *DefaultNotificationService) RegisterChannel(channel NotificationChannel) {
	dns.channels[channel.GetName()] = channel
}

// NotifySuccess sends a success notification
func (dns *DefaultNotificationService) NotifySuccess(publisher string, release Release) error {
	if !dns.config.Enabled {
		return nil
	}
	
	notification := Notification{
		Type:      NotificationTypeSuccess,
		Title:     "Release Published Successfully",
		Message:   fmt.Sprintf("Release %s published successfully to %s", release.Version, publisher),
		Publisher: publisher,
		Release:   release,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"version": release.Version,
			"tag":     release.Tag,
		},
	}
	
	return dns.sendNotification(context.Background(), notification)
}

// NotifyFailure sends a failure notification
func (dns *DefaultNotificationService) NotifyFailure(publisher string, release Release, err error) error {
	if !dns.config.Enabled {
		return nil
	}
	
	notification := Notification{
		Type:      NotificationTypeFailure,
		Title:     "Release Publishing Failed",
		Message:   fmt.Sprintf("Failed to publish release %s to %s: %v", release.Version, publisher, err),
		Publisher: publisher,
		Release:   release,
		Error:     err,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"version": release.Version,
			"tag":     release.Tag,
			"error":   err.Error(),
		},
	}
	
	return dns.sendNotification(context.Background(), notification)
}

// NotifyProgress sends a progress notification
func (dns *DefaultNotificationService) NotifyProgress(publisher string, release Release, progress float64) error {
	if !dns.config.Enabled {
		return nil
	}
	
	notification := Notification{
		Type:      NotificationTypeProgress,
		Title:     "Release Publishing Progress",
		Message:   fmt.Sprintf("Publishing release %s to %s: %.1f%% complete", release.Version, publisher, progress*100),
		Publisher: publisher,
		Release:   release,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"version":  release.Version,
			"tag":      release.Tag,
			"progress": progress,
		},
	}
	
	return dns.sendNotification(context.Background(), notification)
}

// sendNotification sends a notification to all enabled channels
func (dns *DefaultNotificationService) sendNotification(ctx context.Context, notification Notification) error {
	var lastError error
	
	for _, channel := range dns.channels {
		if !channel.IsEnabled() {
			continue
		}
		
		if err := dns.sendWithRetry(ctx, channel, notification); err != nil {
			lastError = err
			log.Printf("Failed to send notification via %s: %v", channel.GetName(), err)
		}
	}
	
	return lastError
}

// sendWithRetry sends a notification with retry logic
func (dns *DefaultNotificationService) sendWithRetry(ctx context.Context, channel NotificationChannel, notification Notification) error {
	policy := dns.config.RetryPolicy
	
	for attempt := 0; attempt <= policy.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := dns.calculateBackoffDelay(attempt, policy)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}
		
		if err := channel.Send(ctx, notification); err != nil {
			if attempt == policy.MaxRetries {
				return fmt.Errorf("failed after %d attempts: %w", policy.MaxRetries+1, err)
			}
			continue
		}
		
		return nil
	}
	
	return fmt.Errorf("unexpected retry loop exit")
}

// calculateBackoffDelay calculates exponential backoff delay
func (dns *DefaultNotificationService) calculateBackoffDelay(attempt int, policy NotificationRetryPolicy) time.Duration {
	delay := float64(policy.BaseDelay) * pow(2.0, float64(attempt-1))
	if delay > float64(policy.MaxDelay) {
		delay = float64(policy.MaxDelay)
	}
	return time.Duration(delay)
}

// LogNotificationChannel implementation

// Send sends a notification to the log
func (lnc *LogNotificationChannel) Send(ctx context.Context, notification Notification) error {
	level := "INFO"
	switch notification.Type {
	case NotificationTypeFailure:
		level = "ERROR"
	case NotificationTypeWarning:
		level = "WARN"
	case NotificationTypeSuccess:
		level = "INFO"
	case NotificationTypeProgress:
		level = "DEBUG"
	}
	
	message := fmt.Sprintf("[%s] %s: %s", level, notification.Title, notification.Message)
	if notification.Error != nil {
		message += fmt.Sprintf(" (Error: %v)", notification.Error)
	}
	
	lnc.logger.Println(message)
	return nil
}

// GetName returns the channel name
func (lnc *LogNotificationChannel) GetName() string {
	return lnc.name
}

// IsEnabled returns whether the channel is enabled
func (lnc *LogNotificationChannel) IsEnabled() bool {
	return lnc.enabled
}

// ConsoleNotificationChannel implementation

// Send sends a notification to the console
func (cnc *ConsoleNotificationChannel) Send(ctx context.Context, notification Notification) error {
	var color string
	var reset string
	
	if cnc.colored {
		reset = "\033[0m"
		switch notification.Type {
		case NotificationTypeSuccess:
			color = "\033[32m" // Green
		case NotificationTypeFailure:
			color = "\033[31m" // Red
		case NotificationTypeWarning:
			color = "\033[33m" // Yellow
		case NotificationTypeProgress:
			color = "\033[34m" // Blue
		default:
			color = "\033[37m" // White
		}
	}
	
	timestamp := notification.Timestamp.Format("15:04:05")
	message := fmt.Sprintf("%s[%s] %s%s: %s%s", 
		color, timestamp, notification.Type, reset, notification.Message, reset)
	
	if notification.Error != nil {
		message += fmt.Sprintf("\n%sError: %v%s", color, notification.Error, reset)
	}
	
	fmt.Println(message)
	return nil
}

// GetName returns the channel name
func (cnc *ConsoleNotificationChannel) GetName() string {
	return cnc.name
}

// IsEnabled returns whether the channel is enabled
func (cnc *ConsoleNotificationChannel) IsEnabled() bool {
	return cnc.enabled
}