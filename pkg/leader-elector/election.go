package leader

import (
	"context"
	"github.com/kubesphere/porterlb/pkg/constant"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
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

func LeaderElector(stopCh <-chan struct{}, client *clientset.Clientset, opts Options) {
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
	go wait.Until(func() {
		leaderelection.RunOrDie(context.Background(), leaderelection.LeaderElectionConfig{
			Lock:            lock,
			ReleaseOnCancel: true,
			LeaseDuration:   time.Duration(opts.LeaseDuration) * time.Second,
			RenewDeadline:   time.Duration(opts.RenewDeadline) * time.Second,
			RetryPeriod:     time.Duration(opts.RetryPeriod) * time.Second,
			Callbacks: leaderelection.LeaderCallbacks{
				OnStartedLeading: func(ctx context.Context) {
					Leader = true
				},
				OnStoppedLeading: func() {
					Leader = false
				},
			},
		})
	}, time.Second, stopCh)
}
