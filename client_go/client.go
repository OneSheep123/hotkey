package hotkey

import (
	"fmt"
	"strings"

	"github.com/jd/platform/hotkey/client-go/cache"
	"github.com/jd/platform/hotkey/client-go/collector"
	"github.com/jd/platform/hotkey/client-go/context"
	"github.com/jd/platform/hotkey/client-go/etcd"
	"github.com/jd/platform/hotkey/client-go/event"
	"github.com/jd/platform/hotkey/client-go/log"
	"github.com/jd/platform/hotkey/client-go/model"
	"github.com/jd/platform/hotkey/client-go/network"
	"github.com/jd/platform/hotkey/client-go/rule"
)

// Client HotKey客户端，对应Java的ClientStarter
type Client struct {
	appName          string
	etcdServer       string
	pushPeriod       int64
	cacheSize        int64
	countPeriod      int

	// 核心组件
	eventBus         *event.EventBus
	configCenter     *etcd.ConfigCenter
	etcdStarter      *etcd.Starter
	ruleHolder       *rule.KeyRuleHolder
	cacheManager     *cache.CacheManager
	cacheFactory     *cache.CacheFactory
	networkManager   *network.NetworkManager
	collectorManager *collector.CollectorManager
	hotKeyStore      *HotKeyStore
	newKeyListener   *NewKeyListener

	// 状态
	started bool
}

// ClientBuilder 客户端构建器，对应Java的ClientStarter.Builder
type ClientBuilder struct {
	appName     string
	etcdServer  string
	pushPeriod  int64
	cacheSize   int64
	countPeriod int
}

// NewClientBuilder 创建客户端构建器
func NewClientBuilder() *ClientBuilder {
	return &ClientBuilder{
		pushPeriod:  model.DefaultPushPeriod,
		cacheSize:   model.DefaultCacheSize,
		countPeriod: model.DefaultCountPeriod,
	}
}

// SetAppName 设置应用名称
func (cb *ClientBuilder) SetAppName(appName string) *ClientBuilder {
	cb.appName = appName
	return cb
}

// SetEtcdServer 设置etcd服务器地址
func (cb *ClientBuilder) SetEtcdServer(etcdServer string) *ClientBuilder {
	cb.etcdServer = etcdServer
	return cb
}

// SetPushPeriod 设置推送间隔（毫秒）
func (cb *ClientBuilder) SetPushPeriod(pushPeriod int64) *ClientBuilder {
	cb.pushPeriod = pushPeriod
	return cb
}

// SetCacheSize 设置缓存大小
func (cb *ClientBuilder) SetCacheSize(cacheSize int64) *ClientBuilder {
	cb.cacheSize = cacheSize
	return cb
}

// SetCountPeriod 设置计数推送间隔（秒）
func (cb *ClientBuilder) SetCountPeriod(countPeriod int) *ClientBuilder {
	cb.countPeriod = countPeriod
	return cb
}

// Build 构建客户端
func (cb *ClientBuilder) Build() (*Client, error) {
	if cb.appName == "" {
		return nil, fmt.Errorf("appName is required")
	}
	if cb.etcdServer == "" {
		return nil, fmt.Errorf("etcdServer is required")
	}

	return &Client{
		appName:     cb.appName,
		etcdServer:  cb.etcdServer,
		pushPeriod:  cb.pushPeriod,
		cacheSize:   cb.cacheSize,
		countPeriod: cb.countPeriod,
	}, nil
}

// Start 启动客户端，对应Java的ClientStarter.startPipeline
func (c *Client) Start() error {
	if c.started {
		return fmt.Errorf("client already started")
	}

	log.Info(c, fmt.Sprintf("Starting HotKey client for app: %s", c.appName))

	// 初始化全局上下文
	context.InitializeFromClient(c.appName, c.etcdServer, c.pushPeriod, c.cacheSize, c.countPeriod)

	// 1. 初始化事件总线
	c.eventBus = event.NewEventBus()

	// 2. 初始化etcd配置中心
	endpoints := strings.Split(c.etcdServer, ",")
	configCenter, err := etcd.NewConfigCenter(endpoints, c.appName)
	if err != nil {
		return fmt.Errorf("failed to create config center: %v", err)
	}
	c.configCenter = configCenter

	// 3. 初始化规则持有者
	c.ruleHolder = rule.NewKeyRuleHolder()

	// 4. 初始化缓存管理器和工厂
	c.cacheManager = cache.NewCacheManager()
	c.cacheFactory = cache.NewCacheFactory(c.ruleHolder)

	// 5. 初始化网络管理器
	c.networkManager = network.NewNetworkManager(c.eventBus, c.appName)

	// 6. 初始化收集器管理器
	c.collectorManager = collector.NewCollectorManager(
		c.ruleHolder,
		c.networkManager, // 实现KeyPusher接口
		c.configCenter,   // 实现ConfigCenter接口
		c.appName,
	)

	// 7. 初始化热key存储
	hotKeyPusher := c.collectorManager.GetHotKeyPusher()
	c.hotKeyStore = NewHotKeyStore(c.cacheFactory, c.ruleHolder, c.collectorManager, hotKeyPusher)

	// 8. 初始化新key监听器
	c.newKeyListener = NewNewKeyListener(c.cacheFactory)

	// 9. 注册事件订阅者
	c.registerEventSubscribers()

	// 10. 启动各个组件
	if err := c.startComponents(); err != nil {
		return fmt.Errorf("failed to start components: %v", err)
	}

	// 11. 设置全局热key存储
	SetGlobalHotKeyStore(c.hotKeyStore)

	c.started = true
	log.Info(c, fmt.Sprintf("HotKey client started successfully for app: %s", c.appName))
	return nil
}

