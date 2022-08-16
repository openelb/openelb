package bgp

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/openelb/openelb/pkg/constant"
	"github.com/openelb/openelb/pkg/speaker/bgp/config"
	"github.com/openelb/openelb/pkg/speaker/bgp/table"
	api "github.com/osrg/gobgp/api"
	"github.com/osrg/gobgp/pkg/server"

	corev1 "k8s.io/api/core/v1"
)

func (b *Bgp) UpdatePolicy(cm *corev1.ConfigMap) error {
	policyConf, ok := cm.Data[constant.OpenELBBgpName]
	if !ok {
		b.log.Info("error in %s configmap, %s missing", constant.OpenELBBgpName)
		return nil
	}
	path, err := writeToTempFile(policyConf)
	defer os.RemoveAll(path)
	if err != nil {
		return err
	}
	newConfig, err := config.ReadConfigfile(path, "toml")
	if err != nil {
		return err
	}
	p := config.ConfigSetToRoutingPolicy(newConfig)
	rp, err := table.NewAPIRoutingPolicyFromConfigStruct(p)
	if err != nil {
		b.log.Error(err, "failed to update policy config")
		return err
	}
	err = b.bgpServer.SetPolicies(context.Background(), &api.SetPoliciesRequest{
		DefinedSets: rp.DefinedSets,
		Policies:    rp.Policies,
	})
	if err != nil {
		b.log.Info("successfully updated policy config")
		return err
	}
	return b.AssignGlobalpolicy(context.Background(), b.bgpServer, &newConfig.Global.ApplyPolicy.Config)
}

func (b *Bgp) AssignGlobalpolicy(ctx context.Context, bgpServer *server.BgpServer, a *config.ApplyPolicyConfig) error {
	toDefaultTable := func(r config.DefaultPolicyType) table.RouteType {
		var def table.RouteType
		switch r {
		case config.DEFAULT_POLICY_TYPE_ACCEPT_ROUTE:
			def = table.ROUTE_TYPE_ACCEPT
		case config.DEFAULT_POLICY_TYPE_REJECT_ROUTE:
			def = table.ROUTE_TYPE_REJECT
		}
		return def
	}
	toPolicies := func(r []string) []*table.Policy {
		p := make([]*table.Policy, 0, len(r))
		for _, n := range r {
			p = append(p, &table.Policy{
				Name: n,
			})
		}
		return p
	}
	def := toDefaultTable(a.DefaultImportPolicy)
	ps := toPolicies(a.ImportPolicyList)
	err := bgpServer.SetPolicyAssignment(ctx, &api.SetPolicyAssignmentRequest{
		Assignment: table.NewAPIPolicyAssignmentFromTableStruct(&table.PolicyAssignment{
			Name:     table.GLOBAL_RIB_NAME,
			Type:     table.POLICY_DIRECTION_IMPORT,
			Policies: ps,
			Default:  def,
		}),
	})
	if err != nil {
		b.log.Info("failed setting policy assignment")
		return err
	}
	def = toDefaultTable(a.DefaultExportPolicy)
	ps = toPolicies(a.ExportPolicyList)
	err = bgpServer.SetPolicyAssignment(ctx, &api.SetPolicyAssignmentRequest{
		Assignment: table.NewAPIPolicyAssignmentFromTableStruct(&table.PolicyAssignment{
			Name:     table.GLOBAL_RIB_NAME,
			Type:     table.POLICY_DIRECTION_EXPORT,
			Policies: ps,
			Default:  def,
		}),
	})
	if err != nil {
		b.log.Info("failed setting policy assignment")
		return err
	}
	return nil
}

func writeToTempFile(val string) (string, error) {
	var path string
	temp, err := ioutil.TempFile(os.TempDir(), "temp")
	if err != nil {
		return path, err
	}
	err = ioutil.WriteFile(temp.Name(), []byte(val), 0644)
	if err != nil {
		return path, err
	}
	path, err = filepath.Abs(temp.Name())
	if err != nil {
		return path, err
	}
	return path, nil
}
