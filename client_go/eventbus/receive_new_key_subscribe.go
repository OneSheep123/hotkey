package eventbus

// ReceiveNewKeySubscribe 监听有新Key推送事件
type ReceiveNewKeySubscribe struct {
	receiveNewKeyListener ReceiveNewKeyListener
}

// NewReceiveNewKeySubscribe 创建新的接收新Key订阅者
func NewReceiveNewKeySubscribe() *ReceiveNewKeySubscribe {
	return &ReceiveNewKeySubscribe{
		receiveNewKeyListener: &DefaultNewKeyListener{},
	}
}

// NewKeyComing 处理新Key事件
func (rnks *ReceiveNewKeySubscribe) NewKeyComing(event *ReceiveNewKeyEvent) {
	hotKeyModel := event.Model
	if hotKeyModel == nil {
		return
	}

	// 收到新Key推送
	if rnks.receiveNewKeyListener != nil {
		rnks.receiveNewKeyListener.NewKey(hotKeyModel)
	}
}

// ReceiveNewKeyListener 接收新Key监听器接口
type ReceiveNewKeyListener interface {
	NewKey(hotKeyModel *HotKeyModel)
}

// DefaultNewKeyListener 默认新Key监听器
type DefaultNewKeyListener struct{}

// NewKey 处理新Key
func (dnkl *DefaultNewKeyListener) NewKey(hotKeyModel *HotKeyModel) {
	// 这里应该实现新Key的处理逻辑
	// 暂时留空，等待具体实现
}
