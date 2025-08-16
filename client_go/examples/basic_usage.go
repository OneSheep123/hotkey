package main

import (
	"context"
	"fmt"
	"log"
	"time"

	hotkey "github.com/jd/platform/hotkey/client-go"
)

func main() {
	// 创建客户端
	client, err := hotkey.NewClientBuilder().
		SetAppName("example-app").
		SetEtcdServer("http://127.0.0.1:2379").
		SetPushPeriod(500).   // 推送间隔500ms
		SetCacheSize(200000). // 缓存容量20万
		Build()

	if err != nil {
		log.Fatal("Failed to create hotkey client:", err)
	}

	// 启动客户端
	if err := client.Start(); err != nil {
		log.Fatal("Failed to start hotkey client:", err)
	}
	defer client.Stop()

	log.Println("HotKey client started successfully")

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

	// 使用示例
	demonstrateUsage()

	// 保持程序运行
	select {}
}

// isClientHealthy 检查客户端是否健康
func isClientHealthy(client *hotkey.Client) bool {
	// 1. 检查客户端是否已启动
	if !client.IsStarted() {
		return false
	}

	// 2. 检查统计信息是否可用
	stats := client.GetStats()
	if stats == nil {
		return false
	}

	// 3. 检查连接状态
	totalConnections, hasTotalConns := stats["totalConnections"].(int)
	activeConnections, hasActiveConns := stats["activeConnections"].(int)

	// 如果没有连接信息，说明还在初始化
	if !hasTotalConns || !hasActiveConns {
		return false
	}

	// 4. 至少需要一个活跃连接（如果配置了worker的话）
	// 如果没有配置worker，totalConnections为0也是正常的
	if totalConnections > 0 && activeConnections == 0 {
		return false
	}

	// 5. 客户端健康
	return true
}

func demonstrateUsage() {
	log.Println("🚀 Starting HotKey usage demonstration...")

	// 1. 检查key是否为热key
	log.Println("\n📋 Step 1: Checking if key is hot")
	key := "user:12345"
	if hotkey.IsHotKey(key) {
		log.Printf("✅ Key %s is hot!", key)

		// 从本地缓存获取值
		value := hotkey.Get(key)
		log.Printf("📦 Hot key value: %v", value)
	} else {
		log.Printf("❄️  Key %s is not hot (this is normal for new keys)", key)
	}

	// 2. 强制设置值（用于演示）
	log.Println("\n📋 Step 2: Setting values")
	log.Printf("🔧 Force setting value for key: %s", key)
	hotkey.ForceSet(key, map[string]interface{}{
		"userId":    "12345",
		"name":      "John Doe",
		"email":     "john@example.com",
		"timestamp": time.Now().Format("2006-01-02 15:04:05"),
	})

	// 验证设置是否成功
	value := hotkey.Get(key)
	if value != nil {
		log.Printf("✅ Value set successfully: %v", value)
	} else {
		log.Printf("❌ Failed to set value")
	}

	// 3. 智能设置值（仅当key是热key时）
	log.Println("\n📋 Step 3: Smart setting (only if key becomes hot)")
	smartKey := "product:999"
	hotkey.SmartSet(smartKey, map[string]interface{}{
		"productId": "999",
		"name":      "Sample Product",
		"price":     99.99,
	})
	log.Printf("🧠 Smart set attempted for key: %s", smartKey)

	// 4. 获取值（如果不存在会上报）
	log.Println("\n📋 Step 4: Getting values with defaults")
	productValue := hotkey.GetValueWithDefaults(smartKey)
	log.Printf("🛍️  Product value: %v", productValue)

	// 5. 演示多个key的操作
	log.Println("\n📋 Step 5: Batch operations demonstration")
	testKeys := []string{"order:1001", "session:abc123", "cache:temp"}

	for i, testKey := range testKeys {
		log.Printf("🔄 Processing key %d/%d: %s", i+1, len(testKeys), testKey)

		// 设置测试数据
		hotkey.ForceSet(testKey, fmt.Sprintf("test_data_%d", i+1))

		// 检查是否为热key
		isHot := hotkey.IsHotKey(testKey)
		log.Printf("   🌡️  Is hot: %v", isHot)

		// 获取值
		val := hotkey.Get(testKey)
		log.Printf("   📦 Value: %v", val)
	}

	// 6. 清理演示
	log.Println("\n📋 Step 6: Cleanup demonstration")
	cleanupKeys := append(testKeys, key, smartKey)
	for _, cleanupKey := range cleanupKeys {
		hotkey.Remove(cleanupKey)
		log.Printf("🗑️  Removed key: %s", cleanupKey)
	}

	// 7. 最终状态检查
	log.Println("\n📋 Step 7: Final status check")
	for _, checkKey := range cleanupKeys {
		val := hotkey.Get(checkKey)
		if val == nil {
			log.Printf("✅ Key %s successfully removed", checkKey)
		} else {
			log.Printf("⚠️  Key %s still has value: %v", checkKey, val)
		}
	}

	log.Println("\n🎉 Usage demonstration completed successfully!")
	log.Println("💡 Tip: In a real application, keys become 'hot' based on access patterns and rules")
}
