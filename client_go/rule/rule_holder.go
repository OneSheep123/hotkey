package rule

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/jd/platform/hotkey/client-go/cache"
	"github.com/jd/platform/hotkey/client-go/event"
	"github.com/jd/platform/hotkey/client-go/model"
)

// KeyRuleHolder 保存key的规则，对应Java的KeyRuleHolder
type KeyRuleHolder struct {
	ruleCacheMap map[int]cache.LocalCache // 保存超时时间和cache的映射
	keyRules     []*model.KeyRule         // 所有的规则
	cacheManager *cache.CacheManager
	mutex        sync.RWMutex
}

// NewKeyRuleHolder 创建新的规则持有者
func NewKeyRuleHolder() *KeyRuleHolder {
	return &KeyRuleHolder{
		ruleCacheMap: make(map[int]cache.LocalCache),
		keyRules:     make([]*model.KeyRule, 0),
		cacheManager: cache.NewCacheManager(),
	}
}

// PutRules 设置所有规则，对应Java的KeyRuleHolder.putRules
func (krh *KeyRuleHolder) PutRules(keyRules []*model.KeyRule) {
	krh.mutex.Lock()
	defer krh.mutex.Unlock()

	// 如果规则为空，清空规则表
	if len(keyRules) == 0 {
		krh.keyRules = make([]*model.KeyRule, 0)
		krh.ruleCacheMap = make(map[int]cache.LocalCache)
		krh.cacheManager.Clear()
		return
	}

	krh.keyRules = make([]*model.KeyRule, len(keyRules))
	copy(krh.keyRules, keyRules)

	// 收集所有的duration
	durationSet := make(map[int]bool)
	for _, rule := range keyRules {
		durationSet[rule.Duration] = true
	}

	// 清除掉那些在ruleCacheMap里存的，但是rule里已没有的
	for duration := range krh.ruleCacheMap {
		if !durationSet[duration] {
			krh.cacheManager.RemoveCache(duration)
			delete(krh.ruleCacheMap, duration)
		}
	}

	// 遍历所有的规则，创建对应的缓存
	for _, keyRule := range keyRules {
		duration := keyRule.Duration
		if _, exists := krh.ruleCacheMap[duration]; !exists {
			cache := krh.cacheManager.GetOrCreateCache(duration)
			krh.ruleCacheMap[duration] = cache
		}
	}
}

// FindByKey 根据key返回对应的LocalCache，对应Java的KeyRuleHolder.findByKey
func (krh *KeyRuleHolder) FindByKey(key string) cache.LocalCache {
	if key == "" {
		return nil
	}

	rule := krh.findRule(key)
	if rule == nil {
		return nil
	}

	krh.mutex.RLock()
	defer krh.mutex.RUnlock()
	return krh.ruleCacheMap[rule.Duration]
}

// Rule 判断该key命中了哪个rule，对应Java的KeyRuleHolder.rule
func (krh *KeyRuleHolder) Rule(key string) string {
	rule := krh.findRule(key)
	if rule != nil {
		return rule.Key
	}
	return ""
}

// Duration 获取该key应该缓存多久，对应Java的KeyRuleHolder.duration
func (krh *KeyRuleHolder) Duration(key string) int {
	rule := krh.findRule(key)
	if rule != nil {
		return rule.Duration
	}
	return 0
}

// IsKeyInRule 判断key是否在配置的要探测的规则内，对应Java的KeyRuleHolder.isKeyInRule
func (krh *KeyRuleHolder) IsKeyInRule(key string) bool {
	if key == "" {
		return false
	}

	krh.mutex.RLock()
	defer krh.mutex.RUnlock()

	// 遍历该app的所有rule，找到与key匹配的rule
	for _, keyRule := range krh.keyRules {
		if keyRule.Key == "*" || key == keyRule.Key ||
			(keyRule.Prefix && strings.HasPrefix(key, keyRule.Key)) {
			return true
		}
	}
	return false
}

