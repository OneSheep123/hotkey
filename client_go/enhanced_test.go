package hotkey

import (
	"testing"
	"time"

	"github.com/jd/platform/hotkey/client-go/cache"
	"github.com/jd/platform/hotkey/client-go/context"
	"github.com/jd/platform/hotkey/client-go/log"
)

func TestEnhancedLogging(t *testing.T) {
	// 测试日志系统
	logger := log.NewDefaultLogger()
	
	// 测试不同级别的日志
	logger.SetLevel(log.DEBUG)
	
	if !logger.IsEnabled(log.DEBUG) {
		t.Error("DEBUG level should be enabled")
	}
	
	if !logger.IsEnabled(log.INFO) {
		t.Error("INFO level should be enabled")
	}
	
	// 测试日志输出
	logger.Debug("TestClass", "This is a debug message")
	logger.Info("TestClass", "This is an info message")
	logger.Warn("TestClass", "This is a warning message")
	logger.Error("TestClass", "This is an error message")
	
	// 测试级别过滤
	logger.SetLevel(log.WARN)
	if logger.IsEnabled(log.DEBUG) {
		t.Error("DEBUG level should be disabled when level is WARN")
	}
	if logger.IsEnabled(log.INFO) {
		t.Error("INFO level should be disabled when level is WARN")
	}
	if !logger.IsEnabled(log.WARN) {
		t.Error("WARN level should be enabled")
	}
}

func TestGlobalLogging(t *testing.T) {
	// 测试全局日志功能
	log.SetLevel(log.INFO)
	
	log.Debug("TestGlobal", "This debug should not appear")
	log.Info("TestGlobal", "This info should appear")
	log.Warn("TestGlobal", "This warning should appear")
	log.Error("TestGlobal", "This error should appear")
	
	// 测试格式化日志
	log.Infof("TestGlobal", "Formatted message: %s = %d", "count", 42)
	log.Errorf("TestGlobal", "Error code: %d, message: %s", 500, "Internal Server Error")
}

func TestMultiLogger(t *testing.T) {
	// 创建多个日志器
	logger1 := log.NewDefaultLogger()
	logger2 := log.NewDefaultLogger()
	
	// 创建多重日志器
	multiLogger := log.NewMultiLogger(logger1, logger2)
	multiLogger.SetLevel(log.INFO)
	
	// 测试多重日志输出
	multiLogger.Info("TestMulti", "This should appear in both loggers")
	
	if !multiLogger.IsEnabled(log.INFO) {
		t.Error("INFO level should be enabled in multi logger")
	}
}

func TestContext(t *testing.T) {
	// 重置上下文
	context.Reset()
	
	// 测试基本设置和获取
	context.SetAppName("test-app")
	context.SetCaffeineSize(100000)
	context.SetPushPeriod(1000)
	context.SetCountPeriod(10)
	context.SetEtcdServer("http://localhost:2379")
	
	if context.GetAppName() != "test-app" {
		t.Errorf("Expected app name 'test-app', got '%s'", context.GetAppName())
	}
	
	if context.GetCaffeineSize() != 100000 {
		t.Errorf("Expected caffeine size 100000, got %d", context.GetCaffeineSize())
	}
	
	if context.GetPushPeriod() != 1000 {
		t.Errorf("Expected push period 1000, got %d", context.GetPushPeriod())
	}
	
	if context.GetCountPeriod() != 10 {
		t.Errorf("Expected count period 10, got %d", context.GetCountPeriod())
	}
	
	if context.GetEtcdServer() != "http://localhost:2379" {
		t.Errorf("Expected etcd server 'http://localhost:2379', got '%s'", context.GetEtcdServer())
	}
}

func TestContextConfig(t *testing.T) {
	context.Reset()
	
	// 测试配置设置和获取
	context.SetConfig("test.string", "hello")
	context.SetConfig("test.int", 42)
	context.SetConfig("test.int64", int64(123456789))
	context.SetConfig("test.bool", true)
	
	// 测试字符串配置
	if value := context.GetConfigString("test.string", "default"); value != "hello" {
		t.Errorf("Expected 'hello', got '%s'", value)
	}
	
	if value := context.GetConfigString("nonexistent", "default"); value != "default" {
		t.Errorf("Expected 'default', got '%s'", value)
	}
	
	// 测试整数配置
	if value := context.GetConfigInt("test.int", 0); value != 42 {
		t.Errorf("Expected 42, got %d", value)
	}
	
	if value := context.GetConfigInt("nonexistent", 100); value != 100 {
		t.Errorf("Expected 100, got %d", value)
	}
	
	// 测试64位整数配置
	if value := context.GetConfigInt64("test.int64", 0); value != 123456789 {
		t.Errorf("Expected 123456789, got %d", value)
	}
	
	// 测试布尔配置
	if value := context.GetConfigBool("test.bool", false); !value {
		t.Error("Expected true, got false")
	}
	
	if value := context.GetConfigBool("nonexistent", false); value {
		t.Error("Expected false, got true")
	}
}

