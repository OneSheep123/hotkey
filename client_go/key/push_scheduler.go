package key

import (
	"time"
)

// PushSchedulerStarter 推送调度器启动器
type PushSchedulerStarter struct {
	pushPeriod  time.Duration
	countPeriod time.Duration
	stopChan    chan struct{}
	collector   IKeyCollector
	pusher      IKeyPusher
	appName     string
}

// NewPushSchedulerStarter 创建新的推送调度器启动器
func NewPushSchedulerStarter(pushPeriod, countPeriod time.Duration, collector IKeyCollector, pusher IKeyPusher, appName string) *PushSchedulerStarter {
	return &PushSchedulerStarter{
		pushPeriod:  pushPeriod,
		countPeriod: countPeriod,
		stopChan:    make(chan struct{}),
		collector:   collector,
		pusher:      pusher,
		appName:     appName,
	}
}

// StartPusher 启动推送器
func (pss *PushSchedulerStarter) StartPusher() {
	if pss.pushPeriod <= 0 {
		pss.pushPeriod = 500 * time.Millisecond
	}

	go func() {
		ticker := time.NewTicker(pss.pushPeriod)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				pss.pushHotKeys()
			case <-pss.stopChan:
				return
			}
		}
	}()
}

// StartCountPusher 启动计数推送器
func (pss *PushSchedulerStarter) StartCountPusher() {
	if pss.countPeriod <= 0 {
		pss.countPeriod = 10 * time.Second
	}

	go func() {
		ticker := time.NewTicker(pss.countPeriod)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				pss.pushCounts()
			case <-pss.stopChan:
				return
			}
		}
	}()
}

// Stop 停止推送调度器
func (pss *PushSchedulerStarter) Stop() {
	close(pss.stopChan)
}

// pushHotKeys 推送热Keys
func (pss *PushSchedulerStarter) pushHotKeys() {
	// 获取收集器实例
	collector := pss.collector
	if collector == nil {
		return
	}

	// 锁定并获取收集结果
	hotKeyModels := collector.LockAndGetResult()
	if len(hotKeyModels) == 0 {
		return
	}

	// 推送热key数据到Worker节点
	if pss.pusher != nil {
		pss.pusher.Send(pss.appName, hotKeyModels)
	}

	// 标记本次推送完成
	collector.FinishOnce()
}

// pushCounts 推送计数数据
func (pss *PushSchedulerStarter) pushCounts() {
	// 这里应该实现计数数据的推送
	// 暂时留空，等待计数收集器实现
}

// StartDefaultPusher 启动默认推送器
func StartDefaultPusher(pushPeriod time.Duration, collector IKeyCollector, pusher IKeyPusher, appName string) *PushSchedulerStarter {
	scheduler := NewPushSchedulerStarter(pushPeriod, 10*time.Second, collector, pusher, appName)
	scheduler.StartPusher()
	scheduler.StartCountPusher()
	return scheduler
}
