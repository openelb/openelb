package ipam

import (
	"context"
	networkv1alpha2 "github.com/openelb/openelb/api/v1alpha2"
	"github.com/openelb/openelb/pkg/constant"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
	"time"
)

var schemev6 = runtime.NewScheme()

func init() {
	_ = v1.AddToScheme(schemev6)
	_ = networkv1alpha2.AddToScheme(schemev6)
}

func TestManager_ConstructAllocateV6(t *testing.T) {
	tests := []struct {
		name         string
		eip          []*networkv1alpha2.Eip
		svc          *v1.Service
		wantErr      bool
		wantAllocate *svcRecord
		wantRelease  *svcRecord
	}{
		{
			name: "svc is nil",
			eip: []*networkv1alpha2.Eip{{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "eip",
				},
				Spec: networkv1alpha2.EipSpec{
					Address: "2001:0db8::0/120",
				},
				Status: networkv1alpha2.EipStatus{},
			}},
		},

		{
			name: "clusterIP service",
			eip: []*networkv1alpha2.Eip{{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "eip",
				},
				Spec: networkv1alpha2.EipSpec{
					Address:  "2001:0db8::0/120",
					Protocol: constant.OpenELBProtocolLayer2,
				},
				Status: networkv1alpha2.EipStatus{
					Used: map[string]string{
						"2001:0db8::0": "test/test",
					},
				},
			}},
			svc: &v1.Service{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testsvc1",
					Namespace: "default",
					Annotations: map[string]string{
						constant.OpenELBEIPAnnotationKeyV1Alpha2: "eip",
						constant.OpenELBAnnotationKey:            constant.OpenELBAnnotationValue,
					},
				},
				Spec: v1.ServiceSpec{
					Type: v1.ServiceTypeClusterIP,
					Ports: []v1.ServicePort{
						{
							Port:       80,
							TargetPort: intstr.FromInt(80),
						},
					},
				},
				Status: v1.ServiceStatus{},
			},
			wantErr:      false,
			wantAllocate: nil,
		},

		{
			name: "clusterIP service - svc status ingress has ip",
			eip: []*networkv1alpha2.Eip{{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "eip",
				},
				Spec: networkv1alpha2.EipSpec{
					Address:  "2001:0db8::0/120",
					Protocol: constant.OpenELBProtocolLayer2,
				},
				Status: networkv1alpha2.EipStatus{
					Used: map[string]string{
						"2001:0db8::0": "default/testsvc",
					},
				},
			}},
			svc: &v1.Service{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testsvc",
					Namespace: "default",
					Annotations: map[string]string{
						constant.OpenELBEIPAnnotationKeyV1Alpha2: "eip",
						constant.OpenELBAnnotationKey:            constant.OpenELBAnnotationValue,
					},
				},
				Spec: v1.ServiceSpec{
					Type: v1.ServiceTypeClusterIP,
					Ports: []v1.ServicePort{
						{
							Port:       80,
							TargetPort: intstr.FromInt(80),
						},
					},
				},
				Status: v1.ServiceStatus{
					LoadBalancer: v1.LoadBalancerStatus{Ingress: []v1.LoadBalancerIngress{{IP: "2001:0db8::0"}}},
				},
			},
			wantErr:      false,
			wantAllocate: nil,
			wantRelease: &svcRecord{
				Key: "default/testsvc",
				Eip: "eip",
				IP:  "2001:0db8::0",
			},
		},

		{
			name: "loadbalancer service but no specify openelb annotions completely",
			eip: []*networkv1alpha2.Eip{{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "eip",
				},
				Spec: networkv1alpha2.EipSpec{
					Address:  "2001:0db8::0/120",
					Protocol: constant.OpenELBProtocolLayer2,
				},
				Status: networkv1alpha2.EipStatus{
					Used: map[string]string{
						"2001:0db8::0": "default/testsvc",
					},
				},
			}},
			svc: &v1.Service{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testsvc",
					Namespace: "default",
					Annotations: map[string]string{
						constant.OpenELBAnnotationKey: constant.OpenELBAnnotationValue,
					},
				},
				Spec: v1.ServiceSpec{
					Type: v1.ServiceTypeLoadBalancer,
					Ports: []v1.ServicePort{
						{
							Port:       80,
							TargetPort: intstr.FromInt(80),
						},
					},
				},
				Status: v1.ServiceStatus{},
			},
			wantErr: false,
			wantRelease: &svcRecord{
				Key: "default/testsvc",
				Eip: "eip",
				IP:  "2001:0db8::0",
			},
		},

		{
			name: "loadbalancer service - default eip",
			eip: []*networkv1alpha2.Eip{{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "eip",
					Annotations: map[string]string{
						constant.OpenELBEIPAnnotationDefaultPool: "true",
					},
				},
				Spec: networkv1alpha2.EipSpec{
					Address:  "2001:0db8::0/120",
					Protocol: constant.OpenELBProtocolLayer2,
				},
				Status: networkv1alpha2.EipStatus{
					Used: map[string]string{
						"2001:0db8::0": "default/testsvc",
					},
				},
			}},
			svc: &v1.Service{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testsvc",
					Namespace: "default",
					Annotations: map[string]string{
						constant.OpenELBAnnotationKey: constant.OpenELBAnnotationValue,
					},
				},
				Spec: v1.ServiceSpec{
					Type: v1.ServiceTypeLoadBalancer,
					Ports: []v1.ServicePort{
						{
							Port:       80,
							TargetPort: intstr.FromInt(80),
						},
					},
				},
				Status: v1.ServiceStatus{},
			},
			wantErr: false,
			wantAllocate: &svcRecord{
				Key: "default/testsvc",
				Eip: "eip",
			},
			wantRelease: &svcRecord{
				Key: "default/testsvc",
				Eip: "eip",
				IP:  "2001:0db8::0",
			},
		},

		{
			name: "loadbalancer service with openelb annotions completely, but eip has no corresponding record",
			eip: []*networkv1alpha2.Eip{{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "eip",
				},
				Spec: networkv1alpha2.EipSpec{
					Address:  "2001:0db8::0/120",
					Protocol: constant.OpenELBProtocolLayer2,
				},
				Status: networkv1alpha2.EipStatus{},
			}},
			svc: &v1.Service{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testsvc",
					Namespace: "default",
					Annotations: map[string]string{
						constant.OpenELBEIPAnnotationKeyV1Alpha2: "eip",
						constant.OpenELBAnnotationKey:            constant.OpenELBAnnotationValue,
					},
				},
				Spec: v1.ServiceSpec{
					Type: v1.ServiceTypeLoadBalancer,
					Ports: []v1.ServicePort{
						{
							Port:       80,
							TargetPort: intstr.FromInt(80),
						},
					},
				},
				Status: v1.ServiceStatus{},
			},
			wantErr: false,
			wantAllocate: &svcRecord{
				Key: "default/testsvc",
				Eip: "eip",
				IP:  "",
			},
		},

		{
			name: "loadbalancer service with openelb annotions completely. eip has corresponding records",
			eip: []*networkv1alpha2.Eip{{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "eip",
				},
				Spec: networkv1alpha2.EipSpec{
					Address:  "2001:0db8::0/120",
					Protocol: constant.OpenELBProtocolLayer2,
				},
				Status: networkv1alpha2.EipStatus{
					Used: map[string]string{
						"2001:0db8::0": "default/testsvc",
					},
				},
			}},
			svc: &v1.Service{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testsvc",
					Namespace: "default",
					Annotations: map[string]string{
						constant.OpenELBEIPAnnotationKeyV1Alpha2: "eip",
						constant.OpenELBAnnotationKey:            constant.OpenELBAnnotationValue,
					},
				},
				Spec: v1.ServiceSpec{
					Type: v1.ServiceTypeLoadBalancer,
					Ports: []v1.ServicePort{
						{
							Port:       80,
							TargetPort: intstr.FromInt(80),
						},
					},
				},
				Status: v1.ServiceStatus{},
			},
			wantErr: false,
			wantAllocate: &svcRecord{
				Key: "default/testsvc",
				Eip: "eip",
				IP:  "",
			},
			wantRelease: &svcRecord{
				Key: "default/testsvc",
				Eip: "eip",
				IP:  "2001:0db8::0",
			},
		},

		{
			name: "specify service spec.loadbalanceIP, but eip has no corresponding record",
			eip: []*networkv1alpha2.Eip{{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "eip",
				},
				Spec: networkv1alpha2.EipSpec{
					Address:  "2001:0db8::0/120",
					Protocol: constant.OpenELBProtocolLayer2,
				},
				Status: networkv1alpha2.EipStatus{},
			}},
			svc: &v1.Service{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testsvc",
					Namespace: "default",
					Annotations: map[string]string{
						constant.OpenELBAnnotationKey:            constant.OpenELBAnnotationValue,
						constant.OpenELBEIPAnnotationKeyV1Alpha2: "eip",
					},
				},
				Spec: v1.ServiceSpec{
					LoadBalancerIP: "2001:0db8::1",
					Type:           v1.ServiceTypeLoadBalancer,
					Ports: []v1.ServicePort{
						{
							Port:       80,
							TargetPort: intstr.FromInt(80),
						},
					},
				},
				Status: v1.ServiceStatus{},
			},
			wantErr: false,
			wantAllocate: &svcRecord{
				Key: "default/testsvc",
				Eip: "eip",
				IP:  "2001:0db8::1",
			},
		},

		{
			name: "specify service spec.loadbalanceIP, but eip has corresponding records",
			eip: []*networkv1alpha2.Eip{{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "eip",
				},
				Spec: networkv1alpha2.EipSpec{
					Address:  "2001:0db8::0/120",
					Protocol: constant.OpenELBProtocolLayer2,
				},
				Status: networkv1alpha2.EipStatus{
					Used: map[string]string{
						"2001:0db8::1": "default/testsvc",
					},
				},
			}},
			svc: &v1.Service{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testsvc",
					Namespace: "default",
					Labels: map[string]string{
						constant.OpenELBEIPAnnotationKeyV1Alpha2: "eip",
					},
					Annotations: map[string]string{
						constant.OpenELBAnnotationKey:            constant.OpenELBAnnotationValue,
						constant.OpenELBEIPAnnotationKeyV1Alpha2: "eip",
					},
				},
				Spec: v1.ServiceSpec{
					LoadBalancerIP: "2001:0db8::1",
					Type:           v1.ServiceTypeLoadBalancer,
					Ports: []v1.ServicePort{
						{
							Port:       80,
							TargetPort: intstr.FromInt(80),
						},
					},
				},
				Status: v1.ServiceStatus{
					LoadBalancer: v1.LoadBalancerStatus{Ingress: []v1.LoadBalancerIngress{{IP: "2001:0db8:85a3::8a2e:0370:0001"}}},
				},
			},
			wantErr:      false,
			wantAllocate: nil,
		},

		{
			name: "specify service annotation.loadbalanceIP 1",
			eip: []*networkv1alpha2.Eip{{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "eip",
				},
				Spec: networkv1alpha2.EipSpec{
					Address:  "2001:0db8::0/120",
					Protocol: constant.OpenELBProtocolLayer2,
				},
				Status: networkv1alpha2.EipStatus{},
			}},
			svc: &v1.Service{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testsvc",
					Namespace: "default",
					Annotations: map[string]string{
						constant.OpenELBAnnotationKey:            constant.OpenELBAnnotationValue,
						constant.OpenELBEIPAnnotationKeyV1Alpha2: "eip",
						constant.OpenELBEIPAnnotationKey:         "2001:0db8::1",
					},
				},
				Spec: v1.ServiceSpec{
					Type: v1.ServiceTypeLoadBalancer,
					Ports: []v1.ServicePort{
						{
							Port:       80,
							TargetPort: intstr.FromInt(80),
						},
					},
				},
				Status: v1.ServiceStatus{},
			},
			wantErr: false,
			wantAllocate: &svcRecord{
				Key: "default/testsvc",
				Eip: "eip",
				IP:  "2001:0db8::1",
			},
		},

		{
			name: "specify service annotation.loadbalanceIP 2",
			eip: []*networkv1alpha2.Eip{{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "eip",
				},
				Spec: networkv1alpha2.EipSpec{
					Address:  "2001:0db8::0/120",
					Protocol: constant.OpenELBProtocolLayer2,
				},
				Status: networkv1alpha2.EipStatus{},
			}},
			svc: &v1.Service{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testsvc",
					Namespace: "default",
					Annotations: map[string]string{
						constant.OpenELBAnnotationKey:            constant.OpenELBAnnotationValue,
						constant.OpenELBEIPAnnotationKeyV1Alpha2: "eip",
						constant.OpenELBEIPAnnotationKey:         "2001:0db8::50",
					},
				},
				Spec: v1.ServiceSpec{
					LoadBalancerIP: "2001:0db8::50",
					Type:           v1.ServiceTypeLoadBalancer,
					Ports: []v1.ServicePort{
						{
							Port:       80,
							TargetPort: intstr.FromInt(80),
						},
					},
				},
				Status: v1.ServiceStatus{},
			},
			wantErr: false,
			wantAllocate: &svcRecord{
				Key: "default/testsvc",
				Eip: "eip",
				IP:  "2001:0db8::50",
			},
		},

		{
			name: "no eip",
			eip:  nil,
			svc: &v1.Service{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testsvc",
					Namespace: "default",
					Annotations: map[string]string{
						constant.OpenELBEIPAnnotationKeyV1Alpha2: "eip",
						constant.OpenELBAnnotationKey:            constant.OpenELBAnnotationValue,
					},
				},
				Spec: v1.ServiceSpec{
					Type: v1.ServiceTypeLoadBalancer,
					Ports: []v1.ServicePort{
						{
							Port:       80,
							TargetPort: intstr.FromInt(80),
						},
					},
				},
				Status: v1.ServiceStatus{},
			},
			wantErr: false,
			wantAllocate: &svcRecord{
				Key: "default/testsvc",
				Eip: "eip",
				IP:  "",
			},
		},

		// =============== release ===============
		{
			name: "service is deleting - no eip record",
			eip:  nil,
			svc: &v1.Service{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testsvc",
					Namespace: "default",
					Annotations: map[string]string{
						constant.OpenELBEIPAnnotationKeyV1Alpha2: "eip",
						constant.OpenELBAnnotationKey:            constant.OpenELBAnnotationValue,
					},
					Finalizers:        []string{constant.FinalizerName},
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
				},
				Spec: v1.ServiceSpec{
					Type: v1.ServiceTypeLoadBalancer,
					Ports: []v1.ServicePort{
						{
							Port:       80,
							TargetPort: intstr.FromInt(80),
						},
					},
				},
				Status: v1.ServiceStatus{
					LoadBalancer: v1.LoadBalancerStatus{Ingress: []v1.LoadBalancerIngress{{IP: "2001:0db8::0"}}},
				},
			},
			wantErr: false,
			wantRelease: &svcRecord{
				Key: "default/testsvc",
				IP:  "2001:0db8::0",
			},
		},

		{
			name: "service is deleting - eip has records",
			eip: []*networkv1alpha2.Eip{{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "eip",
				},
				Spec: networkv1alpha2.EipSpec{
					Address:  "2001:0db8::0/120",
					Protocol: constant.OpenELBProtocolLayer2,
				},
				Status: networkv1alpha2.EipStatus{
					Used: map[string]string{
						"2001:0db8::0": "default/testsvc",
					},
				},
			}},
			svc: &v1.Service{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testsvc",
					Namespace: "default",
					Annotations: map[string]string{
						constant.OpenELBEIPAnnotationKeyV1Alpha2: "eip",
						constant.OpenELBAnnotationKey:            constant.OpenELBAnnotationValue,
					},
					Finalizers:        []string{constant.FinalizerName},
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
				},
				Spec: v1.ServiceSpec{
					Type: v1.ServiceTypeLoadBalancer,
					Ports: []v1.ServicePort{
						{
							Port:       80,
							TargetPort: intstr.FromInt(80),
						},
					},
				},
				Status: v1.ServiceStatus{
					LoadBalancer: v1.LoadBalancerStatus{Ingress: []v1.LoadBalancerIngress{{IP: "2001:0db8::0"}}},
				},
			},
			wantErr: false,
			wantRelease: &svcRecord{
				Key: "default/testsvc",
				Eip: "eip",
				IP:  "2001:0db8::0",
			},
		},

		{
			name: "service is not specify openelb - eip has records",
			eip: []*networkv1alpha2.Eip{{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "eip",
				},
				Spec: networkv1alpha2.EipSpec{
					Address:  "2001:0db8::0/120",
					Protocol: constant.OpenELBProtocolLayer2,
				},
				Status: networkv1alpha2.EipStatus{
					Used: map[string]string{
						"2001:0db8::0": "default/testsvc",
					},
				},
			}},
			svc: &v1.Service{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testsvc",
					Namespace: "default",
					Annotations: map[string]string{
						constant.OpenELBEIPAnnotationKeyV1Alpha2: "eip",
					},
				},
				Spec: v1.ServiceSpec{
					Type: v1.ServiceTypeLoadBalancer,
					Ports: []v1.ServicePort{
						{
							Port:       80,
							TargetPort: intstr.FromInt(80),
						},
					},
				},
				Status: v1.ServiceStatus{
					LoadBalancer: v1.LoadBalancerStatus{Ingress: []v1.LoadBalancerIngress{{IP: "2001:0db8::0"}}},
				},
			},
			wantErr: false,
			wantRelease: &svcRecord{
				Key: "default/testsvc",
				Eip: "eip",
				IP:  "2001:0db8::0",
			},
		},

		{
			name: "normal service update",
			eip: []*networkv1alpha2.Eip{{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "eip",
				},
				Spec: networkv1alpha2.EipSpec{
					Address:  "2001:0db8::0/120",
					Protocol: constant.OpenELBProtocolLayer2,
				},
				Status: networkv1alpha2.EipStatus{
					Used: map[string]string{
						"2001:0db8::0": "default/testsvc",
					},
				},
			}},
			svc: &v1.Service{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testsvc",
					Namespace: "default",
					Labels: map[string]string{
						constant.OpenELBEIPAnnotationKeyV1Alpha2: "eip",
					},
					Annotations: map[string]string{
						constant.OpenELBEIPAnnotationKeyV1Alpha2: "eip",
						constant.OpenELBAnnotationKey:            constant.OpenELBAnnotationValue,
					},
				},
				Spec: v1.ServiceSpec{
					Type: v1.ServiceTypeLoadBalancer,
					Ports: []v1.ServicePort{
						{
							Port:       80,
							TargetPort: intstr.FromInt(80),
						},
					},
				},
				Status: v1.ServiceStatus{
					LoadBalancer: v1.LoadBalancerStatus{Ingress: []v1.LoadBalancerIngress{{IP: "2001:0db8::0"}}},
				},
			},
			wantErr: false,
		},

		{
			name: "normal service update eip",
			eip: []*networkv1alpha2.Eip{
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "eip",
					},
					Spec: networkv1alpha2.EipSpec{
						Address:  "2001:0db8::0/120",
						Protocol: constant.OpenELBProtocolVip,
					},
					Status: networkv1alpha2.EipStatus{},
				},
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "eip1",
					},
					Spec: networkv1alpha2.EipSpec{
						Address:  "2001:0db8::0/120",
						Protocol: constant.OpenELBProtocolLayer2,
					},
					Status: networkv1alpha2.EipStatus{
						Used: map[string]string{
							"2001:0db8::1": "default/testsvc",
						},
					},
				},
			},
			svc: &v1.Service{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testsvc",
					Namespace: "default",
					Annotations: map[string]string{
						constant.OpenELBEIPAnnotationKeyV1Alpha2: "eip",
						constant.OpenELBAnnotationKey:            constant.OpenELBAnnotationValue,
					},
				},
				Spec: v1.ServiceSpec{
					Type: v1.ServiceTypeLoadBalancer,
					Ports: []v1.ServicePort{
						{
							Port:       80,
							TargetPort: intstr.FromInt(80),
						},
					},
				},
				Status: v1.ServiceStatus{
					LoadBalancer: v1.LoadBalancerStatus{Ingress: []v1.LoadBalancerIngress{{IP: "2001:0db8::1"}}},
				},
			},
			wantErr: false,
			wantAllocate: &svcRecord{
				Key: "default/testsvc",
				Eip: "eip",
			},
			wantRelease: &svcRecord{
				Key: "default/testsvc",
				Eip: "eip1",
				IP:  "2001:0db8::1",
			},
		},

		{
			name: "eip bind default namespace by spec.namespace",
			eip: []*networkv1alpha2.Eip{{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "eip",
				},
				Spec: networkv1alpha2.EipSpec{
					Address:    "2001:0db8::0/120",
					Namespaces: []string{"default"},
				},
				Status: networkv1alpha2.EipStatus{},
			}},
			svc: &v1.Service{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testsvc",
					Namespace: "default",
					Annotations: map[string]string{
						constant.OpenELBAnnotationKey: constant.OpenELBAnnotationValue,
					},
				},
				Spec: v1.ServiceSpec{
					Type: v1.ServiceTypeLoadBalancer,
					Ports: []v1.ServicePort{
						{
							Port:       80,
							TargetPort: intstr.FromInt(80),
						},
					},
				},
				Status: v1.ServiceStatus{},
			},
			wantErr: false,
			wantAllocate: &svcRecord{
				Key: "default/testsvc",
				Eip: "eip",
				IP:  "",
			},
			wantRelease: nil,
		},

		{
			name: "eip bind default namespace by spec.namespaceSelector",
			eip: []*networkv1alpha2.Eip{{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "eip",
				},
				Spec: networkv1alpha2.EipSpec{
					Address: "2001:0db8::0/120",
					NamespaceSelector: map[string]string{
						"label": "test",
					},
				},
				Status: networkv1alpha2.EipStatus{},
			}},
			svc: &v1.Service{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testsvc",
					Namespace: "default",
					Annotations: map[string]string{
						constant.OpenELBAnnotationKey: constant.OpenELBAnnotationValue,
					},
				},
				Spec: v1.ServiceSpec{
					Type: v1.ServiceTypeLoadBalancer,
					Ports: []v1.ServicePort{
						{
							Port:       80,
							TargetPort: intstr.FromInt(80),
						},
					},
				},
				Status: v1.ServiceStatus{},
			},
			wantErr: false,
			wantAllocate: &svcRecord{
				Key: "default/testsvc",
				Eip: "eip",
				IP:  "",
			},
			wantRelease: nil,
		},

		{
			name: "multi-eip bind default namespace with different priority",
			eip: []*networkv1alpha2.Eip{{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "eip-1",
				},
				Spec: networkv1alpha2.EipSpec{
					Address:    "2001:0db8::0/120",
					Namespaces: []string{"default"},
					Priority:   20,
				},
				Status: networkv1alpha2.EipStatus{},
			},
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "eip-2",
					},
					Spec: networkv1alpha2.EipSpec{
						Address: "2001:0db8::1/120",
						NamespaceSelector: map[string]string{
							"label": "test",
						},
						Priority: 10,
					},
					Status: networkv1alpha2.EipStatus{},
				}},
			svc: &v1.Service{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testsvc",
					Namespace: "default",
					Annotations: map[string]string{
						constant.OpenELBAnnotationKey: constant.OpenELBAnnotationValue,
					},
				},
				Spec: v1.ServiceSpec{
					Type: v1.ServiceTypeLoadBalancer,
					Ports: []v1.ServicePort{
						{
							Port:       80,
							TargetPort: intstr.FromInt(80),
						},
					},
				},
				Status: v1.ServiceStatus{},
			},
			wantErr: false,
			wantAllocate: &svcRecord{
				Key: "default/testsvc",
				Eip: "eip-2",
				IP:  "",
			},
			wantRelease: nil,
		},

		{
			name: "default eip",
			eip: []*networkv1alpha2.Eip{{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "eip",
					Annotations: map[string]string{
						constant.OpenELBEIPAnnotationDefaultPool: "true",
					},
				},
				Spec: networkv1alpha2.EipSpec{
					Address:  "2001:0db8::0/120",
					Protocol: constant.OpenELBProtocolLayer2,
				},
				Status: networkv1alpha2.EipStatus{},
			}},
			svc: &v1.Service{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testsvc",
					Namespace: "default",
					Annotations: map[string]string{
						constant.OpenELBAnnotationKey: constant.OpenELBAnnotationValue,
					},
				},
				Spec: v1.ServiceSpec{
					Type: v1.ServiceTypeLoadBalancer,
					Ports: []v1.ServicePort{
						{
							Port:       80,
							TargetPort: intstr.FromInt(80),
						},
					},
				},
				Status: v1.ServiceStatus{},
			},
			wantErr: false,
			wantAllocate: &svcRecord{
				Key: "default/testsvc",
				Eip: "eip",
				IP:  "",
			},
			wantRelease: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ns := &v1.Namespace{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "default",
					Labels: map[string]string{
						"label": "test",
					},
				},
			}
			objs := []client.Object{ns}
			for _, e := range tt.eip {
				objs = append(objs, e)
			}

			if tt.svc != nil {
				objs = append(objs, tt.svc)
			}

			cl := fake.NewClientBuilder()
			cl.WithScheme(scheme).WithObjects(objs...)

			m := NewManager(cl.Build())
			m.EventRecorder = &record.FakeRecorder{}
			request, err := m.ConstructRequest(context.Background(), tt.svc)
			if (err != nil) != tt.wantErr {
				t.Errorf("Request.ConstructAllocate() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !reflect.DeepEqual(tt.wantAllocate, request.Allocate) {
				t.Errorf("Request.ConstructAllocate() wantAllocate = %v, Allocate %v", tt.wantAllocate, request.Allocate)
			}

			if !reflect.DeepEqual(tt.wantRelease, request.Release) {
				t.Errorf("Request.ConstructAllocate() wantRelease = %v, Release %v", tt.wantRelease, request.Release)
			}
		})
	}
}

