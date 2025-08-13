package cache

import (
	"sync"
	"time"
)

// CacheFactory 缓存工厂
type CacheFactory struct {
	caches map[int]LocalCache // key: duration(seconds), value: cache
	mu     sync.RWMutex
}

var defaultFactory = &CacheFactory{
	caches: make(map[int]LocalCache),
}

// GetCacheFactory 获取默认缓存工厂
func GetCacheFactory() *CacheFactory {
	return defaultFactory
}

// Build 创建指定持续时间的缓存
func (cf *CacheFactory) Build(durationSeconds int) LocalCache {
	cf.mu.Lock()
	defer cf.mu.Unlock()

	if cache, exists := cf.caches[durationSeconds]; exists {
		return cache
	}

	// 创建新的缓存实例
	duration := time.Duration(durationSeconds) * time.Second
	cache := NewCaffeineCache(duration, 1000)
	cf.caches[durationSeconds] = cache

	return cache
}

// GetCache 根据key获取对应的缓存
func (cf *CacheFactory) GetCache(key string) LocalCache {
	// 这里需要根据key的规则来获取对应的缓存
	// 暂时返回默认缓存，实际实现需要与规则模块集成
	return cf.GetDefaultCache()
}

// GetNonNullCache 获取非空缓存，如果不存在则返回默认缓存
func (cf *CacheFactory) GetNonNullCache(key string) LocalCache {
	cache := cf.GetCache(key)
	if cache == nil {
		return cf.GetDefaultCache()
	}
	return cache
}

// GetDefaultCache 获取默认缓存
func (cf *CacheFactory) GetDefaultCache() LocalCache {
	return NewDefaultCaffeineCache()
}

// Clear 清空所有缓存
func (cf *CacheFactory) Clear() {
	cf.mu.Lock()
	defer cf.mu.Unlock()

	for _, cache := range cf.caches {
		cache.Clear()
	}
	cf.caches = make(map[int]LocalCache)
}
