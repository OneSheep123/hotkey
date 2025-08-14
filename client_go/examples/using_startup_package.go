package main

import (
	"context"
	"log"
	"time"

	hotkey "github.com/jd/platform/hotkey/client-go"
	"github.com/jd/platform/hotkey/client-go/startup"
)

func main() {
	log.Println("🚀 Demonstrating HotKey Startup Package Usage")

	// Create client
	client, err := hotkey.NewClientBuilder().
		SetAppName("startup-package-demo").
		SetEtcdServer("http://127.0.0.1:2379").
		SetPushPeriod(500).
		SetCacheSize(100000).
		Build()

	if err != nil {
		log.Fatal("❌ Failed to create client:", err)
	}

	// Start client
	if err := client.Start(); err != nil {
		log.Fatal("❌ Failed to start client:", err)
	}
	defer client.Stop()

	// Demonstrate different startup strategies
	demonstrateStartupStrategies(client)

	// Demonstrate event-driven startup
	demonstrateEventDrivenStartup(client)

	// Demonstrate advanced options
	demonstrateAdvancedOptions(client)

	log.Println("🎉 All startup package demonstrations completed!")
}

func demonstrateStartupStrategies(client *hotkey.Client) {
	log.Println("\n📋 === Demonstrating Startup Strategies ===")

	// 1. Quick Wait Strategy (Development)
	log.Println("\n🚀 1. Quick Wait Strategy (Development)")
	start := time.Now()
	if err := startup.WaitForReady(client, startup.QuickWait, 0); err != nil {
		log.Printf("⚠️ Quick wait warning: %v (elapsed: %v)", err, time.Since(start))
	} else {
		log.Printf("✅ Quick wait successful (elapsed: %v)", time.Since(start))
	}

	// 2. Health Check Strategy (Testing)
	log.Println("\n🏥 2. Health Check Strategy (Testing)")
	start = time.Now()
	if err := startup.WaitForReady(client, startup.HealthCheck, 10*time.Second); err != nil {
		log.Printf("❌ Health check failed: %v (elapsed: %v)", err, time.Since(start))
	} else {
		log.Printf("✅ Health check successful (elapsed: %v)", time.Since(start))
	}

	// 3. Graceful Wait Strategy (Production)
	log.Println("\n🛡️ 3. Graceful Wait Strategy (Production)")
	start = time.Now()
	if err := startup.WaitForReady(client, startup.GracefulWait, 15*time.Second); err != nil {
		log.Printf("❌ Graceful wait failed: %v (elapsed: %v)", err, time.Since(start))
	} else {
		log.Printf("✅ Graceful wait successful (elapsed: %v)", time.Since(start))
	}

	// Check current health status
	log.Println("\n🔍 Current Health Status:")
	if startup.IsClientHealthy(client) {
		log.Println("✅ Client is healthy")
	} else {
		log.Println("❌ Client is not healthy")
	}

	// Get detailed stats
	stats := startup.GetClientStats(client)
	log.Printf("📊 Client stats: %+v", stats)
}

func demonstrateEventDrivenStartup(client *hotkey.Client) {
	log.Println("\n📋 === Demonstrating Event-Driven Startup ===")

	// Create event-driven startup manager
	eventStartup := startup.NewEventDrivenStartup(client)
	defer eventStartup.Stop()

	// Create event logger for automatic logging
	logger := startup.NewStartupEventLogger(eventStartup)
	defer logger.Stop()

	// Subscribe to events manually for demonstration
	events := eventStartup.Subscribe()
	eventCount := 0
	go func() {
		for event := range events {
			eventCount++
			log.Printf("📡 Event #%d: [%s] %s", eventCount, event.Phase.String(), event.Message)
			if event.Data != nil && len(event.Data) > 0 {
				log.Printf("   📊 Event data: %+v", event.Data)
			}
		}
	}()

	// Start monitoring
	eventStartup.Start()

	// Wait for ready phase
	log.Println("⏳ Waiting for PhaseReady...")
	start := time.Now()
	if err := eventStartup.WaitForPhase(startup.PhaseReady, 20*time.Second); err != nil {
		log.Printf("❌ Event-driven startup failed: %v (elapsed: %v)", err, time.Since(start))
	} else {
		log.Printf("✅ Event-driven startup successful (elapsed: %v)", time.Since(start))
	}

	// Check current phase
	currentPhase := eventStartup.GetCurrentPhase()
	log.Printf("📍 Current phase: %s", currentPhase.String())
	log.Printf("🎯 Is ready: %v", eventStartup.IsReady())

	// Give some time for events to be processed
	time.Sleep(500 * time.Millisecond)
}

