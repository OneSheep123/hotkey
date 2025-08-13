package worker

import (
	"hotkey-client/eventbus"
)

// WorkerChangeSubscriber Worker变化订阅者
type WorkerChangeSubscriber struct{}

// ConnectAll 监听Worker信息变化事件
func (wcs *WorkerChangeSubscriber) ConnectAll(event *eventbus.WorkerInfoChangeEvent) {
	addresses := event.Addresses
	if addresses == nil {
		addresses = []string{}
	}

	holder := GetInstance()
	holder.MergeAndConnectNew(addresses)
}

// ChannelInactive 当连接断开后，删除对应的Worker
func (wcs *WorkerChangeSubscriber) ChannelInactive(event *eventbus.ChannelInactiveEvent) {
	address := event.Address
	// 记录日志
	// logger.Warn("This channel is inactive: %s, trying to remove this connection", address)

	holder := GetInstance()
	holder.DealChannelInactive(address)
}
