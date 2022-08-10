package vip

import (
	"context"
	"fmt"
	"reflect"
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
	log    logr.Logger
	client *clientset.Clientset
	data   map[string]string
	conf   *KeepAlivedConfig
}

type KeepAlivedConfig struct {
	Args []string
}

func (k *KeepAlived) SetBalancer(vip string, _ []corev1.Node) error {
	ip := strings.SplitN(vip, ":", 2)

	if svc, ok := k.data[ip[0]]; ok {
		//TODO: support proxy: https://github.com/aledbf/kube-keepalived-vip#proxy-protocol-mode
		svcArray := strings.Split(svc, ";")
		exist := false
		for _, s := range svcArray {
			if s == ip[1] {
				exist = true
				break
			}
		}
		if !exist {
			k.data[ip[0]] = svc + ";" + ip[1]
		}
	} else {
		k.data[ip[0]] = ip[1]
	}

	return k.updateConfigMap()
}

func (k *KeepAlived) DelBalancer(vip string) error {
	ip := strings.SplitN(vip, ":", 2)
	if len(ip) == 1 {
		delete(k.data, ip[0])
		return k.updateConfigMap()
	}

	if svc, ok := k.data[ip[0]]; ok {
		svcArray := strings.Split(svc, ";")
		if len(svcArray) == 1 {
			delete(k.data, ip[0])
		} else {
			for i, s := range svcArray {
				if s == ip[1] {
					svcArray = append(svcArray[:i], svcArray[i+1:]...)
					break
				}
			}
			k.data[ip[0]] = strings.Join(svcArray, ";")
		}
	}

	return k.updateConfigMap()
}

func (k *KeepAlived) updateConfigMap() error {
	ctx := context.Background()
	cm, err := k.client.CoreV1().ConfigMaps(util.EnvNamespace()).Get(ctx, constant.OpenELBVipConfigMap, metav1.GetOptions{})
	if err == nil {
		if reflect.DeepEqual(cm.Data, k.data) {
			return nil
		}

		cm.Data = k.data
		_, err = k.client.CoreV1().ConfigMaps(util.EnvNamespace()).Update(ctx, cm, metav1.UpdateOptions{})
		return err
	}

	if errors.IsNotFound(err) {
		cm = &corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      constant.OpenELBVipConfigMap,
				Namespace: util.EnvNamespace(),
			},
			Data: k.data,
		}

		k.log.Info(fmt.Sprintf("not found configmap:%s, so create it", cm.Name))
		_, err = k.client.CoreV1().ConfigMaps(cm.Namespace).Create(ctx, cm, metav1.CreateOptions{})
		return err
	}

	k.log.Error(err, fmt.Sprintf("get configmap:%s error", cm.Name))
	return err
}

func (k *KeepAlived) Start(stopCh <-chan struct{}) error {
	dsClient := k.client.AppsV1().DaemonSets(util.EnvNamespace())
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

	cmClient := k.client.CoreV1().ConfigMaps(util.EnvNamespace())
	cm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      constant.OpenELBVipConfigMap,
			Namespace: util.EnvNamespace(),
		},
	}
	k.log.Info(fmt.Sprintf("create ConfigMap %s", cm.Name))
	_, err = cmClient.Create(context.TODO(), cm, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		k.log.Error(err, fmt.Sprintf("create cm:%s error", cm.Name))
		return err
	}

	go func() {
		select {
		case <-stopCh:
			deletePolicy := metav1.DeletePropagationForeground
			deleteOpt := metav1.DeleteOptions{
				PropagationPolicy: &deletePolicy,
			}
			if err = dsClient.Delete(context.TODO(), ds.Name, deleteOpt); err != nil {
				k.log.Error(err, "keepalived daemonSet delete error")
			}

			if err = cmClient.Delete(context.TODO(), cm.Name, deleteOpt); err != nil {
				k.log.Error(err, "keepalived configMap delete error")
			}

			return
		}
	}()

	return nil
}

// User can config Keepalived by ConfigMap to specify the images
// If the ConfigMap exists and the configuration is set, use it,
// 	otherwise, use the default image got from constants.
func (k *KeepAlived) getConfig() (*corev1.ConfigMap, error) {
	return k.client.CoreV1().ConfigMaps(util.EnvNamespace()).
		Get(context.Background(), constant.OpenELBImagesConfigMap, metav1.GetOptions{})
}

func (k *KeepAlived) getImage() string {
	cm, err := k.getConfig()
	if err != nil {
		return constant.OpenELBDefaultKeepAliveImage
	}

	image, exist := cm.Data[constant.OpenELBKeepAliveImage]
	if !exist {
		return constant.OpenELBDefaultKeepAliveImage
	}
	return image
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
		log:    ctrl.Log.WithName("keepalived"),
		client: client,
		conf:   conf,
		data:   map[string]string{},
	}
}