func TestContextInitialization(t *testing.T) {
	context.Reset()
	
	// 测试从客户端初始化
	context.InitializeFromClient("init-app", "http://etcd:2379", 500, 200000, 10)
	
	if context.GetAppName() != "init-app" {
		t.Errorf("Expected 'init-app', got '%s'", context.GetAppName())
	}
	
	// 测试上下文信息
	info := context.GetContextInfo()
	if info["appName"] != "init-app" {
		t.Error("Context info should contain correct app name")
	}
	
	// 测试运行时间
	time.Sleep(10 * time.Millisecond)
	uptime := context.GetUptime()
	if uptime < 10*time.Millisecond {
		t.Error("Uptime should be at least 10ms")
	}
}

func TestContextValidation(t *testing.T) {
	context.Reset()
	
	// 测试无效配置
	err := context.Validate()
	if err == nil {
		t.Error("Validation should fail with empty configuration")
	}
	
	// 设置有效配置
	context.SetAppName("valid-app")
	context.SetEtcdServer("http://localhost:2379")
	context.SetPushPeriod(500)
	context.SetCaffeineSize(100000)
	context.SetCountPeriod(10)
	
	err = context.Validate()
	if err != nil {
		t.Errorf("Validation should pass with valid configuration: %v", err)
	}
	
	// 测试无效的推送间隔
	context.SetPushPeriod(0)
	err = context.Validate()
	if err == nil {
		t.Error("Validation should fail with zero push period")
	}
}

func TestCacheBuilder(t *testing.T) {
	// 测试默认缓存
	defaultCache := cache.Cache()
	if defaultCache == nil {
		t.Error("Default cache should not be nil")
	}
	
	// 测试带过期时间的缓存
	timedCache := cache.CacheWithDuration(60)
	if timedCache == nil {
		t.Error("Timed cache should not be nil")
	}
	
	// 测试带大小的缓存
	sizedCache := cache.CacheWithSize(10000)
	if sizedCache == nil {
		t.Error("Sized cache should not be nil")
	}
	
	// 测试完全配置的缓存
	configCache := cache.CacheWithConfig(64, 10000, 120)
	if configCache == nil {
		t.Error("Config cache should not be nil")
	}
	
	// 测试构建器模式
	builderCache := cache.NewCacheBuilder().
		SetMinSize(32).
		SetMaxSize(5000).
		SetExpireSeconds(90).
		SetInitialCapacity(128).
		BuildDefault()
	
	if builderCache == nil {
		t.Error("Builder cache should not be nil")
	}
	
	// 测试缓存操作
	testKey := "test-key"
	testValue := "test-value"
	
	builderCache.Set(testKey, testValue)
	if value := builderCache.Get(testKey); value != testValue {
		t.Errorf("Expected '%s', got '%v'", testValue, value)
	}
	
	if builderCache.Size() != 1 {
		t.Errorf("Expected cache size 1, got %d", builderCache.Size())
	}
	
	builderCache.Delete(testKey)
	if value := builderCache.Get(testKey); value != nil {
		t.Errorf("Expected nil after delete, got '%v'", value)
	}
}

func TestCacheBuilderError(t *testing.T) {
	// 测试构建器的错误处理
	builder := cache.NewCacheBuilder()
	
	// 设置一些配置
	cache := builder.
		SetMinSize(10).
		SetMaxSize(1000).
		SetExpireSeconds(30).
		BuildDefault()
	
	if cache == nil {
		t.Error("Cache should not be nil even with basic config")
	}
}

func BenchmarkCacheBuilder(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache := cache.NewCacheBuilder().
			SetMaxSize(10000).
			SetExpireSeconds(60).
			BuildDefault()
		
		// 简单的操作测试
		cache.Set("bench-key", "bench-value")
		cache.Get("bench-key")
		cache.Delete("bench-key")
	}
}

func BenchmarkContextOperations(b *testing.B) {
	context.Reset()
	context.SetAppName("bench-app")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		context.SetConfig("bench.key", i)
		context.GetConfigInt("bench.key", 0)
		context.GetContextInfo()
	}
}