func TestManager_AssignIPV6(t *testing.T) {
	eip := &networkv1alpha2.Eip{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: "eip",
		},
		Spec: networkv1alpha2.EipSpec{
			Address:  "2001:0db8::0/120",
			Protocol: constant.OpenELBProtocolLayer2,
		},
		Status: networkv1alpha2.EipStatus{
			FirstIP:  "2001:0db8::00",
			LastIP:   "2001:0db8::ff",
			PoolSize: 256,
			Used:     map[string]string{},
			Usage:    0,
			Ready:    true,
		},
	}
	type fields struct {
		eip *networkv1alpha2.Eip
	}
	type args struct {
		allocate *svcRecord
	}
	tests := []struct {
		name         string
		args         args
		fields       fields
		wantErr      bool
		wantAllocate bool
	}{
		{
			name:    "allocate is nil",
			wantErr: false,
			args: args{
				allocate: nil,
			},
			fields: fields{
				eip: eip,
			},
		},

		{
			name:    "eip not found",
			wantErr: true,
			args: args{
				allocate: &svcRecord{Eip: "xx"},
			},
			fields: fields{
				eip: eip,
			},
		},

		{
			name:         "allocate from eip 1 - static ip",
			wantErr:      false,
			wantAllocate: true,
			args: args{
				allocate: &svcRecord{
					Key: "default/svc",
					Eip: "eip",
					IP:  "2001:0db8::f",
				},
			},
			fields: fields{
				eip: eip,
			},
		},

		{
			name:         "allocate from eip 2 - auto",
			wantErr:      false,
			wantAllocate: true,
			args: args{
				allocate: &svcRecord{
					Key: "default/svc",
					Eip: "eip",
				},
			},
			fields: fields{
				eip: eip,
			},
		},

		{
			name:    "no avliable eip 1 - eip is deleting",
			wantErr: true,
			args: args{
				allocate: &svcRecord{
					Key: "default/svc",
					Eip: "eip",
				},
			},
			fields: fields{
				eip: func() *networkv1alpha2.Eip {
					clone := eip.DeepCopy()
					clone.Finalizers = append(clone.Finalizers, constant.IPAMFinalizerName)
					clone.DeletionTimestamp = &metav1.Time{Time: time.Now()}
					return clone
				}(),
			},
		},

		{
			name:    "no avliable eip 2 - eip is disabled",
			wantErr: true,
			args: args{
				allocate: &svcRecord{
					Key: "default/svc",
					Eip: "eip",
				},
			},
			fields: fields{
				eip: func() *networkv1alpha2.Eip {
					clone := eip.DeepCopy()
					clone.Spec.Disable = true
					return clone
				}(),
			},
		},

		{
			name:    "no avliable eip 3 - out of range",
			wantErr: true,
			args: args{
				allocate: &svcRecord{
					Key: "default/svc",
					Eip: "eip",
					IP:  "2001:0db8::100",
				},
			},
			fields: fields{
				eip: eip,
			},
		},

		{
			name:    "no avliable eip 4 - ippool is full",
			wantErr: true,
			args: args{
				allocate: &svcRecord{
					Key: "default/svc",
					Eip: "eip",
				},
			},
			fields: fields{
				eip: &networkv1alpha2.Eip{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "eip-full",
					},
					Spec: networkv1alpha2.EipSpec{
						Address:  "2001:0db8::101",
						Protocol: constant.OpenELBProtocolLayer2,
					},
					Status: networkv1alpha2.EipStatus{
						FirstIP:  "2001:0db8::100",
						LastIP:   "2001:0db8::101",
						PoolSize: 2,
						Used: map[string]string{
							"2001:0db8::100": "default/test0",
							"2001:0db8::101": "default/test1",
						},
						Usage: 2,
						Ready: true,
					},
				},
			},
		},

		{
			name:         "Allocation records that already exist - share address",
			wantErr:      false,
			wantAllocate: true,
			args: args{
				allocate: &svcRecord{
					Key: "default/svc",
					Eip: "eip",
					IP:  "2001:0db8::1",
				},
			},
			fields: fields{
				eip: func() *networkv1alpha2.Eip {
					clone := eip.DeepCopy()
					clone.Status.Usage = 1
					clone.Status.Used = map[string]string{
						"2001:0db8::1": "default/test",
					}
					return clone
				}(),
			},
		},

		{
			name:         "Allocation records that already exist - return records",
			wantErr:      false,
			wantAllocate: true,
			args: args{
				allocate: &svcRecord{
					Key: "default/svc",
					Eip: "eip",
					IP:  "2001:0db8::100",
				},
			},
			fields: fields{
				eip: func() *networkv1alpha2.Eip {
					clone := eip.DeepCopy()
					clone.Status.Usage = 1
					clone.Status.Used = map[string]string{
						"2001:0db8::100": "default/svc",
					}
					return clone
				}(),
			},
		},

		{
			name:         "Allocation records that already exist but specify a different IP",
			wantErr:      false,
			wantAllocate: true,
			args: args{
				allocate: &svcRecord{
					Key: "default/svc",
					Eip: "eip",
					IP:  "2001:0db8::1",
				},
			},
			fields: fields{
				eip: func() *networkv1alpha2.Eip {
					clone := eip.DeepCopy()
					clone.Status.Usage = 1
					clone.Status.Used = map[string]string{
						"2001:0db8::0": "default/svc",
					}
					return clone
				}(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objs := []client.Object{}
			if tt.fields.eip != nil {
				objs = append(objs, tt.fields.eip)
			}
			cl := fake.NewClientBuilder()
			cl.WithStatusSubresource(objs...).WithScheme(scheme).WithObjects(objs...)

			m := NewManager(cl.Build())
			err := m.AssignIP(context.Background(), tt.args.allocate)
			if (err != nil) != tt.wantErr {
				t.Errorf("Manager.AssignIP() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.fields.eip != nil {
				eip := &networkv1alpha2.Eip{}
				err := m.Get(context.Background(), types.NamespacedName{Name: tt.fields.eip.Name}, eip)
				if err != nil {
					t.Errorf("Manager.AssignIP() get eip error = %v", err)
				}
				if tt.wantAllocate && len(eip.Status.Used) == 0 {
					t.Errorf("Manager.AssignIP() eip.Status %v", eip.Status)
				}

				if !tt.wantAllocate && len(eip.Status.Used) == 1 {
					t.Errorf("Manager.AssignIP() eip.Status %v", eip.Status)
				}
			}
		})
	}
}

