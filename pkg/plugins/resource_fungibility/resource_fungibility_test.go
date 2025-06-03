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
	"testing"

	"github.com/google/go-cmp/cmp"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/informers"
	clientsetfake "k8s.io/client-go/kubernetes/fake"
	restclient "k8s.io/client-go/rest"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/defaultbinder"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/queuesort"
	frameworkruntime "k8s.io/kubernetes/pkg/scheduler/framework/runtime"
	tf "k8s.io/kubernetes/pkg/scheduler/testing/framework"

	llmazcoreapi "github.com/inftyai/llmaz/api/core/v1alpha1"
)

var (
	registeredPlugins = []tf.RegisterPluginFunc{
		tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
		tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
	}
)

func TestResourceFungibility_Filter(t *testing.T) {
	tests := []struct {
		name       string
		pod        *v1.Pod
		nodeLabels map[string]string
		model      *llmazcoreapi.OpenModel

		wantFilterStatus    *framework.Status
		wantPreFilterStatus *framework.Status
	}{
		{
			name:                "pod without model label",
			pod:                 &v1.Pod{},
			wantPreFilterStatus: framework.NewStatus(framework.Skip),
		},
		{
			name: "model not found",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{modelNameLabelKey: "test-model"},
				},
			},
			wantPreFilterStatus: framework.AsStatus(apierrors.NewNotFound(schema.GroupResource{Group: group, Resource: resource}, "test-model")),
		},
		{
			name: "model without inference config",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{modelNameLabelKey: "test-model"},
				},
			},
			model: &llmazcoreapi.OpenModel{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "llmaz.io/v1alpha1",
					Kind:       "OpenModel",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-model",
				},
			},
			wantPreFilterStatus: framework.NewStatus(framework.Skip),
		},
		{
			name: "model without flavors",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{modelNameLabelKey: "test-model"},
				},
			},
			model: &llmazcoreapi.OpenModel{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "llmaz.io/v1alpha1",
					Kind:       "OpenModel",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-model",
				},
				Spec: llmazcoreapi.ModelSpec{
					InferenceConfig: &llmazcoreapi.InferenceConfig{
						Flavors: []llmazcoreapi.Flavor{},
					},
				},
			},
			wantPreFilterStatus: framework.NewStatus(framework.Skip),
		},
		{
			name: "model has flavors but at least one flavor has empty nodeSelector",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{modelNameLabelKey: "test-model"},
				},
			},
			model: &llmazcoreapi.OpenModel{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "llmaz.io/v1alpha1",
					Kind:       "OpenModel",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-model",
				},
				Spec: llmazcoreapi.ModelSpec{
					InferenceConfig: &llmazcoreapi.InferenceConfig{
						Flavors: []llmazcoreapi.Flavor{
							{
								Name: "none",
							},
							{
								Name:         "empty",
								NodeSelector: map[string]string{},
							},
							{
								Name:         "t4",
								NodeSelector: map[string]string{"karpenter.k8s.aws/instance-gpu-name": "t4"},
							},
						},
					},
				},
			},
			wantPreFilterStatus: framework.NewStatus(framework.Skip),
		},
		{
			name: "mismatched flavors between pod and model",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      map[string]string{modelNameLabelKey: "test-model"},
					Annotations: map[string]string{inferenceServiceFlavorsAnnoKey: "a10"},
				},
			},
			model: &llmazcoreapi.OpenModel{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "llmaz.io/v1alpha1",
					Kind:       "OpenModel",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-model",
				},
				Spec: llmazcoreapi.ModelSpec{
					InferenceConfig: &llmazcoreapi.InferenceConfig{
						Flavors: []llmazcoreapi.Flavor{
							{
								Name: "none",
							},
							{
								Name:         "empty",
								NodeSelector: map[string]string{},
							},
							{
								Name:         "t4",
								NodeSelector: map[string]string{"karpenter.k8s.aws/instance-gpu-name": "t4"},
							},
						},
					},
				},
			},
			wantPreFilterStatus: framework.NewStatus(framework.Skip, "flavor \"a10\" not found in model \"test-model\""),
		},
		{
			name: "no suitable node for model flavors",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{modelNameLabelKey: "test-model"},
				},
			},
			model: &llmazcoreapi.OpenModel{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "llmaz.io/v1alpha1",
					Kind:       "OpenModel",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-model",
				},
				Spec: llmazcoreapi.ModelSpec{
					InferenceConfig: &llmazcoreapi.InferenceConfig{
						Flavors: []llmazcoreapi.Flavor{
							{
								Name:         "t4",
								NodeSelector: map[string]string{"karpenter.k8s.aws/instance-gpu-name": "t4"},
							},
						},
					},
				},
			},
			wantFilterStatus: framework.NewStatus(framework.UnschedulableAndUnresolvable),
		},
		{
			name: "no suitable node for inference service flavors",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      map[string]string{modelNameLabelKey: "test-model"},
					Annotations: map[string]string{inferenceServiceFlavorsAnnoKey: "a100"},
				},
			},
			nodeLabels: map[string]string{"karpenter.k8s.aws/instance-gpu-name": "t4"},
			model: &llmazcoreapi.OpenModel{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "llmaz.io/v1alpha1",
					Kind:       "OpenModel",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-model",
				},
				Spec: llmazcoreapi.ModelSpec{
					InferenceConfig: &llmazcoreapi.InferenceConfig{
						Flavors: []llmazcoreapi.Flavor{
							{
								Name:         "t4",
								NodeSelector: map[string]string{"karpenter.k8s.aws/instance-gpu-name": "t4"},
							},
							{
								Name:         "a100",
								NodeSelector: map[string]string{"karpenter.k8s.aws/instance-gpu-name": "a100"},
							},
						},
					},
				},
			},
			wantFilterStatus: framework.NewStatus(framework.UnschedulableAndUnresolvable),
		},
		{
			name: "suitable node for both model and inference service flavors",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      map[string]string{modelNameLabelKey: "test-model"},
					Annotations: map[string]string{inferenceServiceFlavorsAnnoKey: "a100"},
				},
			},
			nodeLabels: map[string]string{"karpenter.k8s.aws/instance-gpu-name": "a100"},
			model: &llmazcoreapi.OpenModel{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "llmaz.io/v1alpha1",
					Kind:       "OpenModel",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-model",
				},
				Spec: llmazcoreapi.ModelSpec{
					InferenceConfig: &llmazcoreapi.InferenceConfig{
						Flavors: []llmazcoreapi.Flavor{
							{
								Name:         "t4",
								NodeSelector: map[string]string{"karpenter.k8s.aws/instance-gpu-name": "t4"},
							},
							{
								Name:         "a100",
								NodeSelector: map[string]string{"karpenter.k8s.aws/instance-gpu-name": "a100"},
							},
						},
					},
				},
			},
		},
		{
			name: "suitable node for model flavors",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{modelNameLabelKey: "test-model"},
				},
			},
			nodeLabels: map[string]string{"karpenter.k8s.aws/instance-gpu-name": "a100"},
			model: &llmazcoreapi.OpenModel{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "llmaz.io/v1alpha1",
					Kind:       "OpenModel",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-model",
				},
				Spec: llmazcoreapi.ModelSpec{
					InferenceConfig: &llmazcoreapi.InferenceConfig{
						Flavors: []llmazcoreapi.Flavor{
							{
								Name:         "t4",
								NodeSelector: map[string]string{"karpenter.k8s.aws/instance-gpu-name": "t4"},
							},
							{
								Name:         "a100",
								NodeSelector: map[string]string{"karpenter.k8s.aws/instance-gpu-name": "a100"},
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			node := v1.Node{ObjectMeta: metav1.ObjectMeta{
				Labels: tc.nodeLabels,
			}}
			nodeInfo := framework.NewNodeInfo()
			nodeInfo.SetNode(&node)

			fr, err := tf.NewFramework(ctx, registeredPlugins, Name, frameworkruntime.WithClientSet(clientsetfake.NewSimpleClientset()))
			if err != nil {
				t.Fatalf("failed to create framework: %v", err)
			}

			scheme := runtime.NewScheme()
			dynClient := dynamicfake.NewSimpleDynamicClient(scheme)
			if tc.model != nil {
				dynClient = dynamicfake.NewSimpleDynamicClient(scheme, tc.model)
			}

			p := &ResourceFungibility{
				handle:    fr,
				dynClient: dynClient,
			}

			state := framework.NewCycleState()
			_, gotStatus := p.PreFilter(ctx, state, tc.pod)
			// As we cannot compare two errors directly due to miss the equal method for how to compare two errors, so just need to compare the reasons.
			if gotStatus.Code() == framework.Error {
				if diff := cmp.Diff(tc.wantPreFilterStatus.Reasons(), gotStatus.Reasons()); diff != "" {
					t.Errorf("unexpected PreFilter status (-want, +got):\n%s", diff)
				}
			} else {
				if diff := cmp.Diff(tc.wantPreFilterStatus, gotStatus); diff != "" {
					t.Errorf("unexpected PreFilter status (-want, +got):\n%s", diff)
				}
			}
			// If PreFilter fails, then Filter will not run.
			if tc.wantPreFilterStatus.IsSuccess() {
				gotStatus = p.Filter(ctx, state, tc.pod, nodeInfo)
				if diff := cmp.Diff(tc.wantFilterStatus, gotStatus); diff != "" {
					t.Errorf("unexpected Filter status (-want, +got):\n%s", diff)
				}
			}
		})
	}
}

func TestNew(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fakeclient := clientsetfake.NewSimpleClientset()
	fr, err := tf.NewFramework(ctx, registeredPlugins, Name,
		frameworkruntime.WithInformerFactory(informers.NewSharedInformerFactory(fakeclient, 0)),
		frameworkruntime.WithKubeConfig(&restclient.Config{}),
		frameworkruntime.WithClientSet(fakeclient))
	if err != nil {
		t.Error(err)
	}

	pl, err := New(ctx, nil, fr)
	if err != nil {
		t.Fatalf("failed to create plugin: %v", err)
	}
	if pl == nil {
		t.Fatalf("plugin is nil")
	}
}
