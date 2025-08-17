package collector

import (
	"log"
	"time"

	"github.com/jd/platform/hotkey/client-go/model"
)

// PushSchedulerStarter 推送调度器启动器，对应Java的PushSchedulerStarter
type PushSchedulerStarter struct {
	handlerFactory *KeyHandlerFactory
	appName        string
	stopChan       chan struct{}
	keyTicker      *time.Ticker
	countTicker    *time.Ticker
}

// NewPushSchedulerStarter 创建推送调度器启动器
func NewPushSchedulerStarter(handlerFactory *KeyHandlerFactory, appName string) *PushSchedulerStarter {
	return &PushSchedulerStarter{
		handlerFactory: handlerFactory,
		appName:        appName,
		stopChan:       make(chan struct{}),
	}
}

// StartPusher 启动Key推送器，对应Java的PushSchedulerStarter.startPusher
func (pss *PushSchedulerStarter) StartPusher(period int64) {
	if period <= 0 {
		period = model.DefaultPushPeriod
	}

	pss.keyTicker = time.NewTicker(time.Duration(period) * time.Millisecond)

	go func() {
		for {
			select {
			case <-pss.keyTicker.C:
				pss.pushHotKeys()
			case <-pss.stopChan:
				return
			}
		}
	}()
}

// StartCountPusher 启动计数推送器，对应Java的PushSchedulerStarter.startCountPusher
func (pss *PushSchedulerStarter) StartCountPusher(period int) {
	if period <= 0 {
		period = model.DefaultCountPeriod
	}

	pss.countTicker = time.NewTicker(time.Duration(period) * time.Second)

	go func() {
		for {
			select {
			case <-pss.countTicker.C:
				pss.pushKeyCount()
			case <-pss.stopChan:
				return
			}
		}
	}()
}

// Stop 停止推送调度器
func (pss *PushSchedulerStarter) Stop() {
	close(pss.stopChan)
	if pss.keyTicker != nil {
		pss.keyTicker.Stop()
	}
	if pss.countTicker != nil {
		pss.countTicker.Stop()
	}
}

// pushHotKeys 推送热key，对应Java的PushSchedulerStarter中的热key推送逻辑
func (pss *PushSchedulerStarter) pushHotKeys() {
	// 步骤1: 获取热key收集器实例
	collector := pss.handlerFactory.GetCollector()

	// 步骤2: 锁定并获取收集结果
	result := collector.LockAndGetResult()
	hotKeyModels, ok := result.([]*model.HotKeyModel)
	if !ok || len(hotKeyModels) == 0 {
		return
	}

	// 步骤3: 推送热key数据到Worker节点
	pusher := pss.handlerFactory.GetPusher()
	err := pusher.Send(pss.appName, hotKeyModels)
	if err != nil {
		log.Printf("Failed to send hot keys: %v", err)
		return
	}

	// 步骤4: 标记本次推送完成
	collector.FinishOnce()
}

// pushKeyCount 推送访问计数，对应Java的PushSchedulerStarter中的计数推送逻辑
func (pss *PushSchedulerStarter) pushKeyCount() {
	// 步骤1: 获取计数收集器实例
	counter := pss.handlerFactory.GetCounter()

	// 步骤2: 锁定并获取收集结果
	result := counter.LockAndGetResult()
	keyCountModels, ok := result.([]*model.KeyCountModel)
	if !ok || len(keyCountModels) == 0 {
		return
	}

	// 步骤3: 推送计数数据到Worker节点
	pusher := pss.handlerFactory.GetPusher()
	err := pusher.SendCount(pss.appName, keyCountModels)
	if err != nil {
		log.Printf("Failed to send key count: %v", err)
		return
	}

	// 步骤4: 标记本次推送完成
	counter.FinishOnce()
}

// HotKeyPusher 热key推送器，对应Java的HotKeyPusher
type HotKeyPusher struct {
	handlerFactory *KeyHandlerFactory
	configCenter   ConfigCenter
	appName        string
}

// ConfigCenter 配置中心接口
type ConfigCenter interface {
	PutWithTTL(key, value string, ttlSeconds int64) error
	Delete(key string) error
}

// NewHotKeyPusher 创建热key推送器
func NewHotKeyPusher(handlerFactory *KeyHandlerFactory, configCenter ConfigCenter, appName string) *HotKeyPusher {
	return &HotKeyPusher{
		handlerFactory: handlerFactory,
		configCenter:   configCenter,
		appName:        appName,
	}
}

