package cache

import (
	"sync"
	"time"

	"github.com/dgraph-io/ristretto"

	"github.com/jd/platform/hotkey/client-go/model"
)

// LocalCache 本地缓存接口，对应Java的LocalCache
type LocalCache interface {
	Get(key string) interface{}
	Set(key string, value interface{})
	Delete(key string)
	Size() int64
	Clear()
}

// RistrettoCache 基于Ristretto的本地缓存实现，对应Java的CaffeineCache
type RistrettoCache struct {
	cache    *ristretto.Cache
	duration time.Duration
}

// NewRistrettoCache 创建新的Ristretto缓存
func NewRistrettoCache(maxSize int64, duration int) (*RistrettoCache, error) {
	config := &ristretto.Config{
		NumCounters: maxSize * 10, // number of keys to track frequency of (10x cache size)
		MaxCost:     maxSize,      // maximum cost of cache (cache size)
		BufferItems: 64,           // number of keys per Get buffer
	}

	cache, err := ristretto.NewCache(config)
	if err != nil {
		return nil, err
	}

	return &RistrettoCache{
		cache:    cache,
		duration: time.Duration(duration) * time.Second,
	}, nil
}

// Get 获取缓存值
func (rc *RistrettoCache) Get(key string) interface{} {
	value, found := rc.cache.Get(key)
	if !found {
		return nil
	}

	// 检查是否为ValueModel
	if vm, ok := value.(*model.ValueModel); ok {
		// 检查是否过期
		if vm.IsExpired() {
			rc.cache.Del(key)
			return nil
		}
		return vm
	}

	return value
}

// Set 设置缓存值
func (rc *RistrettoCache) Set(key string, value interface{}) {
	var ttl time.Duration

	// 如果是ValueModel，使用其内部的duration
	if vm, ok := value.(*model.ValueModel); ok {
		ttl = time.Duration(vm.Duration) * time.Millisecond
	} else {
		ttl = rc.duration
	}

	rc.cache.SetWithTTL(key, value, 1, ttl)
}

// Delete 删除缓存值
func (rc *RistrettoCache) Delete(key string) {
	rc.cache.Del(key)
}

// Size 获取缓存大小
func (rc *RistrettoCache) Size() int64 {
	metrics := rc.cache.Metrics
	return int64(metrics.KeysAdded() - metrics.KeysEvicted())
}

// Clear 清空缓存
func (rc *RistrettoCache) Clear() {
	rc.cache.Clear()
}

// DefaultCache 默认缓存实现，对应Java的DefaultCaffeineCache
type DefaultCache struct {
	cache map[string]interface{}
	mutex sync.RWMutex
}

// NewDefaultCache 创建默认缓存
func NewDefaultCache() *DefaultCache {
	return &DefaultCache{
		cache: make(map[string]interface{}),
	}
}

// Get 获取缓存值
func (dc *DefaultCache) Get(key string) interface{} {
	dc.mutex.RLock()
	defer dc.mutex.RUnlock()
	return dc.cache[key]
}

// Set 设置缓存值
func (dc *DefaultCache) Set(key string, value interface{}) {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()
	dc.cache[key] = value
}

// Delete 删除缓存值
func (dc *DefaultCache) Delete(key string) {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()
	delete(dc.cache, key)
}

// Size 获取缓存大小
func (dc *DefaultCache) Size() int64 {
	dc.mutex.RLock()
	defer dc.mutex.RUnlock()
	return int64(len(dc.cache))
}

// Clear 清空缓存
func (dc *DefaultCache) Clear() {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()
	dc.cache = make(map[string]interface{})
}

// CacheFactory 缓存工厂，对应Java的CacheFactory
type CacheFactory struct {
	defaultCache LocalCache
	ruleHolder   CacheRuleHolder
}

// CacheRuleHolder 缓存规则持有者接口，专门用于缓存工厂
type CacheRuleHolder interface {
	FindByKey(key string) LocalCache
}

// NewCacheFactory 创建缓存工厂
func NewCacheFactory(ruleHolder CacheRuleHolder) *CacheFactory {
	return &CacheFactory{
		defaultCache: NewDefaultCache(),
		ruleHolder:   ruleHolder,
	}
}

// Build 创建本地缓存实例，对应Java的CacheFactory.build
func (cf *CacheFactory) Build(duration int) (LocalCache, error) {
	return NewRistrettoCache(model.DefaultCacheSize, duration)
}

// GetNonNullCache 获取非空缓存，对应Java的CacheFactory.getNonNullCache
func (cf *CacheFactory) GetNonNullCache(key string) LocalCache {
	cache := cf.GetCache(key)
	if cache == nil {
		return cf.defaultCache
	}
	return cache
}

// GetCache 获取缓存，对应Java的CacheFactory.getCache
func (cf *CacheFactory) GetCache(key string) LocalCache {
	if cf.ruleHolder == nil {
		return nil
	}
	return cf.ruleHolder.FindByKey(key)
}

// CacheManager 缓存管理器，管理多个缓存实例
type CacheManager struct {
	caches       map[int]LocalCache // duration -> cache
	defaultCache LocalCache
	mutex        sync.RWMutex
	factory      *CacheFactory
}

