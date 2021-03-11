package balancenetscheduling

import (
	"context"
	"fmt"
	v1 "k8s.io/api/core/v1"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/kubernetes/pkg/features"
	schedutil "k8s.io/kubernetes/pkg/scheduler/util"
	"math"
	"strconv"

	"k8s.io/apimachinery/pkg/runtime"
	v12 "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog/v2"
	framework "k8s.io/kubernetes/pkg/scheduler/framework/v1alpha1"
	"sigs.k8s.io/scheduler-plugins/pkg/apis/config"
)

const (
	// Name is the name of the plugin used in Registry and configurations.
	Name       = "BalanceNetScheduling"
	NodeNetMap = "nodeNetInfo"
)
func Decimal(value float64) float64 {
	value, _ = strconv.ParseFloat(fmt.Sprintf("%.2f", value), 64)
	return value
}

// New initializes and returns a new Coscheduling plugin.
func New(obj runtime.Object, handle framework.FrameworkHandle) (framework.Plugin, error) {
	fmt.Println(obj)
	args, ok := obj.(*config.BalanceNetSchedulingArgs)
	if !ok {
		return nil, fmt.Errorf("want args to be of type CoschedulingArgs, got %T", obj)
	}
	fmt.Printf("args:%v\n", args)
	resourceToWeightMap := make(map[ResourceName]int64, 3)
	for _, weightMap := range args.ResourceToWeightMap {
		resourceToWeightMap[ResourceName(weightMap.Name)] = weightMap.Weight
	}
	podLister := handle.SharedInformerFactory().Core().V1().Pods().Lister()
	return &BalanceNetScheduling{
		frameworkHandler:    handle,
		podLister:           podLister,
		resourceToWeightMap: resourceToWeightMap,
	}, nil
}

// Coscheduling is a plugin that schedules pods in a group.
type BalanceNetScheduling struct {
	frameworkHandler    framework.FrameworkHandle
	podLister           v12.PodLister
	resourceToWeightMap map[ResourceName]int64
}

func (b BalanceNetScheduling) Filter(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeInfo *framework.NodeInfo) *framework.Status {
	//klog.V(1).Infof("均衡Filter 测试,当前Pod:%v,当前节点%v", pod.Name, nodeInfo.Node().Name)

	return nil
}

func (b BalanceNetScheduling) PreFilter(ctx context.Context, state *framework.CycleState, p *v1.Pod) *framework.Status {
	klog.V(1).Info("测试Prefilter------------------------")
	// 由于PreFilter扩展点每个Pod都会执行，我们要保证这个插件只执行一次，即节点的带框数据只初始化一次
	_, err := state.Read(NodeNetMap)
	if err != nil {
		nodeNetRequestMap := make(map[string]int32, 3)
		nodeNetCapacityMap := make(map[string]int32, 3)
		nodeList, _ := b.frameworkHandler.SnapshotSharedLister().NodeInfos().List()
		for _, node := range nodeList {
			nodeNetRequestMap[node.Node().Name] = 0
			nodeNetCapacityMap[node.Node().Name] = 100
		}
		state.Write(NodeNetMap, NewNodeStateData(nodeNetRequestMap, nodeNetCapacityMap))
	}

	return framework.NewStatus(framework.Success)
}

func (b BalanceNetScheduling) PreFilterExtensions() framework.PreFilterExtensions {
	//klog.V(1).Info("测试  PreFilterExtensions")
	return nil
}

func (b BalanceNetScheduling) Name() string {
	return Name
}

type resourceToValueMap map[ResourceName]int64
type ResourceName string

const (
	// CPU, in cores. (500m = .5 cores)
	ResourceCPU ResourceName = "cpu"
	// Memory, in bytes. (500Gi = 500GiB = 500 * 1024 * 1024 * 1024)
	ResourceMemory ResourceName = "mem"
	// Net request size, in bytes (e,g. 5Gi = 5GiB = 5 * 1024 * 1024 * 1024)
	ResourceNet ResourceName = "net"
)

