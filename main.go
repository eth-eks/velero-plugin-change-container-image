/*
Copyright 2017, 2019 the Velero contributors and CloudBees.

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
	"github.com/eth-eks/velero-plugin-change-container-image/internal/plugin"
	"github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/velero/pkg/plugin/framework"
)

func main() {
	framework.NewServer().
		RegisterRestoreItemActionV2("eth-eks/change-container-image", newRestorePluginV2).
		Serve()
}

func newRestorePluginV2(logger logrus.FieldLogger) (interface{}, error) {
	return plugin.NewRestorePluginV2(logger), nil
}
