package network

import (
	"fmt"
	"time"

	"github.com/jd/platform/hotkey/client-go/event"
	"github.com/jd/platform/hotkey/client-go/log"
	"github.com/jd/platform/hotkey/client-go/model"
)

// WorkerRetryConnector Worker重连器，对应Java的WorkerRetryConnector
type WorkerRetryConnector struct {
	netClient    *NettyClient
	workerHolder *WorkerInfoHolder
	stopChan     chan struct{}
	retryTicker  *time.Ticker
}

// NewWorkerRetryConnector 创建Worker重连器
func NewWorkerRetryConnector(netClient *NettyClient, workerHolder *WorkerInfoHolder) *WorkerRetryConnector {
	return &WorkerRetryConnector{
		netClient:    netClient,
		workerHolder: workerHolder,
		stopChan:     make(chan struct{}),
	}
}

// Start 启动重连器，对应Java的WorkerRetryConnector.retryConnectWorkers
func (wrc *WorkerRetryConnector) Start() {
	wrc.retryTicker = time.NewTicker(model.DefaultWorkerRetryInterval * time.Second)

	go func() {
		for {
			select {
			case <-wrc.retryTicker.C:
				wrc.reConnectWorkers()
			case <-wrc.stopChan:
				return
			}
		}
	}()
}

// Stop 停止重连器
func (wrc *WorkerRetryConnector) Stop() {
	if wrc.retryTicker != nil {
		wrc.retryTicker.Stop()
	}
	close(wrc.stopChan)
}

// reConnectWorkers 重连worker，对应Java的WorkerRetryConnector.reConnectWorkers
func (wrc *WorkerRetryConnector) reConnectWorkers() {
	nonConnectedAddresses := wrc.netClient.GetNonConnectedAddresses()
	if len(nonConnectedAddresses) == 0 {
		return
	}

	log.Info(wrc, fmt.Sprintf("Trying to reconnect to these workers: %v", nonConnectedAddresses))
	wrc.netClient.Connect(nonConnectedAddresses)
}

// WorkerChangeSubscriber Worker变化订阅者，对应Java的WorkerChangeSubscriber
type WorkerChangeSubscriber struct {
	workerHolder *WorkerInfoHolder
}

// NewWorkerChangeSubscriber 创建Worker变化订阅者
func NewWorkerChangeSubscriber(workerHolder *WorkerInfoHolder) *WorkerChangeSubscriber {
	return &WorkerChangeSubscriber{
		workerHolder: workerHolder,
	}
}

// HandleWorkerInfoChange 处理Worker信息变化事件
func (wcs *WorkerChangeSubscriber) HandleWorkerInfoChange(event *WorkerInfoChangeEvent) {
	wcs.workerHolder.MergeAndConnectNew(event.Addresses)
}

// HandleChannelInactive 处理连接断开事件
func (wcs *WorkerChangeSubscriber) HandleChannelInactive(event *ChannelInactiveEvent) {
	wcs.workerHolder.DealChannelInactive(event.Address)
}

// 使用event包中的事件类型
type WorkerInfoChangeEvent = event.WorkerInfoChangeEvent
type ChannelInactiveEvent = event.ChannelInactiveEvent

// NetworkManager 网络管理器，整合所有网络相关组件
type NetworkManager struct {
	netClient        *NettyClient
	workerHolder     *WorkerInfoHolder
	retryConnector   *WorkerRetryConnector
	changeSubscriber *WorkerChangeSubscriber
}

// NewNetworkManager 创建网络管理器
func NewNetworkManager(eventBus *event.EventBus, appName string) *NetworkManager {
	netClient := NewNettyClient(eventBus, appName)
	workerHolder := NewWorkerInfoHolder(netClient)
	retryConnector := NewWorkerRetryConnector(netClient, workerHolder)
	changeSubscriber := NewWorkerChangeSubscriber(workerHolder)

	return &NetworkManager{
		netClient:        netClient,
		workerHolder:     workerHolder,
		retryConnector:   retryConnector,
		changeSubscriber: changeSubscriber,
	}
}

// Start 启动网络管理器
func (nm *NetworkManager) Start() error {
	// 启动网络客户端
	if err := nm.netClient.Start(); err != nil {
		return err
	}

	// 启动重连器
	nm.retryConnector.Start()

	return nil
}

// Stop 停止网络管理器
func (nm *NetworkManager) Stop() error {
	nm.retryConnector.Stop()
	return nm.netClient.Stop()
}

// GetNetClient 获取网络客户端
func (nm *NetworkManager) GetNetClient() *NettyClient {
	return nm.netClient
}

// GetWorkerHolder 获取Worker持有者
func (nm *NetworkManager) GetWorkerHolder() *WorkerInfoHolder {
	return nm.workerHolder
}

