package eip

import (
	"fmt"
	"os"

	networkv1alpha1 "github.com/kubesphere/porter/api/v1alpha1"
	"github.com/kubesphere/porter/pkg/nettool"
)

func (r *EipReconciler) AddRule(instance *networkv1alpha1.Eip) error {
	if instance.Spec.Address != "" {
		rule := nettool.NewEIPRule(instance.Spec.Address, 32)
		nodeName := os.Getenv("MY_NODE_NAME")
		if ok, err := rule.IsExist(); err != nil {
			return err
		} else {
			if !ok {
				err = rule.Add()
				if err != nil {
					return err
				}
				r.Event(instance, "Normal", "Rule Created", fmt.Sprintf("Created ip rule for EIP %s in agent %s", instance.Spec.Address, nodeName))
			} else {
				r.Log.Info("Detect rule in node")
				r.Event(instance, "Normal", "Detect rule in node", fmt.Sprintf("Skipped Creating ip rule for EIP %s in agent %s", instance.Spec.Address, nodeName))
			}
		}
	}
	return nil
}

func (r *EipReconciler) DeleteRule(instance *networkv1alpha1.Eip) error {
	if instance.Spec.Address != "" {
		rule := nettool.NewEIPRule(instance.Spec.Address, 32)
		if ok, err := rule.IsExist(); err != nil {
			return err
		} else {
			if ok {
				err = rule.Delete()
				if err != nil {
					return err
				}
				r.Log.Info("Rule is deleted successfully", "rule", rule.ToAgentRule().String())
			} else {
				r.Log.Info("Try to delete a non-exist rule", "rule", rule.ToAgentRule().String())
			}
		}
	}
	return nil
}
