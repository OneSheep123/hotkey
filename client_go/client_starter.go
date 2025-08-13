package main

import (
	"hotkey-client/config"
	"hotkey-client/eventbus"
	"hotkey-client/key"
	"hotkey-client/rule"
	"hotkey-client/worker"
	"time"
)

// ClientStarter 客户端启动器
type ClientStarter struct {
	config *config.ClientConfig
}

// NewClientStarter 创建新的客户端启动器
func NewClientStarter(config *config.ClientConfig) *ClientStarter {
	return &ClientStarter{
		config: config,
	}
}

// StartPipeline 启动管道
func (cs *ClientStarter) StartPipeline() error {
	// 设置全局上下文
	ctx := config.GetContext()
	ctx.SetAppName(cs.config.AppName)
	ctx.SetCacheSize(cs.config.CacheSize)

	// 启动推送调度器
	collector := key.GetInstance().GetCollector()
	pusher := key.GetInstance().GetPusher()
	_ = key.StartDefaultPusher(cs.config.PushPeriod, collector, pusher, cs.config.AppName)

	// 启动Worker重连器
	worker.StartDefaultRetryConnector()

	// 注册事件总线
	cs.registerEventBus()

	// 启动etcd监听（这里暂时留空，等待etcd模块实现）
	// cs.startEtcdWatcher()

	return nil
}

// registerEventBus 注册事件总线
func (cs *ClientStarter) registerEventBus() {
	eventBus := eventbus.GetInstance()

	// 注册Worker变化订阅者
	eventBus.Register(&worker.WorkerChangeSubscriber{})

	// 注册热Key订阅者
	eventBus.Register(&eventbus.ReceiveNewKeySubscribe{})

	// 注册规则持有者
	eventBus.Register(rule.GetInstance())
}

// Builder 构建器模式
type Builder struct {
	appName     string
	etcdServers []string
	pushPeriod  time.Duration
	cacheSize   int
}

// NewBuilder 创建新的构建器
func NewBuilder() *Builder {
	return &Builder{}
}

// SetAppName 设置应用名称
func (b *Builder) SetAppName(appName string) *Builder {
	b.appName = appName
	return b
}

// SetEtcdServers 设置etcd服务器地址
func (b *Builder) SetEtcdServers(servers []string) *Builder {
	b.etcdServers = servers
	return b
}

// SetPushPeriod 设置推送间隔
func (b *Builder) SetPushPeriod(period time.Duration) *Builder {
	b.pushPeriod = period
	return b
}

// SetCacheSize 设置缓存大小
func (b *Builder) SetCacheSize(size int) *Builder {
	b.cacheSize = size
	return b
}

// Build 构建客户端启动器
func (b *Builder) Build() *ClientStarter {
	config := &config.ClientConfig{
		AppName:     b.appName,
		EtcdServers: b.etcdServers,
		PushPeriod:  b.pushPeriod,
		CacheSize:   b.cacheSize,
	}

	// 验证配置
	config.Validate()

	return NewClientStarter(config)
}
