package hotkey

import (
	"log"
	"time"

	"github.com/jd/platform/hotkey/client-go/cache"
	"github.com/jd/platform/hotkey/client-go/collector"
	"github.com/jd/platform/hotkey/client-go/model"
	"github.com/jd/platform/hotkey/client-go/rule"
)

// HotKeyStore 热key存储，对应Java的JdHotKeyStore
type HotKeyStore struct {
	cacheFactory     *cache.CacheFactory
	ruleHolder       *rule.KeyRuleHolder
	collectorManager *collector.CollectorManager
	hotKeyPusher     *collector.HotKeyPusher
}

// NewHotKeyStore 创建热key存储
func NewHotKeyStore(cacheFactory *cache.CacheFactory, ruleHolder *rule.KeyRuleHolder, 
	collectorManager *collector.CollectorManager, hotKeyPusher *collector.HotKeyPusher) *HotKeyStore {
	return &HotKeyStore{
		cacheFactory:     cacheFactory,
		ruleHolder:       ruleHolder,
		collectorManager: collectorManager,
		hotKeyPusher:     hotKeyPusher,
	}
}

// IsHotKey 判断是否是热key，对应Java的JdHotKeyStore.isHotKey
func (hks *HotKeyStore) IsHotKey(key string) bool {
	// 如果不在规则内，直接返回false
	if !hks.inRule(key) {
		return false
	}

	isHot := hks.isHot(key)
	if !isHot {
		// 不是热key，上报一次
		hks.hotKeyPusher.PushWithDefaults(key)
	} else {
		// 是热key，检查是否临近过期
		valueModel := hks.getValueSimple(key)
		if hks.isNearExpire(valueModel) {
			hks.hotKeyPusher.PushWithDefaults(key)
		}
	}

	// 统计计数
	keyHotModel := model.NewKeyHotModel(key, isHot)
	hks.collectorManager.CollectKeyHot(keyHotModel)

	return isHot
}

// Get 从本地缓存取值，对应Java的JdHotKeyStore.get
func (hks *HotKeyStore) Get(key string) interface{} {
	valueModel := hks.getValueSimple(key)
	if valueModel == nil {
		return nil
	}

	value := valueModel.Value
	// 如果是默认值也返回nil
	if intValue, ok := value.(int); ok && intValue == model.MagicNumber {
		return nil
	}

	return value
}

// GetValue 获取value，如果value不存在则上报，对应Java的JdHotKeyStore.getValue
func (hks *HotKeyStore) GetValue(key string, keyType model.KeyType) interface{} {
	// 如果没有为该key配置规则，就不用上报key
	if !hks.inRule(key) {
		return nil
	}

	var userValue interface{}
	valueModel := hks.getValueSimple(key)

	if valueModel == nil {
		hks.hotKeyPusher.Push(key, keyType, 1, false)
	} else {
		// 临近过期了，也发
		if hks.isNearExpire(valueModel) {
			hks.hotKeyPusher.Push(key, keyType, 1, false)
		}

		value := valueModel.Value
		// 如果是默认值，也返回nil
		if intValue, ok := value.(int); ok && intValue == model.MagicNumber {
			userValue = nil
		} else {
			userValue = value
		}
	}

	// 统计计数
	keyHotModel := model.NewKeyHotModel(key, valueModel != nil)
	hks.collectorManager.CollectKeyHot(keyHotModel)

	return userValue
}

// GetValueWithDefaults 使用默认keyType获取value
func (hks *HotKeyStore) GetValueWithDefaults(key string) interface{} {
	return hks.GetValue(key, model.RedisKey)
}

// SmartSet 智能设置值，仅当key是热key时设置，对应Java的JdHotKeyStore.smartSet
func (hks *HotKeyStore) SmartSet(key string, value interface{}) {
	if hks.isHot(key) {
		valueModel := hks.getValueSimple(key)
		if valueModel != nil {
			valueModel.Value = value
		}
	}
}

// ForceSet 强制设置值，对应Java的JdHotKeyStore.forceSet
func (hks *HotKeyStore) ForceSet(key string, value interface{}) {
	duration := hks.ruleHolder.Duration(key)
	if duration <= 0 {
		// 如果没有规则，使用默认缓存
		localCache := hks.cacheFactory.GetNonNullCache(key)
		localCache.Set(key, value)
		return
	}

	valueModel := model.NewValueModel(duration)
	if valueModel != nil {
		valueModel.Value = value
	}
	hks.setValueDirectly(key, valueModel)
}

// Remove 删除某key，会通知整个集群删除，对应Java的JdHotKeyStore.remove
func (hks *HotKeyStore) Remove(key string) {
	localCache := hks.cacheFactory.GetNonNullCache(key)
	localCache.Delete(key)
	hks.hotKeyPusher.Remove(key)
}

// isHot 判断是否是热key，对应Java的JdHotKeyStore.isHot
func (hks *HotKeyStore) isHot(key string) bool {
	return hks.getValueSimple(key) != nil
}

// getValueSimple 仅获取value，如果不存在也不上报热key，对应Java的JdHotKeyStore.getValueSimple
func (hks *HotKeyStore) getValueSimple(key string) *model.ValueModel {
	localCache := hks.cacheFactory.GetCache(key)
	if localCache == nil {
		return nil
	}

	value := localCache.Get(key)
	if value == nil {
		return nil
	}

	valueModel, ok := value.(*model.ValueModel)
	if !ok {
		return nil
	}

	return valueModel
}

