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

package main

import (
	"os"

	// Ensure scheme package is initialized.
	_ "github.com/inftyai/scheduler/api/config/scheme"

	"k8s.io/component-base/cli"
	"k8s.io/kubernetes/cmd/kube-scheduler/app"

	resourceFungibility "github.com/inftyai/scheduler/pkg/plugins/resource_fungibility"
	//+kubebuilder:scaffold:imports
)

func main() {
	command := app.NewSchedulerCommand(
		app.WithPlugin(resourceFungibility.Name, resourceFungibility.New),
	)

	code := cli.Run(command)
	os.Exit(code)
}
