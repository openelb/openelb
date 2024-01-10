package layer2

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"net"
	"sort"
	"time"

	"github.com/hashicorp/memberlist"
	"github.com/openelb/openelb/api/v1alpha2"
	"github.com/openelb/openelb/pkg/constant"
	"github.com/openelb/openelb/pkg/speaker"
	"github.com/openelb/openelb/pkg/util"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

var _ speaker.Speaker = &layer2Speaker{}

func NewSpeaker(client *kubernetes.Clientset, opt *Options, reloadChan chan event.GenericEvent, queue workqueue.Interface) (speaker.Speaker, error) {
	config := memberlist.DefaultLANConfig()
	config.Name = opt.NodeName
	config.BindAddr = opt.BindAddr
	config.BindPort = opt.BindPort
	secret := util.GetSecret()
	if secret == "" {
		secret = opt.SecretKey
	}
	config.SecretKey = []byte(secret)
	eventCh := make(chan memberlist.NodeEvent, 16)
	config.Events = &memberlist.ChannelEventDelegate{Ch: eventCh}
	config.Logger = log.New(logWriter{}, "memberlist", log.LstdFlags)
	list, err := memberlist.Create(config)
	if err != nil {
		return nil, err
	}

	return &layer2Speaker{
		eventCh:    eventCh,
		reloadChan: reloadChan,
		mlist:      list,
		client:     client,
		queue:      queue,
		announcers: map[string]announcer{}}, nil
}

func (l *layer2Speaker) joinMembers() error {
	iplist := []string{}
	pods, err := l.client.CoreV1().Pods(util.EnvNamespace()).List(context.TODO(), v1.ListOptions{
		LabelSelector: labels.SelectorFromSet(labels.Set{"app": "openelb", "component": "speaker"}).String(),
	})
	if err != nil {
		return err
	}

	for _, p := range pods.Items {
		iplist = append(iplist, p.Status.PodIP)
	}

	_, err = l.mlist.Join(iplist)
	return err
}

type layer2Speaker struct {
	mlist      *memberlist.Memberlist
	eventCh    chan memberlist.NodeEvent
	reloadChan chan event.GenericEvent
	client     *kubernetes.Clientset
	queue      workqueue.Interface

	// nic - announcers
	announcers map[string]announcer
}

func (l *layer2Speaker) SetBalancer(ip string, clusterNodes []corev1.Node) error {
	member := map[string]string{}
	for _, m := range l.mlist.Members() {
		member[m.Name] = m.Addr.String()
	}

	nodes := []string{}
	for _, n := range clusterNodes {
		if _, exist := member[n.GetName()]; exist {
			nodes = append(nodes, n.GetName())
		}
	}

	if len(nodes) == 0 {
		klog.Warningf("no suitable nodes to participate in the announced election.")
		return nil
	}

	// Sort the slice by the hash of node + load balancer ips. This
	// produces an ordering of ready nodes that is unique to all the services
	// with the same ip.
	sort.Slice(nodes, func(i, j int) bool {
		hi := sha256.Sum256([]byte(nodes[i] + "#" + ip))
		hj := sha256.Sum256([]byte(nodes[j] + "#" + ip))

		return bytes.Compare(hi[:], hj[:]) < 0
	})

	klog.Infof("node %s wins the right to announce the IP address %s", nodes[0], ip)
	if nodes[0] != util.GetNodeName() {
		return nil
	}

	for _, a := range l.announcers {
		if a.ContainsIP(ip) {
			return a.AddAnnouncedIP(ip)
		}
	}

	return nil
}

func (l *layer2Speaker) DelBalancer(ip string) error {
	for _, a := range l.announcers {
		if a.ContainsIP(ip) {
			return a.DelAnnouncedIP(ip)
		}
	}
	return nil
}

func (l *layer2Speaker) Start(stopCh <-chan struct{}) error {
	if err := l.joinMembers(); err != nil {
		return err
	}

	go wait.Until(l.processEIP, time.Second, stopCh)
	go func() {
		tick := time.NewTicker(5 * time.Second)
		for {
			select {
			case <-stopCh:
				l.unregisterAllAnnouncers()
			case <-l.eventCh:
				evt := corev1.Service{}
				evt.Name = constant.Layer2ReloadServiceName
				evt.Namespace = constant.Layer2ReloadServiceNamespace
				l.reloadChan <- event.GenericEvent{Object: &evt}
			case <-tick.C:
				for _, member := range l.mlist.Members() {
					klog.V(2).Infof("member info %s %s", member.Name, member.Address())
				}
			}
		}
	}()

	return nil
}

func (l *layer2Speaker) processEIP() {
	sync := func() bool {
		key, quit := l.queue.Get()
		if quit {
			return false
		}
		defer l.queue.Done(key)

		if err := l.syncAnnouncers(key.(*v1alpha2.Eip)); err != nil {
			klog.Errorf("Error when sync announcers: %s", err)
		}

		return true
	}

	for sync() {
	}
}

func (l *layer2Speaker) syncAnnouncers(eip *v1alpha2.Eip) error {
	if eip == nil {
		return nil
	}

	netif, err := parseInterface(eip.Spec.Interface, true)
	if err != nil || netif == nil {
		return err
	}

	if !eip.DeletionTimestamp.IsZero() {
		return l.unregisterAnnouncer(netif.Name, eip.Name)
	}

	return l.registerAnnouncer(eip.Name, netif)
}

func (l *layer2Speaker) registerAnnouncer(eipName string, netif *net.Interface) error {
	a, exist := l.announcers[netif.Name]
	if exist {
		if _, ok := a.eips[eipName]; !ok {
			a.eips[eipName] = struct{}{}
			klog.Infof("use interface %s to announce eip %s's arp", netif.Name, eipName)
		}
		return nil
	}

	klog.Infof("use interface %s to announce eip %s's arp", netif.Name, eipName)
	ann, err := newAnnouncer(netif, true)
	if err != nil {
		return fmt.Errorf("new Announcer error. interface %s, error %s", netif.Name, err.Error())
	}

	a = announcer{Announcer: ann, stopCh: make(chan struct{}), eips: map[string]struct{}{eipName: {}}}
	if err := a.Start(a.stopCh); err != nil {
		return err
	}

	l.announcers[netif.Name] = a
	return nil
}

func (l *layer2Speaker) unregisterAnnouncer(name, eipName string) error {
	a, exist := l.announcers[name]
	if !exist {
		return nil
	}

	delete(a.eips, eipName)
	klog.Infof("cancel interface %s to announce eip %s's arp", name, eipName)
	if len(a.eips) == 0 {
		close(a.stopCh)
		delete(l.announcers, name)
	}
	return nil
}

func (l *layer2Speaker) unregisterAllAnnouncers() {
	for _, a := range l.announcers {
		close(a.stopCh)
	}

	l.announcers = map[string]announcer{}
}