func demonstrateAdvancedOptions(client *hotkey.Client) {
	log.Println("\n📋 === Demonstrating Advanced Options ===")

	// 1. Context-aware waiting
	log.Println("\n🔄 1. Context-Aware Waiting")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	notifier := startup.NewClientReadyNotifier(client)
	notifier.Start(ctx)

	start := time.Now()
	if err := notifier.WaitReady(ctx); err != nil {
		log.Printf("❌ Context wait failed: %v (elapsed: %v)", err, time.Since(start))
	} else {
		log.Printf("✅ Context wait successful (elapsed: %v)", time.Since(start))
		log.Printf("🎯 Notifier ready status: %v", notifier.IsReady())
	}

	// 2. Custom wait options
	log.Println("\n⚙️ 2. Custom Wait Options")
	progressCount := 0
	opts := startup.WaitOptions{
		Timeout:       15 * time.Second,
		CheckInterval: 300 * time.Millisecond,
		LogProgress:   true,
		OnProgress: func(attempt int, elapsed time.Duration) {
			progressCount++
			if attempt%5 == 0 {
				log.Printf("   🔄 Custom progress: attempt %d, elapsed %v", attempt, elapsed)
			}
		},
		OnReady: func(elapsed time.Duration) {
			log.Printf("   🎉 Custom ready callback: Client ready in %v!", elapsed)
		},
	}

	start = time.Now()
	if err := startup.WaitWithOptions(client, opts); err != nil {
		log.Printf("❌ Custom wait failed: %v (elapsed: %v)", err, time.Since(start))
	} else {
		log.Printf("✅ Custom wait successful (elapsed: %v)", time.Since(start))
	}
	log.Printf("📊 Progress callbacks called: %d times", progressCount)

	// 3. Manual health checking
	log.Println("\n🔍 3. Manual Health Checking")
	for i := 0; i < 3; i++ {
		healthy := startup.IsClientHealthy(client)
		stats := startup.GetClientStats(client)
		
		log.Printf("   Check #%d: healthy=%v, ready=%v", i+1, healthy, stats["ready"])
		
		if i < 2 {
			time.Sleep(500 * time.Millisecond)
		}
	}

	// 4. Demonstrate different startup phases
	log.Println("\n📊 4. Startup Phase Information")
	phases := []startup.StartupPhase{
		startup.PhaseInitializing,
		startup.PhaseConnecting,
		startup.PhaseReady,
		startup.PhaseFailed,
	}

	for _, phase := range phases {
		log.Printf("   Phase: %s (value: %d)", phase.String(), int(phase))
	}
}

// demonstrateErrorHandling shows how to handle various error scenarios
func demonstrateErrorHandling() {
	log.Println("\n📋 === Demonstrating Error Handling ===")

	// This would be called with a problematic client
	// For demonstration, we'll show the patterns

	log.Println("🛡️ Error Handling Patterns:")
	log.Println("   1. Quick wait with fallback:")
	log.Println("      if err := startup.WaitForReady(client, startup.QuickWait, 0); err != nil {")
	log.Println("          log.Printf(\"Warning: %v\", err)")
	log.Println("          // Continue with degraded functionality")
	log.Println("      }")

	log.Println("   2. Health check with retry:")
	log.Println("      for i := 0; i < 3; i++ {")
	log.Println("          if err := startup.WaitForReady(client, startup.HealthCheck, 10*time.Second); err == nil {")
	log.Println("              break")
	log.Println("          }")
	log.Println("          log.Printf(\"Retry %d failed: %v\", i+1, err)")
	log.Println("      }")

	log.Println("   3. Graceful wait with context:")
	log.Println("      ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)")
	log.Println("      defer cancel()")
	log.Println("      if err := startup.WaitForReady(client, startup.GracefulWait, 30*time.Second); err != nil {")
	log.Println("          if ctx.Err() == context.DeadlineExceeded {")
	log.Println("              log.Println(\"Startup timeout - check etcd connectivity\")")
	log.Println("          }")
	log.Println("      }")
}

func init() {
	// Set up logging format
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
}
