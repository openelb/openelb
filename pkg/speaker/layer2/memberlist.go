package layer2

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"net"
	"sort"
	"strings"

	"github.com/hashicorp/memberlist"
	"github.com/openelb/openelb/api/v1alpha2"
	"github.com/openelb/openelb/pkg/constant"
	"github.com/openelb/openelb/pkg/speaker"
	"github.com/openelb/openelb/pkg/util"
	"github.com/openelb/openelb/pkg/util/iprange"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

var _ speaker.Speaker = &layer2Speaker{}

func NewSpeaker(client *kubernetes.Clientset, opt *Options, reloadChan chan event.GenericEvent) (speaker.Speaker, error) {
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
		announcers: map[string]Announcer{}}, nil
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

	// nic - announcers
	announcers map[string]Announcer
}

func (l *layer2Speaker) SetBalancer(ip string, clusterNodes []corev1.Node) error {
	for _, a := range l.announcers {
		if a.ContainsIP(net.ParseIP(ip)) {
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

			klog.Infof("candidates: [%s]", strings.Join(nodes, ","))
			// Sort the slice by the hash of node + load balancer ips. This
			// produces an ordering of ready nodes that is unique to all the services
			// with the same ip.
			sort.Slice(nodes, func(i, j int) bool {
				hi := sha256.Sum256([]byte(nodes[i] + "#" + ip))
				hj := sha256.Sum256([]byte(nodes[j] + "#" + ip))

				return bytes.Compare(hi[:], hj[:]) < 0
			})

			klog.Infof("[%s] wins the right to announce the IP address %s", nodes[0], ip)
			if nodes[0] != util.GetNodeName() {
				return nil
			}
			return a.AddAnnouncedIP(net.ParseIP(ip))
		}
	}

	klog.Warningf("The announcers of the speakers do not contain the %s", ip)
	return nil
}

func (l *layer2Speaker) DelBalancer(ip string) error {
	for _, a := range l.announcers {
		if a.ContainsIP(net.ParseIP(ip)) {
			return a.DelAnnouncedIP(net.ParseIP(ip))
		}
	}
	return nil
}

func (l *layer2Speaker) Start(stopCh <-chan struct{}) error {
	if err := l.joinMembers(); err != nil {
		return err
	}

	for {
		select {
		case <-stopCh:
			l.unregisterAllAnnouncers()
			return nil
		case <-l.eventCh:
			evt := v1alpha2.Eip{}
			evt.Name = constant.Layer2ReloadEIPName
			evt.Namespace = constant.Layer2ReloadEIPNamespace
			l.reloadChan <- event.GenericEvent{Object: &evt}
		}
	}
}

func (l *layer2Speaker) ConfigureWithEIP(config speaker.Config, deleted bool) error {
	netif, err := speaker.ParseInterface(config.Iface)
	if err != nil || netif == nil {
		return err
	}

	if deleted {
		return l.unregisterAnnouncer(config.Name, netif.Name)
	}
	return l.registerAnnouncer(config.Name, netif, config.IPRange)
}

func (l *layer2Speaker) registerAnnouncer(eipName string, netif *net.Interface, r iprange.Range) error {
	a, exist := l.announcers[netif.Name]
	if !exist {
		// no announcer for the interface, create a new one
		var err error
		a, err = newAnnouncer(netif, r.Family())
		if err != nil {
			return fmt.Errorf("new Announcer error. interface %s, error %s", netif.Name, err.Error())
		}
		klog.Infof("use interface %s to announce eip[%s]", netif.Name, eipName)

		if err := a.Start(); err != nil {
			return err
		}
		l.announcers[netif.Name] = a
	}

	a.RegisterIPRange(eipName, r)
	return nil
}

func (l *layer2Speaker) unregisterAnnouncer(eipName, netifName string) error {
	a, exist := l.announcers[netifName]
	if !exist {
		return nil
	}

	klog.Infof("cancel interface %s to announce eip[%s]'s arp", netifName, eipName)
	a.UnregisterIPRange(eipName)
	if a.Size() == 0 {
		if err := a.Stop(); err != nil {
			return err
		}

		delete(l.announcers, netifName)
	}
	return nil
}

func (l *layer2Speaker) unregisterAllAnnouncers() {
	for _, a := range l.announcers {
		if err := a.Stop(); err != nil {
			klog.Errorf("stop announcer error. %s", err.Error())
		}
	}

	l.announcers = map[string]Announcer{}
}
