package service

import (
	"context"
	"fmt"

	"github.com/kubesphere/porter/pkg/apis/network/v1alpha1"
	"github.com/kubesphere/porter/pkg/bgp/routes"
	"github.com/kubesphere/porter/pkg/strategy"
	"github.com/kubesphere/porter/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ReconcileService) getExternalIP(serv *corev1.Service) (string, error) {
	if len(serv.Spec.ExternalIPs) > 0 {
		return serv.Spec.ExternalIPs[0], nil
	}
	listOptions := &client.ListOptions{}
	eipList := r.Get(context.Background(), listOptions, &v1alpha1.EIPList)
	ipStrategy, _ := strategy.GetStrategy(strategy.DefaultStrategy)
	ip, err := ipStrategy.Select(serv, eipList)
	if err != nil {
		return "", err
	}
	return ip.Spec.Address, nil
}

func (r *ReconcileService) createLB(serv *corev1.Service) error {
	ip, err := getExternalIP(serv)
	if err != nil {
		return err
	}
	localip := util.GetOutboundIP()
	if err := routes.AddRoute(ip, 32, localip); err != nil {
		return err
	}
	log.Info("Routed added successful", "ServiceName", serv.Name, "Namespace", serv.Namespace)
	if err := routes.AddVIP(ip, 32); err != nil {
		log.Error(err, "Failed to create vip")
		return err
	}
	log.Info("VIP added successful", "ServiceName", serv.Name, "Namespace", serv.Namespace)
	log.Info(fmt.Sprintf("Pls visit %s:%d to check it out", ip, serv.Spec.Ports[0].Port))
	return nil
}

func (r *ReconcileService) deleteLB(serv *corev1.Service) error {
	ip, err := getExternalIP(serv)
	if err != nil {
		return err
	}
	log.Info("Routed deleted successful", "ServiceName", serv.Name, "Namespace", serv.Namespace)
	if err := routes.DeleteVIP(ip, 32); err != nil {
		return err
	}
	log.Info("VIP deleted successful", "ServiceName", serv.Name, "Namespace", serv.Namespace)
	return nil
}

func (r *ReconcileService) checkLB(serv *corev1.Service) bool {
	ip, err := getExternalIP(serv)
	if err != nil {
		log.Error(err, "Failed to get ip")
		return false
	}
	return routes.IsRouteAdded(ip, 32)
}
