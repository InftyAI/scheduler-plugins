/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package resourceFungibility

import (
	"context"
	"fmt"

	llmazcoreapi "github.com/inftyai/llmaz/api/core/v1alpha1"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

const (
	Name              = "ResourceFungibility"
	preFilterStateKey = "PreFilter" + Name

	modelNameLabelKey = llmazcoreapi.ModelNameLabelKey
)

var (
	// Following fibonacci, we at most have 8 flavors. This is validated at llmaz.
	scoreWeights = []int32{34, 21, 13, 8, 5, 3, 2, 1}
	totalWeights = 34 + 21 + 13 + 8 + 5 + 3 + 2 + 1

	group    = llmazcoreapi.GroupVersion.Group
	version  = llmazcoreapi.GroupVersion.Version
	resource = "openmodels"
)

type ResourceFungibility struct {
	handle    framework.Handle
	dynClient *dynamic.DynamicClient
}

type flavor struct {
	name          string
	nodeSelectors map[string]string
}

type preFilterState struct {
	// Max item number is 8, which is limited by the API validation.
	inferenceFlavors []flavor
	shouldSkipFilter bool
}

func (s *preFilterState) Clone() framework.StateData {
	if s == nil {
		return nil
	}

	res := preFilterState{}

	for _, f := range s.inferenceFlavors {
		flavor := flavor{
			name:          f.name,
			nodeSelectors: map[string]string{},
		}
		for k, v := range f.nodeSelectors {
			flavor.nodeSelectors[k] = v
		}
	}

	return &res
}

var _ framework.PreFilterPlugin = (*ResourceFungibility)(nil)
var _ framework.FilterPlugin = (*ResourceFungibility)(nil)
var _ framework.PreScorePlugin = (*ResourceFungibility)(nil)
var _ framework.ScorePlugin = (*ResourceFungibility)(nil)

func New(ctx context.Context, _ runtime.Object, handle framework.Handle) (framework.Plugin, error) {
	dynClient, err := dynamic.NewForConfig(handle.KubeConfig())
	if err != nil {
		return nil, err
	}
	return &ResourceFungibility{
		handle:    handle,
		dynClient: dynClient,
	}, nil
}

func (rf *ResourceFungibility) Name() string {
	return Name
}

func (rf *ResourceFungibility) PreFilter(ctx context.Context, state *framework.CycleState, pod *v1.Pod) (*framework.PreFilterResult, *framework.Status) {
	modelName := pod.Labels[modelNameLabelKey]
	if modelName == "" {
		return nil, framework.NewStatus(framework.Skip)
	}

	preFilterState, err := rf.calPreFilterState(ctx, pod)
	if err != nil {
		return nil, framework.AsStatus(err)
	}
	state.Write(preFilterStateKey, preFilterState)
	return nil, nil
}

func (rf *ResourceFungibility) calPreFilterState(ctx context.Context, pod *v1.Pod) (*preFilterState, error) {
	gvr := schema.GroupVersionResource{
		Group:    group,
		Version:  version,
		Resource: resource,
	}

	// modelName shouldn't be empty here, early returned, it's a cluster-scope obj.
	modelName := pod.Labels[modelNameLabelKey]
	unstructuredModel, err := rf.dynClient.Resource(gvr).Get(ctx, modelName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	model := &llmazcoreapi.OpenModel{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredModel.Object, model); err != nil {
		return nil, err
	}

	var shouldSkipFilter bool
	s := preFilterState{}
	for _, f := range model.Spec.InferenceFlavors {
		flavor := flavor{
			name:          string(f.Name),
			nodeSelectors: map[string]string{},
		}
		if len(f.NodeSelector) == 0 {
			// Once nodeSelector is empty, which means all nodes are potential candidates,
			// so we'll skip the Filter stage.
			shouldSkipFilter = true
		}
		for k, v := range f.NodeSelector {
			flavor.nodeSelectors[k] = v
		}
		s.inferenceFlavors = append(s.inferenceFlavors, flavor)
	}

	s.shouldSkipFilter = shouldSkipFilter
	return &s, nil
}

func (rf *ResourceFungibility) PreFilterExtensions() framework.PreFilterExtensions {
	return nil
}

func (rf *ResourceFungibility) Filter(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeInfo *framework.NodeInfo) *framework.Status {
	preFilterState, err := getPreFilterState(state)
	if err != nil {
		return framework.AsStatus(err)
	}

	if preFilterState.shouldSkipFilter {
		return nil
	}

	for _, flavor := range preFilterState.inferenceFlavors {
		for k, v := range flavor.nodeSelectors {
			value, ok := nodeInfo.Node().Labels[k]
			if ok && value == v {
				// At least one flavor matches with the node, success then.
				return nil
			}
		}
	}
	return framework.NewStatus(framework.UnschedulableAndUnresolvable)
}

func (rf *ResourceFungibility) PreScore(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodes []*framework.NodeInfo) *framework.Status {
	modelName := pod.Labels[modelNameLabelKey]
	if modelName == "" {
		return framework.NewStatus(framework.Skip)
	}
	return nil
}

func (rf *ResourceFungibility) Score(ctx context.Context, state *framework.CycleState, p *v1.Pod, nodeName string) (int64, *framework.Status) {
	preFilterState, err := getPreFilterState(state)
	if err != nil {
		return 0, framework.AsStatus(err)
	}

	nodeInfo, err := rf.handle.SnapshotSharedLister().NodeInfos().Get(nodeName)
	if err != nil {
		return 0, framework.AsStatus(fmt.Errorf("getting node %q from Snapshot: %w", nodeName, err))
	}

	node := nodeInfo.Node()

	for i, flavor := range preFilterState.inferenceFlavors {
		for k, v := range flavor.nodeSelectors {
			value, ok := node.Labels[k]
			if ok && value == v {
				// Find the first matched node flavor.
				return (int64(scoreWeights[i]) / int64(totalWeights)) * 100, nil
			}
		}
	}

	// We should not reach here.
	return 0, nil
}

func (rf *ResourceFungibility) ScoreExtensions() framework.ScoreExtensions {
	return nil
}

func getPreFilterState(cycleState *framework.CycleState) (*preFilterState, error) {
	c, err := cycleState.Read(preFilterStateKey)
	if err != nil {
		// preFilterState doesn't exist, likely PreFilter wasn't invoked.
		return nil, fmt.Errorf("reading %q from cycleState: %w", preFilterStateKey, err)
	}

	s, ok := c.(*preFilterState)
	if !ok {
		return nil, fmt.Errorf("%+v convert to resourceFungibility.preFilterState error", c)
	}
	return s, nil
}