func (b BalanceNetScheduling) Score(ctx context.Context, state *framework.CycleState, p *v1.Pod, nodeName string) (int64, *framework.Status) {
	klog.V(1).Info("测试  Score")
	nodeInfo, err := b.frameworkHandler.SnapshotSharedLister().NodeInfos().Get(nodeName)
	if err != nil {
		return 0, framework.NewStatus(framework.Error, fmt.Sprintf("获取节点 %q 错误：%v", nodeName, err))
	}
	node := nodeInfo.Node()
	if node == nil {
		return 0, framework.NewStatus(framework.Error, fmt.Sprintf("找不到该节点,%q", node.Name))
	}
	nodeNetInfo, err := state.Read(NodeNetMap)
	if err != nil {
		return 0, framework.NewStatus(framework.Unschedulable, "没有找到节点网络信息！")
	}
	nodeNetMapData := nodeNetInfo.(*nodeStateData)
	nodeNetCapacity := nodeNetMapData.NodeNetCapacityMap[node.Name]
	nodeNetRequest := 0
	podNetLabel := p.Labels["netRequest"]
	if podNetLabel != "" {
		podNet, err := strconv.Atoi(podNetLabel)
		if err != nil {
			klog.V(1).Info("节点的netRequest label值不能转换为int")
			return 0, nil
		}
		nodeNetRequest += podNet
	}
	podListOfNode := nodeInfo.Pods
	for _, podInfo := range podListOfNode {
		pod := podInfo.Pod
		//if pod.Status.Phase == v1.PodRunning || pod.Status.Phase == v1.PodPending {
		podNetRequestLabel := pod.Labels["netRequest"]
		if podNetRequestLabel == "" {
			//klog.V(1).Infof("该pod %v: 没有指定netRequest", pod.Name)
			continue
		} else {
			podNetRequest, err := strconv.Atoi(podNetRequestLabel)
			if err != nil {
				//klog.V(1).Info("节点的netRequest label值不能转换为int")
				continue
			}
			nodeNetRequest += podNetRequest
		}

		//}
	}

	requested := make(resourceToValueMap, 3)
	allocatable := make(resourceToValueMap, 3)
	requested[ResourceNet] = int64(nodeNetRequest)
	allocatable[ResourceNet] = int64(nodeNetCapacity)

	// v2 NodeResourceBalanceWithNetAllocation
	// 1、计算cpu和mem的request以及capacity
	allocatable[ResourceCPU], requested[ResourceCPU] = calculateResourceAllocatableRequest(nodeInfo, p, v1.ResourceCPU)
	allocatable[ResourceMemory], requested[ResourceMemory] = calculateResourceAllocatableRequest(nodeInfo, p, v1.ResourceMemory)
	//klog.V(1).Infof("node 名字: %v",nodeName)

	score := score(requested, allocatable, b.resourceToWeightMap, nodeName)

	// v1 只关注net
	//s := (1 - float64(nodeNetRequest)/float64(nodeNetCapacity)) * float64(framework.MaxNodeScore)
	//klog.V(1).Infof("节点 %v 的分数是 %v", node.Name, s)
	//klog.V(1).Infof("map ---- %v",b.resourceToWeightMap)
	//if s < 0 {
	//	return 0, nil
	//}
	//return int64(s), nil

	return score, framework.NewStatus(framework.Success, "测试输出！")

	//calculateResourceAllocatableRequest(nodeInfo,p)

	//return 0, nil
}

