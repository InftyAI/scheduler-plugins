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

package util

import (
	"context"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	frameworkruntime "k8s.io/kubernetes/pkg/scheduler/framework/runtime"
)

func SetupPluginWithInformers(
	ctx context.Context,
	pf frameworkruntime.PluginFactory,
	config runtime.Object,
	sharedLister framework.SharedLister,
	objs []runtime.Object,
) (framework.Plugin, error) {
	objs = append([]runtime.Object{&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ""}}}, objs...)
	informerFactory := informers.NewSharedInformerFactory(fake.NewClientset(objs...), 0)
	fh, err := frameworkruntime.NewFramework(ctx, nil, nil,
		frameworkruntime.WithSnapshotSharedLister(sharedLister),
		frameworkruntime.WithInformerFactory(informerFactory))
	if err != nil {
		return nil, err
	}
	p, err := pf(ctx, config, fh)
	if err != nil {
		return nil, err
	}
	informerFactory.Start(ctx.Done())
	informerFactory.WaitForCacheSync(ctx.Done())
	return p, nil
}
