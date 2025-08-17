package network

import (
	"bytes"
	"fmt"
	"net"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jd/platform/hotkey/client-go/event"
	hotlog "github.com/jd/platform/hotkey/client-go/log"
	"github.com/jd/platform/hotkey/client-go/model"
	"github.com/jd/platform/hotkey/client-go/serializer"
)

// Connection 连接信息
type Connection struct {
	Address string
	Conn    net.Conn
	Active  bool
	mutex   sync.RWMutex
}

// IsActive 检查连接是否活跃
func (c *Connection) IsActive() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.Active && c.Conn != nil
}

// SetActive 设置连接状态
func (c *Connection) SetActive(active bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.Active = active
}

// Close 关闭连接
func (c *Connection) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.Active = false
	if c.Conn != nil {
		return c.Conn.Close()
	}
	return nil
}

// NettyClient 网络客户端，对应Java的NettyClient
type NettyClient struct {
	connections     map[string]*Connection
	serializer      *serializer.ProtostuffSerializer
	eventBus        *event.EventBus
	appName         string
	mutex           sync.RWMutex
	stopChan        chan struct{}
	heartbeatTicker *time.Ticker
}

// NewNettyClient 创建新的网络客户端
func NewNettyClient(eventBus *event.EventBus, appName string) *NettyClient {
	return &NettyClient{
		connections: make(map[string]*Connection),
		serializer:  serializer.NewProtostuffSerializer(),
		eventBus:    eventBus,
		appName:     appName,
		stopChan:    make(chan struct{}),
	}
}

// Start 启动客户端
func (nc *NettyClient) Start() error {
	// 启动心跳
	nc.startHeartbeat()
	return nil
}

// Stop 停止客户端
func (nc *NettyClient) Stop() error {
	close(nc.stopChan)
	if nc.heartbeatTicker != nil {
		nc.heartbeatTicker.Stop()
	}

	nc.mutex.Lock()
	defer nc.mutex.Unlock()

	for _, conn := range nc.connections {
		conn.Close()
	}
	nc.connections = make(map[string]*Connection)
	return nil
}

// Connect 连接到指定地址列表，对应Java的NettyClient.connect
func (nc *NettyClient) Connect(addresses []string) bool {
	allSuccess := true

	for _, address := range addresses {
		if nc.hasConnected(address) {
			continue
		}

		parts := strings.Split(address, ":")
		if len(parts) != 2 {
			hotlog.Errorf(nc, "Invalid address format: %s", address)
			allSuccess = false
			continue
		}

		host := parts[0]
		port, err := strconv.Atoi(parts[1])
		if err != nil {
			hotlog.Errorf(nc, "Invalid port in address: %s", address)
			allSuccess = false
			continue
		}

		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 5*time.Second)
		if err != nil {
			hotlog.Error(nc, fmt.Sprintf("Failed to connect to worker: %s, error: %v", address, err))
			nc.putConnection(address, nil)
			allSuccess = false
			continue
		}

		// 设置TCP连接选项，与Java版本保持一致
		if tcpConn, ok := conn.(*net.TCPConn); ok {
			tcpConn.SetKeepAlive(true)
			tcpConn.SetNoDelay(true)
		}

		connection := &Connection{
			Address: address,
			Conn:    conn,
			Active:  true,
		}

		nc.putConnection(address, connection)

		// 启动连接处理，AppName将在连接处理器中发送
		go nc.handleConnection(connection)
	}

	return allSuccess
}

// hasConnected 检查是否已连接到指定地址
func (nc *NettyClient) hasConnected(address string) bool {
	nc.mutex.RLock()
	defer nc.mutex.RUnlock()

	if conn, exists := nc.connections[address]; exists {
		return conn.IsActive()
	}
	return false
}

// putConnection 添加或更新连接
func (nc *NettyClient) putConnection(address string, connection *Connection) {
	nc.mutex.Lock()
	defer nc.mutex.Unlock()

	if connection == nil {
		// 标记为未连接
		nc.connections[address] = &Connection{
			Address: address,
			Conn:    nil,
			Active:  false,
		}
	} else {
		nc.connections[address] = connection
	}
}

