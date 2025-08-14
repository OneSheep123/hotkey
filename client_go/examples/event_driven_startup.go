package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	hotkey "github.com/jd/platform/hotkey/client-go"
)

// StartupPhase 启动阶段枚举
type StartupPhase int

const (
	PhaseInitializing StartupPhase = iota
	PhaseConnecting
	PhaseReady
	PhaseFailed
)

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

// StartupEvent 启动事件
type StartupEvent struct {
	Phase     StartupPhase
	Message   string
	Timestamp time.Time
	Data      map[string]interface{}
}

// EventDrivenStartup 事件驱动的启动管理器
type EventDrivenStartup struct {
	client       *hotkey.Client
	currentPhase StartupPhase
	events       chan StartupEvent
	subscribers  []chan StartupEvent
	mutex        sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
}

// NewEventDrivenStartup 创建事件驱动启动管理器
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

// Subscribe 订阅启动事件
func (eds *EventDrivenStartup) Subscribe() <-chan StartupEvent {
	eds.mutex.Lock()
	defer eds.mutex.Unlock()
	
	subscriber := make(chan StartupEvent, 10)
	eds.subscribers = append(eds.subscribers, subscriber)
	return subscriber
}

// GetCurrentPhase 获取当前阶段
func (eds *EventDrivenStartup) GetCurrentPhase() StartupPhase {
	eds.mutex.RLock()
	defer eds.mutex.RUnlock()
	return eds.currentPhase
}

// IsReady 检查是否就绪
func (eds *EventDrivenStartup) IsReady() bool {
	return eds.GetCurrentPhase() == PhaseReady
}

// Start 开始启动过程
func (eds *EventDrivenStartup) Start() {
	go eds.eventDispatcher()
	go eds.phaseMonitor()
}

// Stop 停止启动管理器
func (eds *EventDrivenStartup) Stop() {
	eds.cancel()
	close(eds.events)
	
	eds.mutex.Lock()
	for _, subscriber := range eds.subscribers {
		close(subscriber)
	}
	eds.mutex.Unlock()
}

// WaitForPhase 等待特定阶段
func (eds *EventDrivenStartup) WaitForPhase(targetPhase StartupPhase, timeout time.Duration) error {
	if eds.GetCurrentPhase() == targetPhase {
		return nil
	}

	subscriber := eds.Subscribe()
	defer func() {
		// 清理订阅者
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

// publishEvent 发布事件
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

// eventDispatcher 事件分发器
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
					// 订阅者缓冲区满，跳过
				}
			}
			eds.mutex.Unlock()
			
			log.Printf("📡 Phase: %s - %s", event.Phase.String(), event.Message)
			
		case <-eds.ctx.Done():
			return
		}
	}
}

// phaseMonitor 阶段监控器
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

// checkAndUpdatePhase 检查并更新阶段
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

	// 客户端就绪
	if currentPhase != PhaseReady {
		eds.publishEvent(PhaseReady, 
			fmt.Sprintf("Client ready with %d/%d active connections", activeConnections, totalConnections),
			stats)
	}
}

func main() {
	// 创建客户端
	client, err := hotkey.NewClientBuilder().
		SetAppName("event-driven-sample").
		SetEtcdServer("http://127.0.0.1:12379").
		SetPushPeriod(500).
		SetCacheSize(200000).
		Build()

	if err != nil {
		log.Fatal("Failed to create hotkey client:", err)
	}

	// 创建事件驱动启动管理器
	startup := NewEventDrivenStartup(client)
	defer startup.Stop()

	// 订阅启动事件
	events := startup.Subscribe()
	go func() {
		for event := range events {
			log.Printf("🎯 Event: [%s] %s", event.Phase.String(), event.Message)
			if event.Data != nil {
				if data, err := json.Marshal(event.Data); err == nil {
					log.Printf("   Data: %s", string(data))
				}
			}
		}
	}()

	// 启动客户端
	if err := client.Start(); err != nil {
		log.Fatal("Failed to start hotkey client:", err)
	}
	defer client.Stop()

	// 开始监控
	startup.Start()

	log.Println("🚀 Starting event-driven client initialization...")

	// 等待客户端就绪
	if err := startup.WaitForPhase(PhaseReady, 30*time.Second); err != nil {
		log.Fatal("❌ Client startup failed:", err)
	}

	log.Println("✅ Client is ready! Starting HTTP server...")

	// 启动HTTP服务器
	startEventDrivenServer(client, startup)
}

func startEventDrivenServer(client *hotkey.Client, startup *EventDrivenStartup) {
	// 状态接口
	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		status := map[string]interface{}{
			"phase":     startup.GetCurrentPhase().String(),
			"ready":     startup.IsReady(),
			"timestamp": time.Now().Format(time.RFC3339),
		}

		if startup.IsReady() {
			status["stats"] = client.GetStats()
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(status)
	})

	// 热key检测接口
	http.HandleFunc("/check", func(w http.ResponseWriter, r *http.Request) {
		if !startup.IsReady() {
			http.Error(w, "Service not ready", http.StatusServiceUnavailable)
			return
		}

		key := r.URL.Query().Get("key")
		if key == "" {
			http.Error(w, "key parameter required", http.StatusBadRequest)
			return
		}

		result := map[string]interface{}{
			"key":       key,
			"isHot":     hotkey.IsHotKey(key),
			"phase":     startup.GetCurrentPhase().String(),
			"timestamp": time.Now().Format(time.RFC3339),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	})

	log.Println("🌐 Event-driven server running on :8080")
	log.Println("📋 Endpoints:")
	log.Println("  GET /status           - Get startup status")
	log.Println("  GET /check?key=xxx    - Check hot key")

	log.Fatal(http.ListenAndServe(":8080", nil))
}
