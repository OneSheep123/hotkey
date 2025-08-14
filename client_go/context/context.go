package context

import (
	"fmt"
	"sync"
	"time"

	"github.com/jd/platform/hotkey/client-go/model"
)

// Context 全局上下文，对应Java的Context
type Context struct {
	appName      string
	caffeineSize int64
	pushPeriod   int64
	countPeriod  int
	etcdServer   string
	startTime    time.Time
	mutex        sync.RWMutex
	
	// 扩展配置
	config map[string]interface{}
}

// 全局上下文实例
var globalContext *Context
var contextOnce sync.Once

// GetGlobalContext 获取全局上下文实例
func GetGlobalContext() *Context {
	contextOnce.Do(func() {
		globalContext = &Context{
			caffeineSize: model.DefaultCacheSize,
			pushPeriod:   model.DefaultPushPeriod,
			countPeriod:  model.DefaultCountPeriod,
			startTime:    time.Now(),
			config:       make(map[string]interface{}),
		}
	})
	return globalContext
}

// SetAppName 设置应用名称，对应Java的Context.APP_NAME
func SetAppName(appName string) {
	ctx := GetGlobalContext()
	ctx.mutex.Lock()
	defer ctx.mutex.Unlock()
	ctx.appName = appName
}

// GetAppName 获取应用名称
func GetAppName() string {
	ctx := GetGlobalContext()
	ctx.mutex.RLock()
	defer ctx.mutex.RUnlock()
	return ctx.appName
}

// SetCaffeineSize 设置缓存大小，对应Java的Context.CAFFEINE_SIZE
func SetCaffeineSize(size int64) {
	ctx := GetGlobalContext()
	ctx.mutex.Lock()
	defer ctx.mutex.Unlock()
	ctx.caffeineSize = size
}

// GetCaffeineSize 获取缓存大小
func GetCaffeineSize() int64 {
	ctx := GetGlobalContext()
	ctx.mutex.RLock()
	defer ctx.mutex.RUnlock()
	return ctx.caffeineSize
}

// SetPushPeriod 设置推送间隔
func SetPushPeriod(period int64) {
	ctx := GetGlobalContext()
	ctx.mutex.Lock()
	defer ctx.mutex.Unlock()
	ctx.pushPeriod = period
}

// GetPushPeriod 获取推送间隔
func GetPushPeriod() int64 {
	ctx := GetGlobalContext()
	ctx.mutex.RLock()
	defer ctx.mutex.RUnlock()
	return ctx.pushPeriod
}

// SetCountPeriod 设置计数间隔
func SetCountPeriod(period int) {
	ctx := GetGlobalContext()
	ctx.mutex.Lock()
	defer ctx.mutex.Unlock()
	ctx.countPeriod = period
}

// GetCountPeriod 获取计数间隔
func GetCountPeriod() int {
	ctx := GetGlobalContext()
	ctx.mutex.RLock()
	defer ctx.mutex.RUnlock()
	return ctx.countPeriod
}

// SetEtcdServer 设置etcd服务器地址
func SetEtcdServer(server string) {
	ctx := GetGlobalContext()
	ctx.mutex.Lock()
	defer ctx.mutex.Unlock()
	ctx.etcdServer = server
}

// GetEtcdServer 获取etcd服务器地址
func GetEtcdServer() string {
	ctx := GetGlobalContext()
	ctx.mutex.RLock()
	defer ctx.mutex.RUnlock()
	return ctx.etcdServer
}

// GetStartTime 获取启动时间
func GetStartTime() time.Time {
	ctx := GetGlobalContext()
	ctx.mutex.RLock()
	defer ctx.mutex.RUnlock()
	return ctx.startTime
}

// GetUptime 获取运行时间
func GetUptime() time.Duration {
	return time.Since(GetStartTime())
}

// SetConfig 设置配置项
func SetConfig(key string, value interface{}) {
	ctx := GetGlobalContext()
	ctx.mutex.Lock()
	defer ctx.mutex.Unlock()
	ctx.config[key] = value
}