// Stop 停止客户端
func (c *Client) Stop() error {
	if !c.started {
		return nil
	}

	log.Info(c, fmt.Sprintf("Stopping HotKey client for app: %s", c.appName))

	// 停止各个组件
	if c.collectorManager != nil {
		c.collectorManager.Stop()
	}
	if c.networkManager != nil {
		c.networkManager.Stop()
	}
	if c.etcdStarter != nil {
		c.etcdStarter.Stop()
	}
	if c.configCenter != nil {
		c.configCenter.Close()
	}

	// 清理全局状态
	SetGlobalHotKeyStore(nil)

	c.started = false
	log.Info(c, fmt.Sprintf("HotKey client stopped for app: %s", c.appName))
	return nil
}

// registerEventSubscribers 注册事件订阅者
func (c *Client) registerEventSubscribers() {
	// 注册规则变化订阅者
	c.eventBus.Subscribe(c.ruleHolder)

	// 注册Worker变化订阅者
	workerSubscriber := c.networkManager.GetChangeSubscriber()
	c.eventBus.Subscribe(workerSubscriber)

	// 注册新key接收订阅者
	newKeySubscriber := event.NewDefaultReceiveNewKeySubscriber(c.newKeyListener)
	c.eventBus.Subscribe(newKeySubscriber)
}

// startComponents 启动各个组件
func (c *Client) startComponents() error {
	// 1. 启动网络管理器
	if err := c.networkManager.Start(); err != nil {
		return fmt.Errorf("failed to start network manager: %v", err)
	}

	// 2. 启动etcd启动器
	c.etcdStarter = etcd.NewStarter(c.configCenter, c.eventBus, c.appName)
	if err := c.etcdStarter.Start(); err != nil {
		return fmt.Errorf("failed to start etcd starter: %v", err)
	}

	// 3. 启动收集器管理器
	c.collectorManager.Start(c.pushPeriod, c.countPeriod)

	return nil
}

// GetAppName 获取应用名称
func (c *Client) GetAppName() string {
	return c.appName
}

// GetRuleHolder 获取规则持有者
func (c *Client) GetRuleHolder() *rule.KeyRuleHolder {
	return c.ruleHolder
}

// GetCacheFactory 获取缓存工厂
func (c *Client) GetCacheFactory() *cache.CacheFactory {
	return c.cacheFactory
}

// GetNetworkManager 获取网络管理器
func (c *Client) GetNetworkManager() *network.NetworkManager {
	return c.networkManager
}

// GetHotKeyStore 获取热key存储
func (c *Client) GetHotKeyStore() *HotKeyStore {
	return c.hotKeyStore
}

// IsStarted 检查是否已启动
func (c *Client) IsStarted() bool {
	return c.started
}

// GetStats 获取客户端统计信息
func (c *Client) GetStats() map[string]interface{} {
	stats := make(map[string]interface{})
	
	stats["appName"] = c.appName
	stats["started"] = c.started
	
	if c.ruleHolder != nil {
		stats["rules"] = len(c.ruleHolder.GetRules())
		stats["cacheStats"] = c.ruleHolder.GetCacheStats()
	}
	
	if c.networkManager != nil {
		connections := c.networkManager.GetNetClient().GetConnections()
		activeCount := 0
		for _, conn := range connections {
			if conn.IsActive() {
				activeCount++
			}
		}
		stats["totalConnections"] = len(connections)
		stats["activeConnections"] = activeCount
	}
	
	return stats
}



// 便利函数

// NewClient 创建新的客户端（使用默认配置）
func NewClient(appName, etcdServer string) (*Client, error) {
	return NewClientBuilder().
		SetAppName(appName).
		SetEtcdServer(etcdServer).
		Build()
}

// StartClient 启动客户端的便利函数
func StartClient(appName, etcdServer string) (*Client, error) {
	client, err := NewClient(appName, etcdServer)
	if err != nil {
		return nil, err
	}
	
	err = client.Start()
	if err != nil {
		return nil, err
	}
	
	return client, nil
}
