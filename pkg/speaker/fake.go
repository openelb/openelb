package speaker

import (
	"sync"

	"github.com/openelb/openelb/pkg/util/set"
	corev1 "k8s.io/api/core/v1"
)

type Fake struct {
	lock     sync.Mutex
	nextHops map[string]set.Set
}

func NewFake() *Fake {
	return &Fake{
		nextHops: make(map[string]set.Set),
	}
}

func (f *Fake) SetBalancer(ip string, nexthops []corev1.Node) error {
	f.lock.Lock()
	defer f.lock.Unlock()

	var names []string
	for _, nexthop := range nexthops {
		names = append(names, nexthop.Name)
	}

	if len(names) == 0 {
		f.nextHops[ip] = nil
	} else {
		f.nextHops[ip] = set.FromArray(names)
	}

	return nil
}

func (f *Fake) Equal(ip string, nexthops []string) bool {
	f.lock.Lock()
	defer f.lock.Unlock()

	tmp, ok := f.nextHops[ip]
	if ok {
		if tmp == nil && len(nexthops) == 0 {
			return true
		}

		if tmp == nil || len(nexthops) == 0 {
			return false
		}
	} else {
		if len(nexthops) == 0 {
			return true
		}
		return false
	}

	return set.FromArray(nexthops).Equals(f.nextHops[ip])
}

func (f *Fake) DelBalancer(ip string) error {
	f.lock.Lock()
	defer f.lock.Unlock()

	delete(f.nextHops, ip)

	return nil
}

func (f *Fake) Start(stopCh <-chan struct{}) error {
	return nil
}
