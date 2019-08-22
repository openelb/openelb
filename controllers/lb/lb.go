package lb

import (
	"context"
	"fmt"
	"strings"

	"github.com/kubesphere/porter/api/v1alpha1"
	"github.com/kubesphere/porter/pkg/bgp/routes"
	portererror "github.com/kubesphere/porter/pkg/errors"
	"github.com/kubesphere/porter/pkg/kubeutil"
	"github.com/kubesphere/porter/pkg/strategy"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

func (r *ServiceReconciler) getExternalIP(serv *corev1.Service, useField bool) (string, error) {
	if len(serv.Status.LoadBalancer.Ingress) > 0 {
		if r.getEIPByString(serv.Status.LoadBalancer.Ingress[0].IP) != nil {
			return serv.Status.LoadBalancer.Ingress[0].IP, nil
		}
		return "", portererror.NewEIPNotFoundError(strings.Join(serv.Spec.ExternalIPs, ";"))
	} else {
		if useField {
			return "", portererror.NewEIPNotFoundError("")
		}
	}
	eipList := &v1alpha1.EipList{}
	err := r.List(context.Background(), eipList)
	if err != nil {
		return "", err
	}
	ipStrategy, _ := strategy.GetStrategy(strategy.DefaultStrategy)
	ip, err := ipStrategy.Select(serv, eipList)
	if err != nil {
		return "", err
	}
	return ip.Spec.Address, nil
}

func (r *ServiceReconciler) getEIPByString(ip string) *v1alpha1.Eip {
	eipList := &v1alpha1.EipList{}
	err := r.List(context.Background(), eipList)
	if err != nil {
		r.Log.Error(err, "Faided to get EIP list")
		return nil
	}
	for _, eip := range eipList.Items {
		if eip.Spec.Address == ip {
			return &eip
		}
	}
	return nil
}

func (r *ServiceReconciler) createLB(serv *corev1.Service) error {
	ip, err := r.getExternalIP(serv, false)
	if err != nil {
		return err
	}
	nexthops, err := r.getServiceNodesIP(serv)
	if err != nil {
		if errors.IsNotFound(err) {
			r.Log.Info("Detect no available endpoints now", "ServiceName", serv.GetName(), "Namespace", serv.GetNamespace())
			return nil
		}
		r.Log.Error(nil, "Failed to get ip of nodes where endpoints locate in")
		return err
	}
	if nexthops == nil {
		r.Log.Info("No endpoints is ready now")
		return nil
	}
	if err := routes.AddRoutes(ip, 32, nexthops); err != nil {
		return err
	}
	r.Log.Info("Routed added successfully", "ServiceName", serv.Name, "Namespace", serv.Namespace)
	for _, nexthop := range nexthops {
		r.Log.Info("Add Route to ", "ip", nexthop)
	}
	r.Event(serv, corev1.EventTypeNormal, "BGP Route Pulished", "Route to external-ip added successfully")
	err = r.markEIPPorts(ip, serv.Spec.Ports, true)
	if err != nil {
		r.Log.Error(nil, "failed to mark ports of ip used")
		return err
	}
	exist := false
	for _, item := range serv.Status.LoadBalancer.Ingress {
		if item.IP == ip {
			exist = true
			break
		}
	}
	if !exist {
		serv.Status.LoadBalancer.Ingress = append(serv.Status.LoadBalancer.Ingress, corev1.LoadBalancerIngress{
			IP: ip,
		})
		err = r.Status().Update(context.Background(), serv)
		if err != nil {
			r.Log.Error(nil, "failed to update LoadBalancer of service", "ServiceName", serv.Name, "Namespace", serv.Namespace)
			return err
		}
		r.Event(serv, corev1.EventTypeNormal, "LB Created", fmt.Sprintf("Successfully assign EIP %s", ip))
	}
	r.Log.Info(fmt.Sprintf("Pls visit %s:%d to check it out", ip, serv.Spec.Ports[0].Port))
	return nil
}

func (r *ServiceReconciler) deleteLB(serv *corev1.Service) error {
	ip, err := r.getExternalIP(serv, true)
	if err != nil {
		if _, ok := err.(portererror.EIPNotFoundError); ok {
			r.Log.Info("Have not assign a ip, skip deleting LB")
			return nil
		}
		return err
	}
	nodeIPs, err := r.getServiceNodesIP(serv)
	if err != nil {
		if errors.IsNotFound(err) {
			r.Log.Info("Endpoints is disappearing,try to delete ip in global table")
			err := routes.DeleteAllRoutesOfIP(ip)
			if err != nil {
				return err
			}
		} else {
			r.Log.Error(nil, "error in get nodes ip when try to deleting bgp routes")
			return err
		}
	} else {
		err = routes.DeleteRoutes(ip, nodeIPs)
		if err != nil {
			r.Log.Error(nil, "Failed to delete routes ", "nexthops", nodeIPs)
		}
	}
	err = r.markEIPPorts(ip, serv.Spec.Ports, false)
	if err != nil {
		r.Log.Error(nil, "failed to update status of eip", "ServiceName", serv.Name, "Namespace", serv.Namespace, "ip", ip)
		return err
	}
	r.Log.Info("Routed deleted successful", "ServiceName", serv.Name, "Namespace", serv.Namespace)
	return nil
}

func (r *ServiceReconciler) getServiceNodesIP(serv *corev1.Service) ([]string, error) {
	return kubeutil.GetServiceNodesIP(r.Client, serv)
}

func (r *ServiceReconciler) markEIPPorts(ip string, ports []corev1.ServicePort, used bool) error {
	eip := r.getEIPByString(ip)
	eip.Status.Occupied = used
	return r.Status().Update(context.Background(), eip)
}