// GetChangeSubscriber 获取变化订阅者
func (nm *NetworkManager) GetChangeSubscriber() *WorkerChangeSubscriber {
	return nm.changeSubscriber
}

// SendHotKeys 发送热key数据到Worker，对应Java的NettyKeyPusher.send
func (nm *NetworkManager) SendHotKeys(appName string, hotKeys []*model.HotKeyModel) error {
	if len(hotKeys) == 0 {
		return nil
	}

	// 步骤1: 设置统一的时间戳
	now := time.Now().UnixMilli()

	// 步骤2: 按Worker节点分组热key数据
	connectionGroups := make(map[*Connection][]*model.HotKeyModel)

	// 步骤3: 遍历所有热key，进行负载均衡分发
	for _, hotKey := range hotKeys {
		// 设置创建时间，确保时间窗口的一致性
		hotKey.CreateTime = now

		// 根据key的hash值选择对应的Worker节点
		conn := nm.workerHolder.ChooseConnection(hotKey.Key)
		if conn == nil {
			continue
		}

		// 将当前热key添加到对应Worker的数据列表中
		connectionGroups[conn] = append(connectionGroups[conn], hotKey)
	}

	// 步骤4: 批量发送数据到各个Worker节点
	var lastError error
	successCount := 0
	totalBatches := len(connectionGroups)

	for conn, batch := range connectionGroups {
		// 创建热key消息对象
		msg := model.NewHotKeyMsg(model.RequestNewKey, appName)
		msg.HotKeyModels = batch

		err := nm.netClient.sendMessage(conn, msg)
		if err != nil {
			// 详细的错误处理和日志记录
			log.Error(nm, fmt.Sprintf("Failed to send %d hot keys to worker %s: %v",
				len(batch), conn.Address, err))
			lastError = err

			// 标记连接为不活跃
			conn.SetActive(false)
		} else {
			successCount++
			if log.IsDebugEnabled() {
				log.Debug(nm, fmt.Sprintf("Successfully sent %d hot keys to worker %s",
					len(batch), conn.Address))
			}
		}
	}

	// 记录发送统计
	log.Info(nm, fmt.Sprintf("Hot key batch send completed: %d/%d workers succeeded, total keys: %d",
		successCount, totalBatches, len(hotKeys)))

	return lastError
}

// SendKeyCount 发送访问计数数据到Worker，对应Java的NettyKeyPusher.sendCount
func (nm *NetworkManager) SendKeyCount(appName string, countModels []*model.KeyCountModel) error {
	if len(countModels) == 0 {
		return nil
	}

	// 步骤1: 设置统一的时间戳
	now := time.Now().UnixMilli()

	// 步骤2: 按Worker节点分组计数数据
	connectionGroups := make(map[*Connection][]*model.KeyCountModel)

	// 步骤3: 遍历所有计数数据，进行负载均衡分发
	for _, countModel := range countModels {
		// 设置创建时间，确保统计时间窗口的一致性
		countModel.CreateTime = now

		// 根据规则key选择对应的Worker节点
		conn := nm.workerHolder.ChooseConnection(countModel.RuleKey)
		if conn == nil {
			continue
		}

		// 将当前计数数据添加到对应Worker的数据列表中
		connectionGroups[conn] = append(connectionGroups[conn], countModel)
	}

	// 步骤4: 批量发送计数数据到各个Worker节点
	var lastError error
	successCount := 0
	totalBatches := len(connectionGroups)

	for conn, batch := range connectionGroups {
		// 创建计数消息对象
		msg := model.NewHotKeyMsg(model.RequestHitCount, appName)
		msg.KeyCountModels = batch

		err := nm.netClient.sendMessage(conn, msg)
		if err != nil {
			// 详细的错误处理和日志记录
			log.Error(nm, fmt.Sprintf("Failed to send %d count models to worker %s: %v",
				len(batch), conn.Address, err))
			lastError = err

			// 标记连接为不活跃
			conn.SetActive(false)
		} else {
			successCount++
			if log.IsDebugEnabled() {
				log.Debug(nm, fmt.Sprintf("Successfully sent %d count models to worker %s",
					len(batch), conn.Address))
			}
		}
	}

	// 记录发送统计
	log.Info(nm, fmt.Sprintf("Count batch send completed: %d/%d workers succeeded, total models: %d",
		successCount, totalBatches, len(countModels)))

	return lastError
}

// 实现KeyPusher接口
func (nm *NetworkManager) Send(appName string, hotKeys []*model.HotKeyModel) error {
	return nm.SendHotKeys(appName, hotKeys)
}

func (nm *NetworkManager) SendCount(appName string, countModels []*model.KeyCountModel) error {
	return nm.SendKeyCount(appName, countModels)
}
