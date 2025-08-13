package eventbus

import (
	"reflect"
	"sync"
)

// Event 事件接口
type Event interface {
	Type() string
}

// EventBus 事件总线
type EventBus struct {
	subscribers map[string][]interface{}
	mu          sync.RWMutex
}

var defaultEventBus = NewEventBus()

// NewEventBus 创建新的事件总线
func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[string][]interface{}),
	}
}

// GetInstance 获取默认事件总线实例
func GetInstance() *EventBus {
	return defaultEventBus
}

// Register 注册事件订阅者
func (eb *EventBus) Register(subscriber interface{}) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	// 使用反射获取订阅者的方法
	subscriberType := reflect.TypeOf(subscriber)
	for i := 0; i < subscriberType.NumMethod(); i++ {
		method := subscriberType.Method(i)
		// 检查方法是否有正确的签名
		if method.Type.NumIn() == 2 && method.Type.NumOut() == 0 {
			// 第一个参数是接收者，第二个参数是事件
			eventType := method.Type.In(1)
			if eventType.Implements(reflect.TypeOf((*Event)(nil)).Elem()) {
				eventName := eventType.Name()
				eb.subscribers[eventName] = append(eb.subscribers[eventName], subscriber)
			}
		}
	}
}

// Unregister 取消注册事件订阅者
func (eb *EventBus) Unregister(subscriber interface{}) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	subscriberType := reflect.TypeOf(subscriber)
	for i := 0; i < subscriberType.NumMethod(); i++ {
		method := subscriberType.Method(i)
		if method.Type.NumIn() == 2 && method.Type.NumOut() == 0 {
			eventType := method.Type.In(1)
			if eventType.Implements(reflect.TypeOf((*Event)(nil)).Elem()) {
				eventName := eventType.Name()
				if subscribers, exists := eb.subscribers[eventName]; exists {
					for j, sub := range subscribers {
						if sub == subscriber {
							eb.subscribers[eventName] = append(subscribers[:j], subscribers[j+1:]...)
							break
						}
					}
				}
			}
		}
	}
}

// Post 发布事件
func (eb *EventBus) Post(event Event) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	eventType := reflect.TypeOf(event).Name()
	if subscribers, exists := eb.subscribers[eventType]; exists {
		for _, subscriber := range subscribers {
			// 异步处理事件
			go eb.handleEvent(subscriber, event)
		}
	}
}

// handleEvent 处理单个事件
func (eb *EventBus) handleEvent(subscriber interface{}, event Event) {
	defer func() {
		if r := recover(); r != nil {
			// 记录错误日志
			// logger.Error("Event handling panic: %v", r)
		}
	}()

	subscriberValue := reflect.ValueOf(subscriber)
	eventValue := reflect.ValueOf(event)

	// 查找匹配的方法
	subscriberType := subscriberValue.Type()
	for i := 0; i < subscriberType.NumMethod(); i++ {
		method := subscriberType.Method(i)
		if method.Type.NumIn() == 2 && method.Type.NumOut() == 0 {
			eventType := method.Type.In(1)
			if eventType == eventValue.Type() {
				method.Func.Call([]reflect.Value{subscriberValue, eventValue})
				break
			}
		}
	}
}

// GetSubscriberCount 获取订阅者数量
func (eb *EventBus) GetSubscriberCount(eventType string) int {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	if subscribers, exists := eb.subscribers[eventType]; exists {
		return len(subscribers)
	}
	return 0
}
