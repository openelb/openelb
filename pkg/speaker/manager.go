package speaker

import (
	"context"
	"fmt"
	"net"
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/openelb/openelb/api/v1alpha2"
	"github.com/openelb/openelb/pkg/constant"
	"github.com/openelb/openelb/pkg/util"
	"github.com/openelb/openelb/pkg/util/iprange"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type speakerWithCancelFunc struct {
	Speaker
	cancel context.CancelFunc
}

type Manager struct {
	client.Client
	record.EventRecorder

	mgr       manager.Manager
	speakers  map[string]speakerWithCancelFunc
	pools     map[string]*v1alpha2.Eip
	waitGroup sync.WaitGroup
	errChan   chan error
}

func NewSpeakerManager(mgr manager.Manager) *Manager {
	return &Manager{
		mgr:           mgr,
		Client:        mgr.GetClient(),
		EventRecorder: mgr.GetEventRecorderFor("speakerManager"),
		speakers:      make(map[string]speakerWithCancelFunc, 0),
		pools:         make(map[string]*v1alpha2.Eip, 0),
		errChan:       make(chan error),
	}
}

func (m *Manager) Start(ctx context.Context) (err error) {
	ictx, cancelFunc := context.WithCancel(context.TODO())
	errCh := make(chan error)
	defer close(errCh)

	m.waitGroup.Add(1)
	go func() {
		defer m.waitGroup.Done()
		if err := m.mgr.Start(ictx); err != nil {
			errCh <- err
		}
	}()

	// The ctx (signals.SetupSignalHandler()) is to control the entire program life cycle,
	// The ictx(internal context)  is created here to control the life cycle of the controller-manager(all controllers, sharedInformer, webhook etc.)
	// when config changed, stop server and renew context, start new server
	for {
		select {
		case <-ctx.Done():
			cancelFunc()
			m.waitGroup.Wait()
			return nil
		case err = <-errCh:
		case err = <-m.errChan:
			cancelFunc()
			for _, s := range m.speakers {
				if s.cancel != nil {
					s.cancel()
				}
			}
			m.waitGroup.Wait()
			return err
		}
	}
}

// TODO: Dynamically configure the speaker through configmap
func (m *Manager) RegisterSpeaker(ctx context.Context, name string, speaker Speaker) error {
	if s, exist := m.speakers[name]; exist && s.cancel != nil {
		s.cancel()
	}

	m.waitGroup.Add(1)
	ctxChild, cancel := context.WithCancel(ctx)
	s := speakerWithCancelFunc{Speaker: speaker, cancel: cancel}

	go func() {
		defer m.waitGroup.Done()
		if err := s.Start(ctxChild.Done()); err != nil {
			s.cancel()
			klog.Errorf("speaker %s start failed: %s", name, err.Error())
			m.errChan <- err
		}
	}()

	m.speakers[name] = s
	return nil
}

func (m *Manager) UnRegisterSpeaker(name string) {
	if s, exist := m.speakers[name]; exist && s.cancel != nil {
		s.cancel()
	}

	delete(m.speakers, name)
}

func (m *Manager) HandleEIP(ctx context.Context, eip *v1alpha2.Eip) error {
	if eip == nil {
		return nil
	}

	if _, exist := m.speakers[eip.GetProtocol()]; !exist {
		return fmt.Errorf("no registered speaker:[%s] eip:[%s]", eip.GetProtocol(), eip.GetName())
	}

	oldData, exist := m.pools[eip.GetName()]
	if !exist {
		klog.V(1).Infof("start to set balancer with new eip:%s", eip.GetName())
		if err := m.setBalancerWithEIP(ctx, eip); err != nil {
			return err
		}
		m.pools[eip.GetName()] = eip
		return nil
	}

	// delete eip - cancel advertise
	if !eip.DeletionTimestamp.IsZero() {
		klog.V(1).Infof("delete balancer with deleting eip:%s", eip.GetName())
		if err := m.delBalancerWithEIP(ctx, oldData); err != nil {
			return err
		}
		delete(m.pools, eip.GetName())
		return nil
	}

	// update speaker configurate
	if m.isSpeakerConfigUpdate(eip.Spec, oldData.Spec) {
		klog.V(1).Infof("update protocol with eip:%s", eip.GetName())
		if err := m.delBalancerWithEIP(ctx, oldData); err != nil {
			return err
		}
		delete(m.pools, eip.GetName())

		if err := m.setBalancerWithEIP(ctx, eip); err != nil {
			return err
		}
		m.pools[eip.GetName()] = eip
		return nil
	}

	// update status - for update service ip record
	if !reflect.DeepEqual(eip.Status.Used, oldData.Status.Used) {
		klog.V(1).Infof("update status with eip:%s", eip.GetName())
		add, del := util.DiffMaps(oldData.Status.Used, eip.Status.Used)
		if err := m.delBalancer(ctx, eip.GetProtocol(), del); err != nil {
			return err
		}
		if err := m.setBalancer(ctx, eip.GetProtocol(), add); err != nil {
			return err
		}
		m.pools[eip.GetName()] = eip
	}

	klog.V(1).Infof("no need to handle eip:%s", eip.GetName())
	return nil
}

// update speaker configurate
// protocol change or interface change
func (m *Manager) isSpeakerConfigUpdate(old, new v1alpha2.EipSpec) bool {
	if old.Protocol != new.Protocol {
		return true
	}

	if new.Protocol != constant.OpenELBProtocolBGP && old.Interface != new.Interface {
		return true
	}
	return false
}

func (m *Manager) delBalancerWithEIP(ctx context.Context, eip *v1alpha2.Eip) error {
	if err := m.delBalancer(ctx, eip.GetProtocol(), eip.Status.Used); err != nil {
		return err
	}

	r, err := iprange.ParseRange(eip.Spec.Address)
	if err != nil {
		return err
	}

	c := Config{Name: eip.Name, Iface: eip.Spec.Interface, IPRange: r}
	if err := m.speakers[eip.GetProtocol()].ConfigureWithEIP(c, true); err != nil {
		m.Event(eip, corev1.EventTypeWarning, "ConfigSpeakerFailed", err.Error())
		return err
	}
	m.Event(eip, corev1.EventTypeNormal, "ConfigSpeaker", fmt.Sprintf("unconfig openelb %s speaker successfully", eip.GetProtocol()))
	return nil
}

func (m *Manager) delBalancer(ctx context.Context, protocol string, usage map[string]string) error {
	for ip, svcs := range usage {
		if err := m.speakers[protocol].DelBalancer(ip); err != nil {
			m.addSvcEventRecorder(ctx, svcs, corev1.EventTypeWarning, "DelBalancer", err.Error())
			return err
		}

		m.addSvcEventRecorder(ctx, svcs, corev1.EventTypeNormal, "DelBalancer", "success to withdraw announcement for service")
	}

	return nil
}

func (m *Manager) setBalancerWithEIP(ctx context.Context, eip *v1alpha2.Eip) error {
	r, err := iprange.ParseRange(eip.Spec.Address)
	if err != nil {
		return err
	}

	c := Config{Name: eip.Name, Iface: eip.Spec.Interface, IPRange: r}
	if err := m.speakers[eip.GetProtocol()].ConfigureWithEIP(c, false); err != nil {
		m.Event(eip, corev1.EventTypeWarning, "ConfigSpeakerFailed", err.Error())
		return err
	}
	m.Event(eip, corev1.EventTypeNormal, "ConfigSpeaker", fmt.Sprintf("config openelb %s speaker successfully", eip.GetProtocol()))

	if err := m.setBalancer(ctx, eip.GetProtocol(), eip.Status.Used); err != nil {
		return err
	}
	return nil
}

func (m *Manager) setBalancer(ctx context.Context, protocol string, usage map[string]string) error {
	for ip, value := range usage {
		nodes, err := m.getServiceNodes(ctx, ip, value)
		if err != nil {
			return err
		}

		if len(nodes) == 0 {
			warnStr := fmt.Sprintf("delete balancer with no available nodes for service ip %s:%s.", ip, value)
			m.addSvcEventRecorder(ctx, value, corev1.EventTypeWarning, "SetBalancer", warnStr)
			klog.Warning(warnStr)

			return m.speakers[protocol].DelBalancer(ip)
		}

		nodeNames := []string{}
		for _, node := range nodes {
			nodeNames = append(nodeNames, node.Name)
		}
		sort.Slice(nodeNames, func(i, j int) bool {
			return nodeNames[i] < nodeNames[j]
		})
		if err := m.speakers[protocol].SetBalancer(ip, nodes); err != nil {
			m.addSvcEventRecorder(ctx, value, corev1.EventTypeWarning, "SetBalancer", err.Error())
			return err
		}

		m.addSvcEventRecorder(ctx, value, corev1.EventTypeNormal, "SetBalancer", fmt.Sprintf("success to add nexthops [%s]", strings.Join(nodeNames, ", ")))
	}
	return nil
}

func (m *Manager) addSvcEventRecorder(ctx context.Context, services, eventType, reason, message string) {
	for _, str := range strings.Split(services, ";") {
		svcInfo := strings.Split(str, "/")
		if len(svcInfo) != 2 {
			continue
		}

		svc := &corev1.Service{}
		if err := m.Get(ctx, types.NamespacedName{Namespace: svcInfo[0], Name: svcInfo[1]}, svc); err != nil {
			klog.Warningf("get service %s failed, err: %s", services, err.Error())
			continue
		}

		m.Event(svc, eventType, reason, message)
	}
}

func (m *Manager) HandleService(ctx context.Context, svc *corev1.Service) error {
	if svc == nil || !svc.DeletionTimestamp.IsZero() {
		return nil
	}

	eip, exist := m.pools[m.getSvcEIPUsed(svc)]
	if !exist || eip == nil {
		return nil
	}

	addr, err := iprange.ParseRange(eip.Spec.Address)
	if err != nil {
		return err
	}

	ingress := map[string]string{}
	for _, ip := range svc.Status.LoadBalancer.Ingress {
		value, ok := eip.Status.Used[ip.IP]
		if value == svc.GetNamespace()+"/"+svc.GetName() {
			ingress[ip.IP] = value
			continue
		}

		if ok {
			value += ";"
		}

		if addr.Contains(net.ParseIP(ip.IP)) {
			ingress[ip.IP] = value + svc.GetNamespace() + "/" + svc.GetName()
		}
	}

	if err := m.setBalancer(ctx, eip.GetProtocol(), ingress); err != nil {
		return err
	}
	return nil
}

// todo: annotions
func (m *Manager) getSvcEIPUsed(svc *corev1.Service) string {
	eipName := svc.Labels[constant.OpenELBEIPAnnotationKeyV1Alpha2]
	if eipName != "" {
		return eipName
	}

	return svc.Annotations[constant.OpenELBEIPAnnotationKeyV1Alpha2]
}

func (m *Manager) getServiceNodes(ctx context.Context, ip, svcs string) ([]corev1.Node, error) {
	nodeSets := map[string]corev1.Node{}
	nodeList := &corev1.NodeList{}
	if err := m.List(ctx, nodeList); err != nil {
		return nil, err
	}

	share := false
	svcArray := strings.Split(svcs, ";")
	if len(svcArray) > 1 {
		share = true
	}

	for _, str := range svcArray {
		//1. filter endpoints
		svcInfo := strings.Split(str, "/")
		if len(svcInfo) != 2 {
			continue
		}

		svc := &corev1.Service{}
		if err := m.Get(ctx, types.NamespacedName{Namespace: svcInfo[0], Name: svcInfo[1]}, svc); err != nil {
			return nil, err
		}
		endpoints := &corev1.Endpoints{}
		if err := m.Get(ctx, types.NamespacedName{Namespace: svc.GetNamespace(), Name: svc.GetName()}, endpoints); err != nil {
			return nil, err
		}

		//2. get next hops
		if svc.Spec.ExternalTrafficPolicy == corev1.ServiceExternalTrafficPolicyTypeLocal {
			if share {
				klog.Warningf("service %s's ExternalTrafficPolicyType is Local, but specify %s as a shared ip", svc.GetName(), ip)
			}

			active := make(map[string]bool)
			for _, subnet := range endpoints.Subsets {
				for _, addr := range subnet.Addresses {
					if addr.NodeName == nil {
						continue
					}
					active[*addr.NodeName] = true
				}
			}

			if len(active) == 0 {
				klog.Warningf("service %s's ExternalTrafficPolicyType is Local, and endpoint don't have nodeName, Please make sure the endpoints are configured correctly", svc.GetName())
				continue
			}

			for _, node := range nodeList.Items {
				if active[node.Name] {
					nodeSets[node.Name] = node
				}
			}

		} else {
			for _, node := range nodeList.Items {
				nodeSets[node.Name] = node
			}
		}
	}

	resultNodes := []corev1.Node{}
	for _, node := range nodeSets {
		resultNodes = append(resultNodes, node)
	}
	return resultNodes, nil
}

func (m *Manager) ResyncEIPSpeaker(ctx context.Context) error {
	eips := &v1alpha2.EipList{}
	if err := m.Client.List(ctx, eips, &client.ListOptions{}); err != nil {
		return err
	}

	for _, e := range eips.Items {
		if e.Spec.Protocol != constant.OpenELBProtocolLayer2 {
			continue
		}

		if err := m.setBalancer(ctx, constant.OpenELBProtocolLayer2, e.Status.Used); err != nil {
			klog.Warningf("resync speaker error: %s", err.Error())
		}
	}

	return nil
}