// handleConnection 处理连接，对应Java的NettyClientHandler
func (nc *NettyClient) handleConnection(connection *Connection) {
	hotlog.Info(nc, fmt.Sprintf("Starting connection handler for %s", connection.Address))

	defer func() {
		hotlog.Info(nc, fmt.Sprintf("Connection handler for %s is closing", connection.Address))
		connection.Close()
		// 发布连接断开事件
		nc.eventBus.Publish(&event.ChannelInactiveEvent{
			Address: connection.Address,
		})
	}()

	// 等待连接完全建立，然后发送AppName，对应Java的channelActive
	time.Sleep(100 * time.Millisecond) // 确保连接完全建立
	hotlog.Debug(nc, fmt.Sprintf("Sending app name to %s", connection.Address))
	err := nc.sendAppName(connection)
	if err != nil {
		hotlog.Error(nc, fmt.Sprintf("Failed to send app name to %s: %v", connection.Address, err))
		return
	} else {
		hotlog.Debug(nc, fmt.Sprintf("Successfully sent app name to %s", connection.Address))
	}

	buffer := &bytes.Buffer{}

	for {
		select {
		case <-nc.stopChan:
			return
		default:
			// 设置读取超时为1秒，更频繁地检查停止信号
			connection.Conn.SetReadDeadline(time.Now().Add(1 * time.Second))

			// 读取数据
			data := make([]byte, 4096) // 增加缓冲区大小
			n, err := connection.Conn.Read(data)
			if err != nil {
				// 检查是否是超时错误
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					// 超时不是致命错误，继续循环
					continue
				}
				hotlog.Error(nc, fmt.Sprintf("Connection read error from %s: %v", connection.Address, err))
				return
			}

			if n > 0 {
				hotlog.Debug(nc, fmt.Sprintf("Received %d bytes from %s", n, connection.Address))
				buffer.Write(data[:n])
				// 处理完整的消息
				nc.processMessages(buffer, connection)
			}
		}
	}
}

// processMessages 处理消息
func (nc *NettyClient) processMessages(buffer *bytes.Buffer, connection *Connection) {
	delimiter := []byte(model.Delimiter)

	for {
		data := buffer.Bytes()
		index := bytes.Index(data, delimiter)
		if index == -1 {
			// 没有完整的消息
			break
		}

		// 提取完整的消息
		messageData := data[:index]
		buffer.Next(index + len(delimiter))

		// 处理消息
		nc.handleMessage(messageData, connection)
	}
}

// handleMessage 处理单个消息
func (nc *NettyClient) handleMessage(data []byte, connection *Connection) {
	hotlog.Debug(nc, fmt.Sprintf("Received message from %s, data length: %d", connection.Address, len(data)))

	msg, err := nc.serializer.Deserialize(data, reflect.TypeOf(&model.HotKeyMsg{}))
	if err != nil {
		hotlog.Error(nc, fmt.Sprintf("Failed to deserialize message from %s: %v, data: %s", connection.Address, err, string(data)))
		return
	}

	hotKeyMsg, ok := msg.(*model.HotKeyMsg)
	if !ok {
		hotlog.Error(nc, fmt.Sprintf("Invalid message type from %s", connection.Address))
		return
	}

	hotlog.Debug(nc, fmt.Sprintf("Received message type %v from %s", hotKeyMsg.MessageType, connection.Address))

	switch hotKeyMsg.MessageType {
	case model.Pong:
		hotlog.Debug(nc, fmt.Sprintf("Received heartbeat pong from %s", connection.Address))

	case model.ResponseNewKey:
		hotlog.Info(nc, fmt.Sprintf("Received new key response from %s with %d keys",
			connection.Address, len(hotKeyMsg.HotKeyModels)))
		if len(hotKeyMsg.HotKeyModels) > 0 {
			for _, hotKeyModel := range hotKeyMsg.HotKeyModels {
				hotlog.Info(nc, fmt.Sprintf("Processing hot key: %s, Remove: %t", hotKeyModel.Key, hotKeyModel.Remove))
				nc.eventBus.Publish(&event.ReceiveNewKeyEvent{
					Model: hotKeyModel,
				})
			}
		}

	default:
		hotlog.Warn(nc, fmt.Sprintf("Unknown message type %v from %s", hotKeyMsg.MessageType, connection.Address))
	}
}

