package e2eutil

import (
	"context"
	"time"

	networkv1alpha1 "github.com/kubesphere/porter/pkg/apis/network/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CleanEIPList(dynclient client.Client) error {
	eiplist := &networkv1alpha1.EIPList{}
	err := dynclient.List(context.TODO(), nil, eiplist)
	if err != nil {
		if errors.IsNotFound(err) || errors.IsUnexpectedObjectError(err) {
			return nil
		}
	} else {
		return err
	}
	for _, eip := range eiplist.Items {
		if len(eip.GetFinalizers()) > 0 {
			eip.Finalizers = nil
			err = dynclient.Update(context.TODO(), &eip)
			if err != nil {
				return err
			}
		} else {
			dynclient.Delete(context.Background(), &eip)
		}
	}
	return nil

}

func EnsureNamespaceClean(nsname string, k8sclient client.Client) error {
	ns := &corev1.Namespace{}
	err := k8sclient.Get(context.Background(), types.NamespacedName{Name: nsname}, ns)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	err = k8sclient.Delete(context.TODO(), ns)
	if err != nil {
		return err
	}
	return WaitForDeletion(k8sclient, ns, 10*time.Second, 30*time.Second)
}
