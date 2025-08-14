package main

import (
	"fmt"
	"time"

	hotkey "github.com/jd/platform/hotkey/client-go"
	"github.com/jd/platform/hotkey/client-go/cache"
	"github.com/jd/platform/hotkey/client-go/context"
	"github.com/jd/platform/hotkey/client-go/log"
)

func main() {
	// 1. 配置日志系统
	setupLogging()

	// 2. 创建增强配置的客户端
	client := createEnhancedClient()

	// 3. 启动客户端
	if err := client.Start(); err != nil {
		log.Error(main, fmt.Sprintf("Failed to start hotkey client: %v", err))
		return
	}
	defer client.Stop()

	// 4. 等待初始化完成
	time.Sleep(3 * time.Second)

	// 5. 演示增强功能
	demonstrateEnhancedFeatures(client)

	// 6. 保持运行
	log.Info(main, "Enhanced client is running. Press Ctrl+C to stop.")
	select {}
}

// setupLogging 设置日志系统
func setupLogging() {
	// 设置日志级别为DEBUG，显示详细信息
	log.SetLevel(log.DEBUG)
	
	// 可以创建文件日志器
	// fileLogger, err := log.NewFileLogger("hotkey-client.log")
	// if err == nil {
	//     // 创建多重日志器，同时输出到控制台和文件
	//     multiLogger := log.NewMultiLogger(log.GetGlobalLogger(), fileLogger)
	//     log.SetGlobalLogger(multiLogger)
	// }

	log.Info("main", "Logging system initialized")
}

// createEnhancedClient 创建增强配置的客户端
func createEnhancedClient() *hotkey.Client {
	client, err := hotkey.NewClientBuilder().
		SetAppName("enhanced-go-client").
		SetEtcdServer("http://127.0.0.1:2379").
		SetPushPeriod(300).     // 更频繁的推送
		SetCacheSize(500000).   // 更大的缓存
		SetCountPeriod(5).      // 更频繁的计数上报
		Build()

	if err != nil {
		log.Error("main", fmt.Sprintf("Failed to create client: %v", err))
		panic(err)
	}

	log.Info("main", "Enhanced client created successfully")
	return client
}

// demonstrateEnhancedFeatures 演示增强功能
func demonstrateEnhancedFeatures(client *hotkey.Client) {
	log.Info("main", "=== Demonstrating Enhanced Features ===")

	// 1. 演示上下文功能
	demonstrateContext()

	// 2. 演示缓存构建器
	demonstrateCacheBuilder()

	// 3. 演示客户端统计
	demonstrateClientStats(client)

	// 4. 演示热key操作
	demonstrateHotKeyOperations()

	// 5. 演示配置管理
	demonstrateConfigManagement()
}

// demonstrateContext 演示上下文功能
func demonstrateContext() {
	log.Info("main", "--- Context Demo ---")

	// 获取上下文信息
	contextInfo := context.GetContextInfo()
	log.Info("main", fmt.Sprintf("Context info: %+v", contextInfo))

	// 设置自定义配置
	context.SetConfig("custom.feature.enabled", true)
	context.SetConfig("custom.timeout", 30)
	context.SetConfig("custom.endpoint", "https://api.example.com")

	// 获取配置
	enabled := context.GetConfigBool("custom.feature.enabled", false)
	timeout := context.GetConfigInt("custom.timeout", 10)
	endpoint := context.GetConfigString("custom.endpoint", "")

	log.Info("main", fmt.Sprintf("Custom config - enabled: %v, timeout: %d, endpoint: %s", 
		enabled, timeout, endpoint))

	// 显示运行时间
	uptime := context.GetUptime()
	log.Info("main", fmt.Sprintf("Client uptime: %v", uptime))
}

// demonstrateCacheBuilder 演示缓存构建器
func demonstrateCacheBuilder() {
	log.Info("main", "--- Cache Builder Demo ---")

	// 使用默认缓存
	defaultCache := cache.Cache()
	log.Info("main", "Created default cache")

	// 使用指定过期时间的缓存
	timedCache := cache.CacheWithDuration(120) // 2分钟过期
	log.Info("main", "Created timed cache with 120s expiration")

	// 使用指定大小的缓存
	sizedCache := cache.CacheWithSize(100000)
	log.Info("main", "Created sized cache with 100k capacity")

	// 使用完全自定义的缓存
	customCache := cache.CacheWithConfig(64, 50000, 300) // 最小64，最大5万，5分钟过期
	log.Info("main", "Created custom cache")

	// 使用构建器模式
	builderCache := cache.NewCacheBuilder().
		SetMinSize(32).
		SetMaxSize(10000).
		SetExpireSeconds(180).
		SetInitialCapacity(256).
		BuildDefault()
	log.Info("main", "Created builder cache")

	// 测试缓存操作
	testKey := "cache-test-key"
	testValue := "cache-test-value"

	defaultCache.Set(testKey, testValue)
	if value := defaultCache.Get(testKey); value != nil {
		log.Info("main", fmt.Sprintf("Cache test successful: %v", value))
	}

	log.Info("main", fmt.Sprintf("Cache sizes - default: %d, timed: %d, sized: %d, custom: %d, builder: %d",
		defaultCache.Size(), timedCache.Size(), sizedCache.Size(), customCache.Size(), builderCache.Size()))
}

