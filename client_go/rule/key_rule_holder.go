package rule

import (
	"hotkey-client/cache"
	"hotkey-client/eventbus"
	"strings"
	"sync"
)

// KeyRuleHolder Key规则持有者
type KeyRuleHolder struct {
	rules    []*eventbus.KeyRule
	cacheMap map[int]cache.LocalCache
	mu       sync.RWMutex
}

var defaultHolder = &KeyRuleHolder{
	rules:    make([]*eventbus.KeyRule, 0),
	cacheMap: make(map[int]cache.LocalCache),
}

// GetInstance 获取默认实例
func GetInstance() *KeyRuleHolder {
	return defaultHolder
}

// PutRules 设置所有规则
func (krh *KeyRuleHolder) PutRules(keyRules []*eventbus.KeyRule) {
	krh.mu.Lock()
	defer krh.mu.Unlock()

	// 如果规则为空，清空规则表
	if len(keyRules) == 0 {
		krh.rules = nil
		krh.cacheMap = nil
		return
	}

	krh.rules = make([]*eventbus.KeyRule, len(keyRules))
	copy(krh.rules, keyRules)

	// 收集所有持续时间
	durationSet := make(map[int]bool)
	for _, rule := range keyRules {
		durationSet[rule.Duration] = true
	}

	// 清理不再使用的缓存
	for duration := range krh.cacheMap {
		if !durationSet[duration] {
			delete(krh.cacheMap, duration)
		}
	}

	// 为每个持续时间创建缓存
	for _, keyRule := range keyRules {
		duration := keyRule.Duration
		if _, exists := krh.cacheMap[duration]; !exists {
			localCache := cache.GetCacheFactory().Build(duration)
			krh.cacheMap[duration] = localCache
		}
	}
}

// FindByKey 根据key返回对应的LocalCache
func (krh *KeyRuleHolder) FindByKey(key string) cache.LocalCache {
	if key == "" {
		return nil
	}

	keyRule := krh.findRule(key)
	if keyRule == nil {
		return nil
	}

	krh.mu.RLock()
	defer krh.mu.RUnlock()

	return krh.cacheMap[keyRule.Duration]
}

// Rule 判断该key命中了哪个rule
func (krh *KeyRuleHolder) Rule(key string) string {
	keyRule := krh.findRule(key)
	if keyRule != nil {
		return keyRule.Key
	}
	return ""
}

// Duration 获取该key应该缓存多久
func (krh *KeyRuleHolder) Duration(key string) int {
	keyRule := krh.findRule(key)
	if keyRule != nil {
		return keyRule.Duration
	}
	return 0
}

// IsKeyInRule 判断key是否在配置的要探测的规则内
func (krh *KeyRuleHolder) IsKeyInRule(key string) bool {
	if key == "" {
		return false
	}

	krh.mu.RLock()
	defer krh.mu.RUnlock()

	// 遍历该app的所有rule，找到与key匹配的rule
	for _, keyRule := range krh.rules {
		if keyRule.Key == "*" || key == keyRule.Key ||
			(keyRule.Prefix && strings.HasPrefix(key, keyRule.Key)) {
			return true
		}
	}
	return false
}

// findRule 查找匹配的规则
func (krh *KeyRuleHolder) findRule(key string) *eventbus.KeyRule {
	krh.mu.RLock()
	defer krh.mu.RUnlock()

	var prefix *eventbus.KeyRule
	var common *eventbus.KeyRule

	for _, keyRule := range krh.rules {
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

// RuleChange 监听规则变化事件
func (krh *KeyRuleHolder) RuleChange(event *eventbus.KeyRuleInfoChangeEvent) {
	// 记录日志
	// logger.Info("New rules info is: %v", event.KeyRules)

	ruleList := event.KeyRules
	if ruleList == nil {
		return
	}

	krh.PutRules(ruleList)
}
