package framework

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	e2epod "k8s.io/kubernetes/test/e2e/framework/pod"
)

type PodClient struct {
	f *Framework
	*e2epod.PodClient
}

func (f *Framework) PodClient() *PodClient {
	return f.PodClientNS(f.Namespace.Name)
}

func (f *Framework) PodClientNS(namespace string) *PodClient {
	return &PodClient{f, e2epod.PodClientNS(f.Framework, namespace)}
}

func (c *PodClient) Create(pod *corev1.Pod) *corev1.Pod {
	return c.PodClient.Create(pod)
}

func (c *PodClient) CreateSync(pod *corev1.Pod) *corev1.Pod {
	return c.PodClient.CreateSync(pod)
}

func (c *PodClient) Delete(name string) error {
	return c.PodClient.Delete(context.Background(), name, metav1.DeleteOptions{})
}

func (c *PodClient) DeleteSync(name string) {
	c.PodClient.DeleteSync(name, metav1.DeleteOptions{}, timeout)
}

func (c *PodClient) WaitForRunning(name string) {
	err := e2epod.WaitTimeoutForPodRunningInNamespace(c.f.ClientSet, name, c.f.Namespace.Name, timeout)
	ExpectNoError(err)
}

func (c *PodClient) WaitForNotFound(name string) {
	err := e2epod.WaitForPodNotFoundInNamespace(c.f.ClientSet, name, c.f.Namespace.Name, timeout)
	ExpectNoError(err)
}

func MakePod(ns, name string, labels, annotations map[string]string, image string, command, args []string) *corev1.Pod {
	if image == "" {
		image = PauseImage
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   ns,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: corev1.PodSpec{
			HostNetwork: true,
			Containers: []corev1.Container{
				{
					Name:            "container",
					Image:           image,
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command:         command,
					Args:            args,
				},
			},
		},
	}
	pod.Spec.TerminationGracePeriodSeconds = new(int64)
	*pod.Spec.TerminationGracePeriodSeconds = 3

	return pod
}
