package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	hotkey "github.com/jd/platform/hotkey/client-go"
	"github.com/jd/platform/hotkey/client-go/model"
)

func main() {
	// 创建客户端，配置与Java版本兼容
	client, err := hotkey.NewClientBuilder().
		SetAppName("sample").
		SetEtcdServer("http://127.0.0.1:12379").
		SetPushPeriod(500).   // 与Java版本相同的推送间隔
		SetCacheSize(200000). // 与Java版本相同的缓存大小
		SetCountPeriod(10).   // 与Java版本相同的计数间隔
		Build()

	if err != nil {
		log.Fatal("Failed to create hotkey client:", err)
	}

	// 启动客户端
	if err := client.Start(); err != nil {
		log.Fatal("Failed to start hotkey client:", err)
	}
	defer client.Stop()

	log.Println("Advanced HotKey client started successfully")

	// 等待初始化完成 - 使用健康检查轮询
	if err := waitForClientReady(client, 30*time.Second); err != nil {
		log.Fatal("Client failed to become ready:", err)
	}

	// 启动HTTP服务器，提供与Java版本相同的API
	startHTTPServer(client)
}

// waitForClientReady 等待客户端准备就绪，使用健康检查轮询
func waitForClientReady(client *hotkey.Client, timeout time.Duration) error {
	log.Println("Waiting for client to be ready...")

	ticker := time.NewTicker(100 * time.Millisecond) // 每100ms检查一次
	defer ticker.Stop()

	timeoutChan := time.After(timeout)

	for {
		select {
		case <-ticker.C:
			if isClientReady(client) {
				log.Println("Client is ready!")
				return nil
			}
		case <-timeoutChan:
			return fmt.Errorf("timeout waiting for client to be ready after %v", timeout)
		}
	}
}

// isClientReady 检查客户端是否准备就绪
func isClientReady(client *hotkey.Client) bool {
	// 检查客户端是否已启动
	if !client.IsStarted() {
		return false
	}

	// 检查统计信息是否可用
	stats := client.GetStats()
	if stats == nil {
		return false
	}

	// 检查是否有活跃连接
	if activeConnections, ok := stats["activeConnections"].(int); ok && activeConnections > 0 {
		log.Printf("Client ready with %d active connections", activeConnections)
		return true
	}

	// 如果没有活跃连接，但有总连接数，说明正在连接中
	if totalConnections, ok := stats["totalConnections"].(int); ok && totalConnections > 0 {
		log.Printf("Client connecting... (%d total connections)", totalConnections)
	}

	return false
}

