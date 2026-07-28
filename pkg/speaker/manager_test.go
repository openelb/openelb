package speaker

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestManager_setBalancer_WithdrawsStaleRouteWhenNoNodesLeft ensures that once a
// service's ExternalTrafficPolicy=Local endpoints no longer resolve to any node
// (e.g. during a pod rollout), the previously advertised route is withdrawn
// instead of being left stale until the speaker restarts.
func TestManager_setBalancer_WithdrawsStaleRouteWhenNoNodesLeft(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	node := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node1"}}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "svc1", Namespace: "default"},
		Spec:       corev1.ServiceSpec{ExternalTrafficPolicy: corev1.ServiceExternalTrafficPolicyTypeLocal},
	}
	nodeName := "node1"
	endpoints := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{Name: "svc1", Namespace: "default"},
		Subsets: []corev1.EndpointSubset{{
			Addresses: []corev1.EndpointAddress{{IP: "10.0.0.1", NodeName: &nodeName}},
		}},
	}

	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(node, svc, endpoints).Build()
	fakeSpeaker := NewFake()
	m := &Manager{
		Client:        cl,
		EventRecorder: &record.FakeRecorder{},
		speakers:      map[string]speakerWithCancelFunc{"fake": {Speaker: fakeSpeaker}},
	}

	usage := map[string]string{"1.2.3.4": "default/svc1"}
	if err := m.setBalancer(context.Background(), "fake", usage); err != nil {
		t.Fatalf("setBalancer() error = %v", err)
	}
	if !fakeSpeaker.Equal("1.2.3.4", []string{"node1"}) {
		t.Fatalf("expected route to node1 to be advertised")
	}

	// The pod backing the endpoint moves off node1 and hasn't landed anywhere yet.
	updated := &corev1.Endpoints{}
	if err := cl.Get(context.Background(), client.ObjectKeyFromObject(endpoints), updated); err != nil {
		t.Fatal(err)
	}
	updated.Subsets = nil
	if err := cl.Update(context.Background(), updated); err != nil {
		t.Fatal(err)
	}

	if err := m.setBalancer(context.Background(), "fake", usage); err != nil {
		t.Fatalf("setBalancer() error = %v", err)
	}
	if !fakeSpeaker.Equal("1.2.3.4", nil) {
		t.Fatalf("expected stale route to be withdrawn once no nodes are available")
	}
}
