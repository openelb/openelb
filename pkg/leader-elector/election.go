package leader

import (
	"context"
	"github.com/kubesphere/porter/pkg/constant"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"os"
	"time"
)

var (
	Leader bool
)

func envNamespace() string {
	ns := os.Getenv(constant.EnvPorterNamespace)
	if ns == "" {
		return constant.PorterNamespace
	}
	return ns
}

func envNodename() string {
	name := os.Getenv(constant.EnvNodeName)
	if name == "" {
		panic("env NODE_NAME should not be empty")
	}
	return name
}

func LeaderElector(client *clientset.Clientset) {
	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      constant.PorterSpeakerLocker,
			Namespace: envNamespace(),
		},
		Client: client.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: envNodename(),
		},
	}

	// start the leader election code loop
	go leaderelection.RunOrDie(context.Background(), leaderelection.LeaderElectionConfig{
		Lock:            lock,
		ReleaseOnCancel: true,
		LeaseDuration:   10 * time.Second,
		RenewDeadline:   5 * time.Second,
		RetryPeriod:     2 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				Leader = true
			},
			OnStoppedLeading: func() {
				Leader = false
			},
		},
	})
}