// sendAppName 发送应用名称
func (nc *NettyClient) sendAppName(connection *Connection) error {
	msg := model.NewHotKeyMsg(model.AppName, nc.appName)
	return nc.sendMessage(connection, msg)
}

// sendMessage 发送消息
func (nc *NettyClient) sendMessage(connection *Connection, msg *model.HotKeyMsg) error {
	data, err := nc.serializer.EncodeWithDelimiter(msg)
	if err != nil {
		return err
	}

	_, err = connection.Conn.Write(data)
	if err != nil {
		return err
	}

	// 确保数据立即发送，对应Java的writeAndFlush
	if tcpConn, ok := connection.Conn.(*net.TCPConn); ok {
		// TCP连接会自动刷新，但我们可以设置NoDelay确保立即发送
		tcpConn.SetNoDelay(true)
	}

	return nil
}

// startHeartbeat 启动心跳
func (nc *NettyClient) startHeartbeat() {
	nc.heartbeatTicker = time.NewTicker(model.DefaultHeartbeatInterval * time.Second)

	go func() {
		for {
			select {
			case <-nc.heartbeatTicker.C:
				nc.sendHeartbeat()
			case <-nc.stopChan:
				return
			}
		}
	}()
}

// sendHeartbeat 发送心跳
func (nc *NettyClient) sendHeartbeat() {
	nc.mutex.RLock()
	connections := make([]*Connection, 0, len(nc.connections))
	for _, conn := range nc.connections {
		if conn.IsActive() {
			connections = append(connections, conn)
		}
	}
	nc.mutex.RUnlock()

	pingMsg := model.NewHotKeyMsg(model.Ping, nc.appName)

	for _, conn := range connections {
		err := nc.sendMessage(conn, pingMsg)
		if err != nil {
			hotlog.Errorf(nc, "Failed to send heartbeat to %s: %v", conn.Address, err)
			conn.SetActive(false)
		}
	}
}

// GetConnections 获取所有连接
func (nc *NettyClient) GetConnections() map[string]*Connection {
	nc.mutex.RLock()
	defer nc.mutex.RUnlock()

	result := make(map[string]*Connection)
	for addr, conn := range nc.connections {
		result[addr] = conn
	}
	return result
}

// GetActiveConnections 获取活跃连接
func (nc *NettyClient) GetActiveConnections() []*Connection {
	nc.mutex.RLock()
	defer nc.mutex.RUnlock()

	var active []*Connection
	for _, conn := range nc.connections {
		if conn.IsActive() {
			active = append(active, conn)
		}
	}
	return active
}

// GetNonConnectedAddresses 获取未连接的地址列表
func (nc *NettyClient) GetNonConnectedAddresses() []string {
	nc.mutex.RLock()
	defer nc.mutex.RUnlock()

	var addresses []string
	for addr, conn := range nc.connections {
		if !conn.IsActive() {
			addresses = append(addresses, addr)
		}
	}
	return addresses
}

// WorkerInfoHolder Worker信息持有者，对应Java的WorkerInfoHolder
type WorkerInfoHolder struct {
	workers   []*WorkerServer
	netClient *NettyClient
	mutex     sync.RWMutex
}

// WorkerServer Worker服务器信息
type WorkerServer struct {
	Address string
	Conn    *Connection
}

// NewWorkerInfoHolder 创建Worker信息持有者
func NewWorkerInfoHolder(netClient *NettyClient) *WorkerInfoHolder {
	return &WorkerInfoHolder{
		workers:   make([]*WorkerServer, 0),
		netClient: netClient,
	}
}

// GetWorkers 获取所有Worker
func (wih *WorkerInfoHolder) GetWorkers() []*WorkerServer {
	wih.mutex.RLock()
	defer wih.mutex.RUnlock()

	result := make([]*WorkerServer, len(wih.workers))
	copy(result, wih.workers)
	return result
}

