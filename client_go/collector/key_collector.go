package collector

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/jd/platform/hotkey/client-go/model"
)

// KeyCollector Key收集器接口，对应Java的IKeyCollector
type KeyCollector interface {
	Collect(item interface{})
	LockAndGetResult() interface{}
	FinishOnce()
}

// TurnKeyCollector 轮转Key收集器，对应Java的TurnKeyCollector
type TurnKeyCollector struct {
	map0       sync.Map // 使用sync.Map替代ConcurrentHashMap
	map1       sync.Map
	atomicLong int64 // 使用atomic操作
}

// NewTurnKeyCollector 创建轮转Key收集器
func NewTurnKeyCollector() *TurnKeyCollector {
	return &TurnKeyCollector{}
}

// Collect 收集热key，对应Java的TurnKeyCollector.collect
func (tkc *TurnKeyCollector) Collect(item interface{}) {
	hotKeyModel, ok := item.(*model.HotKeyModel)
	if !ok {
		return
	}

	key := hotKeyModel.Key
	if key == "" {
		return
	}

	currentMap := tkc.getCurrentMap()

	// 尝试添加新的hotKeyModel，如果已存在则累加count
	if existing, loaded := currentMap.LoadOrStore(key, hotKeyModel); loaded {
		if existingModel, ok := existing.(*model.HotKeyModel); ok {
			existingModel.Add(hotKeyModel.GetCount())
		}
	}
}

// LockAndGetResult 锁定并获取结果，对应Java的TurnKeyCollector.lockAndGetResult
func (tkc *TurnKeyCollector) LockAndGetResult() interface{} {
	// 自增后，对应的map就会停止被写入，等待被读取
	atomic.AddInt64(&tkc.atomicLong, 1)

	var targetMap *sync.Map
	if atomic.LoadInt64(&tkc.atomicLong)%2 == 0 {
		targetMap = &tkc.map1
	} else {
		targetMap = &tkc.map0
	}

	// 收集结果
	var result []*model.HotKeyModel
	targetMap.Range(func(key, value interface{}) bool {
		if hotKeyModel, ok := value.(*model.HotKeyModel); ok {
			result = append(result, hotKeyModel)
		}
		return true
	})

	// 清空map
	tkc.clearMap(targetMap)

	return result
}

// FinishOnce 完成一次处理，对应Java的TurnKeyCollector.finishOnce
func (tkc *TurnKeyCollector) FinishOnce() {
	// 在Go版本中，这个方法可以为空，因为我们使用atomic操作
}

// getCurrentMap 获取当前应该写入的map
func (tkc *TurnKeyCollector) getCurrentMap() *sync.Map {
	if atomic.LoadInt64(&tkc.atomicLong)%2 == 0 {
		return &tkc.map0
	}
	return &tkc.map1
}

// clearMap 清空指定的map
func (tkc *TurnKeyCollector) clearMap(m *sync.Map) {
	m.Range(func(key, value interface{}) bool {
		m.Delete(key)
		return true
	})
}

// TurnCountCollector 轮转计数收集器，对应Java的TurnCountCollector
type TurnCountCollector struct {
	hitMap0    sync.Map
	hitMap1    sync.Map
	atomicLong int64
	ruleHolder RuleHolder
}

// RuleHolder 规则持有者接口
type RuleHolder interface {
	Rule(key string) string
}

// HitCount 命中计数
type HitCount struct {
	hotHitCount   int64
	totalHitCount int64
}

// NewTurnCountCollector 创建轮转计数收集器
func NewTurnCountCollector(ruleHolder RuleHolder) *TurnCountCollector {
	return &TurnCountCollector{
		ruleHolder: ruleHolder,
	}
}

// Collect 收集计数，对应Java的TurnCountCollector.collect
func (tcc *TurnCountCollector) Collect(item interface{}) {
	keyHotModel, ok := item.(*model.KeyHotModel)
	if !ok {
		return
	}

	currentMap := tcc.getCurrentMap()
	tcc.put(keyHotModel.Key, keyHotModel.IsHot, currentMap)
}

