package lb

import (
	"context"
	"fmt"
	"github.com/kubesphere/porter/pkg/constant"
	portererror "github.com/kubesphere/porter/pkg/errors"
	"github.com/kubesphere/porter/pkg/kubeutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

func (r *ServiceReconciler) findEIP(svc *corev1.Service) string {
	if svc.Annotations != nil {
		if ip, ok := svc.Annotations[constant.PorterEIPAnnotationKey]; ok {
			return ip
		}
	}

	if svc.Spec.LoadBalancerIP != "" {
		return svc.Spec.LoadBalancerIP
	}

	return ""
}

func (r *ServiceReconciler) ensureEIP(serv *corev1.Service, foundOrError bool) (string, bool, error) {
	if ip := r.findEIP(serv); ip != "" {
		status := r.IPAM.CheckEIPStatus(ip)
		if !status.Exist {
			return "", false, portererror.NewEIPNotFoundError(ip)
		}
		if !status.Used {
			r.Log.Info("Service has eip but not in pool", "Service", serv.Name, "eip", ip)
			_, err := r.IPAM.AssignSpecifyIP(serv, ip)
			if err != nil {
				r.Log.Info("Failed to mark eip as used", "eip", ip)
				return "", false, err
			}
		}
		return ip, false, nil
	}

	if foundOrError {
		return "", false, portererror.NewEIPNotFoundError("")
	}
	resp, err := r.IPAM.AssignIP(serv)
	if err != nil {
		r.Log.Error(nil, "Failed to get an ip from pool")
		return "", true, err
	}
	return resp.Address, true, nil
}

func (r *ServiceReconciler) addRoutes(ip string, prefix uint32, nexthops []string) error {
	toAdd, toDelete, err := r.ReconcileRoutes(ip, prefix, nexthops)
	if err != nil {
		return err
	}
	err = r.AddMultiRoutes(ip, prefix, toAdd)
	if err != nil {
		return err
	}
	err = r.DeleteMultiRoutes(ip, prefix, toDelete)
	if err != nil {
		return err
	}
	return nil
}

func (r *ServiceReconciler) advertiseIP(serv *corev1.Service, ip string, nexthops []string) error {

	if err := r.addRoutes(ip, 32, nexthops); err != nil {
		return err
	}
	r.Log.Info("Routed added successfully")
	for _, nexthop := range nexthops {
		r.Log.Info("Add Route to ", "ip", nexthop)
	}
	r.Event(serv, corev1.EventTypeNormal, "BGP Route Pulished", "Route to external-ip added successfully")
	return nil
}

func (r *ServiceReconciler) createLB(serv *corev1.Service) error {
	nexthops, err := r.getServiceNodesIP(serv)
	if err != nil {
		if errors.IsNotFound(err) {
			r.Log.Info("Detect no available endpoints now")
			return nil
		}
		r.Log.Error(nil, "Failed to get ip of nodes where endpoints locate in")
		return err
	}
	if nexthops == nil {
		r.Log.Info("No endpoints is ready now")
		return nil
	}
	ip, newAssign, err := r.ensureEIP(serv, false)
	if err != nil {
		return err
	}

	protocol := r.IPAM.ProtocolForEIP(ip)
	switch protocol {
	case constant.PorterProtocolBGP:
		err = r.advertiseIP(serv, ip, nexthops)
	case constant.PorterProtocolLayer2:
		err = r.announcer.SetBalancer(ip, nexthops[0])
	default:
		r.Log.Info("invalid protocol", "protocol", protocol, "ip", ip)
		err = portererror.NewEIPProtocolNotFoundError()
	}

	if err != nil {
		if newAssign {
			r.Log.Info("Failed to advertise ip,try to revoke ip", "ip", ip)
			revokeErr := r.IPAM.RevokeIP(ip)
			if revokeErr != nil {
				panic(revokeErr)
			}
		}
		return err
	}

	if err = r.updateService(serv, ip, newAssign); err != nil {
		return err
	}
	r.Log.Info(fmt.Sprintf("Pls visit %s:%d to check it out", ip, serv.Spec.Ports[0].Port))
	return nil
}

func (r *ServiceReconciler) updateService(serv *corev1.Service, ip string, newAssign bool) error {
	found := false
	for _, item := range serv.Status.LoadBalancer.Ingress {
		if item.IP == ip {
			found = true
			break
		}
	}
	if !found {
		r.Log.V(2).Info("Updating service status")
		serv.Status.LoadBalancer.Ingress = append(serv.Status.LoadBalancer.Ingress, corev1.LoadBalancerIngress{
			IP: ip,
		})
		if err := r.Status().Update(context.TODO(), serv); err != nil {
			if newAssign {
				r.IPAM.RevokeIP(ip)
			}
			return err
		}
	}
	r.Log.V(2).Info("Updating service metadata")
	if serv.Annotations == nil {
		serv.Annotations = make(map[string]string)
	}
	if store, ok := serv.Annotations[constant.PorterEIPAnnotationKey]; !ok || store != ip {
		serv.Annotations[constant.PorterEIPAnnotationKey] = ip
	}
	if _, ok := serv.Annotations[constant.PorterProtocolAnnotationKey]; !ok {
		serv.Annotations[constant.PorterProtocolAnnotationKey] = r.IPAM.ProtocolForEIP(ip)
	}
	if err := r.Update(context.Background(), serv); err != nil {
		r.Log.Error(err, "Faided to add annotations")
		return err
	}
	r.Event(serv, corev1.EventTypeNormal, "LB Created", fmt.Sprintf("Successfully assign EIP %s", ip))
	return nil
}

func (r *ServiceReconciler) deleteLB(serv *corev1.Service) error {
	ip, _, err := r.ensureEIP(serv, true)
	if err != nil {
		if _, ok := err.(portererror.EIPNotFoundError); ok {
			r.Log.Info("Have not assign a ip, skip deleting LB")
			return nil
		}
		return err
	}
	err = r.IPAM.RevokeIP(ip)
	if err != nil {
		r.Log.Info("Failed to revoke ip on service", "ip", ip)
		return err
	}

	protocol := r.IPAM.ProtocolForEIP(ip)
	switch protocol {
	case constant.PorterProtocolBGP:
		nodeIPs, err := r.getServiceNodesIP(serv)
		if err != nil {
			if errors.IsNotFound(err) {
				r.Log.Info("Endpoints is disappearing,try to delete ip in global table")
				err := r.DeleteAllRoutesOfIP(ip)
				if err != nil {
					return err
				}
			} else {
				r.Log.Error(nil, "error in get nodes ip when try to deleting bgp routes")
				return err
			}
		} else {
			err = r.DeleteMultiRoutes(ip, 32, nodeIPs)
			if err != nil {
				r.Log.Error(nil, "Failed to delete routes ", "nexthops", nodeIPs)
			}
		}
		r.Log.Info("Routed deleted successful")
	case constant.PorterProtocolLayer2:
		r.announcer.DeleteBalancer(ip)
	default:
		return portererror.NewEIPProtocolNotFoundError()
	}
	return nil
}

func (r *ServiceReconciler) getServiceNodesIP(serv *corev1.Service) ([]string, error) {
	return kubeutil.GetServiceNodesIP(r.Client, serv)
}
