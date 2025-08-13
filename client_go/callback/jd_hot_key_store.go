package callback

import (
	"hotkey-client/cache"
	"hotkey-client/eventbus"
	"hotkey-client/key"
	"hotkey-client/rule"
	"time"
)

// JdHotKeyStore 热Key存储
type JdHotKeyStore struct{}

// IsHotKey 判断是否是热Key
func (jhs *JdHotKeyStore) IsHotKey(keyStr string) bool {
	defer func() {
		if r := recover(); r != nil {
			// 记录错误日志
			// logger.Error("IsHotKey panic: %v", r)
		}
	}()

	if !jhs.inRule(keyStr) {
		return false
	}

	isHot := jhs.isHot(keyStr)
	if !isHot {
		// 推送Key到Worker
		key.GetInstance().GetPusher().Send("", []*eventbus.HotKeyModel{
			{
				Key:   keyStr,
				Count: 1,
			},
		})
	} else {
		valueModel := jhs.getValueSimple(keyStr)
		// 判断是否临近过期
		if jhs.isNearExpire(valueModel) {
			key.GetInstance().GetPusher().Send("", []*eventbus.HotKeyModel{
				{
					Key:   keyStr,
					Count: 1,
				},
			})
		}
	}

	// 统计计数
	key.GetInstance().GetCounter().Collect(&eventbus.HotKeyModel{
		Key:    keyStr,
		Count:  1,
		Remove: false,
	})

	return isHot
}

// Get 从本地缓存获取值
func (jhs *JdHotKeyStore) Get(keyStr string) interface{} {
	value := jhs.getValueSimple(keyStr)
	if value == nil {
		return nil
	}

	object := value.Value
	// 如果是默认值也返回null
	if object == nil {
		return nil
	}

	return object
}

// SmartSet 智能设置值（仅当key是热Key时）
func (jhs *JdHotKeyStore) SmartSet(keyStr string, value interface{}) {
	if jhs.isHot(keyStr) {
		valueModel := jhs.getValueSimple(keyStr)
		if valueModel == nil {
			return
		}
		valueModel.Value = value
	}
}

// ForceSet 强制设置值
func (jhs *JdHotKeyStore) ForceSet(keyStr string, value interface{}) {
	valueModel := jhs.defaultValue(keyStr)
	if valueModel == nil {
		return
	}
	valueModel.Value = value
	jhs.setValueDirectly(keyStr, valueModel)
}

// GetValue 获取值，如果不存在则推送到Worker
func (jhs *JdHotKeyStore) GetValue(keyStr string, keyType string) interface{} {
	defer func() {
		if r := recover(); r != nil {
			// 记录错误日志
			// logger.Error("GetValue panic: %v", r)
		}
	}()

	// 如果没有为该key配置规则，就不用上报key
	if !jhs.inRule(keyStr) {
		return nil
	}

	value := jhs.getValueSimple(keyStr)
	if value == nil {
		// 推送到Worker
		key.GetInstance().GetPusher().Send("", []*eventbus.HotKeyModel{
			{
				Key:     keyStr,
				KeyType: keyType,
				Count:   1,
			},
		})
	} else {
		// 临近过期了，也推送
		if jhs.isNearExpire(value) {
			key.GetInstance().GetPusher().Send("", []*eventbus.HotKeyModel{
				{
					Key:     keyStr,
					KeyType: keyType,
					Count:   1,
				},
			})
		}
	}

	// 统计计数
	key.GetInstance().GetCounter().Collect(&eventbus.HotKeyModel{
		Key:    keyStr,
		Count:  1,
		Remove: false,
	})

	if value != nil {
		return value.Value
	}
	return nil
}

// Remove 删除Key
func (jhs *JdHotKeyStore) Remove(keyStr string) {
	jhs.getCache(keyStr).Delete(keyStr)
	// 通知Worker删除
	key.GetInstance().GetPusher().Send("", []*eventbus.HotKeyModel{
		{
			Key:    keyStr,
			Count:  1,
			Remove: true,
		},
	})
}

// 私有方法

// isNearExpire 是否临近过期
func (jhs *JdHotKeyStore) isNearExpire(valueModel *ValueModel) bool {
	if valueModel == nil {
		return true
	}
	// 判断是否过期时间小于2秒
	return valueModel.CreateTime+int64(valueModel.Duration)-time.Now().UnixMilli() <= 2000
}

// isHot 判断是否是热Key
func (jhs *JdHotKeyStore) isHot(keyStr string) bool {
	return jhs.getValueSimple(keyStr) != nil
}

// getValueSimple 获取值模型
func (jhs *JdHotKeyStore) getValueSimple(keyStr string) *ValueModel {
	object, exists := jhs.getCache(keyStr).Get(keyStr)
	if !exists || object == nil {
		return nil
	}
	if valueModel, ok := object.(*ValueModel); ok {
		return valueModel
	}
	return nil
}

// setValueDirectly 直接设置值
func (jhs *JdHotKeyStore) setValueDirectly(keyStr string, value interface{}) {
	jhs.getCache(keyStr).Set(keyStr, value)
}

// getCache 获取缓存
func (jhs *JdHotKeyStore) getCache(keyStr string) cache.LocalCache {
	return cache.GetCacheFactory().GetNonNullCache(keyStr)
}

// inRule 判断Key是否在规则内
func (jhs *JdHotKeyStore) inRule(keyStr string) bool {
	return cache.GetCacheFactory().GetCache(keyStr) != nil
}

// defaultValue 创建默认值模型
func (jhs *JdHotKeyStore) defaultValue(keyStr string) *ValueModel {
	duration := rule.GetInstance().Duration(keyStr)
	if duration <= 0 {
		return nil
	}

	return &ValueModel{
		CreateTime: time.Now().UnixMilli(),
		Duration:   int64(duration) * 1000, // 转换为毫秒
		Value:      nil,
	}
}

// 全局实例
var defaultStore = &JdHotKeyStore{}

// GetInstance 获取默认实例
func GetInstance() *JdHotKeyStore {
	return defaultStore
}