// LockAndGetResult 锁定并获取结果，对应Java的TurnCountCollector.lockAndGetResult
func (tcc *TurnCountCollector) LockAndGetResult() interface{} {
	// 自增后，对应的map就会停止被写入，等待被读取
	atomic.AddInt64(&tcc.atomicLong, 1)

	var targetMap *sync.Map
	if atomic.LoadInt64(&tcc.atomicLong)%2 == 0 {
		targetMap = &tcc.hitMap1
	} else {
		targetMap = &tcc.hitMap0
	}

	// 收集结果
	result := tcc.convertToKeyCountModels(targetMap)

	// 清空map
	tcc.clearMap(targetMap)

	return result
}

// FinishOnce 完成一次处理
func (tcc *TurnCountCollector) FinishOnce() {
	// 在Go版本中，这个方法可以为空
}

// put 添加计数，对应Java的TurnCountCollector.put
func (tcc *TurnCountCollector) put(key string, isHot bool, targetMap *sync.Map) {
	// 如key是pin_的前缀，则存储pin_
	rule := tcc.ruleHolder.Rule(key)
	// 不在规则内的不处理
	if rule == "" {
		return
	}

	nowTime := tcc.nowTime()
	// rule + 分隔符 + 2020-10-23 21:11:22
	mapKey := rule + model.CountDelimiter + nowTime

	// 获取或创建HitCount
	hitCountInterface, _ := targetMap.LoadOrStore(mapKey, &HitCount{})
	hitCount := hitCountInterface.(*HitCount)

	if isHot {
		atomic.AddInt64(&hitCount.hotHitCount, 1)
	}
	atomic.AddInt64(&hitCount.totalHitCount, 1)
}

// getCurrentMap 获取当前应该写入的map
func (tcc *TurnCountCollector) getCurrentMap() *sync.Map {
	if atomic.LoadInt64(&tcc.atomicLong)%2 == 0 {
		return &tcc.hitMap0
	}
	return &tcc.hitMap1
}

// clearMap 清空指定的map
func (tcc *TurnCountCollector) clearMap(m *sync.Map) {
	m.Range(func(key, value interface{}) bool {
		m.Delete(key)
		return true
	})
}

// convertToKeyCountModels 转换为KeyCountModel列表
func (tcc *TurnCountCollector) convertToKeyCountModels(targetMap *sync.Map) []*model.KeyCountModel {
	var result []*model.KeyCountModel

	targetMap.Range(func(key, value interface{}) bool {
		mapKey := key.(string)
		hitCount := value.(*HitCount)

		keyCountModel := &model.KeyCountModel{
			RuleKey:       mapKey,
			TotalHitCount: int(atomic.LoadInt64(&hitCount.totalHitCount)),
			HotHitCount:   int(atomic.LoadInt64(&hitCount.hotHitCount)),
		}
		result = append(result, keyCountModel)
		return true
	})

	return result
}

// nowTime 获取当前时间字符串
func (tcc *TurnCountCollector) nowTime() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

// KeyHandlerFactory Key处理器工厂，对应Java的KeyHandlerFactory
type KeyHandlerFactory struct {
	keyCollector   *TurnKeyCollector
	countCollector *TurnCountCollector
	keyPusher      KeyPusher
}

// KeyPusher Key推送器接口
type KeyPusher interface {
	Send(appName string, hotKeys []*model.HotKeyModel) error
	SendCount(appName string, countModels []*model.KeyCountModel) error
}

// NewKeyHandlerFactory 创建Key处理器工厂
func NewKeyHandlerFactory(ruleHolder RuleHolder, keyPusher KeyPusher) *KeyHandlerFactory {
	return &KeyHandlerFactory{
		keyCollector:   NewTurnKeyCollector(),
		countCollector: NewTurnCountCollector(ruleHolder),
		keyPusher:      keyPusher,
	}
}

// GetCollector 获取Key收集器
func (khf *KeyHandlerFactory) GetCollector() *TurnKeyCollector {
	return khf.keyCollector
}

// GetCounter 获取计数收集器
func (khf *KeyHandlerFactory) GetCounter() *TurnCountCollector {
	return khf.countCollector
}

// GetPusher 获取推送器
func (khf *KeyHandlerFactory) GetPusher() KeyPusher {
	return khf.keyPusher
}
