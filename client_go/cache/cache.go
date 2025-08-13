package cache

import (
	"time"

	gocache "github.com/patrickmn/go-cache"
)

// Cache 基于go-cache的缓存实现
type Cache struct {
	cache *gocache.Cache
}

// NewCache 创建新的缓存实例
func NewCache(defaultExpiration, cleanupInterval time.Duration) *Cache {
	return &Cache{
		cache: gocache.New(defaultExpiration, cleanupInterval),
	}
}

// Get 获取缓存值
func (c *Cache) Get(key string) (interface{}, bool) {
	return c.cache.Get(key)
}

// Set 设置缓存值
func (c *Cache) Set(key string, value interface{}, ttl time.Duration) {
	c.cache.Set(key, value, ttl)
}

// Delete 删除缓存值
func (c *Cache) Delete(key string) {
	c.cache.Delete(key)
}

// Flush 清空缓存
func (c *Cache) Flush() {
	c.cache.Flush()
}

// ItemCount 获取缓存项数量
func (c *Cache) ItemCount() int {
	return c.cache.ItemCount()
}

// GetWithTTL 获取缓存值及其TTL
func (c *Cache) GetWithTTL(key string) (interface{}, time.Time, bool) {
	return c.cache.GetWithExpiration(key)
}
