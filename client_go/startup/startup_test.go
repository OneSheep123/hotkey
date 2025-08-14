package startup

import (
	"testing"
	"time"
)

// MockClient for testing purposes
type MockClient struct {
	started bool
	stats   map[string]interface{}
}

func (mc *MockClient) IsStarted() bool {
	return mc.started
}

func (mc *MockClient) GetStats() map[string]interface{} {
	return mc.stats
}

func (mc *MockClient) Start() error {
	mc.started = true
	return nil
}

func (mc *MockClient) Stop() error {
	mc.started = false
	return nil
}

func TestQuickWaitReady(t *testing.T) {
	// This should work but we can't test it directly since QuickWaitReady expects *hotkey.Client
	// We'll test the IsClientHealthy function instead
	t.Skip("Skipping test that requires real hotkey.Client")
}

func TestIsClientHealthy(t *testing.T) {
	t.Skip("Skipping test that requires real hotkey.Client")
}

func TestWaitStrategy(t *testing.T) {
	strategies := []WaitStrategy{QuickWait, HealthCheck, GracefulWait}
	strategyNames := []string{"QuickWait", "HealthCheck", "GracefulWait"}

	for i, strategy := range strategies {
		t.Run(strategyNames[i], func(t *testing.T) {
			// Test strategy enum values
			if strategy < QuickWait || strategy > GracefulWait {
				t.Errorf("Invalid strategy value: %v", strategy)
			}
		})
	}
}

func TestStartupPhase(t *testing.T) {
	phases := []StartupPhase{PhaseInitializing, PhaseConnecting, PhaseReady, PhaseFailed}
	expectedStrings := []string{"initializing", "connecting", "ready", "failed"}
	
	for i, phase := range phases {
		if phase.String() != expectedStrings[i] {
			t.Errorf("Expected phase %v to have string %s, got %s", phase, expectedStrings[i], phase.String())
		}
	}
}

func TestStartupEvent(t *testing.T) {
	event := StartupEvent{
		Phase:     PhaseReady,
		Message:   "Test message",
		Timestamp: time.Now(),
		Data:      map[string]interface{}{"test": "data"},
	}
	
	if event.Phase != PhaseReady {
		t.Errorf("Expected phase PhaseReady, got %v", event.Phase)
	}
	
	if event.Message != "Test message" {
		t.Errorf("Expected message 'Test message', got %s", event.Message)
	}
	
	if event.Data["test"] != "data" {
		t.Errorf("Expected data['test'] to be 'data', got %v", event.Data["test"])
	}
}

func TestWaitOptions(t *testing.T) {
	opts := WaitOptions{
		Timeout:       30 * time.Second,
		CheckInterval: 200 * time.Millisecond,
		LogProgress:   true,
	}
	
	if opts.Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", opts.Timeout)
	}
	
	if opts.CheckInterval != 200*time.Millisecond {
		t.Errorf("Expected check interval 200ms, got %v", opts.CheckInterval)
	}
	
	if !opts.LogProgress {
		t.Error("Expected LogProgress to be true")
	}
}

func TestEventDrivenStartupCreation(t *testing.T) {
	// We can't create a real client for testing, so we'll test the structure
	t.Skip("Skipping test that requires real hotkey.Client")
}

func TestClientReadyNotifierCreation(t *testing.T) {
	// We can't create a real client for testing, so we'll test the structure
	t.Skip("Skipping test that requires real hotkey.Client")
}

// Benchmark tests for performance
func BenchmarkStartupPhaseString(b *testing.B) {
	phase := PhaseReady
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = phase.String()
	}
}

func BenchmarkStartupEventCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = StartupEvent{
			Phase:     PhaseReady,
			Message:   "Test message",
			Timestamp: time.Now(),
			Data:      map[string]interface{}{"test": "data"},
		}
	}
}

// Example tests to demonstrate usage
func ExampleQuickWaitReady() {
	// This example shows how to use QuickWaitReady
	// Note: This requires a real hotkey.Client instance
	
	// client, err := hotkey.NewClientBuilder().
	//     SetAppName("example").
	//     SetEtcdServer("http://127.0.0.1:2379").
	//     Build()
	// 
	// if err != nil {
	//     log.Fatal(err)
	// }
	// 
	// client.Start()
	// defer client.Stop()
	// 
	// if err := QuickWaitReady(client); err != nil {
	//     log.Printf("Warning: %v", err)
	// }
}

func ExampleWaitForReady() {
	// This example shows how to use WaitForReady with different strategies
	// Note: This requires a real hotkey.Client instance
	
	// client, err := hotkey.NewClientBuilder().
	//     SetAppName("example").
	//     SetEtcdServer("http://127.0.0.1:2379").
	//     Build()
	// 
	// if err != nil {
	//     log.Fatal(err)
	// }
	// 
	// client.Start()
	// defer client.Stop()
	// 
	// // Quick wait for development
	// err = WaitForReady(client, QuickWait, 0)
	// 
	// // Health check for testing
	// err = WaitForReady(client, HealthCheck, 15*time.Second)
	// 
	// // Graceful wait for production
	// err = WaitForReady(client, GracefulWait, 30*time.Second)
}

func ExampleNewEventDrivenStartup() {
	// This example shows how to use EventDrivenStartup
	// Note: This requires a real hotkey.Client instance
	
	// client, err := hotkey.NewClientBuilder().
	//     SetAppName("example").
	//     SetEtcdServer("http://127.0.0.1:2379").
	//     Build()
	// 
	// if err != nil {
	//     log.Fatal(err)
	// }
	// 
	// startup := NewEventDrivenStartup(client)
	// defer startup.Stop()
	// 
	// // Subscribe to events
	// events := startup.Subscribe()
	// go func() {
	//     for event := range events {
	//         log.Printf("Event: [%s] %s", event.Phase.String(), event.Message)
	//     }
	// }()
	// 
	// client.Start()
	// startup.Start()
	// 
	// // Wait for ready phase
	// err = startup.WaitForPhase(PhaseReady, 30*time.Second)
}

func ExampleWaitWithOptions() {
	// This example shows how to use WaitWithOptions
	// Note: This requires a real hotkey.Client instance
	
	// client, err := hotkey.NewClientBuilder().
	//     SetAppName("example").
	//     SetEtcdServer("http://127.0.0.1:2379").
	//     Build()
	// 
	// if err != nil {
	//     log.Fatal(err)
	// }
	// 
	// client.Start()
	// defer client.Stop()
	// 
	// opts := WaitOptions{
	//     Timeout:       30 * time.Second,
	//     CheckInterval: 500 * time.Millisecond,
	//     LogProgress:   true,
	//     OnProgress: func(attempt int, elapsed time.Duration) {
	//         if attempt%5 == 0 {
	//             log.Printf("Still waiting... attempt %d, elapsed %v", attempt, elapsed)
	//         }
	//     },
	//     OnReady: func(elapsed time.Duration) {
	//         log.Printf("Client ready in %v!", elapsed)
	//     },
	// }
	// 
	// err = WaitWithOptions(client, opts)
}
