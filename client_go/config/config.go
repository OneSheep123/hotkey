package config

import (
	"sync"
	"time"
)

// ClientConfig 客户端配置
type ClientConfig struct {
	AppName     string        `json:"appName" yaml:"appName"`
	EtcdServers []string      `json:"etcdServers" yaml:"etcdServers"`
	PushPeriod  time.Duration `json:"pushPeriod" yaml:"pushPeriod"`
	CacheSize   int           `json:"cacheSize" yaml:"cacheSize"`
}

// Context 全局上下文
type Context struct {
	AppName   string
	CacheSize int
	mu        sync.RWMutex
}

var globalContext = &Context{}

// GetContext 获取全局上下文
func GetContext() *Context {
	return globalContext
}

// SetAppName 设置应用名称
func (c *Context) SetAppName(appName string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.AppName = appName
}

// GetAppName 获取应用名称
func (c *Context) GetAppName() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.AppName
}

// SetCacheSize 设置缓存大小
func (c *Context) SetCacheSize(size int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.CacheSize = size
}

// GetCacheSize 获取缓存大小
func (c *Context) GetCacheSize() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.CacheSize
}

// DefaultConfig 返回默认配置
func DefaultConfig() *ClientConfig {
	return &ClientConfig{
		AppName:     "default-app",
		EtcdServers: []string{"127.0.0.1:2379"},
		PushPeriod:  500 * time.Millisecond,
		CacheSize:   200000,
	}
}

// Validate 验证配置
func (c *ClientConfig) Validate() error {
	if c.AppName == "" {
		c.AppName = "default-app"
	}
	if len(c.EtcdServers) == 0 {
		c.EtcdServers = []string{"127.0.0.1:2379"}
	}
	if c.PushPeriod <= 0 {
		c.PushPeriod = 500 * time.Millisecond
	}
	if c.CacheSize <= 0 {
		c.CacheSize = 200000
	}
	return nil
}
