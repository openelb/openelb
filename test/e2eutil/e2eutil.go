// Copyright 2018 The Kubesphere Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package e2eutil

import (
	"context"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strconv"
	"time"

	"github.com/kubesphere/porter/pkg/kubeutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func WaitForController(c client.Client, namespace, name string, retryInterval, timeout time.Duration) error {
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		controller := &appsv1.Deployment{}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		err = c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, controller)
		if apierrors.IsNotFound(err) {
			log.Println("Cannot find controller")
			return false, nil
		}
		if err != nil {
			return false, err
		}
		if controller.Status.ReadyReplicas == 1 {
			log.Println("Controller is not ready")
			return true, nil
		}
		return false, nil
	})
	return err
}

func WaitForDeletion(dynclient client.Client, obj runtime.Object, retryInterval, timeout time.Duration) error {
	err := dynclient.Delete(context.TODO(), obj)
	if err != nil {
		return err
	}
	key, err := client.ObjectKeyFromObject(obj)
	if err != nil {
		return err
	}

	kind := obj.GetObjectKind().GroupVersionKind().Kind
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	err = wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		err = dynclient.Get(ctx, key, obj)
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		if err != nil {
			return false, err
		}
		fmt.Printf("Waiting for %s %s to be deleted\n", kind, key)
		return false, nil
	})
	if err != nil {
		return err
	}
	fmt.Printf("%s %s was deleted\n", kind, key)
	return nil
}

func GetLogOfPod(rest *rest.RESTClient, namespace, name string, logOptions *corev1.PodLogOptions, out io.Writer) error {
	req := rest.Get().Namespace(namespace).Name(name).SubResource("log").Param("follow", strconv.FormatBool(logOptions.Follow)).
		Param("container", logOptions.Container).
		Param("previous", strconv.FormatBool(logOptions.Previous)).
		Param("timestamps", strconv.FormatBool(logOptions.Timestamps))
	if logOptions.SinceSeconds != nil {
		req.Param("sinceSeconds", strconv.FormatInt(*logOptions.SinceSeconds, 10))
	}
	if logOptions.SinceTime != nil {
		req.Param("sinceTime", logOptions.SinceTime.Format(time.RFC3339))
	}
	if logOptions.LimitBytes != nil {
		req.Param("limitBytes", strconv.FormatInt(*logOptions.LimitBytes, 10))
	}
	if logOptions.TailLines != nil {
		req.Param("tailLines", strconv.FormatInt(*logOptions.TailLines, 10))
	}
	readCloser, err := req.Stream()
	if err != nil {
		return err
	}
	defer readCloser.Close()
	_, err = io.Copy(out, readCloser)
	return err
}

func GetServiceNodesIP(c client.Client, namespaceName types.NamespacedName) ([]string, error) {
	service := &corev1.Service{}
	err := c.Get(context.TODO(), namespaceName, service)
	if err != nil {
		return nil, err
	}
	return kubeutil.GetServiceNodesIP(c, service)
}

func KubectlApply(filename string) error {
	cmd := exec.Command("kubectl", "apply", "-f", filename)
	str, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("kubectl apply failed, error :%s\n", str)
	}
	return err
}

func KubectlDelete(filename string) error {
	ctx, cancle := context.WithTimeout(context.Background(), time.Second*30)
	cmd := exec.CommandContext(ctx, "kubectl", "delete", "-f", filename)
	defer cancle()
	output, err := cmd.CombinedOutput()
	log.Println(string(output))
	return err
}

func DeleteNamespace(c client.Client, ns string) error {
	namespace := &corev1.Namespace{}
	namespace.Name = ns
	return c.Delete(context.TODO(), namespace)
}
