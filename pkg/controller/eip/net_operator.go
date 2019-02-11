package eip

import (
	"fmt"
	"os"

	networkv1alpha1 "github.com/kubesphere/porter/pkg/apis/network/v1alpha1"
	"github.com/kubesphere/porter/pkg/controller/eip/nettool"
)

func (r *ReconcileEIP) AddRule(instance *networkv1alpha1.EIP) error {
	if instance.Spec.Address != "" {
		rule := nettool.NewEIPRule(instance.Spec.Address, 32)
		nodename := os.Getenv("MY_NODE_NAME")
		if yes, err := rule.IsExist(); err != nil {
			return err
		} else {
			if !yes {
				err = rule.Add()
				if err != nil {
					return err
				}
				r.recorder.Event(instance, "Normal", "Rule Created", fmt.Sprintf("Created ip rule for EIP %s in agent %s", instance.Spec.Address, nodename))
			} else {
				log.Info("Detect rule in node")
				r.recorder.Event(instance, "Normal", "Detect rule in node", fmt.Sprintf("Skipped Creating ip rule for EIP %s in agent %s", instance.Spec.Address, nodename))
			}
		}
	}
	return nil
}

func (r *ReconcileEIP) DelRule(instance *networkv1alpha1.EIP) error {
	if instance.Spec.Address != "" {
		rule := nettool.NewEIPRule(instance.Spec.Address, 32)
		if yes, err := rule.IsExist(); err != nil {
			return err
		} else {
			if yes {
				err = rule.Delete()
				if err != nil {
					return err
				}
				log.Info("Rule is deleted successfully")
			} else {
				log.Info("Try to delete a non-exist rule")
			}
		}
	}
	return nil
}
