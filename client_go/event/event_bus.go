package event

import (
	"reflect"
	"sync"

	"github.com/jd/platform/hotkey/client-go/model"
)

// EventBus 事件总线，对应Java的EventBusCenter
type EventBus struct {
	subscribers map[reflect.Type][]interface{}
	mutex       sync.RWMutex
}

// NewEventBus 创建新的事件总线
func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[reflect.Type][]interface{}),
	}
}

// Subscribe 订阅事件，对应Java的EventBusCenter.register
func (eb *EventBus) Subscribe(subscriber interface{}) {
	eb.mutex.Lock()
	defer eb.mutex.Unlock()

	subscriberType := reflect.TypeOf(subscriber)
	subscriberValue := reflect.ValueOf(subscriber)

	// 查找所有以"Handle"开头的方法
	for i := 0; i < subscriberType.NumMethod(); i++ {
		method := subscriberType.Method(i)
		methodType := method.Type

		// 检查方法签名：func (receiver) HandleXxx(event EventType)
		if methodType.NumIn() == 2 && methodType.NumOut() == 0 {
			eventType := methodType.In(1)
			if eventType.Kind() == reflect.Ptr {
				eventType = eventType.Elem()
			}

			// 创建处理器函数
			handler := func(event interface{}) {
				eventValue := reflect.ValueOf(event)
				if eventValue.Type().AssignableTo(methodType.In(1)) {
					subscriberValue.Method(i).Call([]reflect.Value{eventValue})
				}
			}

			eb.subscribers[eventType] = append(eb.subscribers[eventType], handler)
		}
	}
}

// Unsubscribe 取消订阅，对应Java的EventBusCenter.unregister
func (eb *EventBus) Unsubscribe(subscriber interface{}) {
	eb.mutex.Lock()
	defer eb.mutex.Unlock()

	subscriberValue := reflect.ValueOf(subscriber)

	for eventType, handlers := range eb.subscribers {
		var newHandlers []interface{}
		for _, handler := range handlers {
			handlerValue := reflect.ValueOf(handler)
			if handlerValue.Pointer() != subscriberValue.Pointer() {
				newHandlers = append(newHandlers, handler)
			}
		}
		eb.subscribers[eventType] = newHandlers
	}
}

// Publish 发布事件，对应Java的EventBusCenter.post
func (eb *EventBus) Publish(event interface{}) {
	eb.mutex.RLock()
	defer eb.mutex.RUnlock()

	eventType := reflect.TypeOf(event)
	if eventType.Kind() == reflect.Ptr {
		eventType = eventType.Elem()
	}

	if handlers, exists := eb.subscribers[eventType]; exists {
		for _, handler := range handlers {
			if handlerFunc, ok := handler.(func(interface{})); ok {
				go handlerFunc(event) // 异步处理
			}
		}
	}
}

// 事件定义

// WorkerInfoChangeEvent Worker信息变化事件，对应Java的WorkerInfoChangeEvent
type WorkerInfoChangeEvent struct {
	Addresses []string
}

// ReceiveNewKeyEvent 接收新key事件，对应Java的ReceiveNewKeyEvent
type ReceiveNewKeyEvent struct {
	Model *model.HotKeyModel
}

// KeyRuleInfoChangeEvent Key规则信息变化事件，对应Java的KeyRuleInfoChangeEvent
type KeyRuleInfoChangeEvent struct {
	KeyRules []*model.KeyRule
}

// ChannelInactiveEvent 连接断开事件，对应Java的ChannelInactiveEvent
type ChannelInactiveEvent struct {
	Address string
}

// 事件订阅者接口

// WorkerChangeSubscriber Worker变化订阅者，对应Java的WorkerChangeSubscriber
type WorkerChangeSubscriber interface {
	HandleWorkerInfoChange(event *WorkerInfoChangeEvent)
}

// ReceiveNewKeySubscriber 新key接收订阅者，对应Java的ReceiveNewKeySubscribe
type ReceiveNewKeySubscriber interface {
	HandleReceiveNewKey(event *ReceiveNewKeyEvent)
}