// setValueDirectly 纯粹的本地缓存，无需该key是热key，对应Java的JdHotKeyStore.setValueDirectly
func (hks *HotKeyStore) setValueDirectly(key string, value interface{}) {
	localCache := hks.cacheFactory.GetNonNullCache(key)
	localCache.Set(key, value)
}

// inRule 判断这个key是否在被探测的规则范围内，对应Java的JdHotKeyStore.inRule
func (hks *HotKeyStore) inRule(key string) bool {
	return hks.cacheFactory.GetCache(key) != nil
}

// isNearExpire 是否临近过期，对应Java的JdHotKeyStore.isNearExpire
func (hks *HotKeyStore) isNearExpire(valueModel *model.ValueModel) bool {
	// 判断是否过期时间小于2秒，小于2秒的话也发送
	if valueModel == nil {
		return true
	}
	return valueModel.IsNearExpire()
}

// 全局热key存储实例
var globalHotKeyStore *HotKeyStore

// SetGlobalHotKeyStore 设置全局热key存储实例
func SetGlobalHotKeyStore(store *HotKeyStore) {
	globalHotKeyStore = store
}

// GetGlobalHotKeyStore 获取全局热key存储实例
func GetGlobalHotKeyStore() *HotKeyStore {
	return globalHotKeyStore
}

// 全局API函数，对应Java的JdHotKeyStore静态方法

// IsHotKey 全局判断是否是热key
func IsHotKey(key string) bool {
	if globalHotKeyStore == nil {
		log.Println("Warning: HotKeyStore not initialized")
		return false
	}
	return globalHotKeyStore.IsHotKey(key)
}

// Get 全局从本地缓存取值
func Get(key string) interface{} {
	if globalHotKeyStore == nil {
		log.Println("Warning: HotKeyStore not initialized")
		return nil
	}
	return globalHotKeyStore.Get(key)
}

// GetValue 全局获取value
func GetValue(key string, keyType model.KeyType) interface{} {
	if globalHotKeyStore == nil {
		log.Println("Warning: HotKeyStore not initialized")
		return nil
	}
	return globalHotKeyStore.GetValue(key, keyType)
}

// GetValueWithDefaults 全局使用默认keyType获取value
func GetValueWithDefaults(key string) interface{} {
	if globalHotKeyStore == nil {
		log.Println("Warning: HotKeyStore not initialized")
		return nil
	}
	return globalHotKeyStore.GetValueWithDefaults(key)
}

// SmartSet 全局智能设置值
func SmartSet(key string, value interface{}) {
	if globalHotKeyStore == nil {
		log.Println("Warning: HotKeyStore not initialized")
		return
	}
	globalHotKeyStore.SmartSet(key, value)
}

// ForceSet 全局强制设置值
func ForceSet(key string, value interface{}) {
	if globalHotKeyStore == nil {
		log.Println("Warning: HotKeyStore not initialized")
		return
	}
	globalHotKeyStore.ForceSet(key, value)
}

// Remove 全局删除key
func Remove(key string) {
	if globalHotKeyStore == nil {
		log.Println("Warning: HotKeyStore not initialized")
		return
	}
	globalHotKeyStore.Remove(key)
}

// NewKeyListener 新key监听器实现，对应Java的DefaultNewKeyListener
type NewKeyListener struct {
	cacheFactory *cache.CacheFactory
}

// NewNewKeyListener 创建新key监听器
func NewNewKeyListener(cacheFactory *cache.CacheFactory) *NewKeyListener {
	return &NewKeyListener{
		cacheFactory: cacheFactory,
	}
}

// NewKey 处理新key，对应Java的DefaultNewKeyListener.newKey
func (nkl *NewKeyListener) NewKey(hotKeyModel *model.HotKeyModel) {
	now := time.Now().UnixMilli()
	
	// 如果key到达时已经过去1秒了，记录一下
	if hotKeyModel.CreateTime != 0 && abs(now-hotKeyModel.CreateTime) > 1000 {
		log.Printf("Warning: the key comes too late: %s now %d keyCreateAt %d", 
			hotKeyModel.Key, now, hotKeyModel.CreateTime)
	}

	if hotKeyModel.Remove {
		// 如果是删除事件，就直接删除
		nkl.deleteKey(hotKeyModel.Key)
		return
	}

	// 已经是热key了，又推过来同样的热key，做个日志记录，并刷新一下
	if globalHotKeyStore != nil && globalHotKeyStore.isHot(hotKeyModel.Key) {
		log.Printf("Warning: receive repeat hot key: %s at %d", hotKeyModel.Key, now)
	}

	nkl.addKey(hotKeyModel.Key)
}

// addKey 添加key
func (nkl *NewKeyListener) addKey(key string) {
	// 根据规则创建ValueModel
	duration := 0
	if globalHotKeyStore != nil {
		duration = globalHotKeyStore.ruleHolder.Duration(key)
	}
	
	if duration <= 0 {
		// 不符合任何规则
		nkl.deleteKey(key)
		return
	}

	valueModel := model.NewValueModel(duration)
	if valueModel == nil {
		nkl.deleteKey(key)
		return
	}

	// 如果原来该key已经存在了，那么value就被重置，过期时间也会被重置
	if globalHotKeyStore != nil {
		globalHotKeyStore.setValueDirectly(key, valueModel)
	}
}

// deleteKey 删除key
func (nkl *NewKeyListener) deleteKey(key string) {
	localCache := nkl.cacheFactory.GetNonNullCache(key)
	localCache.Delete(key)
}

// abs 返回绝对值
func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}
