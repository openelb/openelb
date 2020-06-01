package e2eutil

import (
	"context"
	"os/exec"
	"strings"
	"time"

	networkv1alpha1 "github.com/kubesphere/porter/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CleanEIPList(dynclient client.Client) error {
	eiplist := &networkv1alpha1.EipList{}
	err := dynclient.List(context.TODO(), eiplist)
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

func EnsureNamespaceClean(nsname string) error {
	return wait.Poll(5*time.Second, 30*time.Second, func() (bool, error) {
		cmd := exec.Command("kubectl", "get", "all", "-n", nsname)
		str, err := cmd.CombinedOutput()
		if err != nil {
			return false, err
		}
		if strings.Contains(string(str), "No resources found") {
			return true, nil
		}
		return false, nil
	})
}