// KeyRuleSubscriber 规则变化订阅者，对应Java的KeyRuleHolder
type KeyRuleSubscriber interface {
	HandleKeyRuleInfoChange(event *KeyRuleInfoChangeEvent)
}

// ChannelInactiveSubscriber 连接断开订阅者
type ChannelInactiveSubscriber interface {
	HandleChannelInactive(event *ChannelInactiveEvent)
}

// 默认事件处理器实现

// DefaultWorkerChangeSubscriber 默认Worker变化处理器
type DefaultWorkerChangeSubscriber struct {
	workerManager WorkerManager
}

// WorkerManager Worker管理器接口
type WorkerManager interface {
	MergeAndConnectNew(addresses []string)
	DealChannelInactive(address string)
}

// NewDefaultWorkerChangeSubscriber 创建默认Worker变化处理器
func NewDefaultWorkerChangeSubscriber(workerManager WorkerManager) *DefaultWorkerChangeSubscriber {
	return &DefaultWorkerChangeSubscriber{
		workerManager: workerManager,
	}
}

// HandleWorkerInfoChange 处理Worker信息变化
func (s *DefaultWorkerChangeSubscriber) HandleWorkerInfoChange(event *WorkerInfoChangeEvent) {
	s.workerManager.MergeAndConnectNew(event.Addresses)
}

// HandleChannelInactive 处理连接断开
func (s *DefaultWorkerChangeSubscriber) HandleChannelInactive(event *ChannelInactiveEvent) {
	s.workerManager.DealChannelInactive(event.Address)
}

// DefaultReceiveNewKeySubscriber 默认新key接收处理器
type DefaultReceiveNewKeySubscriber struct {
	keyListener NewKeyListener
}

// NewKeyListener 新key监听器接口
type NewKeyListener interface {
	NewKey(hotKeyModel *model.HotKeyModel)
}

// NewDefaultReceiveNewKeySubscriber 创建默认新key接收处理器
func NewDefaultReceiveNewKeySubscriber(keyListener NewKeyListener) *DefaultReceiveNewKeySubscriber {
	return &DefaultReceiveNewKeySubscriber{
		keyListener: keyListener,
	}
}

// HandleReceiveNewKey 处理接收新key事件
func (s *DefaultReceiveNewKeySubscriber) HandleReceiveNewKey(event *ReceiveNewKeyEvent) {
	if event.Model != nil && s.keyListener != nil {
		s.keyListener.NewKey(event.Model)
	}
}

// DefaultKeyRuleSubscriber 默认规则变化处理器
type DefaultKeyRuleSubscriber struct {
	ruleHolder RuleHolder
}

// RuleHolder 规则持有者接口
type RuleHolder interface {
	PutRules(rules []*model.KeyRule)
}

// NewDefaultKeyRuleSubscriber 创建默认规则变化处理器
func NewDefaultKeyRuleSubscriber(ruleHolder RuleHolder) *DefaultKeyRuleSubscriber {
	return &DefaultKeyRuleSubscriber{
		ruleHolder: ruleHolder,
	}
}

// HandleKeyRuleInfoChange 处理规则信息变化
func (s *DefaultKeyRuleSubscriber) HandleKeyRuleInfoChange(event *KeyRuleInfoChangeEvent) {
	if s.ruleHolder != nil {
		s.ruleHolder.PutRules(event.KeyRules)
	}
}

// EventBusCenter 全局事件总线中心，对应Java的EventBusCenter
var EventBusCenter = NewEventBus()

// Register 注册订阅者到全局事件总线
func Register(subscriber interface{}) {
	EventBusCenter.Subscribe(subscriber)
}

// Unregister 从全局事件总线取消注册订阅者
func Unregister(subscriber interface{}) {
	EventBusCenter.Unsubscribe(subscriber)
}

// Post 发布事件到全局事件总线
func Post(event interface{}) {
	EventBusCenter.Publish(event)
}