// GetConfig 获取配置项
func GetConfig(key string) (interface{}, bool) {
	ctx := GetGlobalContext()
	ctx.mutex.RLock()
	defer ctx.mutex.RUnlock()
	value, exists := ctx.config[key]
	return value, exists
}

// GetConfigString 获取字符串配置项
func GetConfigString(key string, defaultValue string) string {
	if value, exists := GetConfig(key); exists {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return defaultValue
}

// GetConfigInt 获取整数配置项
func GetConfigInt(key string, defaultValue int) int {
	if value, exists := GetConfig(key); exists {
		if i, ok := value.(int); ok {
			return i
		}
	}
	return defaultValue
}

// GetConfigInt64 获取64位整数配置项
func GetConfigInt64(key string, defaultValue int64) int64 {
	if value, exists := GetConfig(key); exists {
		if i, ok := value.(int64); ok {
			return i
		}
	}
	return defaultValue
}

// GetConfigBool 获取布尔配置项
func GetConfigBool(key string, defaultValue bool) bool {
	if value, exists := GetConfig(key); exists {
		if b, ok := value.(bool); ok {
			return b
		}
	}
	return defaultValue
}

// GetAllConfig 获取所有配置
func GetAllConfig() map[string]interface{} {
	ctx := GetGlobalContext()
	ctx.mutex.RLock()
	defer ctx.mutex.RUnlock()
	
	result := make(map[string]interface{})
	for k, v := range ctx.config {
		result[k] = v
	}
	return result
}

// Reset 重置上下文（主要用于测试）
func Reset() {
	globalContext = nil
	contextOnce = sync.Once{}
}

// InitializeFromClient 从客户端初始化上下文
func InitializeFromClient(appName, etcdServer string, pushPeriod int64, cacheSize int64, countPeriod int) {
	SetAppName(appName)
	SetEtcdServer(etcdServer)
	SetPushPeriod(pushPeriod)
	SetCaffeineSize(cacheSize)
	SetCountPeriod(countPeriod)
}

// GetContextInfo 获取上下文信息
func GetContextInfo() map[string]interface{} {
	ctx := GetGlobalContext()
	ctx.mutex.RLock()
	defer ctx.mutex.RUnlock()
	
	return map[string]interface{}{
		"appName":      ctx.appName,
		"caffeineSize": ctx.caffeineSize,
		"pushPeriod":   ctx.pushPeriod,
		"countPeriod":  ctx.countPeriod,
		"etcdServer":   ctx.etcdServer,
		"startTime":    ctx.startTime,
		"uptime":       time.Since(ctx.startTime).String(),
		"config":       ctx.config,
	}
}

// Validate 验证上下文配置
func Validate() error {
	ctx := GetGlobalContext()
	ctx.mutex.RLock()
	defer ctx.mutex.RUnlock()
	
	if ctx.appName == "" {
		return fmt.Errorf("appName is required")
	}
	if ctx.etcdServer == "" {
		return fmt.Errorf("etcdServer is required")
	}
	if ctx.pushPeriod <= 0 {
		return fmt.Errorf("pushPeriod must be positive")
	}
	if ctx.caffeineSize <= 0 {
		return fmt.Errorf("caffeineSize must be positive")
	}
	if ctx.countPeriod <= 0 {
		return fmt.Errorf("countPeriod must be positive")
	}
	
	return nil
}

// 常用配置键名常量
const (
	ConfigKeyLogLevel        = "log.level"
	ConfigKeyLogFile         = "log.file"
	ConfigKeyMetricsEnabled  = "metrics.enabled"
	ConfigKeyMetricsPort     = "metrics.port"
	ConfigKeyHealthPort      = "health.port"
	ConfigKeyDebugMode       = "debug.mode"
	ConfigKeyMaxConnections  = "network.maxConnections"
	ConfigKeyConnectTimeout  = "network.connectTimeout"
	ConfigKeyHeartbeatInterval = "network.heartbeatInterval"
)
