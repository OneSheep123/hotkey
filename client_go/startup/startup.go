// Package startup provides utilities for graceful HotKey client startup and health checking.
// 
// This package extracts the most useful startup methods from examples and makes them
// available as a public API for production use.
package startup

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	hotkey "github.com/jd/platform/hotkey/client-go"
)

// WaitStrategy defines different waiting strategies for client startup
type WaitStrategy int

const (
	// QuickWait - Fast startup check, suitable for development (max 6 seconds)
	QuickWait WaitStrategy = iota
	// HealthCheck - Standard health check, suitable for testing (configurable timeout)
	HealthCheck
	// GracefulWait - Context-managed graceful wait, suitable for production
	GracefulWait
)

// WaitForReady waits for the HotKey client to be ready using the specified strategy.
//
// Parameters:
//   - client: The HotKey client instance
//   - strategy: The waiting strategy to use
//   - timeout: Maximum time to wait (ignored for QuickWait strategy)
//
// Returns error if the client fails to become ready within the timeout.
func WaitForReady(client *hotkey.Client, strategy WaitStrategy, timeout time.Duration) error {
	switch strategy {
	case QuickWait:
		return QuickWaitReady(client)
	case HealthCheck:
		return HealthCheckWait(client, timeout)
	case GracefulWait:
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		return GracefulWaitReady(ctx, client)
	default:
		return fmt.Errorf("unknown wait strategy: %v", strategy)
	}
}

// QuickWaitReady performs a fast startup check suitable for development environments.
// Maximum wait time is approximately 6 seconds (30 checks * 200ms).
func QuickWaitReady(client *hotkey.Client) error {
	maxChecks := 30 // About 6 seconds
	
	for i := 0; i < maxChecks; i++ {
		if client.IsStarted() {
			stats := client.GetStats()
			if stats != nil {
				log.Printf("✅ Client ready after %d checks", i+1)
				return nil
			}
		}
		
		if i%5 == 0 && i > 0 {
			log.Printf("⏳ Quick check in progress... (%d/%d)", i, maxChecks)
		}
		
		time.Sleep(200 * time.Millisecond)
	}
	
	return fmt.Errorf("client not ready after %d quick checks", maxChecks)
}

// HealthCheckWait performs a comprehensive health check with configurable timeout.
// Suitable for testing and production environments.
func HealthCheckWait(client *hotkey.Client, timeout time.Duration) error {
	log.Printf("🏥 Starting health check (timeout: %v)...", timeout)
	
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	
	timeoutChan := time.After(timeout)
	checkCount := 0
	
	for {
		select {
		case <-ticker.C:
			checkCount++
			if IsClientHealthy(client) {
				log.Printf("✅ Client healthy after %d checks", checkCount)
				return nil
			}
			
			// Log progress every 10 checks
			if checkCount%10 == 0 {
				log.Printf("🔍 Health check in progress... (check #%d)", checkCount)
			}
			
		case <-timeoutChan:
			return fmt.Errorf("health check timeout after %v (%d checks performed)", timeout, checkCount)
		}
	}
}

// GracefulWaitReady performs a context-managed graceful wait suitable for production.
func GracefulWaitReady(ctx context.Context, client *hotkey.Client) error {
	notifier := NewClientReadyNotifier(client)
	notifier.Start(ctx)
	return notifier.WaitReady(ctx)
}

// IsClientHealthy checks if the HotKey client is in a healthy state.
// This is a comprehensive health check that verifies:
// 1. Client is started
// 2. Statistics are available
// 3. Connection state is valid
func IsClientHealthy(client *hotkey.Client) bool {
	// 1. Check if client is started
	if !client.IsStarted() {
		return false
	}
	
	// 2. Check if statistics are available
	stats := client.GetStats()
	if stats == nil {
		return false
	}
	
	// 3. Check connection state
	totalConnections, hasTotalConns := stats["totalConnections"].(int)
	activeConnections, hasActiveConns := stats["activeConnections"].(int)
	
	// If connection info is not available, still initializing
	if !hasTotalConns || !hasActiveConns {
		return false
	}
	
	// If no workers configured, that's fine (totalConnections == 0)
	// If workers are configured, need at least one active connection
	if totalConnections > 0 && activeConnections == 0 {
		return false
	}
	
	// Client is healthy
	return true
}

