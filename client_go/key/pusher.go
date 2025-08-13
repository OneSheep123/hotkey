package key

import (
	"hotkey-client/eventbus"
)

// IKeyPusher Key推送器接口
type IKeyPusher interface {
	Send(appName string, list []*eventbus.HotKeyModel)
	SendCount(appName string, list []interface{})
}

// NettyKeyPusher Netty网络推送器实现
type NettyKeyPusher struct{}

// NewNettyKeyPusher 创建新的Netty推送器
func NewNettyKeyPusher() *NettyKeyPusher {
	return &NettyKeyPusher{}
}

// Send 发送热Key数据
func (nkp *NettyKeyPusher) Send(appName string, list []*eventbus.HotKeyModel) {
	// 这里应该实现通过Netty发送数据到Worker的逻辑
	// 暂时留空，等待Netty模块实现
}

// SendCount 发送计数数据
func (nkp *NettyKeyPusher) SendCount(appName string, list []interface{}) {
	// 这里应该实现通过Netty发送计数数据到Worker的逻辑
	// 暂时留空，等待Netty模块实现
}

// DefaultKeyHandler 默认Key处理器
type DefaultKeyHandler struct {
	keyPusher  IKeyPusher
	collector  IKeyCollector
	keyCounter IKeyCollector
}

// NewDefaultKeyHandler 创建新的默认Key处理器
func NewDefaultKeyHandler() *DefaultKeyHandler {
	return &DefaultKeyHandler{
		keyPusher:  NewNettyKeyPusher(),
		collector:  NewTurnKeyCollector(),
		keyCounter: NewTurnKeyCollector(), // 这里应该使用专门的计数收集器
	}
}

// KeyPusher 获取Key推送器
func (dkh *DefaultKeyHandler) KeyPusher() IKeyPusher {
	return dkh.keyPusher
}

// KeyCollector 获取Key收集器
func (dkh *DefaultKeyHandler) KeyCollector() IKeyCollector {
	return dkh.collector
}

// KeyCounter 获取Key计数器
func (dkh *DefaultKeyHandler) KeyCounter() IKeyCollector {
	return dkh.keyCounter
}

// KeyHandlerFactory Key处理器工厂
type KeyHandlerFactory struct {
	handler *DefaultKeyHandler
}

var defaultFactory = &KeyHandlerFactory{
	handler: NewDefaultKeyHandler(),
}

// GetInstance 获取默认实例
func GetInstance() *KeyHandlerFactory {
	return defaultFactory
}

// GetPusher 获取推送器
func (khf *KeyHandlerFactory) GetPusher() IKeyPusher {
	return khf.handler.KeyPusher()
}

// GetCollector 获取收集器
func (khf *KeyHandlerFactory) GetCollector() IKeyCollector {
	return khf.handler.KeyCollector()
}

// GetCounter 获取计数器
func (khf *KeyHandlerFactory) GetCounter() IKeyCollector {
	return khf.handler.KeyCounter()
}
