package key

import (
	"hotkey-client/eventbus"
	"sync"
	"sync/atomic"
)

// IKeyCollector Key收集器接口
type IKeyCollector interface {
	Collect(key *eventbus.HotKeyModel)
	LockAndGetResult() []*eventbus.HotKeyModel
	FinishOnce()
}

// TurnKeyCollector 轮流Key收集器
type TurnKeyCollector struct {
	map0    map[string]*eventbus.HotKeyModel
	map1    map[string]*eventbus.HotKeyModel
	counter int64
	mu      sync.Mutex
}

// NewTurnKeyCollector 创建新的轮流Key收集器
func NewTurnKeyCollector() *TurnKeyCollector {
	return &TurnKeyCollector{
		map0: make(map[string]*eventbus.HotKeyModel),
		map1: make(map[string]*eventbus.HotKeyModel),
	}
}

// Collect 收集热Key
func (tkc *TurnKeyCollector) Collect(hotKeyModel *eventbus.HotKeyModel) {
	key := hotKeyModel.Key
	if key == "" {
		return
	}

	tkc.mu.Lock()
	defer tkc.mu.Unlock()

	if atomic.LoadInt64(&tkc.counter)%2 == 0 {
		// 写入map0
		if model, exists := tkc.map0[key]; exists {
			model.Count += hotKeyModel.Count
		} else {
			tkc.map0[key] = hotKeyModel
		}
	} else {
		// 写入map1
		if model, exists := tkc.map1[key]; exists {
			model.Count += hotKeyModel.Count
		} else {
			tkc.map1[key] = hotKeyModel
		}
	}
}

// LockAndGetResult 锁定并获取结果
func (tkc *TurnKeyCollector) LockAndGetResult() []*eventbus.HotKeyModel {
	tkc.mu.Lock()
	defer tkc.mu.Unlock()

	// 自增计数器
	atomic.AddInt64(&tkc.counter, 1)

	var result []*eventbus.HotKeyModel
	if atomic.LoadInt64(&tkc.counter)%2 == 0 {
		// 返回map1的结果并清空
		result = make([]*eventbus.HotKeyModel, 0, len(tkc.map1))
		for _, model := range tkc.map1 {
			result = append(result, model)
		}
		tkc.map1 = make(map[string]*eventbus.HotKeyModel)
	} else {
		// 返回map0的结果并清空
		result = make([]*eventbus.HotKeyModel, 0, len(tkc.map0))
		for _, model := range tkc.map0 {
			result = append(result, model)
		}
		tkc.map0 = make(map[string]*eventbus.HotKeyModel)
	}

	return result
}

// FinishOnce 完成一次收集
func (tkc *TurnKeyCollector) FinishOnce() {
	// 这里可以添加一些完成后的逻辑
}