// IsClientReady is an alias for IsClientHealthy for backward compatibility
func IsClientReady(client *hotkey.Client) bool {
	return IsClientHealthy(client)
}

// ClientReadyNotifier provides a context-aware way to wait for client readiness.
// This is suitable for production environments where graceful shutdown is important.
type ClientReadyNotifier struct {
	client    *hotkey.Client
	readyChan chan struct{}
	once      sync.Once
	isReady   bool
	mutex     sync.RWMutex
}

// NewClientReadyNotifier creates a new client ready notifier.
func NewClientReadyNotifier(client *hotkey.Client) *ClientReadyNotifier {
	return &ClientReadyNotifier{
		client:    client,
		readyChan: make(chan struct{}),
	}
}

// Start begins monitoring the client status.
func (crn *ClientReadyNotifier) Start(ctx context.Context) {
	go crn.monitor(ctx)
}

// WaitReady waits for the client to become ready.
func (crn *ClientReadyNotifier) WaitReady(ctx context.Context) error {
	select {
	case <-crn.readyChan:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// IsReady returns true if the client is ready.
func (crn *ClientReadyNotifier) IsReady() bool {
	crn.mutex.RLock()
	defer crn.mutex.RUnlock()
	return crn.isReady
}

// monitor continuously monitors the client status.
func (crn *ClientReadyNotifier) monitor(ctx context.Context) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if IsClientHealthy(crn.client) {
				crn.once.Do(func() {
					crn.mutex.Lock()
					crn.isReady = true
					crn.mutex.Unlock()
					close(crn.readyChan)
					log.Println("✅ Client is ready!")
				})
				return
			}
		}
	}
}

// GetClientStats returns formatted client statistics for monitoring.
func GetClientStats(client *hotkey.Client) map[string]interface{} {
	stats := client.GetStats()
	if stats == nil {
		return map[string]interface{}{
			"status": "unavailable",
			"ready":  false,
		}
	}

	// Add readiness status
	stats["ready"] = IsClientHealthy(client)
	stats["timestamp"] = time.Now().Format(time.RFC3339)
	
	return stats
}

// WaitOptions provides configuration for advanced waiting scenarios.
type WaitOptions struct {
	Timeout       time.Duration
	CheckInterval time.Duration
	LogProgress   bool
	OnProgress    func(attempt int, elapsed time.Duration)
	OnReady       func(elapsed time.Duration)
}

// WaitWithOptions provides advanced waiting with custom options.
func WaitWithOptions(client *hotkey.Client, opts WaitOptions) error {
	if opts.Timeout == 0 {
		opts.Timeout = 30 * time.Second
	}
	if opts.CheckInterval == 0 {
		opts.CheckInterval = 200 * time.Millisecond
	}

	startTime := time.Now()
	ticker := time.NewTicker(opts.CheckInterval)
	defer ticker.Stop()
	
	timeoutChan := time.After(opts.Timeout)
	attempt := 0
	
	for {
		select {
		case <-ticker.C:
			attempt++
			elapsed := time.Since(startTime)
			
			if IsClientHealthy(client) {
				if opts.OnReady != nil {
					opts.OnReady(elapsed)
				}
				if opts.LogProgress {
					log.Printf("✅ Client ready after %v (%d attempts)", elapsed, attempt)
				}
				return nil
			}
			
			if opts.OnProgress != nil {
				opts.OnProgress(attempt, elapsed)
			}
			
			if opts.LogProgress && attempt%10 == 0 {
				log.Printf("⏳ Waiting for client... (attempt %d, elapsed %v)", attempt, elapsed)
			}
			
		case <-timeoutChan:
			elapsed := time.Since(startTime)
			return fmt.Errorf("timeout after %v (%d attempts)", elapsed, attempt)
		}
	}
}
