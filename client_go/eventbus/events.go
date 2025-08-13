package eventbus

import (
	"time"
)

// WorkerInfoChangeEvent Worker信息变化事件
type WorkerInfoChangeEvent struct {
	Addresses []string `json:"addresses"`
}

func (e *WorkerInfoChangeEvent) Type() string {
	return "WorkerInfoChangeEvent"
}

// ReceiveNewKeyEvent 接收到新Key事件
type ReceiveNewKeyEvent struct {
	Model *HotKeyModel `json:"model"`
}

func (e *ReceiveNewKeyEvent) Type() string {
	return "ReceiveNewKeyEvent"
}

// KeyRuleInfoChangeEvent Key规则信息变化事件
type KeyRuleInfoChangeEvent struct {
	KeyRules []*KeyRule `json:"keyRules"`
}

func (e *KeyRuleInfoChangeEvent) Type() string {
	return "KeyRuleInfoChangeEvent"
}

// ChannelInactiveEvent 连接断开事件
type ChannelInactiveEvent struct {
	Address string `json:"address"`
}

func (e *ChannelInactiveEvent) Type() string {
	return "ChannelInactiveEvent"
}

// HotKeyModel 热Key模型
type HotKeyModel struct {
	AppName    string    `json:"appName"`
	Key        string    `json:"key"`
	KeyType    string    `json:"keyType"`
	Count      int64     `json:"count"`
	Remove     bool      `json:"remove"`
	CreateTime time.Time `json:"createTime"`
}

// KeyRule Key规则
type KeyRule struct {
	Key       string `json:"key"`
	Prefix    bool   `json:"prefix"`
	Duration  int    `json:"duration"`
	Interval  int    `json:"interval"`
	Threshold int    `json:"threshold"`
}
