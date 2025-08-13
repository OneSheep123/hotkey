package com.jd.platform.hotkey.common.configcenter.etcd;

import cn.hutool.core.collection.CollectionUtil;
import com.google.protobuf.ByteString;
import com.ibm.etcd.api.KeyValue;
import com.ibm.etcd.api.LeaseGrantResponse;
import com.ibm.etcd.api.RangeResponse;
import com.ibm.etcd.client.KvStoreClient;
import com.ibm.etcd.client.kv.KvClient;
import com.ibm.etcd.client.lease.LeaseClient;
import com.ibm.etcd.client.lease.PersistentLease;
import com.ibm.etcd.client.lock.LockClient;
import com.jd.platform.hotkey.common.configcenter.IConfigCenter;

import java.util.List;
import java.util.concurrent.ExecutionException;

import static java.util.concurrent.TimeUnit.SECONDS;

/**
 * JD HotKey项目的etcd客户端实现类
 * 
 * 功能说明：
 * 1. 配置中心：提供key-value的存储和读取功能
 * 2. 服务发现：支持服务注册、发现和健康检查
 * 3. 分布式锁：提供分布式锁机制
 * 4. 租约管理：支持TTL（Time To Live）和自动续约
 * 5. 事件监听：支持key变化的事件监听和前缀监听
 * 
 * 核心特性：
 * - 基于IBM etcd-java客户端库
 * - 支持同步和异步操作
 * - 提供租约自动续约机制
 * - 支持前缀查询和监听
 * 
 * @author wuweifeng wrote on 2019-12-06
 * @version 1.0
 */
public class JdEtcdClient implements IConfigCenter {

    /**
     * KV存储客户端：负责key-value的增删改查操作
     */
    private KvClient kvClient;
    
    /**
     * 租约客户端：负责租约的创建、续约、撤销等操作
     * 租约用于实现TTL（Time To Live）功能，key会在租约到期后自动删除
     */
    private LeaseClient leaseClient;
    
    /**
     * 分布式锁客户端：提供分布式锁机制
     * 用于协调多个节点之间的并发访问
     */
    private LockClient lockClient;

    /**
     * 构造函数：初始化etcd客户端的各个组件
     * 
     * @param kvStoreClient etcd存储客户端，包含KV、租约、锁等子客户端
     */
    public JdEtcdClient(KvStoreClient kvStoreClient) {
        // 获取KV操作客户端
        this.kvClient = kvStoreClient.getKvClient();
        // 获取租约管理客户端
        this.leaseClient = kvStoreClient.getLeaseClient();
        // 获取分布式锁客户端
        this.lockClient = kvStoreClient.getLockClient();
    }


    /**
     * 获取租约客户端
     * @return 租约客户端实例，用于管理TTL和自动续约
     */
    public LeaseClient getLeaseClient() {
        return leaseClient;
    }

    /**
     * 设置租约客户端
     * @param leaseClient 新的租约客户端实例
     */
    public void setLeaseClient(LeaseClient leaseClient) {
        this.leaseClient = leaseClient;
    }

    /**
     * 获取KV存储客户端
     * @return KV客户端实例，用于key-value的增删改查
     */
    public KvClient getKvClient() {
        return kvClient;
    }

    /**
     * 设置KV存储客户端
     * @param kvClient 新的KV客户端实例
     */
    public void setKvClient(KvClient kvClient) {
        this.kvClient = kvClient;
    }

    /**
     * 获取分布式锁客户端
     * @return 锁客户端实例，用于分布式锁操作
     */
    public LockClient getLockClient() {
        return lockClient;
    }

    /**
     * 设置分布式锁客户端
     * @param lockClient 新的锁客户端实例
     */
    public void setLockClient(LockClient lockClient) {
        this.lockClient = lockClient;
    }

    @Override
    public void put(String key, String value) {
        // 将key和value转换为ByteString格式，然后同步写入etcd
        // sync()确保操作完成后再返回，提供可靠性保证
        kvClient.put(ByteString.copyFromUtf8(key), ByteString.copyFromUtf8(value)).sync();
    }

    @Override
    public void put(String key, String value, long leaseId) {
        // 将key-value与已存在的租约关联
        // 注意：此方法不设置租约的TTL，租约的生存时间必须在其他地方预先设置
        // 当关联的租约到期时，该key会自动从etcd中删除
        // 常用于实现临时配置或会话管理
        kvClient.put(ByteString.copyFromUtf8(key), ByteString.copyFromUtf8(value), leaseId).sync();
    }

    @Override
    public void revoke(long leaseId) {
        // 撤销指定的租约，立即删除与该租约关联的所有key
        // 这是一个异步操作，不需要等待完成
        leaseClient.revoke(leaseId);
    }

    @Override
    public long putAndGrant(String key, String value, long ttl) {
        // 组合操作：创建租约 + 写入key-value
        // 1. 首先创建一个指定TTL的租约
        // 2. 然后将key-value与该租约关联
        // 3. 返回租约ID，便于后续管理
        LeaseGrantResponse lease = leaseClient.grant(ttl).sync();
        put(key, value, lease.getID());
        return lease.getID();
    }

