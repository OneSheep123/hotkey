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

// ClientReadyNotifier 客户端就绪通知器
type ClientReadyNotifier struct {
	client     *hotkey.Client
	readyChan  chan struct{}
	once       sync.Once
	isReady    bool
	mutex      sync.RWMutex
}

// NewClientReadyNotifier 创建客户端就绪通知器
func NewClientReadyNotifier(client *hotkey.Client) *ClientReadyNotifier {
	return &ClientReadyNotifier{
		client:    client,
		readyChan: make(chan struct{}),
	}
}

// Start 开始监控客户端状态
func (crn *ClientReadyNotifier) Start(ctx context.Context) {
	go crn.monitor(ctx)
}

// WaitReady 等待客户端就绪
func (crn *ClientReadyNotifier) WaitReady(ctx context.Context) error {
	select {
	case <-crn.readyChan:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// IsReady 检查客户端是否就绪
func (crn *ClientReadyNotifier) IsReady() bool {
	crn.mutex.RLock()
	defer crn.mutex.RUnlock()
	return crn.isReady
}

// monitor 监控客户端状态
func (crn *ClientReadyNotifier) monitor(ctx context.Context) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if crn.checkClientReady() {
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

// checkClientReady 检查客户端是否准备就绪
func (crn *ClientReadyNotifier) checkClientReady() bool {
	if !crn.client.IsStarted() {
		return false
	}

	stats := crn.client.GetStats()
	if stats == nil {
		return false
	}

	// 检查是否有活跃连接
	if activeConnections, ok := stats["activeConnections"].(int); ok && activeConnections > 0 {
		return true
	}

	return false
}

func main() {
	// 创建客户端
	client, err := hotkey.NewClientBuilder().
		SetAppName("graceful-sample").
		SetEtcdServer("http://127.0.0.1:12379").
		SetPushPeriod(500).
		SetCacheSize(200000).
		SetCountPeriod(10).
		Build()

	if err != nil {
		log.Fatal("Failed to create hotkey client:", err)
	}

	// 启动客户端
	if err := client.Start(); err != nil {
		log.Fatal("Failed to start hotkey client:", err)
	}
	defer client.Stop()

	log.Println("🚀 HotKey client started, waiting for ready...")

	// 创建就绪通知器
	notifier := NewClientReadyNotifier(client)
	
	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 开始监控
	notifier.Start(ctx)

	// 等待客户端就绪
	if err := notifier.WaitReady(ctx); err != nil {
		log.Fatal("❌ Client failed to become ready:", err)
	}

	// 启动HTTP服务器
	startGracefulHTTPServer(client, notifier)
}

func startGracefulHTTPServer(client *hotkey.Client, notifier *ClientReadyNotifier) {
	// 健康检查接口 - 更详细的状态
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		
		health := map[string]interface{}{
			"status":    "unknown",
			"timestamp": time.Now().Format(time.RFC3339),
		}

		if !client.IsStarted() {
			health["status"] = "starting"
			w.WriteHeader(http.StatusServiceUnavailable)
		} else if !notifier.IsReady() {
			health["status"] = "initializing"
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			health["status"] = "ready"
			health["stats"] = client.GetStats()
			w.WriteHeader(http.StatusOK)
		}

		json.NewEncoder(w).Encode(health)
	})

	// 就绪检查接口
	http.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		if notifier.IsReady() {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ready"))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("not ready"))
		}
	})

	// 热key检测接口 - 带就绪检查
	http.HandleFunc("/hotkey", func(w http.ResponseWriter, r *http.Request) {
		if !notifier.IsReady() {
			http.Error(w, "Service not ready", http.StatusServiceUnavailable)
			return
		}

		key := r.URL.Query().Get("key")
		if key == "" {
			http.Error(w, "key parameter is required", http.StatusBadRequest)
			return
		}

		result := map[string]interface{}{
			"key":   key,
			"isHot": hotkey.IsHotKey(key),
			"timestamp": time.Now().Format(time.RFC3339),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	})

	// 批量检测接口
	http.HandleFunc("/hotkeys/batch", func(w http.ResponseWriter, r *http.Request) {
		if !notifier.IsReady() {
			http.Error(w, "Service not ready", http.StatusServiceUnavailable)
			return
		}

		if r.Method != "POST" {
			http.Error(w, "POST method required", http.StatusMethodNotAllowed)
			return
		}

		var request struct {
			Keys []string `json:"keys"`
		}

		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		results := make([]map[string]interface{}, len(request.Keys))
		for i, key := range request.Keys {
			results[i] = map[string]interface{}{
				"key":   key,
				"isHot": hotkey.IsHotKey(key),
			}
		}

		response := map[string]interface{}{
			"results":   results,
			"timestamp": time.Now().Format(time.RFC3339),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	// 统计信息接口
	http.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		stats := client.GetStats()
		stats["ready"] = notifier.IsReady()
		stats["timestamp"] = time.Now().Format(time.RFC3339)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
	})

	log.Println("🌐 HTTP server starting on :8080")
	log.Println("📋 Available endpoints:")
	log.Println("  GET  /health              - Detailed health check")
	log.Println("  GET  /ready               - Simple readiness check")
	log.Println("  GET  /hotkey?key=xxx      - Check if key is hot")
	log.Println("  POST /hotkeys/batch       - Batch check hot keys")
	log.Println("  GET  /stats               - Get client statistics")

	// 启动模拟流量
	go simulateGracefulTraffic(notifier)

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func simulateGracefulTraffic(notifier *ClientReadyNotifier) {
	// 等待服务就绪
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := notifier.WaitReady(ctx); err != nil {
		log.Printf("⚠️  Traffic simulation stopped: %v", err)
		return
	}

	log.Println("🚦 Starting traffic simulation...")

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	keys := []string{"user:1001", "product:2001", "order:3001", "user:1001", "product:2001"}
	index := 0

	for range ticker.C {
		if !notifier.IsReady() {
			log.Println("⏸️  Service not ready, pausing traffic simulation")
			continue
		}

		key := keys[index%len(keys)]
		isHot := hotkey.IsHotKey(key)
		
		log.Printf("🔍 Simulated check: %s -> isHot: %v", key, isHot)

		// 模拟设置一些数据
		if !isHot {
			hotkey.ForceSet(key, fmt.Sprintf("data_%d", time.Now().Unix()))
		}

		index++
	}
}
