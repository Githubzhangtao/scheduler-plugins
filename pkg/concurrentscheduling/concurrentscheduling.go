package concurrentscheduling

import (
	"context"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	v12 "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog/v2"
	framework "k8s.io/kubernetes/pkg/scheduler/framework/v1alpha1"
	util "sigs.k8s.io/scheduler-plugins/pkg/util"
	"sync"
)

var (
	once               sync.Once
	concurrentInstance *ConcurrentScheduling
	//PriorityList       = make(map[string]int64, 10)
)

const (
	Name = "ConcurrentScheduling"
)

type ConcurrentScheduling struct {
	frameworkHandler framework.FrameworkHandle
	podLister        v12.PodLister
	sync.RWMutex
	// 给定的pod返回node
	PodToNode map[string]string
	thePod    string
	//Queue     *workqueue.Type
	Queue     *util.Type
}

func (c *ConcurrentScheduling) PreFilter(ctx context.Context, state *framework.CycleState, pod *v1.Pod) *framework.Status {
	klog.V(2).Infof("PreFilter测试，当前Pod:%v,队列长度:%v", pod.Name, c.Queue.Len())
	// 判断是否属于某一个rs或者deployment
	//if len(pod.OwnerReferences) <= 0 {
	//	return nil
	//}
	//owener := pod.OwnerReferences[0]
	//if owener.Kind != "ReplicaSet" {
	//	klog.V(2).Infof("pod %v 不属于任何RS", pod.Name)
	//	return nil
	//}

	//klog.V(1).Infof("当前Pod:%v队列：%v",pod.Name,c.Queue.Len())

	//if c.thePod == "" || len(c.thePod) <= 0{
	//	c.thePod = pod.Name
	//	klog.V(2).Infof("新的ThePod:%v,队列长度为:%v", c.thePod, c.Queue.Len())
	//	return nil
	//}
	//if  c.Queue.Len() <= 0{
	//	c.thePod = pod.Name
	//	klog.V(2).Infof("当前队列长度:%v",c.Queue.Len())
	//	return nil
	//}

	if c.thePod == "" || len(c.thePod) <= 0 || c.Queue.Len() <= 0 {
		c.thePod = pod.Name
		klog.V(2).Infof("新的ThePod:%v,队列长度为:%v", c.thePod, c.Queue.Len())
		return nil
	}

	klog.V(2).Infof("当前Pod %v是NormalPod，队列长度为:%v,队列:%v", pod.Name, c.Queue.Len(),c.Queue.Len())
	nodeName, _ := c.Queue.Get()
	c.PodToNode[pod.Name] = nodeName.(framework.NodeScore).Name

	//if c.thePod == pod.Name {
	//	klog.V(2).Infof("当前pod %v是ThePod，需要全流程！", pod.Name)
	//	return nil
	//} else {
	//	klog.V(1).Infof("当前Pod %v是NormalPod，队列长度为:%v", pod.Name, c.Queue.Len())
	//	nodeName, _ := c.Queue.Get()
	//
	//}

	//if c.Queue.Len() <= 0 {
	//	klog.V(2).Infof("当前队列没有节点！需等待下一次调度！")
	//	return framework.NewStatus(framework.UnschedulableAndUnresolvable)
	//}
	/*nodeName, _ := c.Queue.Get()


	if c.Queue.Len() == 0 {
		c.thePod = ""
		klog.V(2).Infof("取完最后一个节点，需要重新调度一个pod")
	}
	c.PodToNode[pod.Name] = nodeName.(framework.NodeScore).Name*/
	//klog.V(1).Infof("PodToNode：%v,nodeName:%v,队列:%v",c.PodToNode,nodeName,c.Queue)

	return nil
}

// Filter代码逻辑在V2版本中已经放在generic_scheduler.go中
func (c *ConcurrentScheduling) Filter(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeInfo *framework.NodeInfo) *framework.Status {
	klog.V(2).Infof("并行Filter 测试,ThePod:%v，当前Pod:%v,当前节点%v",c.thePod, pod.Name, nodeInfo.Node().Name)
	// V1版本需要以下代码，关闭注释。V2版本不需要以下代码，注释掉。
	nodeName := c.PodToNode[pod.Name]
	if nodeName == "" {
		klog.V(2).Infof("pod %v 是ThePod需要进行完整的预选和优选流程！", pod.Name)
		return nil
	}
	// 获取打分列表，如果没有列表表示是第一个pod运行，返回success继续运行下面的逻辑，如果获取到打分列表表示可以并行运行
	if nodeInfo.Node().Name != nodeName {
		//klog.V(2).Infof("node %v，不是pod %v 想要调度的节点", nodeInfo.Node().Name, pod.Name)
		//klog.V(2).Infof("当前节点:%v,指定调度的节点:%v", nodeInfo.Node().Name, nodeName)
		return framework.NewStatus(framework.Unschedulable, "不是我们想调度的节点")
	} else {
		klog.V(2).Infof("是我们要调度的节点:%v", nodeInfo.Node().Name)
	}
	return nil
}

func (c *ConcurrentScheduling) PreFilterExtensions() framework.PreFilterExtensions {
	klog.V(2).Info("PreFilterExtensions 测试")
	return nil
}
func (c *ConcurrentScheduling) Score(ctx context.Context, state *framework.CycleState, p *v1.Pod, nodeName string) (int64, *framework.Status) {
	klog.V(2).Info("Score 测试")
	return 0, nil
}

func (c *ConcurrentScheduling) ScoreExtensions() framework.ScoreExtensions {
	klog.V(2).Info("ScoreExtensions 测试")
	return c
}
func (c *ConcurrentScheduling) NormalizeScore(ctx context.Context, state *framework.CycleState, p *v1.Pod, scores framework.NodeScoreList) *framework.Status {
	klog.V(2).Info("NormalizeScore 测试")
	return nil
}

func (c *ConcurrentScheduling) Bind(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) *framework.Status {
	klog.V(2).Info("Bind 测试")
	return nil
}

func (c *ConcurrentScheduling) Name() string {
	return Name
}

var _ framework.PreFilterPlugin = &ConcurrentScheduling{}
var _ framework.FilterPlugin = &ConcurrentScheduling{}
var _ framework.BindPlugin = &ConcurrentScheduling{}
var _ framework.ScorePlugin = &ConcurrentScheduling{}
var _ framework.ScoreExtensions = &ConcurrentScheduling{}

//var _ framewor = &ConcurrentScheduling{}

func GetConcurrentScheduler() *ConcurrentScheduling {
	return concurrentInstance
}

// New initializes and returns a new Coscheduling plugin.
func New(obj runtime.Object, handle framework.FrameworkHandle) (framework.Plugin, error) {
	once.Do(func() {
		queue := util.New()
		//queue.Add("k8s1.19.7node-control-plane")
		//queue.Add("k8s1.19.7node-worker")
		//queue.Add("k8s1.19.7node-worker2")
		//queue.Add("k8s1.19.7node-worker3")
		//queue.Add("k8s1.19.7node-worker4")
		//queue.Add("k8s1.19.7node-worker5")
		//queue.Add("k8s1.19.7node-worker6")
		podLister := handle.SharedInformerFactory().Core().V1().Pods().Lister()
		concurrentInstance = &ConcurrentScheduling{
			frameworkHandler: handle,
			podLister:        podLister,
			Queue:            queue,
			PodToNode:        make(map[string]string, 7),
			thePod:           "",
		}
	})
	return concurrentInstance, nil

}
