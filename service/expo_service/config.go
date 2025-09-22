package expo_service

import (
	"time"
)

// Config represents the configuration for Expo push service
type Config struct {
	// Authentication
	AccessToken string `yaml:"access_token" json:"access_token"` // Expo Access Token (required for production)

	// HTTP client settings
	Timeout    time.Duration `yaml:"timeout" json:"timeout"`         // Request timeout
	MaxRetries int           `yaml:"max_retries" json:"max_retries"` // Maximum number of retries
	BaseDelay  time.Duration `yaml:"base_delay" json:"base_delay"`   // Base delay for exponential backoff

	// Push notification settings
	DefaultSound    string `yaml:"default_sound" json:"default_sound"`       // Default sound for notifications
	DefaultTTL      int    `yaml:"default_ttl" json:"default_ttl"`           // Default TTL in seconds
	DefaultPriority string `yaml:"default_priority" json:"default_priority"` // Default priority (normal/high)

	// Batch processing settings
	BatchSize int `yaml:"batch_size" json:"batch_size"` // Batch size for bulk operations

	// Rate limiting
	MaxConcurrency int `yaml:"max_concurrency" json:"max_concurrency"` // Maximum concurrent requests
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Timeout:         30 * time.Second,
		MaxRetries:      3,
		BaseDelay:       1 * time.Second,
		DefaultSound:    "default",
		DefaultTTL:      3600, // 1 hour
		DefaultPriority: "normal",
		BatchSize:       100,
		MaxConcurrency:  6, // Recommended by Expo
	}
}

// ApplyDefaults applies default values to missing configuration fields
func (c *Config) ApplyDefaults() {
	defaults := DefaultConfig()

	if c.Timeout == 0 {
		c.Timeout = defaults.Timeout
	}
	if c.MaxRetries == 0 {
		c.MaxRetries = defaults.MaxRetries
	}
	if c.BaseDelay == 0 {
		c.BaseDelay = defaults.BaseDelay
	}
	if c.DefaultSound == "" {
		c.DefaultSound = defaults.DefaultSound
	}
	if c.DefaultTTL == 0 {
		c.DefaultTTL = defaults.DefaultTTL
	}
	if c.DefaultPriority == "" {
		c.DefaultPriority = defaults.DefaultPriority
	}
	if c.BatchSize == 0 {
		c.BatchSize = defaults.BatchSize
	}
	if c.MaxConcurrency == 0 {
		c.MaxConcurrency = defaults.MaxConcurrency
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Timeout < 0 {
		c.Timeout = DefaultConfig().Timeout
	}
	if c.MaxRetries < 0 {
		c.MaxRetries = 0
	}
	if c.BaseDelay < 0 {
		c.BaseDelay = DefaultConfig().BaseDelay
	}
	if c.BatchSize <= 0 {
		c.BatchSize = DefaultConfig().BatchSize
	}
	if c.BatchSize > MaxMessagesPerRequest {
		c.BatchSize = MaxMessagesPerRequest
	}
	if c.MaxConcurrency <= 0 {
		c.MaxConcurrency = DefaultConfig().MaxConcurrency
	}

	// Validate priority
	if c.DefaultPriority != "normal" && c.DefaultPriority != "high" {
		c.DefaultPriority = "normal"
	}

	return nil
}