// ChooseConnection 根据key选择连接，对应Java的WorkerInfoHolder.chooseChannel
func (wih *WorkerInfoHolder) ChooseConnection(key string) *Connection {
	wih.mutex.RLock()
	defer wih.mutex.RUnlock()

	size := len(wih.workers)
	if key == "" || size == 0 {
		return nil
	}

	// 使用key的hash值选择worker
	hash := hashCode(key)
	index := abs(hash) % size

	worker := wih.workers[index]
	if worker.Conn != nil && worker.Conn.IsActive() {
		return worker.Conn
	}
	return nil
}

// MergeAndConnectNew 合并并连接新的worker，对应Java的WorkerInfoHolder.mergeAndConnectNew
func (wih *WorkerInfoHolder) MergeAndConnectNew(allAddresses []string) {
	wih.removeNoneUsed(allAddresses)

	// 去连接那些在etcd里有，但是list里没有的
	needConnectWorkers := wih.newWorkers(allAddresses)
	if len(needConnectWorkers) == 0 {
		return
	}

	hotlog.Infof(wih, "New workers: %v", needConnectWorkers)

	// 连接新的worker
	wih.netClient.Connect(needConnectWorkers)

	// 更新worker列表
	wih.updateWorkerList(allAddresses)

	// 排序保证一致性
	wih.sortWorkers()
}

// DealChannelInactive 处理连接断开，对应Java的WorkerInfoHolder.dealChannelInactive
func (wih *WorkerInfoHolder) DealChannelInactive(address string) {
	wih.mutex.Lock()
	defer wih.mutex.Unlock()

	for i, worker := range wih.workers {
		if worker.Address == address {
			wih.workers = append(wih.workers[:i], wih.workers[i+1:]...)
			break
		}
	}
}

// updateWorkerList 更新worker列表
func (wih *WorkerInfoHolder) updateWorkerList(addresses []string) {
	wih.mutex.Lock()
	defer wih.mutex.Unlock()

	connections := wih.netClient.GetConnections()

	// 重建worker列表
	wih.workers = make([]*WorkerServer, 0, len(addresses))
	for _, addr := range addresses {
		worker := &WorkerServer{
			Address: addr,
			Conn:    connections[addr],
		}
		wih.workers = append(wih.workers, worker)
	}
}

// newWorkers 获取需要新连接的worker地址
func (wih *WorkerInfoHolder) newWorkers(allAddresses []string) []string {
	wih.mutex.RLock()
	defer wih.mutex.RUnlock()

	existingSet := make(map[string]bool)
	for _, worker := range wih.workers {
		existingSet[worker.Address] = true
	}

	var newAddresses []string
	for _, addr := range allAddresses {
		if !existingSet[addr] {
			newAddresses = append(newAddresses, addr)
		}
	}

	return newAddresses
}

// removeNoneUsed 移除不再使用的worker
func (wih *WorkerInfoHolder) removeNoneUsed(allAddresses []string) {
	wih.mutex.Lock()
	defer wih.mutex.Unlock()

	addressSet := make(map[string]bool)
	for _, addr := range allAddresses {
		addressSet[addr] = true
	}

	var keepWorkers []*WorkerServer
	for _, worker := range wih.workers {
		if addressSet[worker.Address] {
			keepWorkers = append(keepWorkers, worker)
		} else {
			hotlog.Infof(wih, "Worker remove: %s", worker.Address)
			if worker.Conn != nil {
				worker.Conn.Close()
			}
		}
	}

	wih.workers = keepWorkers
}

// sortWorkers 对worker进行排序
func (wih *WorkerInfoHolder) sortWorkers() {
	wih.mutex.Lock()
	defer wih.mutex.Unlock()

	sort.Slice(wih.workers, func(i, j int) bool {
		return wih.workers[i].Address < wih.workers[j].Address
	})
}

// hashCode 计算字符串的hash值
func hashCode(s string) int {
	h := 0
	for _, c := range s {
		h = 31*h + int(c)
	}
	return h
}

// abs 返回绝对值
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