func startHTTPServer(client *hotkey.Client) {
	// 热key检测接口，对应Java版本的 /hotKey
	http.HandleFunc("/hotKey", func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("key")
		if key == "" {
			http.Error(w, "key parameter is required", http.StatusBadRequest)
			return
		}

		if hotkey.IsHotKey(key) {
			w.Write([]byte("isHot"))
		} else {
			w.Write([]byte("noHot"))
		}
	})

	// 获取值接口
	http.HandleFunc("/getValue", func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("key")
		if key == "" {
			http.Error(w, "key parameter is required", http.StatusBadRequest)
			return
		}

		value := hotkey.GetValueWithDefaults(key)
		if value == nil {
			w.Write([]byte("null"))
		} else {
			json.NewEncoder(w).Encode(value)
		}
	})

	// 设置值接口
	http.HandleFunc("/setValue", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "POST method required", http.StatusMethodNotAllowed)
			return
		}

		key := r.URL.Query().Get("key")
		value := r.URL.Query().Get("value")
		if key == "" {
			http.Error(w, "key parameter is required", http.StatusBadRequest)
			return
		}

		hotkey.ForceSet(key, value)
		w.Write([]byte("success"))
	})

	// 删除key接口，对应Java版本的DELETE请求
	http.HandleFunc("/deleteKey", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			http.Error(w, "DELETE method required", http.StatusMethodNotAllowed)
			return
		}

		key := r.URL.Query().Get("key")
		if key == "" {
			http.Error(w, "key parameter is required", http.StatusBadRequest)
			return
		}

		hotkey.Remove(key)
		w.Write([]byte("1"))
	})

	// 添加规则接口，对应Java版本的 /addRulePath
	http.HandleFunc("/addRulePath", func(w http.ResponseWriter, r *http.Request) {
		appName := r.URL.Query().Get("appName")
		if appName == "" {
			appName = client.GetAppName()
		}

		// 创建默认规则
		defaultRule := &model.KeyRule{
			Key:       "*",
			Prefix:    false,
			Interval:  5,
			Threshold: 10,
			Duration:  60,
			Desc:      "Default rule for all keys",
		}

		rules := []*model.KeyRule{defaultRule}
		rulesJSON, err := json.Marshal(rules)
		if err != nil {
			http.Error(w, "Failed to marshal rules", http.StatusInternalServerError)
			return
		}

		// 这里应该调用etcd配置中心设置规则
		// 为了演示，我们直接返回成功
		log.Printf("Would set rules for app %s: %s", appName, string(rulesJSON))
		w.Write([]byte("success"))
	})

	// 获取客户端状态接口
	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		stats := client.GetStats()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
	})

	// 健康检查接口
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if client.IsStarted() {
			w.Write([]byte("OK"))
		} else {
			http.Error(w, "Client not started", http.StatusServiceUnavailable)
		}
	})

	// 模拟业务接口，展示热key的实际使用
	http.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		userID := r.URL.Query().Get("id")
		if userID == "" {
			http.Error(w, "id parameter is required", http.StatusBadRequest)
			return
		}

		userKey := "user:" + userID

		// 检查是否为热key
		if hotkey.IsHotKey(userKey) {
			// 从本地缓存获取用户数据
			userData := hotkey.Get(userKey)
			if userData != nil {
				log.Printf("Hit hot key cache for user: %s", userID)
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"id":     userID,
					"data":   userData,
					"source": "hot_cache",
				})
				return
			}
		}

		// 模拟从数据库获取用户数据
		userData := map[string]interface{}{
			"id":      userID,
			"name":    "User " + userID,
			"email":   "user" + userID + "@example.com",
			"created": time.Now().Format("2006-01-02 15:04:05"),
		}

		// 设置到缓存中
		hotkey.SmartSet(userKey, userData)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":     userID,
			"data":   userData,
			"source": "database",
		})
	})

	log.Println("HTTP server starting on :8080")
	log.Println("Available endpoints:")
	log.Println("  GET  /hotKey?key=xxx        - Check if key is hot")
	log.Println("  GET  /getValue?key=xxx      - Get value from cache")
	log.Println("  POST /setValue?key=xxx&value=yyy - Set value to cache")
	log.Println("  DELETE /deleteKey?key=xxx   - Delete key from cache")
	log.Println("  GET  /addRulePath?appName=xxx - Add default rule")
	log.Println("  GET  /status                - Get client status")
	log.Println("  GET  /health                - Health check")
	log.Println("  GET  /user?id=xxx           - User data example")

	// 启动定时任务，模拟业务访问
	go simulateBusinessTraffic()

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func simulateBusinessTraffic() {
	time.Sleep(5 * time.Second) // 等待服务器启动

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	userIDs := []string{"123", "456", "789", "123", "456", "123"} // 模拟热点用户
	index := 0

	for range ticker.C {
		userID := userIDs[index%len(userIDs)]
		userKey := "user:" + userID

		// 模拟访问
		isHot := hotkey.IsHotKey(userKey)
		log.Printf("Simulated access to user:%s, isHot: %v", userID, isHot)

		// 如果不是热key，设置一些数据
		if !isHot {
			userData := map[string]string{
				"id":   userID,
				"name": "User " + userID,
			}
			hotkey.ForceSet(userKey, userData)
		}

		index++
	}
}
