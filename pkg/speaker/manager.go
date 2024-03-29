package speaker

import (
	"context"
	"fmt"
	"net"
	"reflect"
	"strings"

	"github.com/openelb/openelb/api/v1alpha2"
	"github.com/openelb/openelb/pkg/constant"
	"github.com/openelb/openelb/pkg/util"
	"github.com/openelb/openelb/pkg/util/iprange"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type speaker struct {
	Speaker
	ch chan struct{}
}

type Manager struct {
	client.Client

	speakers map[string]speaker
	pools    map[string]*v1alpha2.Eip
}

func NewSpeakerManager(c client.Client) *Manager {
	return &Manager{
		Client:   c,
		speakers: make(map[string]speaker, 0),
		pools:    make(map[string]*v1alpha2.Eip, 0),
	}
}

func (m *Manager) RegisterSpeaker(name string, s Speaker) error {
	t := speaker{
		Speaker: s,
		ch:      make(chan struct{}),
	}

	if err := s.Start(t.ch); err != nil {
		return err
	}

	m.speakers[name] = t
	return nil
}

func (m *Manager) UnRegisterSpeaker(name string) {
	t, ok := m.speakers[name]
	if ok {
		close(t.ch)
	}
	delete(m.speakers, name)
}

func (m *Manager) GetSpeaker(name string) Speaker {
	t, ok := m.speakers[name]
	if ok {
		return t.Speaker
	}

	return nil
}

func (m *Manager) HandleEIP(ctx context.Context, eip *v1alpha2.Eip) error {
	if eip == nil {
		return nil
	}

	if m.GetSpeaker(eip.GetProtocol()) == nil {
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
		if err := m.delBalancerWithEIP(oldData); err != nil {
			return err
		}
		delete(m.pools, eip.GetName())
		return nil
	}

	// update speaker configurate
	if m.isSpeakerConfigUpdate(eip.Spec, oldData.Spec) {
		klog.V(1).Infof("update protocol with eip:%s", eip.GetName())
		if err := m.delBalancerWithEIP(oldData); err != nil {
			return err
		}

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
		if err := m.delBalancer(eip.GetProtocol(), del); err != nil {
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
	return (old.Protocol == new.Protocol && new.Protocol == constant.OpenELBProtocolLayer2 &&
		old.Interface != new.Interface) || old.Protocol != new.Protocol
}

func (m *Manager) delBalancerWithEIP(eip *v1alpha2.Eip) error {
	if err := m.delBalancer(eip.GetProtocol(), eip.Status.Used); err != nil {
		return err
	}

	return m.speakers[eip.GetProtocol()].ConfigureWithEIP(eip, true)
}

func (m *Manager) delBalancer(protocol string, usage map[string]string) error {
	for ip, value := range usage {
		if protocol == constant.OpenELBProtocolVip {
			ip = fmt.Sprintf("%s:%s", ip, value)
		}

		if err := m.speakers[protocol].DelBalancer(ip); err != nil {
			return err
		}
	}

	return nil
}

func (m *Manager) setBalancerWithEIP(ctx context.Context, eip *v1alpha2.Eip) error {
	if err := m.speakers[eip.GetProtocol()].ConfigureWithEIP(eip, false); err != nil {
		return err
	}

	return m.setBalancer(ctx, eip.GetProtocol(), eip.Status.Used)
}

func (m *Manager) setBalancer(ctx context.Context, protocol string, usage map[string]string) error {
	for ip, value := range usage {
		nodes, err := m.getServiceNodes(ctx, value)
		if err != nil {
			return err
		}

		if protocol == constant.OpenELBProtocolVip {
			ip = fmt.Sprintf("%s:%s", ip, value)
		}

		if err := m.speakers[protocol].SetBalancer(ip, nodes); err != nil {
			return err
		}
	}
	return nil
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
		value, ok := ingress[ip.IP]
		if ok {
			value += ";"
		}

		if addr.Contains(net.ParseIP(ip.IP)) {
			ingress[ip.IP] = value + svc.GetNamespace() + "/" + svc.GetName()
		}
	}

	return m.setBalancer(ctx, eip.GetProtocol(), ingress)
}

// todo: annotions
func (m *Manager) getSvcEIPUsed(svc *corev1.Service) string {
	eipName := svc.Labels[constant.OpenELBEIPAnnotationKeyV1Alpha2]
	if eipName != "" {
		return eipName
	}

	return svc.Annotations[constant.OpenELBEIPAnnotationKeyV1Alpha2]
}

func (m *Manager) getServiceNodes(ctx context.Context, svcString string) ([]corev1.Node, error) {
	//1. filter endpoints
	svc := &corev1.Service{}
	active := make(map[string]bool)
	for _, str := range strings.Split(svcString, ";") {
		svcInfo := strings.Split(str, "/")
		if len(svcInfo) != 2 {
			continue
		}

		if err := m.Get(ctx, types.NamespacedName{Namespace: svcInfo[0], Name: svcInfo[1]}, svc); err != nil {
			return nil, err
		}

		endpoints := &corev1.Endpoints{}
		if err := m.Get(ctx, types.NamespacedName{Namespace: svc.GetNamespace(), Name: svc.GetName()}, endpoints); err != nil {
			return nil, err
		}

		for _, subnet := range endpoints.Subsets {
			for _, addr := range subnet.Addresses {
				if addr.NodeName == nil {
					continue
				}
				active[*addr.NodeName] = true
			}
		}
	}

	//2. get next hops
	nodeList := &corev1.NodeList{}
	if err := m.List(ctx, nodeList); err != nil {
		return nil, err
	}

	resultNodes := make([]corev1.Node, 0)
	if svc.Spec.ExternalTrafficPolicy == corev1.ServiceExternalTrafficPolicyTypeLocal && len(active) > 0 {
		for _, node := range nodeList.Items {
			if active[node.Name] {
				resultNodes = append(resultNodes, node)
			}
		}

		return resultNodes, nil
	}

	if svc.Spec.ExternalTrafficPolicy == corev1.ServiceExternalTrafficPolicyTypeLocal && len(active) == 0 {
		klog.Warningf("service %s's ExternalTrafficPolicyType is Local, and endpoint don't have nodeName, Please make sure the endpoints are configured correctly", svc.GetName())
	}

	for _, node := range nodeList.Items {
		if util.NodeReady(&node) {
			resultNodes = append(resultNodes, node)
		}
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

		m.setBalancer(ctx, constant.OpenELBProtocolLayer2, e.Status.Used)
	}

	return nil
}