func TestManager_ReleaseIPV6(t *testing.T) {
	eip := &networkv1alpha2.Eip{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: "eip",
		},
		Spec: networkv1alpha2.EipSpec{
			Address:  "2001:0db8::0/120",
			Protocol: constant.OpenELBProtocolLayer2,
		},
		Status: networkv1alpha2.EipStatus{
			Used: map[string]string{
				"2001:0db8::1": "default/testsvc",
			},
		},
	}
	type fields struct {
		eip *networkv1alpha2.Eip
	}
	type args struct {
		release *svcRecord
	}
	tests := []struct {
		name       string
		args       args
		fields     fields
		wantErr    bool
		wantDelete bool
	}{
		{
			name:    "release is nil",
			wantErr: false,
			args: args{
				release: nil,
			},
			fields: fields{
				eip: eip,
			},
		},

		{
			name:    "eip is not exist",
			wantErr: false,
			args: args{
				release: &svcRecord{
					Key: "default/testsvc",
					Eip: "eip",
				},
			},
			fields: fields{
				eip: nil,
			},
		},

		{
			name:       "eip is deleting",
			wantDelete: false,
			wantErr:    false,
			args: args{
				release: &svcRecord{},
			},
			fields: fields{
				eip: eip,
			},
		},

		{
			name:       "eip no record",
			wantDelete: false,
			wantErr:    false,
			args: args{
				release: &svcRecord{
					Key: "default/no-svc",
					Eip: "eip",
				},
			},
			fields: fields{
				eip: eip,
			},
		},

		{
			name:       "release ip from eip",
			wantErr:    false,
			wantDelete: true,
			args: args{
				release: &svcRecord{
					Key: "default/testsvc",
					Eip: "eip",
				},
			},
			fields: fields{
				eip: eip,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objs := []client.Object{}
			if tt.fields.eip != nil {
				objs = append(objs, tt.fields.eip)
			}
			cl := fake.NewClientBuilder()
			cl.WithStatusSubresource(objs...).WithScheme(scheme).WithObjects(objs...)

			m := NewManager(cl.Build())
			if err := m.ReleaseIP(context.Background(), tt.args.release); (err != nil) != tt.wantErr {
				t.Errorf("Manager.ReleaseIP() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.fields.eip != nil {
				eip := &networkv1alpha2.Eip{}
				err := m.Get(context.Background(), types.NamespacedName{Name: tt.fields.eip.Name}, eip)
				if err != nil {
					t.Errorf("Manager.ReleaseIP() get eip error = %v", err)
				}
				if tt.wantDelete && len(eip.Status.Used) != 0 {
					t.Errorf("Manager.ReleaseIP() eip.Status %v", eip.Status)
				}

				if !tt.wantDelete && len(eip.Status.Used) == 0 {
					t.Errorf("Manager.ReleaseIP() eip.Status %v", eip.Status)
				}
			}
		})
	}
}
