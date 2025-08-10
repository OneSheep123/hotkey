package com.jd.platform.hotkey.client.core.key;

import com.jd.platform.hotkey.client.Context;
import com.jd.platform.hotkey.client.core.worker.WorkerInfoHolder;
import com.jd.platform.hotkey.client.log.JdLogger;
import com.jd.platform.hotkey.common.model.HotKeyModel;
import com.jd.platform.hotkey.common.model.HotKeyMsg;
import com.jd.platform.hotkey.common.model.KeyCountModel;
import com.jd.platform.hotkey.common.model.typeenum.MessageType;
import io.netty.channel.Channel;

import java.net.InetSocketAddress;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

/**
 * Netty网络推送器实现类
 * 负责将收集到的热key数据和计数数据通过Netty网络框架推送到Worker节点
 * 
 * 核心功能：
 * 1. 热key数据推送：将半秒内收集的热key按hash分发到不同Worker
 * 2. 计数数据推送：将10秒内收集的访问计数按hash分发到不同Worker
 * 3. 负载均衡：通过key的hash值选择对应的Worker节点
 * 4. 批量发送：将同一Worker的数据打包后一次性发送，提高网络效率
 * 
 * @author wuweifeng wrote on 2020-01-06
 * @version 1.0
 */
public class NettyKeyPusher implements IKeyPusher {

    @Override
    public void send(String appName, List<HotKeyModel> list) {
        // 步骤1: 设置统一的时间戳
        // 为所有热key设置相同的创建时间，便于Worker端进行时间窗口计算
        long now = System.currentTimeMillis();

        // 步骤2: 按Worker节点分组热key数据
        // 使用HashMap<Channel, List<HotKeyModel>>结构，将热key按目标Worker进行分组
        // 这样设计的好处是：同一Worker的数据可以批量发送，减少网络开销
        Map<Channel, List<HotKeyModel>> map = new HashMap<>();
        
        // 步骤3: 遍历所有热key，进行负载均衡分发
        for(HotKeyModel model : list) {
            // 设置创建时间，确保时间窗口的一致性
            model.setCreateTime(now);
            
            // 根据key的hash值选择对应的Worker节点
            // WorkerInfoHolder.chooseChannel()实现了基于key的hash分片策略
            // 相同的key总是路由到同一个Worker，保证数据一致性
            Channel channel = WorkerInfoHolder.chooseChannel(model.getKey());
            
            // 如果没有可用的Worker连接，跳过当前key
            if (channel == null) {
                continue;
            }

            // 使用computeIfAbsent确保线程安全地创建或获取Worker对应的数据列表
            // 如果该Worker还没有对应的列表，则创建一个新的ArrayList
            List<HotKeyModel> newList = map.computeIfAbsent(channel, k -> new ArrayList<>());
            // 将当前热key添加到对应Worker的数据列表中
            newList.add(model);
        }

        // 步骤4: 批量发送数据到各个Worker节点
        for (Channel channel : map.keySet()) {
            try {
                // 获取当前Worker对应的所有热key数据
                List<HotKeyModel> batch = map.get(channel);
                
                // 创建热key消息对象
                // MessageType.REQUEST_NEW_KEY表示这是一个新的热key请求
                // Context.APP_NAME标识发送方应用名称，用于Worker端的数据路由
                HotKeyMsg hotKeyMsg = new HotKeyMsg(MessageType.REQUEST_NEW_KEY, Context.APP_NAME);
                
                // 设置热key数据到消息中
                hotKeyMsg.setHotKeyModels(batch);
                
                // 通过Netty Channel发送消息并等待发送完成
                // writeAndFlush()将消息写入缓冲区并立即刷新
                // sync()确保消息发送完成后再继续，提供可靠性保证
                channel.writeAndFlush(hotKeyMsg).sync();
                
            } catch (Exception e) {
                // 异常处理：记录发送失败的Worker地址信息
                try {
                    // 获取远程Worker的IP地址，便于问题排查
                    InetSocketAddress insocket = (InetSocketAddress) channel.remoteAddress();
                    JdLogger.error(getClass(),"flush error " + insocket.getAddress().getHostAddress());
                } catch (Exception ex) {
                    // 如果获取地址失败，记录通用错误信息
                    JdLogger.error(getClass(),"flush error");
                }

            }
        }

    }

    @Override
    public void sendCount(String appName, List<KeyCountModel> list) {
        // 步骤1: 设置统一的时间戳
        // 为所有计数数据设置相同的创建时间，便于Worker端进行统计时间窗口计算
        // 注意：计数数据是10秒收集一次，与热key的0.5秒不同
        long now = System.currentTimeMillis();
        
        // 步骤2: 按Worker节点分组计数数据
        // 使用HashMap<Channel, List<KeyCountModel>>结构，将计数数据按目标Worker进行分组
        // 计数数据包含key的访问次数统计信息，用于Worker端的热度分析
        Map<Channel, List<KeyCountModel>> map = new HashMap<>();
        
        // 步骤3: 遍历所有计数数据，进行负载均衡分发
        for(KeyCountModel model : list) {
            // 设置创建时间，确保统计时间窗口的一致性
            model.setCreateTime(now);
            
            // 根据规则key选择对应的Worker节点
            // 注意：计数数据使用ruleKey而不是普通的key进行路由
            // ruleKey通常是规则名称，确保相同规则的数据总是发送到同一个Worker
            Channel channel = WorkerInfoHolder.chooseChannel(model.getRuleKey());
            
            // 如果没有可用的Worker连接，跳过当前计数数据
            if (channel == null) {
                continue;
            }

            // 使用computeIfAbsent确保线程安全地创建或获取Worker对应的计数数据列表
            // 如果该Worker还没有对应的列表，则创建一个新的ArrayList
            List<KeyCountModel> newList = map.computeIfAbsent(channel, k -> new ArrayList<>());
            // 将当前计数数据添加到对应Worker的数据列表中
            newList.add(model);
        }

        // 步骤4: 批量发送计数数据到各个Worker节点
        for (Channel channel : map.keySet()) {
            try {
                // 获取当前Worker对应的所有计数数据
                List<KeyCountModel> batch = map.get(channel);
                
                // 创建计数消息对象
                // MessageType.REQUEST_HIT_COUNT表示这是一个访问计数请求
                // Context.APP_NAME标识发送方应用名称，用于Worker端的数据路由
                HotKeyMsg hotKeyMsg = new HotKeyMsg(MessageType.REQUEST_HIT_COUNT, Context.APP_NAME);
                
                // 设置计数数据到消息中
                hotKeyMsg.setKeyCountModels(batch);
                
                // 通过Netty Channel发送消息并等待发送完成
                // writeAndFlush()将消息写入缓冲区并立即刷新
                // sync()确保消息发送完成后再继续，提供可靠性保证
                channel.writeAndFlush(hotKeyMsg).sync();
                
            } catch (Exception e) {
                // 异常处理：记录发送失败的Worker地址信息
                try {
                    // 获取远程Worker的IP地址，便于问题排查
                    InetSocketAddress insocket = (InetSocketAddress) channel.remoteAddress();
                    JdLogger.error(getClass(),"flush error " + insocket.getAddress().getHostAddress());
                } catch (Exception ex) {
                    // 如果获取地址失败，记录通用错误信息
                    JdLogger.error(getClass(),"flush error");
                }

            }
        }
    }

}
