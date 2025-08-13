package worker

import (
	"net"
	"sort"
	"sync"
)

// Server Worker服务器信息
type Server struct {
	Address string
	Conn    net.Conn
}

func (s *Server) String() string {
	return s.Address
}

// WorkerInfoHolder Worker信息持有者
type WorkerInfoHolder struct {
	workers []*Server
	mu      sync.RWMutex
}

var defaultHolder = &WorkerInfoHolder{
	workers: make([]*Server, 0),
}

// GetInstance 获取默认实例
func GetInstance() *WorkerInfoHolder {
	return defaultHolder
}

// GetWorkers 获取所有Worker
func (wh *WorkerInfoHolder) GetWorkers() []*Server {
	wh.mu.RLock()
	defer wh.mu.RUnlock()

	result := make([]*Server, len(wh.workers))
	copy(result, wh.workers)
	return result
}

// HasConnected 判断某个Worker是否已经连接
func (wh *WorkerInfoHolder) HasConnected(address string) bool {
	wh.mu.RLock()
	defer wh.mu.RUnlock()

	for _, server := range wh.workers {
		if server.Address == address {
			return wh.channelIsOk(server.Conn)
		}
	}
	return false
}

// GetNonConnectedWorkers 获取未连接的Worker地址列表
func (wh *WorkerInfoHolder) GetNonConnectedWorkers() []string {
	wh.mu.RLock()
	defer wh.mu.RUnlock()

	var list []string
	for _, server := range wh.workers {
		if !wh.channelIsOk(server.Conn) {
			list = append(list, server.Address)
		}
	}
	return list
}

// ChooseChannel 根据key选择对应的Worker连接
func (wh *WorkerInfoHolder) ChooseChannel(key string) net.Conn {
	wh.mu.RLock()
	defer wh.mu.RUnlock()

	size := len(wh.workers)
	if key == "" || size == 0 {
		return nil
	}

	// 使用hash算法选择Worker
	index := int(hash(key)) % size
	if index < 0 {
		index = -index
	}

	if index < len(wh.workers) {
		return wh.workers[index].Conn
	}
	return nil
}

// MergeAndConnectNew 合并并连接新的Worker
func (wh *WorkerInfoHolder) MergeAndConnectNew(allAddresses []string) {
	wh.mu.Lock()
	defer wh.mu.Unlock()

	wh.removeNoneUsed(allAddresses)

	// 连接新的Worker
	needConnectWorkers := wh.newWorkers(allAddresses)
	if len(needConnectWorkers) == 0 {
		return
	}

	// 这里应该调用NettyClient连接新的Worker
	// NettyClient.GetInstance().Connect(needConnectWorkers)

	// 排序
	sort.Slice(wh.workers, func(i, j int) bool {
		return wh.workers[i].Address < wh.workers[j].Address
	})
}

// DealChannelInactive 处理连接断开事件
func (wh *WorkerInfoHolder) DealChannelInactive(address string) {
	wh.mu.Lock()
	defer wh.mu.Unlock()

	for i, server := range wh.workers {
		if server.Address == address {
			wh.workers = append(wh.workers[:i], wh.workers[i+1:]...)
			break
		}
	}
}

// Put 添加新的Worker
func (wh *WorkerInfoHolder) Put(address string, conn net.Conn) {
	wh.mu.Lock()
	defer wh.mu.Unlock()

	// 检查是否已存在
	for _, server := range wh.workers {
		if server.Address == address {
			server.Conn = conn
			return
		}
	}

	// 添加新的Worker
	server := &Server{
		Address: address,
		Conn:    conn,
	}
	wh.workers = append(wh.workers, server)
}

// 私有方法

func (wh *WorkerInfoHolder) channelIsOk(conn net.Conn) bool {
	return conn != nil
}

func (wh *WorkerInfoHolder) removeNoneUsed(allAddresses []string) {
	var newWorkers []*Server

	for _, server := range wh.workers {
		exist := false
		for _, address := range allAddresses {
			if server.Address == address {
				exist = true
				break
			}
		}
		if exist {
			newWorkers = append(newWorkers, server)
		} else {
			// 关闭连接
			if server.Conn != nil {
				server.Conn.Close()
			}
		}
	}

	wh.workers = newWorkers
}

func (wh *WorkerInfoHolder) newWorkers(allAddresses []string) []string {
	existingSet := make(map[string]bool)
	for _, server := range wh.workers {
		existingSet[server.Address] = true
	}

	var newWorkers []string
	for _, address := range allAddresses {
		if !existingSet[address] {
			newWorkers = append(newWorkers, address)
		}
	}

	return newWorkers
}

// hash 简单的hash函数
func hash(s string) uint32 {
	h := uint32(0)
	for i := 0; i < len(s); i++ {
		h = 31*h + uint32(s[i])
	}
	return h
}
