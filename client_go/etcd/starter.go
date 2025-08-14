package etcd

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/jd/platform/hotkey/client-go/event"
	hotlog "github.com/jd/platform/hotkey/client-go/log"
	"github.com/jd/platform/hotkey/client-go/model"
)

// Starter Etcd启动器，对应Java的EtcdStarter
type Starter struct {
	configCenter *ConfigCenter
	eventBus     *event.EventBus
	appName      string
	stopChan     chan struct{}
}

// NewStarter 创建新的Etcd启动器
func NewStarter(configCenter *ConfigCenter, eventBus *event.EventBus, appName string) *Starter {
	return &Starter{
		configCenter: configCenter,
		eventBus:     eventBus,
		appName:      appName,
		stopChan:     make(chan struct{}),
	}
}

// Start 启动Etcd相关的监听，对应Java的EtcdStarter.start
func (s *Starter) Start() error {
	// 30秒检测worker变化
	go s.fetchWorkerInfo()

	// 一次性初始化任务，拉到规则之后结束
	go s.fetchRule()

	// 监听规则变化
	go s.startWatchRule()

	// 监听热key事件，只监听手工添加、删除的key
	go s.startWatchHotKey()

	return nil
}

// Stop 停止Etcd启动器
func (s *Starter) Stop() {
	close(s.stopChan)
}

// fetchWorkerInfo 每隔30秒拉取worker信息，对应Java的EtcdStarter.fetchWorkerInfo
func (s *Starter) fetchWorkerInfo() {
	ticker := time.NewTicker(model.DefaultEtcdFetchInterval * time.Second)
	defer ticker.Stop()

	// 立即执行一次
	s.fetch()

	for {
		select {
		case <-ticker.C:
			s.fetch()
		case <-s.stopChan:
			return
		}
	}
}

// fetch 拉取worker信息，对应Java的EtcdStarter.fetch
func (s *Starter) fetch() {
	hotlog.Info(s, "Trying to connect to etcd and fetch worker info")

	addresses, err := s.configCenter.GetWorkerAddresses()
	if err != nil {
		hotlog.Error(s, fmt.Sprintf("Error fetching worker addresses: %v", err))
		return
	}

	if len(addresses) == 0 {
		hotlog.Warn(s, "Very important warn !!! workers ip info is null!!!")
	}

	hotlog.Info(s, fmt.Sprintf("Worker info list is: %v", addresses))

	// 发布worker信息变更事件
	s.notifyWorkerChange(addresses)
}

// notifyWorkerChange 通知worker变化
func (s *Starter) notifyWorkerChange(addresses []string) {
	event := &event.WorkerInfoChangeEvent{
		Addresses: addresses,
	}
	s.eventBus.Publish(event)
}

// fetchRule 拉取规则信息，对应Java的EtcdStarter.fetchRule
func (s *Starter) fetchRule() {
	ticker := time.NewTicker(model.DefaultRuleFetchInterval * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			log.Println("Trying to connect to etcd and fetch rule info")
			success := s.fetchRuleFromEtcd()
			if success {
				// 拉取已存在的热key
				s.fetchExistHotKey()
				return // 成功后退出
			}
		case <-s.stopChan:
			return
		}
	}
}

// fetchRuleFromEtcd 从etcd拉取规则，对应Java的EtcdStarter.fetchRuleFromEtcd
func (s *Starter) fetchRuleFromEtcd() bool {
	rules, err := s.configCenter.GetRules()
	if err != nil {
		log.Printf("Error fetching rules: %v", err)
		return false
	}

	s.notifyRuleChange(rules)
	return true
}

// notifyRuleChange 通知规则变化
func (s *Starter) notifyRuleChange(rules []*model.KeyRule) {
	event := &event.KeyRuleInfoChangeEvent{
		KeyRules: rules,
	}
	s.eventBus.Publish(event)
}

// fetchExistHotKey 拉取已存在的热key，对应Java的EtcdStarter.fetchExistHotKey
func (s *Starter) fetchExistHotKey() {
	log.Println("--- begin fetch exist hotKey from etcd ----")

	hotKeys, err := s.configCenter.GetExistingHotKeys()
	if err != nil {
		log.Printf("Error fetching existing hot keys: %v", err)
		return
	}

	for key, createTime := range hotKeys {
		hotKeyModel := &model.HotKeyModel{
			BaseModel: model.BaseModel{
				Key:        key,
				CreateTime: createTime,
			},
			Remove: false,
		}

		event := &event.ReceiveNewKeyEvent{
			Model: hotKeyModel,
		}
		s.eventBus.Publish(event)
	}
}

// startWatchRule 异步监听rule规则变化，对应Java的EtcdStarter.startWatchRule
func (s *Starter) startWatchRule() {
	log.Println("--- begin watch rule change ----")

	rulePath := model.GetRulePath(s.appName)
	err := s.configCenter.Watch(rulePath, func(event *WatchEvent) {
		log.Printf("Rules info changed. begin to fetch new infos. rule change is %v", event)
		// 全量拉取rule信息
		s.fetchRuleFromEtcd()
	})

	if err != nil {
		log.Printf("Error watching rules: %v", err)
	}
}

// startWatchHotKey 异步开始监听热key变化信息，对应Java的EtcdStarter.startWatchHotKey
func (s *Starter) startWatchHotKey() {
	log.Println("--- begin watch hotKey change ----")

	hotKeyPath := model.GetHotKeyPath(s.appName)
	err := s.configCenter.WatchPrefix(hotKeyPath, func(watchEvent *WatchEvent) {
		s.handleHotKeyEvent(watchEvent)
	})

	if err != nil {
		log.Printf("Error watching hot keys: %v", err)
	}
}

// handleHotKeyEvent 处理热key事件
func (s *Starter) handleHotKeyEvent(watchEvent *WatchEvent) {
	// 提取key名称
	prefix := model.GetHotKeyPath(s.appName) + "/"
	if !strings.HasPrefix(watchEvent.Key, prefix) {
		return
	}

	key := strings.TrimPrefix(watchEvent.Key, prefix)
	if key == "" {
		return
	}

	hotKeyModel := &model.HotKeyModel{
		BaseModel: model.BaseModel{
			Key: key,
		},
	}

	if watchEvent.Type == EventTypeDelete {
		// 删除事件
		hotKeyModel.Remove = true
		log.Printf("Etcd receive delete key: %s", key)
	} else {
		// 新增事件
		hotKeyModel.Remove = false
		value := watchEvent.Value

		log.Printf("Etcd receive new key: %s --value: %s", key, value)

		// 如果这是一个删除指令，就什么也不干
		if value == model.DefaultDeleteValue {
			return
		}

		// 手工创建的value是时间戳
		if timestamp, err := strconv.ParseInt(value, 10, 64); err == nil {
			hotKeyModel.CreateTime = timestamp
		} else {
			hotKeyModel.CreateTime = time.Now().UnixMilli()
		}
	}

	// 发布事件
	event := &event.ReceiveNewKeyEvent{
		Model: hotKeyModel,
	}
	s.eventBus.Publish(event)
}