// demonstrateClientStats 演示客户端统计
func demonstrateClientStats(client *hotkey.Client) {
	log.Info("main", "--- Client Stats Demo ---")

	stats := client.GetStats()
	log.Info("main", fmt.Sprintf("Client statistics: %+v", stats))

	// 获取规则信息
	ruleHolder := client.GetRuleHolder()
	rules := ruleHolder.GetRules()
	log.Info("main", fmt.Sprintf("Current rules count: %d", len(rules)))

	// 获取缓存统计
	cacheStats := ruleHolder.GetCacheStats()
	log.Info("main", fmt.Sprintf("Cache statistics: %+v", cacheStats))

	// 获取网络管理器信息
	networkManager := client.GetNetworkManager()
	connections := networkManager.GetNetClient().GetConnections()
	activeConnections := networkManager.GetNetClient().GetActiveConnections()
	
	log.Info("main", fmt.Sprintf("Network - total connections: %d, active: %d", 
		len(connections), len(activeConnections)))
}

// demonstrateHotKeyOperations 演示热key操作
func demonstrateHotKeyOperations() {
	log.Info("main", "--- Hot Key Operations Demo ---")

	// 测试不同类型的key
	testKeys := []string{
		"user:12345",
		"product:67890",
		"order:abcdef",
		"session:xyz123",
	}

	for _, key := range testKeys {
		// 检查是否为热key
		isHot := hotkey.IsHotKey(key)
		log.Info("main", fmt.Sprintf("Key %s is hot: %v", key, isHot))

		// 如果不是热key，强制设置一些数据
		if !isHot {
			testData := map[string]interface{}{
				"key":       key,
				"timestamp": time.Now().Unix(),
				"data":      fmt.Sprintf("test-data-for-%s", key),
			}
			hotkey.ForceSet(key, testData)
			log.Info("main", fmt.Sprintf("Force set data for key: %s", key))
		}

		// 获取值
		value := hotkey.Get(key)
		if value != nil {
			log.Info("main", fmt.Sprintf("Retrieved value for %s: %v", key, value))
		}

		// 智能设置（仅当是热key时）
		hotkey.SmartSet(key, fmt.Sprintf("smart-value-%d", time.Now().Unix()))
	}

	// 演示删除操作
	hotkey.Remove("user:12345")
	log.Info("main", "Removed key: user:12345")
}

// demonstrateConfigManagement 演示配置管理
func demonstrateConfigManagement() {
	log.Info("main", "--- Config Management Demo ---")

	// 设置各种类型的配置
	context.SetConfig(context.ConfigKeyLogLevel, "DEBUG")
	context.SetConfig(context.ConfigKeyMetricsEnabled, true)
	context.SetConfig(context.ConfigKeyMetricsPort, 9090)
	context.SetConfig(context.ConfigKeyHealthPort, 8080)
	context.SetConfig(context.ConfigKeyDebugMode, true)
	context.SetConfig(context.ConfigKeyMaxConnections, 100)
	context.SetConfig(context.ConfigKeyConnectTimeout, 5000)
	context.SetConfig(context.ConfigKeyHeartbeatInterval, 30)

	// 读取配置
	logLevel := context.GetConfigString(context.ConfigKeyLogLevel, "INFO")
	metricsEnabled := context.GetConfigBool(context.ConfigKeyMetricsEnabled, false)
	metricsPort := context.GetConfigInt(context.ConfigKeyMetricsPort, 8080)
	debugMode := context.GetConfigBool(context.ConfigKeyDebugMode, false)

	log.Info("main", fmt.Sprintf("Config - logLevel: %s, metrics: %v:%d, debug: %v", 
		logLevel, metricsEnabled, metricsPort, debugMode))

	// 显示所有配置
	allConfig := context.GetAllConfig()
	log.Info("main", fmt.Sprintf("All configuration: %+v", allConfig))

	// 验证配置
	if err := context.Validate(); err != nil {
		log.Error("main", fmt.Sprintf("Context validation failed: %v", err))
	} else {
		log.Info("main", "Context validation passed")
	}
}

// 模拟业务场景
func simulateBusinessScenario() {
	log.Info("main", "--- Business Scenario Simulation ---")

	// 模拟用户访问模式
	userIDs := []string{"1001", "1002", "1003", "1001", "1002", "1001"} // 1001是热点用户

	for i, userID := range userIDs {
		userKey := fmt.Sprintf("user:%s", userID)
		
		// 模拟业务逻辑
		if hotkey.IsHotKey(userKey) {
			// 热key，从本地缓存获取
			userData := hotkey.Get(userKey)
			log.Info("main", fmt.Sprintf("Access %d: Hot user %s, data: %v", i+1, userID, userData))
		} else {
			// 非热key，模拟从数据库获取并缓存
			userData := map[string]interface{}{
				"id":       userID,
				"name":     fmt.Sprintf("User%s", userID),
				"lastSeen": time.Now().Unix(),
			}
			hotkey.ForceSet(userKey, userData)
			log.Info("main", fmt.Sprintf("Access %d: Cold user %s, loaded from DB", i+1, userID))
		}

		time.Sleep(500 * time.Millisecond) // 模拟访问间隔
	}
}
