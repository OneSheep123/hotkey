package cache

import (
	"sync"
	"time"
)

// LocalCache 本地缓存接口
type LocalCache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{})
	SetWithTTL(key string, value interface{}, ttl time.Duration)
	Delete(key string)
	Clear()
	Size() int
}

// CaffeineCache Caffeine缓存实现
type CaffeineCache struct {
	cache *Cache
	ttl   time.Duration
	mu    sync.RWMutex
}

// NewCaffeineCache 创建新的Caffeine缓存
func NewCaffeineCache(duration time.Duration, maxSize int) *CaffeineCache {
	return &CaffeineCache{
		cache: NewCache(duration, maxSize),
		ttl:   duration,
	}
}

// Get 获取缓存值
func (c *CaffeineCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cache.Get(key)
}

// Set 设置缓存值
func (c *CaffeineCache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache.Set(key, value, c.ttl)
}

// SetWithTTL 设置缓存值并指定TTL
func (c *CaffeineCache) SetWithTTL(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache.Set(key, value, ttl)
}

// Delete 删除缓存值
func (c *CaffeineCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache.Delete(key)
}

// Clear 清空缓存
func (c *CaffeineCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache.Flush()
}

// Size 获取缓存大小
func (c *CaffeineCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cache.ItemCount()
}

// DefaultCaffeineCache 默认缓存实现
type DefaultCaffeineCache struct {
	*CaffeineCache
}

// NewDefaultCaffeineCache 创建默认缓存
func NewDefaultCaffeineCache() *DefaultCaffeineCache {
	return &DefaultCaffeineCache{
		CaffeineCache: NewCaffeineCache(60*time.Second, 1000),
	}
}