    @Override
    public long setLease(String key, long leaseId) {
        // 如果key之前没有租约，现在会与指定租约关联
        // 如果key已有租约，会替换为新的租约
        kvClient.setLease(ByteString.copyFromUtf8(key), leaseId);
        return leaseId;
    }

    @Override
    public void delete(String key) {
        // 删除指定的key及其对应的value
        // sync()确保删除操作完成后再返回
        kvClient.delete(ByteString.copyFromUtf8(key)).sync();
    }

    @Override
    public String get(String key) {
        // 根据key查询对应的value
        // 1. 将key转换为ByteString格式
        // 2. 同步查询etcd，等待响应
        // 3. 从响应中提取KeyValue列表
        RangeResponse rangeResponse = kvClient.get(ByteString.copyFromUtf8(key)).sync();
        List<KeyValue> keyValues = rangeResponse.getKvsList();

        // 如果key不存在，返回null
        if (CollectionUtil.isEmpty(keyValues)) {
            return null;
        }
        // 返回第一个匹配的value，转换为UTF-8字符串
        return keyValues.get(0).getValue().toStringUtf8();
    }

    @Override
    public KeyValue getKv(String key) {
        // 获取完整的KeyValue对象，包含key、value、版本、创建时间等元信息
        // 与get()方法不同，这里返回的是完整的KeyValue对象，而不仅仅是value字符串
        RangeResponse rangeResponse = kvClient.get(ByteString.copyFromUtf8(key)).sync();
        List<KeyValue> keyValues = rangeResponse.getKvsList();
        if (CollectionUtil.isEmpty(keyValues)) {
            return null;
        }
        return keyValues.get(0);
    }

    @Override
    public List<KeyValue> getPrefix(String key) {
        // 前缀查询：获取所有以指定key为前缀的键值对
        // 例如：getPrefix("/jd/apps/") 会返回所有以"/jd/apps/"开头的key
        // 常用于批量获取配置或服务列表
        RangeResponse rangeResponse = kvClient.get(ByteString.copyFromUtf8(key)).asPrefix().sync();
        return rangeResponse.getKvsList();
    }

    @Override
    public KvClient.WatchIterator watch(String key) {
        // 监听指定key的变化事件
        // 当key被创建、修改或删除时，会触发相应的事件
        // 返回WatchIterator，可以通过迭代器获取变化事件
        return kvClient.watch(ByteString.copyFromUtf8(key)).start();
    }

    @Override
    public KvClient.WatchIterator watchPrefix(String key) {
        // 前缀监听：监听所有以指定key为前缀的键值对变化
        // 常用于监听配置变化或服务状态变化
        // 例如：监听"/jd/apps/"前缀可以监听到所有应用的状态变化
        return kvClient.watch(ByteString.copyFromUtf8(key)).asPrefix().start();
    }

    @Override
    public long keepAlive(String key, String value, int frequencySecs, int minTtl) throws Exception {
        // 创建持久租约并自动续约，同时写入key-value
        // 
        // 参数说明：
        // - frequencySecs: 续约频率（秒），每隔多少秒自动续约一次
        // - minTtl: 最小租期（秒），租约的TTL不会低于这个值
        // 
        // 工作流程：
        // 1. 创建一个持久租约，使用当前时间戳作为租约ID
        // 2. 设置自动续约频率和最小TTL
        // 3. 启动租约维护，开始自动续约
        // 4. 等待3秒获取租约ID
        // 5. 将key-value与该租约关联
        // 6. 返回租约ID
        PersistentLease lease = leaseClient.maintain().leaseId(System.currentTimeMillis()).keepAliveFreq(frequencySecs).minTtl(minTtl).start();
        long newId = lease.get(3L, SECONDS);
        put(key, value, newId);
        return newId;
    }

    @Override
    public long buildAliveLease(int frequencySecs, int minTtl) throws Exception {
        // 创建持久租约但不写入key-value
        // 与keepAlive()的区别是：只创建租约，不执行put操作
        // 适用于需要先创建租约，后续再决定如何使用的情况
        PersistentLease lease = leaseClient.maintain().leaseId(System.currentTimeMillis()).keepAliveFreq(frequencySecs).minTtl(minTtl).start();

        return lease.get(3L, SECONDS);
    }

    @Override
    public long buildNormalLease(long ttl) {
        // 创建普通租约（非持久租约）
        // 普通租约不会自动续约，到期后会自动删除
        // 适用于临时配置或一次性会话
        LeaseGrantResponse lease = leaseClient.grant(ttl).sync();
        return lease.getID();
    }

    @Override
    public long timeToLive(long leaseId) {
        // 查询指定租约的剩余生存时间
        // 返回值表示租约还有多少秒过期
        // 如果租约不存在或已过期，返回0
        try {
            return leaseClient.ttl(leaseId).get().getTTL();
        } catch (InterruptedException | ExecutionException e) {
            // 异常处理：如果查询失败，返回0表示租约无效
            e.printStackTrace();
            return 0L;
        }
    }
}
