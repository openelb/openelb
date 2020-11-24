package speaker

import (
	"github.com/projectcalico/libcalico-go/lib/set"
	"sync"
)

type Speaker interface {
	SetBalancer(ip string, nexthops []string) error
	DelBalancer(ip string) error
	Start(stopCh <-chan struct{}) error
}

type Fake struct {
	lock     sync.Mutex
	nextHops map[string]set.Set
}

func NewFake() *Fake {
	return &Fake{
		nextHops: make(map[string]set.Set),
	}
}

func (f *Fake) SetBalancer(ip string, nexthops []string) error {
	f.lock.Lock()
	defer f.lock.Unlock()

	if len(nexthops) == 0 {
		f.nextHops[ip] = nil
	} else {
		f.nextHops[ip] = set.FromArray(nexthops)
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

type speaker struct {
	s  Speaker
	ch chan struct{}
}

var (
	speakers map[string]speaker
	lock     sync.Mutex
)

func RegisteSpeaker(name string, s Speaker) error {
	lock.Lock()
	defer lock.Unlock()

	t := speaker{
		s:  s,
		ch: make(chan struct{}),
	}

	err := s.Start(t.ch)
	if err == nil {
		speakers[name] = t
		return nil
	}

	return err
}

func UnRegisteSpeaker(name string) {
	lock.Lock()
	defer lock.Unlock()

	t, ok := speakers[name]
	if ok {
		close(t.ch)
	}
	delete(speakers, name)
}

func GetSpeaker(name string) Speaker {
	lock.Lock()
	defer lock.Unlock()

	t, ok := speakers[name]
	if ok {
		return t.s
	}

	return nil
}

func init() {
	speakers = make(map[string]speaker, 0)
}
