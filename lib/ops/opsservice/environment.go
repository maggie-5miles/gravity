/*
Copyright 2018 Gravitational, Inc.

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

package opsservice

import (
	"context"

	"github.com/gravitational/gravity/lib/constants"
	"github.com/gravitational/gravity/lib/defaults"
	"github.com/gravitational/gravity/lib/kubernetes"
	"github.com/gravitational/gravity/lib/ops"
	"github.com/gravitational/gravity/lib/storage"

	"github.com/gravitational/rigging"
	"github.com/gravitational/trace"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// GetClusterEnvironmentVariables retrieves the cluster environment variables
func (o *Operator) GetClusterEnvironmentVariables(key ops.SiteKey) (env storage.EnvironmentVariables, err error) {
	client, err := o.GetKubeClient()
	if err != nil {
		return nil, trace.Wrap(err)
	}
	configmap, err := client.CoreV1().ConfigMaps(defaults.KubeSystemNamespace).
		Get(constants.ClusterEnvironmentMap, metav1.GetOptions{})
	if err != nil {
		return nil, rigging.ConvertError(err)
	}
	env = storage.NewEnvironment(configmap.Data)
	return env, nil
}

// UpdateClusterEnvironmentVariables updates environment variables in cluster.
func (o *Operator) UpdateClusterEnvironmentVariables(req ops.UpdateClusterEnvironmentVariablesRequest) error {
	client, err := o.GetKubeClient()
	if err != nil {
		return trace.Wrap(err)
	}
	configmaps := client.CoreV1().ConfigMaps(defaults.KubeSystemNamespace)
	err = kubernetes.Retry(context.TODO(), func() error {
		return trace.Wrap(updateClusterEnvironment(configmaps, req.Env.GetKeyValues()))
	})
	return trace.Wrap(err)
}

func updateClusterEnvironment(client corev1.ConfigMapInterface, keyValues map[string]string) error {
	configmap, err := client.Get(constants.ClusterEnvironmentMap, metav1.GetOptions{})
	if err != nil {
		if !trace.IsNotFound(rigging.ConvertError(err)) {
			return trace.Wrap(err)
		}
		err = rigging.ConvertError(err)
	}
	if trace.IsNotFound(err) {
		configmap = &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      constants.ClusterEnvironmentMap,
				Namespace: defaults.KubeSystemNamespace,
			},
			Data: keyValues,
		}
		configmap, err = client.Create(configmap)
		if err != nil {
			return trace.Wrap(err)
		}
	} else {
		configmap.Data = keyValues
	}
	_, err = client.Update(configmap)
	if err != nil {
		return trace.Wrap(err)
	}
	return nil
}