// findRule 遍历该app的所有rule，找到与key匹配的rule
// 优先级：全匹配 -> prefix匹配 -> * 通配，对应Java的KeyRuleHolder.findRule
func (krh *KeyRuleHolder) findRule(key string) *model.KeyRule {
	krh.mutex.RLock()
	defer krh.mutex.RUnlock()

	var prefix *model.KeyRule
	var common *model.KeyRule

	for _, keyRule := range krh.keyRules {
		if key == keyRule.Key {
			return keyRule
		}
		if keyRule.Prefix && strings.HasPrefix(key, keyRule.Key) {
			prefix = keyRule
		}
		if keyRule.Key == "*" {
			common = keyRule
		}
	}

	if prefix != nil {
		return prefix
	}
	return common
}

// HandleKeyRuleInfoChange 处理规则信息变化事件，对应Java的KeyRuleHolder.ruleChange
func (krh *KeyRuleHolder) HandleKeyRuleInfoChange(event *event.KeyRuleInfoChangeEvent) {
	log.Printf("New rules info is: %v", event.KeyRules)
	if event.KeyRules != nil {
		krh.PutRules(event.KeyRules)
	}
}

// GetRules 获取所有规则
func (krh *KeyRuleHolder) GetRules() []*model.KeyRule {
	krh.mutex.RLock()
	defer krh.mutex.RUnlock()

	rules := make([]*model.KeyRule, len(krh.keyRules))
	copy(rules, krh.keyRules)
	return rules
}

// GetCacheStats 获取缓存统计信息
func (krh *KeyRuleHolder) GetCacheStats() map[int]int64 {
	return krh.cacheManager.GetCacheStats()
}

// Clear 清空所有规则和缓存
func (krh *KeyRuleHolder) Clear() {
	krh.mutex.Lock()
	defer krh.mutex.Unlock()

	krh.keyRules = make([]*model.KeyRule, 0)
	krh.ruleCacheMap = make(map[int]cache.LocalCache)
	krh.cacheManager.Clear()
}

// ValidateRule 验证规则的有效性
func (krh *KeyRuleHolder) ValidateRule(rule *model.KeyRule) error {
	if rule == nil {
		return fmt.Errorf("rule cannot be nil")
	}
	if rule.Key == "" {
		return fmt.Errorf("rule key cannot be empty")
	}
	if rule.Duration <= 0 {
		return fmt.Errorf("rule duration must be positive")
	}
	if rule.Interval <= 0 {
		return fmt.Errorf("rule interval must be positive")
	}
	if rule.Threshold <= 0 {
		return fmt.Errorf("rule threshold must be positive")
	}
	return nil
}

// AddRule 添加单个规则
func (krh *KeyRuleHolder) AddRule(rule *model.KeyRule) error {
	if err := krh.ValidateRule(rule); err != nil {
		return err
	}

	krh.mutex.Lock()
	defer krh.mutex.Unlock()

	// 检查是否已存在相同的规则
	for i, existingRule := range krh.keyRules {
		if existingRule.Key == rule.Key && existingRule.Prefix == rule.Prefix {
			// 更新现有规则
			krh.keyRules[i] = rule
			return nil
		}
	}

	// 添加新规则
	krh.keyRules = append(krh.keyRules, rule)

	// 创建对应的缓存
	if _, exists := krh.ruleCacheMap[rule.Duration]; !exists {
		cache := krh.cacheManager.GetOrCreateCache(rule.Duration)
		krh.ruleCacheMap[rule.Duration] = cache
	}

	return nil
}

// RemoveRule 移除规则
func (krh *KeyRuleHolder) RemoveRule(key string, prefix bool) bool {
	krh.mutex.Lock()
	defer krh.mutex.Unlock()

	for i, rule := range krh.keyRules {
		if rule.Key == key && rule.Prefix == prefix {
			// 移除规则
			krh.keyRules = append(krh.keyRules[:i], krh.keyRules[i+1:]...)

			// 检查是否还有其他规则使用相同的duration
			duration := rule.Duration
			stillUsed := false
			for _, r := range krh.keyRules {
				if r.Duration == duration {
					stillUsed = true
					break
				}
			}

			// 如果没有其他规则使用这个duration，移除对应的缓存
			if !stillUsed {
				krh.cacheManager.RemoveCache(duration)
				delete(krh.ruleCacheMap, duration)
			}

			return true
		}
	}
	return false
}
