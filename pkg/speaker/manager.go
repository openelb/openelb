package speaker

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/go-logr/logr"
	"github.com/openelb/openelb/api/v1alpha2"
	"github.com/openelb/openelb/pkg/constant"
	"github.com/openelb/openelb/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type handleResult int

const (
	none handleResult = iota
	wantDelete
	wantStore
	wantReset
)

type recordInfo struct {
	ips       []string
	eip       string
	protocols string
	speaker   string
}

func (r *recordInfo) String() string {
	return fmt.Sprintf("{eip:%s, protocol:%s, speaker:%s, ips:%s}",
		r.eip, r.protocols, r.speaker, strings.Join(r.ips, ";"))
}

type speaker struct {
	Speaker
	ch chan struct{}
}

type Manager struct {
	sync.RWMutex
	client.Client
	logr.Logger

	// map[name]Speaker
	speakers map[string]speaker

	// map[eip.name]EIP
	eips map[string]*v1alpha2.Eip

	// map[NamespacedName]recordInfo
	ips map[string]*recordInfo
}

func NewSpeakerManager(c client.Client, l logr.Logger) *Manager {
	return &Manager{
		Client:   c,
		Logger:   l,
		speakers: make(map[string]speaker, 0),
		eips:     make(map[string]*v1alpha2.Eip, 0),
		ips:      make(map[string]*recordInfo, 0),
	}
}