// NewCacheManager 创建缓存管理器
func NewCacheManager() *CacheManager {
	return &CacheManager{
		caches:       make(map[int]LocalCache),
		defaultCache: NewDefaultCache(),
	}
}

// GetOrCreateCache 获取或创建指定duration的缓存
func (cm *CacheManager) GetOrCreateCache(duration int) LocalCache {
	cm.mutex.RLock()
	cache, exists := cm.caches[duration]
	cm.mutex.RUnlock()

	if exists {
		return cache
	}

	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// 双重检查
	if cache, exists := cm.caches[duration]; exists {
		return cache
	}

	// 创建新缓存
	newCache, err := NewRistrettoCache(model.DefaultCacheSize, duration)
	if err != nil {
		return cm.defaultCache
	}

	cm.caches[duration] = newCache
	return newCache
}

// RemoveCache 移除指定duration的缓存
func (cm *CacheManager) RemoveCache(duration int) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	if cache, exists := cm.caches[duration]; exists {
		cache.Clear()
		delete(cm.caches, duration)
	}
}

// GetDefaultCache 获取默认缓存
func (cm *CacheManager) GetDefaultCache() LocalCache {
	return cm.defaultCache
}

// Clear 清空所有缓存
func (cm *CacheManager) Clear() {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	for _, cache := range cm.caches {
		cache.Clear()
	}
	cm.caches = make(map[int]LocalCache)
	cm.defaultCache.Clear()
}

// GetCacheStats 获取缓存统计信息
func (cm *CacheManager) GetCacheStats() map[int]int64 {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	stats := make(map[int]int64)
	for duration, cache := range cm.caches {
		stats[duration] = cache.Size()
	}
	return stats
}

// CacheBuilder 缓存构建器，对应Java的CaffeineBuilder
type CacheBuilder struct {
	minSize       int64
	maxSize       int64
	expireSeconds int
	initialCap    int64
}

// NewCacheBuilder 创建缓存构建器
func NewCacheBuilder() *CacheBuilder {
	return &CacheBuilder{
		minSize:       128,
		maxSize:       model.DefaultCacheSize,
		expireSeconds: 60,
		initialCap:    128,
	}
}

// SetMinSize 设置最小大小
func (cb *CacheBuilder) SetMinSize(minSize int64) *CacheBuilder {
	cb.minSize = minSize
	return cb
}

// SetMaxSize 设置最大大小
func (cb *CacheBuilder) SetMaxSize(maxSize int64) *CacheBuilder {
	cb.maxSize = maxSize
	return cb
}

// SetExpireSeconds 设置过期时间（秒）
func (cb *CacheBuilder) SetExpireSeconds(expireSeconds int) *CacheBuilder {
	cb.expireSeconds = expireSeconds
	return cb
}

// SetInitialCapacity 设置初始容量
func (cb *CacheBuilder) SetInitialCapacity(initialCap int64) *CacheBuilder {
	cb.initialCap = initialCap
	return cb
}

// Build 构建缓存
func (cb *CacheBuilder) Build() (LocalCache, error) {
	return NewRistrettoCacheWithConfig(cb.maxSize, cb.expireSeconds, cb.initialCap)
}

// BuildDefault 构建默认缓存
func (cb *CacheBuilder) BuildDefault() LocalCache {
	cache, err := cb.Build()
	if err != nil {
		return NewDefaultCache()
	}
	return cache
}

// NewRistrettoCacheWithConfig 创建带配置的Ristretto缓存
func NewRistrettoCacheWithConfig(maxSize int64, duration int, initialCap int64) (*RistrettoCache, error) {
	config := &ristretto.Config{
		NumCounters: maxSize * 10, // number of keys to track frequency of (10x cache size)
		MaxCost:     maxSize,      // maximum cost of cache (cache size)
		BufferItems: 64,           // number of keys per Get buffer
	}

	cache, err := ristretto.NewCache(config)
	if err != nil {
		return nil, err
	}

	return &RistrettoCache{
		cache:    cache,
		duration: time.Duration(duration) * time.Second,
	}, nil
}

// 便利方法，对应Java的CaffeineBuilder静态方法

// Cache 创建默认缓存
func Cache() LocalCache {
	return NewCacheBuilder().BuildDefault()
}

// CacheWithDuration 创建指定过期时间的缓存
func CacheWithDuration(duration int) LocalCache {
	return NewCacheBuilder().
		SetExpireSeconds(duration).
		BuildDefault()
}

// CacheWithSize 创建指定大小的缓存
func CacheWithSize(maxSize int64) LocalCache {
	return NewCacheBuilder().
		SetMaxSize(maxSize).
		BuildDefault()
}

// CacheWithConfig 创建指定配置的缓存
func CacheWithConfig(minSize, maxSize int64, expireSeconds int) LocalCache {
	return NewCacheBuilder().
		SetMinSize(minSize).
		SetMaxSize(maxSize).
		SetExpireSeconds(expireSeconds).
		BuildDefault()
}
