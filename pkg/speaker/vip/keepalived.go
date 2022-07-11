package vip

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"github.com/openelb/openelb/pkg/constant"
	"github.com/openelb/openelb/pkg/speaker"
	"github.com/openelb/openelb/pkg/util"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
)

type KeepAlived struct {
	log       logr.Logger
	clientset *clientset.Clientset
	cm        *corev1.ConfigMap
	conf      *KeepAlivedConfig
}

type KeepAlivedConfig struct {
	Args []string
}

func (k *KeepAlived) SetBalancer(configMap string, nexthops []corev1.Node) error {
	ip := strings.SplitN(configMap, ":", 2)
	k.cm = &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      constant.OpenELBConfigMap,
			Namespace: util.EnvNamespace(),
		},
		Data: map[string]string{
			ip[0]: ip[1],
		},
	}
	k.log.Info(fmt.Sprintf("cm %s", k.cm.String()))
	var err error
	if oldCm, err := k.clientset.CoreV1().ConfigMaps(k.cm.ObjectMeta.Namespace).Get(context.TODO(), k.cm.ObjectMeta.Name, metav1.GetOptions{}); errors.IsNotFound(err) {
		k.cm, err = k.clientset.CoreV1().ConfigMaps(k.cm.ObjectMeta.Namespace).Create(context.TODO(), k.cm, metav1.CreateOptions{})
		k.log.Info(fmt.Sprintf("create cm %s", k.cm.ObjectMeta.Name))
		if err != nil {
			k.log.Error(err, "create cm error")
		}
	} else {
		for oldk, oldv := range oldCm.Data {
			if _, ok := k.cm.Data[oldk]; !ok {
				k.cm.Data[oldk] = oldv
			}
		}
		k.cm, err = k.clientset.CoreV1().ConfigMaps(k.cm.Namespace).Update(context.TODO(), k.cm, metav1.UpdateOptions{})
	}

	return err
}

func (k *KeepAlived) DelBalancer(configMap string) error {
	var err error
	if _, err = k.clientset.CoreV1().ConfigMaps(k.cm.ObjectMeta.Namespace).Get(context.TODO(), k.cm.ObjectMeta.Name, metav1.GetOptions{}); err == nil {
		ip := strings.SplitN(configMap, ":", 2)
		delete(k.cm.Data, ip[0])
		k.cm, err = k.clientset.CoreV1().ConfigMaps(k.cm.ObjectMeta.Namespace).Update(context.TODO(), k.cm, metav1.UpdateOptions{})
	}
	return err
}

func (k *KeepAlived) Start(stopCh <-chan struct{}) error {
	dsClient := k.clientset.AppsV1().DaemonSets(util.EnvNamespace())
	ds, err := dsClient.Get(context.TODO(), constant.OpenELBVipName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			ds, err = dsClient.Create(context.TODO(), k.generateVIPDaemonSet(), metav1.CreateOptions{})
			if err != nil {
				k.log.Error(err, "keepalived daemonSet create error")
				return err
			}
			k.log.Info(fmt.Sprintf("keepalived daemonSet %s created successfully", ds.Name))
		} else {
			k.log.Error(err, "keepalived daemonSet get error")
			return err
		}
	}

	go func() {
		select {
		case <-stopCh:
			deletePolicy := metav1.DeletePropagationForeground
			if err = dsClient.Delete(context.TODO(), ds.Name, metav1.DeleteOptions{
				PropagationPolicy: &deletePolicy,
			}); err != nil {
				k.log.Error(err, "keepalived daemonSet delete error")
			}
		}
	}()

	return err
}

// User can config NodeProxy by ConfigMap to specify the proxy and forward images
// If the ConfigMap exists and the configuration is set, use it,
// 	otherwise, use the default image got from constants.
func (k *KeepAlived) getConfig() (*corev1.ConfigMap, error) {
	return k.clientset.CoreV1().ConfigMaps(util.EnvNamespace()).Get(context.Background(), constant.OpenELBConfigMap, metav1.GetOptions{})
}

func (k *KeepAlived) getImage() string {
	cm, err := k.getConfig()
	if err != nil {
		return constant.OpenELBKeepAliveImageName
	}
	return cm.Data[constant.OpenELBKeepAliveImageConfigMapKey]
}

func (k *KeepAlived) generateVIPDaemonSet() *appv1.DaemonSet {
	var privileged = true
	return &appv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constant.OpenELBVipName,
			Namespace: util.EnvNamespace(),
			Labels: map[string]string{
				"app": constant.OpenELBVipName,
			},
		},
		Spec: appv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": constant.OpenELBVipName,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{
					"app": constant.OpenELBVipName,
				}},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "modules",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/lib/modules",
								},
							},
						},
						{
							Name: "dev",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/dev",
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Image:           k.getImage(),
							Name:            constant.OpenELBVipName,
							ImagePullPolicy: corev1.PullIfNotPresent,
							SecurityContext: &corev1.SecurityContext{
								Privileged: &privileged,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									MountPath: "/lib/modules",
									Name:      "modules",
									ReadOnly:  true,
								},
								{
									MountPath: "/dev",
									Name:      "dev",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name: "POD_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
								{
									Name: "POD_NAMESPACE",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.namespace",
										},
									},
								},
							},
							Args: k.conf.Args,
						},
					},
					ServiceAccountName: constant.OpenELBServiceAccountName,
					HostNetwork:        true,
				},
			},
		},
	}
}

var _ speaker.Speaker = &KeepAlived{}

func NewKeepAlived(client *clientset.Clientset, conf *KeepAlivedConfig) *KeepAlived {
	return &KeepAlived{
		log:       ctrl.Log.WithName("keepalived"),
		clientset: client,
		conf:      conf,
	}
}
