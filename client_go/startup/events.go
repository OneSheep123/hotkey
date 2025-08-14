package startup

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	hotkey "github.com/jd/platform/hotkey/client-go"
)

// StartupPhase represents different phases of client startup
type StartupPhase int

const (
	// PhaseInitializing - Client is initializing
	PhaseInitializing StartupPhase = iota
	// PhaseConnecting - Client is connecting to workers
	PhaseConnecting
	// PhaseReady - Client is ready for use
	PhaseReady
	// PhaseFailed - Client startup failed
	PhaseFailed
)

// String returns the string representation of StartupPhase
func (p StartupPhase) String() string {
	switch p {
	case PhaseInitializing:
		return "initializing"
	case PhaseConnecting:
		return "connecting"
	case PhaseReady:
		return "ready"
	case PhaseFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// StartupEvent represents a startup event
type StartupEvent struct {
	Phase     StartupPhase
	Message   string
	Timestamp time.Time
	Data      map[string]interface{}
}

// EventDrivenStartup provides event-driven startup management with detailed phase tracking
type EventDrivenStartup struct {
	client       *hotkey.Client
	currentPhase StartupPhase
	events       chan StartupEvent
	subscribers  []chan StartupEvent
	mutex        sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
}

// NewEventDrivenStartup creates a new event-driven startup manager
func NewEventDrivenStartup(client *hotkey.Client) *EventDrivenStartup {
	ctx, cancel := context.WithCancel(context.Background())
	return &EventDrivenStartup{
		client:       client,
		currentPhase: PhaseInitializing,
		events:       make(chan StartupEvent, 100),
		subscribers:  make([]chan StartupEvent, 0),
		ctx:          ctx,
		cancel:       cancel,
	}
}

// Subscribe subscribes to startup events
func (eds *EventDrivenStartup) Subscribe() <-chan StartupEvent {
	eds.mutex.Lock()
	defer eds.mutex.Unlock()
	
	subscriber := make(chan StartupEvent, 10)
	eds.subscribers = append(eds.subscribers, subscriber)
	return subscriber
}

// GetCurrentPhase returns the current startup phase
func (eds *EventDrivenStartup) GetCurrentPhase() StartupPhase {
	eds.mutex.RLock()
	defer eds.mutex.RUnlock()
	return eds.currentPhase
}

// IsReady returns true if the client is ready
func (eds *EventDrivenStartup) IsReady() bool {
	return eds.GetCurrentPhase() == PhaseReady
}

// Start begins the startup monitoring process
func (eds *EventDrivenStartup) Start() {
	go eds.eventDispatcher()
	go eds.phaseMonitor()
}

// Stop stops the startup manager
func (eds *EventDrivenStartup) Stop() {
	eds.cancel()
	close(eds.events)
	
	eds.mutex.Lock()
	for _, subscriber := range eds.subscribers {
		close(subscriber)
	}
	eds.mutex.Unlock()
}

// WaitForPhase waits for a specific startup phase
func (eds *EventDrivenStartup) WaitForPhase(targetPhase StartupPhase, timeout time.Duration) error {
	if eds.GetCurrentPhase() == targetPhase {
		return nil
	}

	subscriber := eds.Subscribe()
	defer func() {
		// Clean up subscriber
		eds.mutex.Lock()
		for i, sub := range eds.subscribers {
			if sub == subscriber {
				eds.subscribers = append(eds.subscribers[:i], eds.subscribers[i+1:]...)
				break
			}
		}
		eds.mutex.Unlock()
	}()

	timeoutChan := time.After(timeout)
	
	for {
		select {
		case event := <-subscriber:
			if event.Phase == targetPhase {
				return nil
			}
			if event.Phase == PhaseFailed {
				return fmt.Errorf("startup failed: %s", event.Message)
			}
		case <-timeoutChan:
			return fmt.Errorf("timeout waiting for phase %s", targetPhase.String())
		case <-eds.ctx.Done():
			return eds.ctx.Err()
		}
	}
}

// publishEvent publishes a startup event
func (eds *EventDrivenStartup) publishEvent(phase StartupPhase, message string, data map[string]interface{}) {
	event := StartupEvent{
		Phase:     phase,
		Message:   message,
		Timestamp: time.Now(),
		Data:      data,
	}

	select {
	case eds.events <- event:
	case <-eds.ctx.Done():
	}
}

// eventDispatcher dispatches events to subscribers
func (eds *EventDrivenStartup) eventDispatcher() {
	for {
		select {
		case event := <-eds.events:
			eds.mutex.Lock()
			eds.currentPhase = event.Phase
			for _, subscriber := range eds.subscribers {
				select {
				case subscriber <- event:
				default:
					// Subscriber buffer full, skip
				}
			}
			eds.mutex.Unlock()
			
			log.Printf("📡 Phase: %s - %s", event.Phase.String(), event.Message)
			
		case <-eds.ctx.Done():
			return
		}
	}
}

// phaseMonitor monitors and updates startup phases
func (eds *EventDrivenStartup) phaseMonitor() {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	eds.publishEvent(PhaseInitializing, "Starting client initialization", nil)

	for {
		select {
		case <-ticker.C:
			eds.checkAndUpdatePhase()
		case <-eds.ctx.Done():
			return
		}
	}
}

// checkAndUpdatePhase checks current state and updates phase if needed
func (eds *EventDrivenStartup) checkAndUpdatePhase() {
	currentPhase := eds.GetCurrentPhase()
	
	if currentPhase == PhaseReady || currentPhase == PhaseFailed {
		return
	}

	if !eds.client.IsStarted() {
		if currentPhase != PhaseInitializing {
			eds.publishEvent(PhaseInitializing, "Client not started yet", nil)
		}
		return
	}

	stats := eds.client.GetStats()
	if stats == nil {
		if currentPhase != PhaseConnecting {
			eds.publishEvent(PhaseConnecting, "Waiting for client stats", nil)
		}
		return
	}

	totalConnections, hasTotalConns := stats["totalConnections"].(int)
	activeConnections, hasActiveConns := stats["activeConnections"].(int)

	if !hasTotalConns || !hasActiveConns {
		if currentPhase != PhaseConnecting {
			eds.publishEvent(PhaseConnecting, "Waiting for connection info", nil)
		}
		return
	}

	if totalConnections == 0 {
		if currentPhase != PhaseConnecting {
			eds.publishEvent(PhaseConnecting, "No worker connections configured", nil)
		}
		return
	}

	if activeConnections == 0 {
		if currentPhase != PhaseConnecting {
			eds.publishEvent(PhaseConnecting, 
				fmt.Sprintf("Connecting to workers (%d configured)", totalConnections), 
				map[string]interface{}{
					"totalConnections": totalConnections,
					"activeConnections": activeConnections,
				})
		}
		return
	}

	// Client is ready
	if currentPhase != PhaseReady {
		eds.publishEvent(PhaseReady, 
			fmt.Sprintf("Client ready with %d/%d active connections", activeConnections, totalConnections),
			stats)
	}
}

// StartupEventLogger provides a convenient way to log startup events
type StartupEventLogger struct {
	events <-chan StartupEvent
	done   chan struct{}
}

// NewStartupEventLogger creates a new startup event logger
func NewStartupEventLogger(startup *EventDrivenStartup) *StartupEventLogger {
	events := startup.Subscribe()
	logger := &StartupEventLogger{
		events: events,
		done:   make(chan struct{}),
	}
	
	go logger.logEvents()
	return logger
}

// Stop stops the event logger
func (sel *StartupEventLogger) Stop() {
	close(sel.done)
}

// logEvents logs startup events with formatted output
func (sel *StartupEventLogger) logEvents() {
	for {
		select {
		case event := <-sel.events:
			sel.logEvent(event)
		case <-sel.done:
			return
		}
	}
}

// logEvent logs a single startup event
func (sel *StartupEventLogger) logEvent(event StartupEvent) {
	emoji := sel.getPhaseEmoji(event.Phase)
	log.Printf("%s [%s] %s", emoji, event.Phase.String(), event.Message)
	
	if event.Data != nil && len(event.Data) > 0 {
		log.Printf("   📊 Data: %+v", event.Data)
	}
}

// getPhaseEmoji returns an emoji for the given phase
func (sel *StartupEventLogger) getPhaseEmoji(phase StartupPhase) string {
	switch phase {
	case PhaseInitializing:
		return "🚀"
	case PhaseConnecting:
		return "🔗"
	case PhaseReady:
		return "✅"
	case PhaseFailed:
		return "❌"
	default:
		return "❓"
	}
}
