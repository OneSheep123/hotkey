package com.jd.platform.hotkey.client.core.key;

import cn.hutool.core.collection.CollectionUtil;
import cn.hutool.core.thread.NamedThreadFactory;
import com.jd.platform.hotkey.client.Context;
import com.jd.platform.hotkey.common.model.HotKeyModel;
import com.jd.platform.hotkey.common.model.KeyCountModel;

import java.util.Collections;
import java.util.List;
import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.TimeUnit;

/**
 * 定时推送一批key到worker
 * @author wuweifeng wrote on 2020-01-06
 * @version 1.0
 */
public class PushSchedulerStarter {

    /**
     * 每0.5秒推送一次待测key
     */
    public static void startPusher(Long period) {
        if (period == null || period <= 0) {
            period = 500L;
        }
        @SuppressWarnings("PMD.ThreadPoolCreationRule")
        ScheduledExecutorService scheduledExecutorService = Executors.newSingleThreadScheduledExecutor(new NamedThreadFactory("hotkey-pusher-service-executor", true));
        scheduledExecutorService.scheduleAtFixedRate(() -> {
            // 步骤1: 获取热key收集器实例
            // 通过工厂模式获取实现了IKeyCollector接口的收集器
            IKeyCollector<HotKeyModel, HotKeyModel> collectHK = KeyHandlerFactory.getCollector();
            
            // 步骤2: 锁定并获取收集结果
            // lockAndGetResult()会锁定当前收集器，返回已收集的热key列表，并清空收集器
            // 这样可以避免在推送过程中继续收集新数据，保证数据一致性
            List<HotKeyModel> hotKeyModels = collectHK.lockAndGetResult();
            
            // 步骤3: 检查是否有数据需要推送
            // 只有当收集到的热key列表不为空时才进行推送，避免无效的网络传输
            if(CollectionUtil.isNotEmpty(hotKeyModels)){
                // 步骤4: 推送热key数据到Worker节点
                // 通过KeyHandlerFactory获取推送器，将收集到的热key批量发送到Worker
                // Context.APP_NAME标识当前应用名称，用于Worker端的数据路由
                KeyHandlerFactory.getPusher().send(Context.APP_NAME, hotKeyModels);
                
                // 步骤5: 标记本次推送完成
                // finishOnce()通知收集器本次推送已完成，可以开始下一轮收集
                // 这是双缓冲机制的关键，确保收集和推送的交替进行
                collectHK.finishOnce();
            }

        },0, period, TimeUnit.MILLISECONDS);
    }

    /**
     * 每10秒推送一次数量统计
     */

    public static void startCountPusher(Integer period) {
        if (period == null || period <= 0) {
            period = 10;
        }
        @SuppressWarnings("PMD.ThreadPoolCreationRule")
        ScheduledExecutorService scheduledExecutorService = Executors.newSingleThreadScheduledExecutor(new NamedThreadFactory("hotkey-count-pusher-service-executor", true));
        scheduledExecutorService.scheduleAtFixedRate(() -> {
            IKeyCollector<KeyHotModel, KeyCountModel> collectHK = KeyHandlerFactory.getCounter();
            List<KeyCountModel> keyCountModels = collectHK.lockAndGetResult();
            if(CollectionUtil.isNotEmpty(keyCountModels)){
                KeyHandlerFactory.getPusher().sendCount(Context.APP_NAME, keyCountModels);
                collectHK.finishOnce();
            }
        },0, period, TimeUnit.SECONDS);
    }

}