// Push 推送热key，对应Java的HotKeyPusher.push
func (hkp *HotKeyPusher) Push(key string, keyType model.KeyType, count int, remove bool) {
	if count <= 0 {
		count = 1
	}
	if key == "" {
		return
	}

	hotKeyModel := model.NewHotKeyModel(key, hkp.appName, keyType)
	hotKeyModel.SetCount(int64(count))
	hotKeyModel.Remove = remove

	if remove {
		// 如果是删除key，就直接发到etcd去，不用做聚合
		hkp.removeFromEtcd(key)
	} else {
		// 如果key是规则内的要被探测的key，就积累等待传送
		if hkp.isKeyInRule(key) {
			collector := hkp.handlerFactory.GetCollector()
			collector.Collect(hotKeyModel)
		}
	}
}

// PushWithDefaults 使用默认参数推送
func (hkp *HotKeyPusher) PushWithDefaults(key string) {
	hkp.Push(key, model.RedisKey, 1, false)
}

// Remove 删除热key
func (hkp *HotKeyPusher) Remove(key string) {
	hkp.Push(key, model.RedisKey, 1, true)
}

// removeFromEtcd 从etcd删除热key，对应Java的删除逻辑
func (hkp *HotKeyPusher) removeFromEtcd(key string) {
	keyPath := model.GetKeyPath(hkp.appName, key)
	recordPath := model.GetKeyRecordPath(hkp.appName, key)

	// 先设置删除标记（TTL=1秒）
	err := hkp.configCenter.PutWithTTL(keyPath, model.DefaultDeleteValue, 1)
	if err != nil {
		log.Printf("Failed to put delete marker for key %s: %v", key, err)
	}

	// 删除key
	err = hkp.configCenter.Delete(keyPath)
	if err != nil {
		log.Printf("Failed to delete key %s: %v", key, err)
	}

	// 删除记录
	err = hkp.configCenter.Delete(recordPath)
	if err != nil {
		log.Printf("Failed to delete record for key %s: %v", key, err)
	}
}

// isKeyInRule 检查key是否在规则内（这里需要依赖规则持有者）
func (hkp *HotKeyPusher) isKeyInRule(key string) bool {
	// 这里需要访问规则持有者，暂时返回true
	// 在实际实现中，应该注入RuleHolder并调用其IsKeyInRule方法
	return true
}

// CollectorManager 收集器管理器，整合所有收集相关组件
type CollectorManager struct {
	handlerFactory *KeyHandlerFactory
	scheduler      *PushSchedulerStarter
	hotKeyPusher   *HotKeyPusher
}

// NewCollectorManager 创建收集器管理器
func NewCollectorManager(ruleHolder CollectorRuleHolder, keyPusher KeyPusher, configCenter ConfigCenter, appName string) *CollectorManager {
	handlerFactory := NewKeyHandlerFactory(ruleHolder, keyPusher)
	scheduler := NewPushSchedulerStarter(handlerFactory, appName)
	hotKeyPusher := NewHotKeyPusher(handlerFactory, configCenter, appName)

	return &CollectorManager{
		handlerFactory: handlerFactory,
		scheduler:      scheduler,
		hotKeyPusher:   hotKeyPusher,
	}
}

// Start 启动收集器管理器
func (cm *CollectorManager) Start(pushPeriod int64, countPeriod int) {
	cm.scheduler.StartPusher(pushPeriod)
	cm.scheduler.StartCountPusher(countPeriod)
}

// Stop 停止收集器管理器
func (cm *CollectorManager) Stop() {
	cm.scheduler.Stop()
}

// GetHandlerFactory 获取处理器工厂
func (cm *CollectorManager) GetHandlerFactory() *KeyHandlerFactory {
	return cm.handlerFactory
}

// GetHotKeyPusher 获取热key推送器
func (cm *CollectorManager) GetHotKeyPusher() *HotKeyPusher {
	return cm.hotKeyPusher
}

// CollectKey 收集key
func (cm *CollectorManager) CollectKey(hotKeyModel *model.HotKeyModel) {
	cm.handlerFactory.GetCollector().Collect(hotKeyModel)
}

// CollectKeyHot 收集key访问统计
func (cm *CollectorManager) CollectKeyHot(keyHotModel *model.KeyHotModel) {
	cm.handlerFactory.GetCounter().Collect(keyHotModel)
}
