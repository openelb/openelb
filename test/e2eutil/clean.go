package e2eutil

import (
	"context"

	networkv1alpha1 "github.com/kubesphere/porter/pkg/apis/network/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
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
