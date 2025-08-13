package worker

import (
	"time"
)

// WorkerRetryConnector Worker重连器
type WorkerRetryConnector struct {
	retryInterval time.Duration
	stopChan      chan struct{}
}

// NewWorkerRetryConnector 创建新的Worker重连器
func NewWorkerRetryConnector(retryInterval time.Duration) *WorkerRetryConnector {
	return &WorkerRetryConnector{
		retryInterval: retryInterval,
		stopChan:      make(chan struct{}),
	}
}

// Start 启动重连器
func (wrc *WorkerRetryConnector) Start() {
	go wrc.retryConnectWorkers()
}

// Stop 停止重连器
func (wrc *WorkerRetryConnector) Stop() {
	close(wrc.stopChan)
}

// retryConnectWorkers 重连Worker的定时任务
func (wrc *WorkerRetryConnector) retryConnectWorkers() {
	ticker := time.NewTicker(wrc.retryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			wrc.reConnectWorkers()
		case <-wrc.stopChan:
			return
		}
	}
}

// reConnectWorkers 重连Worker
func (wrc *WorkerRetryConnector) reConnectWorkers() {
	holder := GetInstance()
	nonList := holder.GetNonConnectedWorkers()

	if len(nonList) == 0 {
		return
	}

	// 记录日志
	// logger.Info("Trying to reConnect to these workers: %v", nonList)

	// 这里应该调用NettyClient连接Worker
	// NettyClient.GetInstance().Connect(nonList)
}

// StartDefaultRetryConnector 启动默认的重连器
func StartDefaultRetryConnector() {
	// 默认30秒重连一次
	connector := NewWorkerRetryConnector(30 * time.Second)
	connector.Start()
}
