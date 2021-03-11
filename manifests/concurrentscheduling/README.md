1、伪并行调度算法有俩个版本，区别如下
    V1：只能降低优选耗时，不能降低预选耗时，因为是通过PreFilter实现预选过程中的节点过滤
    V2：可以降低预选和优选耗时，在V1的基础上改进，在源码kubernetes@v1.19.0\pkg\scheduler\core\generic_scheduler.go的findNodesThatFitPod()方法中
        进行判断，对NormalPod直接跳过预选阶段
    
2、V1版本使用scheduler-config-v1.yaml配置文件启动，V2版本使用scheduler-config-v2.yaml配置文件启动

3、V1版本注释掉源码kubernetes@v1.19.0\pkg\scheduler\core\generic_scheduler.go的findNodesThatFitPod()中的侵入代码，V2版本注释掉concurrentscheduling.go中Prefilter的逻辑代码


    
    