func score(requested resourceToValueMap, allocatable resourceToValueMap, weightMap map[ResourceName]int64, nodeName string) int64 {

	max := 1.0
	min := -1.0
	//for _, v := range weightMap {
	//	max += float64(v)
	//}
	//max = max / float64(3)

	remain := make(resourceToValueMap, 3)
	for k, v := range allocatable {
		rv := v - requested[k]
		remain[k] = int64(math.Max(0, float64(rv)))
	}

	r1 := 0.0
	var weightAll int64 = 0
	for k, v := range allocatable {
		weightAll += weightMap[k]
		r1 += float64(weightMap[k]*remain[k]) / float64(v)
	}
	r1 = r1 / float64(weightAll)

	a := float64(requested[ResourceCPU])/float64(allocatable[ResourceCPU]) - float64(requested[ResourceMemory])/float64(allocatable[ResourceMemory])
	b := float64(requested[ResourceCPU])/float64(allocatable[ResourceCPU]) - float64(requested[ResourceNet])/float64(allocatable[ResourceNet])
	c := float64(requested[ResourceMemory])/float64(allocatable[ResourceMemory]) - float64(requested[ResourceNet])/float64(allocatable[ResourceNet])

	r2 := (math.Abs(a) + math.Abs(b) + math.Abs(c)) / float64(3)

	//klog.V(1).Infof("allocatable: %v",allocatable)
	//klog.V(1).Infof("requested: %v",requested)
	//klog.V(1).Infof("remain:%v",remain)
	//klog.V(1).Infof("r1:%v",r1)
	//klog.V(1).Infof("r2:%v",r2)
	//
	//klog.V(1).Infof("r1-r2:%v",r1-r2)
	score := (r1 - r2 - min) / (max - min)
	score = score * 100
	klog.V(1).Infof("nodeName is %v,r1: %v,r2: %v,score:%v,weightAll:%v,max:%v,min:%v", nodeName, Decimal(r1),Decimal(r2), Decimal(score),weightAll, max, min)
	//klog.V(1).Infof("requested:%v,allocatable:%v,remain:%v",requested,allocatable,remain)

	return int64(score)
}

func calculateResourceAllocatableRequest(nodeInfo *framework.NodeInfo, pod *v1.Pod, resource v1.ResourceName) (int64, int64) {
	allocatable := nodeInfo.Allocatable
	podRequest := calculatePodResourceRequest(pod, resource)
	switch resource {
	case v1.ResourceCPU:
		return allocatable.MilliCPU, (nodeInfo.NonZeroRequested.MilliCPU + podRequest)
	case v1.ResourceMemory:
		return allocatable.Memory, (nodeInfo.NonZeroRequested.Memory + podRequest)

	default:
		return 0, 0
	}
	klog.Infof("requested resource %v not considered for node score calculation", resource)
	return 0, 0
}

// calculatePodResourceRequest returns the total non-zero requests. If Overhead is defined for the pod and the
// PodOverhead feature is enabled, the Overhead is added to the result.
// podResourceRequest = max(sum(podSpec.Containers), podSpec.InitContainers) + overHead
func calculatePodResourceRequest(pod *v1.Pod, resource v1.ResourceName) int64 {
	var podRequest int64
	for i := range pod.Spec.Containers {
		container := &pod.Spec.Containers[i]
		value := schedutil.GetNonzeroRequestForResource(resource, &container.Resources.Requests)
		podRequest += value
	}

	for i := range pod.Spec.InitContainers {
		initContainer := &pod.Spec.InitContainers[i]
		value := schedutil.GetNonzeroRequestForResource(resource, &initContainer.Resources.Requests)
		if podRequest < value {
			podRequest = value
		}
	}

	// If Overhead is being utilized, add to the total requests for the pod
	if pod.Spec.Overhead != nil && utilfeature.DefaultFeatureGate.Enabled(features.PodOverhead) {
		if quantity, found := pod.Spec.Overhead[resource]; found {
			podRequest += quantity.Value()
		}
	}

	return podRequest
}

func (b BalanceNetScheduling) ScoreExtensions() framework.ScoreExtensions {
	klog.V(1).Info("测试  ScoreExtensions")
	return nil
}

var _ framework.ScorePlugin = &BalanceNetScheduling{}
var _ framework.FilterPlugin = &BalanceNetScheduling{}

var _ framework.PreFilterPlugin = &BalanceNetScheduling{}

// nodeStateData 用来存放节点的网络带宽容量和使用情况
type nodeStateData struct {
	NodeNetRequestMap  map[string]int32 `json:"nodeNetRequestMap"`
	NodeNetCapacityMap map[string]int32 `json:"nodeNetCapacityMap"`
}

func NewNodeStateData(nodeNetRequestMap map[string]int32, nodeNetCapacityMap map[string]int32) framework.StateData {
	return &nodeStateData{
		NodeNetRequestMap:  nodeNetRequestMap,
		NodeNetCapacityMap: nodeNetCapacityMap,
	}
}

func (d *nodeStateData) Clone() framework.StateData {
	return d
}