func (m *Manager) RegisterSpeaker(name string, s Speaker) error {
	t := speaker{
		Speaker: s,
		ch:      make(chan struct{}),
	}

	err := s.Start(t.ch)
	if err == nil {
		m.speakers[name] = t
		return nil
	}

	return err
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

func (m *Manager) HandleEIP(eip *v1alpha2.Eip) error {
	if eip == nil {
		return nil
	}

	if eip.GetProtocol() == constant.OpenELBProtocolVip && m.GetSpeaker(eip.GetProtocol()) == nil {
		m.Info(fmt.Sprintf("no registered speaker:[%s] eip:[%s]", eip.GetProtocol(), eip.GetName()))
		return nil
	}

	m.Lock()
	defer m.Unlock()
	_, exist := m.eips[eip.GetName()]
	if exist {
		if !eip.DeletionTimestamp.IsZero() {
			m.V(3).Info(fmt.Sprintf("deleting eip:[%s]", eip.GetName()))
			delete(m.eips, eip.GetName())
		}

		return nil
	}

	if !eip.DeletionTimestamp.IsZero() {
		return nil
	}

	// TODO: layer2 validate NIC infos
	m.V(3).Info(fmt.Sprintf("store eip:[%s]", eip.GetName()))
	m.eips[eip.GetName()] = eip
	return nil
}

func (m *Manager) HandleService(svc *corev1.Service) error {
	if svc == nil {
		return nil
	}

	svcNSName := types.NamespacedName{Namespace: svc.GetNamespace(), Name: svc.GetName()}.String()

	// get local record info, if localRecord is nil -- no record exists(initing)
	localRecord, exist := m.ips[svcNSName]
	if !exist && svc.Spec.Type != corev1.ServiceTypeLoadBalancer {
		m.V(3).Info(fmt.Sprintf("openelb no record about servic:[%s] type:[%s]", svcNSName, svc.Spec.Type))
		return nil
	}

	// deleting svc
	if !svc.DeletionTimestamp.IsZero() {
		if localRecord == nil {
			m.Info(fmt.Sprintf("service:%s is deleting, so don't handler it", svcNSName))
			return nil
		}
		m.V(3).Info(fmt.Sprintf("deleting svc:[%s - %s], so delete LoadBalancer", svc.GetName(), localRecord.String()))
		return m.delLoadBalancer(svc, localRecord)
	}

	// update service from TypeLoadBalancer to other type
	if exist && svc.Spec.Type != corev1.ServiceTypeLoadBalancer {
		if localRecord == nil {
			m.Info(fmt.Sprintf("local:%s record exists but no data. changed svc'type is %s", svcNSName, svc.Spec.Type))
			return nil
		}
		m.V(3).Info(fmt.Sprintf("change service:[%s] type from LoadBalancer to %s, so delete LoadBalancer", svc.GetName(), svc.Spec.Type))
		return m.delLoadBalancer(svc, localRecord)
	}

	// get service record info
	svcRecord := m.getSvcRecordInfo(svc)
	if svcRecord == nil {
		m.V(3).Info(fmt.Sprintf("get service:[%s] record info error", svc.GetName()))
		return nil
	}

	return m.handleSvcBalance(svc, localRecord, svcRecord)
}

func (m *Manager) handleSvcBalance(svc *corev1.Service, localRecord, svcRecord *recordInfo) error {
	switch m.getHandleResult(localRecord, svcRecord) {
	case wantDelete:
		m.V(1).Info("delLoadBalancer " + localRecord.String())
		return m.delLoadBalancer(svc, localRecord)
	case wantStore:
		m.V(1).Info("setLoadBalancer " + svcRecord.String())
		return m.setLoadBalancer(svc, svcRecord)
	case wantReset:
		m.V(1).Info(fmt.Sprintf("resetLoadBalancer localRecord:%s svcRecord:%s", localRecord.String(), svcRecord.String()))
		return m.resetLoadBalancer(svc, localRecord, svcRecord)

	default:
		// endpoint update
		if svc.Spec.ExternalTrafficPolicy == corev1.ServiceExternalTrafficPolicyTypeLocal {
			m.V(1).Info("Local svc.externalTrafficPolicy updateLoadBalancer")
			return m.updateLoadBalancer(svc, svcRecord)
		}
		m.V(1).Info("handler do nothing")
	}

	return nil
}

func (m *Manager) getHandleResult(localRecord, svcRecord *recordInfo) handleResult {
	if reflect.DeepEqual(localRecord, svcRecord) {
		return none
	}

	if localRecord == nil {
		return wantStore
	}

	if svcRecord == nil {
		return wantDelete
	}

	m.V(4).Info(fmt.Sprintf("local record is: %s  service record is: %s", localRecord.String(), svcRecord.String()))
	if !reflect.DeepEqual(localRecord.ips, svcRecord.ips) {
		if len(svcRecord.ips) == 0 {
			return wantDelete
		}

		if len(localRecord.ips) == 0 {
			return wantStore
		}

		return wantReset
	}

	if localRecord.eip != svcRecord.eip {
		if svcRecord.eip == "" {
			return wantDelete
		}

		if localRecord.eip == "" {
			return wantStore
		}

		return wantReset
	}

	if svcRecord.protocols != localRecord.protocols {
		if svcRecord.protocols == "" {
			return wantDelete
		}

		if localRecord.protocols == "" {
			return wantStore
		}

		return wantReset
	}

	return none
}

func (m *Manager) getSvcRecordInfo(svc *corev1.Service) *recordInfo {
	if svc == nil || svc.Annotations == nil {
		return nil
	}

	r := &recordInfo{}
	if value, ok := svc.Annotations[constant.OpenELBAnnotationKey]; !ok || value != constant.OpenELBAnnotationValue {
		return r
	}

	if protocol, ok := svc.Annotations[constant.OpenELBProtocolAnnotationKey]; ok {
		r.protocols = protocol
	}

	if eipname, ok := svc.Annotations[constant.OpenELBEIPAnnotationKeyV1Alpha2]; ok {
		r.eip = eipname
	}

	for _, v := range svc.Status.LoadBalancer.Ingress {
		r.ips = append(r.ips, v.IP)
	}

	eip, exist := m.eips[r.eip]
	if !exist || eip == nil {
		return r
	}

	r.speaker = eip.GetSpeakerName()
	r.protocols = eip.GetProtocol()
	return r
}

func getRecordValue(record *recordInfo) string {
	return record.eip + "/" + strings.Join(record.ips, ";")
}

func (m *Manager) resetLoadBalancer(svc *corev1.Service, localRecord, svcRecord *recordInfo) error {
	if err := m.delLoadBalancer(svc, localRecord); err != nil {
		return err
	}

	return m.setLoadBalancer(svc, svcRecord)
}

func (m *Manager) delLoadBalancer(svc *corev1.Service, record *recordInfo) error {
	if record == nil || len(record.ips) == 0 {
		return nil
	}

	if svc == nil {
		return nil
	}

	sp, ok := m.speakers[record.speaker]
	if !ok {
		return fmt.Errorf("there is no speaker:%s\n", record.speaker)
	}

	for _, addr := range record.ips {
		if record.protocols == constant.OpenELBProtocolVip {
			addr = fmt.Sprintf("%s:%s", addr, svc.Namespace+"/"+svc.Name)
		}

		if err := sp.DelBalancer(addr); err != nil {
			return err
		}
	}

	delete(m.ips, svc.GetNamespace()+"/"+svc.GetName())
	return nil
}

func (m *Manager) updateLoadBalancer(svc *corev1.Service, record *recordInfo) error {
	return m.setLoadBalancer(svc, record)
}

func (m *Manager) setLoadBalancer(svc *corev1.Service, record *recordInfo) error {
	if record == nil || len(record.ips) == 0 {
		return nil
	}

	if svc == nil {
		return nil
	}

	nodes, err := m.getServiceNodes(svc)
	if err != nil {
		return err
	}

	var announceNodes []corev1.Node
	announceNodes = append(announceNodes, nodes...)
	// todo: layer2 mode

	sp, ok := m.speakers[record.speaker]
	if !ok {
		return fmt.Errorf("there is no speaker:%s\n", record.speaker)
	}

	for _, addr := range record.ips {
		if record.protocols == constant.OpenELBProtocolVip {
			addr = fmt.Sprintf("%s:%s", addr, svc.Namespace+"/"+svc.Name)
		}

		if err := sp.SetBalancer(addr, announceNodes); err != nil {
			return err
		}
	}

	m.ips[svc.GetNamespace()+"/"+svc.GetName()] = record
	return nil
}

func (m *Manager) getServiceNodes(svc *corev1.Service) ([]corev1.Node, error) {
	//1. filter endpoints
	endpoints := &corev1.Endpoints{}
	err := m.Get(context.TODO(), types.NamespacedName{Namespace: svc.GetNamespace(), Name: svc.GetName()}, endpoints)
	if err != nil {
		return nil, err
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

	//2. get next hops
	nodeList := &corev1.NodeList{}
	err = m.List(context.TODO(), nodeList)
	if err != nil {
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
		clone := svc.DeepCopy()
		clone.Spec.ExternalTrafficPolicy = corev1.ServiceExternalTrafficPolicyTypeCluster
		_ = m.Update(context.Background(), clone)
		m.V(1).Info(fmt.Sprintf("endpoint don't have nodeName, so cannot set externalTrafficPolicy to Local"))
	}

	for _, node := range nodeList.Items {
		if util.NodeReady(&node) {
			resultNodes = append(resultNodes, node)
		}
	}

	return resultNodes, nil
}
