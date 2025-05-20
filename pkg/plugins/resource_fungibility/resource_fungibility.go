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
	"math"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/kubernetes/pkg/scheduler/framework"

	llmazcoreapi "github.com/inftyai/llmaz/api/core/v1alpha1"
)

const (
	Name     = "ResourceFungibility"
	stateKey = Name

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

// TODO: get the inference service to extract the flavors.
type ResourceFungibility struct {
	handle    framework.Handle
	dynClient *dynamic.DynamicClient
}

type flavor struct {
	name          string
	nodeSelectors map[string]string
}

type state struct {
	// Max item number is 8, which is limited by the API validation.
	inferenceFlavors []flavor
	shouldSkip       bool
}

func (s *state) Clone() framework.StateData {
	if s == nil {
		return nil
	}

	res := state{}

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

func (rf *ResourceFungibility) PreFilter(ctx context.Context, cycleState *framework.CycleState, pod *v1.Pod) (*framework.PreFilterResult, *framework.Status) {
	state := state{}
	defer func() {
		cycleState.Write(stateKey, &state)
	}()

	modelName := pod.Labels[modelNameLabelKey]
	if modelName == "" {
		state.shouldSkip = true
		return nil, framework.NewStatus(framework.Skip)
	}

	err := rf.calPreFilterState(ctx, pod, &state)
	if err != nil {
		return nil, framework.AsStatus(err)
	}

	if state.shouldSkip {
		return nil, framework.NewStatus(framework.Skip)
	}
	return nil, nil
}

func (rf *ResourceFungibility) calPreFilterState(ctx context.Context, pod *v1.Pod, s *state) error {
	gvr := schema.GroupVersionResource{
		Group:    group,
		Version:  version,
		Resource: resource,
	}

	// modelName shouldn't be empty here, early returned, it's a cluster-scope obj.
	modelName := pod.Labels[modelNameLabelKey]
	unstructuredModel, err := rf.dynClient.Resource(gvr).Get(ctx, modelName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	model := &llmazcoreapi.OpenModel{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredModel.Object, model); err != nil {
		return err
	}

	if model.Spec.InferenceConfig == nil {
		s.shouldSkip = true
		return nil
	}

	for _, f := range model.Spec.InferenceConfig.Flavors {
		flavor := flavor{
			name:          string(f.Name),
			nodeSelectors: map[string]string{},
		}
		if len(f.NodeSelector) == 0 {
			// Once nodeSelector is empty, which means all nodes are potential candidates,
			// so we'll skip the Filter stage.
			s.shouldSkip = true
			return nil
		}

		for k, v := range f.NodeSelector {
			flavor.nodeSelectors[k] = v
		}
		s.inferenceFlavors = append(s.inferenceFlavors, flavor)
	}
	return nil
}

func (rf *ResourceFungibility) PreFilterExtensions() framework.PreFilterExtensions {
	return nil
}

func (rf *ResourceFungibility) Filter(ctx context.Context, cycleState *framework.CycleState, pod *v1.Pod, nodeInfo *framework.NodeInfo) *framework.Status {
	state, err := getState(cycleState)
	if err != nil {
		return framework.AsStatus(err)
	}

	for _, flavor := range state.inferenceFlavors {
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

func (rf *ResourceFungibility) PreScore(ctx context.Context, cycleState *framework.CycleState, pod *v1.Pod, nodes []*framework.NodeInfo) *framework.Status {
	state, err := getState(cycleState)
	if err != nil {
		return framework.AsStatus(err)
	}
	if state.shouldSkip {
		return framework.NewStatus(framework.Skip)
	}
	return nil
}

func (rf *ResourceFungibility) Score(ctx context.Context, cycleState *framework.CycleState, p *v1.Pod, nodeInfo *framework.NodeInfo) (int64, *framework.Status) {
	state, err := getState(cycleState)
	if err != nil {
		return 0, framework.AsStatus(err)
	}

	node := nodeInfo.Node()

	fmt.Println("Starting to calculate scores")

	for i, flavor := range state.inferenceFlavors {
		fmt.Println("flavor name is %s", flavor.name)
		for k, v := range flavor.nodeSelectors {
			fmt.Println("node label is %s, %s", k, v)
			value, ok := node.Labels[k]
			if ok && value == v {
				fmt.Println("per score is %d", scoreWeights[i])
				fmt.Println("final scores is %d", float32(scoreWeights[i])/float32(totalWeights)*100)
				// Find the first matched node flavor.
				return int64(math.Round(float64(scoreWeights[i]) / float64(totalWeights) * 100)), nil
			}
		}
	}

	fmt.Println("No matched flavor found, unexpected.")
	// We should not reach here.
	return 0, nil
}

func (rf *ResourceFungibility) ScoreExtensions() framework.ScoreExtensions {
	return nil
}

func getState(cycleState *framework.CycleState) (*state, error) {
	c, err := cycleState.Read(stateKey)
	if err != nil {
		return nil, fmt.Errorf("reading %q from cycleState: %w", stateKey, err)
	}

	s, ok := c.(*state)
	if !ok {
		return nil, fmt.Errorf("%+v convert to resourceFungibility.state error", c)
	}
	return s, nil
}
