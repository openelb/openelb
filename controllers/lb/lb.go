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

func (r *ServiceReconciler) ensureEIP(serv *corev1.Service, foundOrError bool) (string, error) {
	if ip := r.findEIP(serv); ip != "" {
		status := r.DS.GetEIPStatus(ip)
		if !status.Exist {
			return "", portererror.PorterError{Code: portererror.EIPNotExist}
		}
		if !status.Used {
			r.Log.Info("Service has eip but not in pool", "Service", serv.Name, "eip", ip)
			_, err := r.DS.AssignSpecifyIP(ip, serv.Annotations[constant.PorterProtocolAnnotationKey], serv.Name, serv.Namespace)
			if err != nil {
				r.Log.Info("Failed to mark eip as used", "eip", ip)
				return "", err
			}
		}
		return ip, nil
	}

	if foundOrError {
		return "", portererror.PorterError{Code: portererror.EIPNotExist}
	}

	resp, err := r.DS.AssignIP(serv.Name, serv.Namespace, serv.Annotations[constant.PorterProtocolAnnotationKey])
	if err != nil {
		r.Log.Error(nil, "Failed to get an ip from pool")
		return "", err
	}
	return resp.Address, nil
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

	ip, err := r.ensureEIP(serv, false)
	if err != nil {
		return err
	}

	if err = r.updateService(serv, ip, false); err != nil {
		return err
	}

	err = r.DS.SetBalancer(ip, nexthops)
	if err == nil {
		for _, nexthop := range nexthops {
			r.Log.Info("Add Route to ", "ip", nexthop)
		}
	}

	r.Log.Info(fmt.Sprintf("Pls visit %s:%d to check it out", ip, serv.Spec.Ports[0].Port))
	return nil
}

func (r *ServiceReconciler) updateService(serv *corev1.Service, ip string, delete bool) error {
	log := r.Log.WithValues("ip", ip, "namespace", serv.Namespace, "name", serv.Name)

	found := false
	var tmpIngress []corev1.LoadBalancerIngress
	for _, item := range serv.Status.LoadBalancer.Ingress {
		if item.IP == ip {
			found = true
			if !delete {
				break
			} else {
				continue
			}
		}
		tmpIngress = append(tmpIngress, item)
	}

	if found && !delete {
		return nil
	}

	if !delete {
		log.V(2).Info("Updating service status & metadata")
		serv.Status.LoadBalancer.Ingress = append(serv.Status.LoadBalancer.Ingress, corev1.LoadBalancerIngress{
			IP: ip,
		})

		if serv.Annotations == nil {
			serv.Annotations = make(map[string]string)
		}
		serv.Annotations[constant.PorterEIPAnnotationKey] = ip
	} else {
		log.V(2).Info("remove ingress from service")
		serv.Status.LoadBalancer.Ingress = tmpIngress
	}

	if err := r.Status().Update(context.TODO(), serv); err != nil {
		r.DS.UnassignIP(ip)
		return err
	} else if delete {
		return nil
	}

	r.Event(serv, corev1.EventTypeNormal, "LB Created", fmt.Sprintf("Successfully assign EIP %s", ip))
	return nil
}

func (r *ServiceReconciler) deleteLB(serv *corev1.Service) error {
	log := r.Log.WithValues("namespace", serv.Namespace, "name", serv.Name)
	ip, err := r.ensureEIP(serv, true)
	if err != nil {
		log.Info("EIP not exist, so we can delete service safely")
		return nil
	}

	if err = r.DS.DelBalancer(ip); err != nil {
		log.Info("Failed to delete router on service", "ip", ip)
		return err
	}

	if err = r.DS.UnassignIP(ip); err != nil {
		log.Info("Failed to revoke ip on service", "ip", ip)
		return err
	}

	return nil
}

func (r *ServiceReconciler) getServiceNodesIP(serv *corev1.Service) ([]string, error) {
	return kubeutil.GetServiceNodesIP(r.Client, serv)
}
