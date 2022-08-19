package bgp

import (
	"context"
	"errors"
	"os"

	bgpapi "github.com/openelb/openelb/api/v1alpha2"
	"github.com/osrg/gobgp/pkg/config"
	"github.com/spf13/viper"
)

func (b *Bgp) HandleBgpGlobalConfig(global *bgpapi.BgpConf, rack string, delete bool) error {
	b.rack = rack
	if delete {
		return b.bgpServer.StopBgp(context.Background(), nil)
	}
	return b.CreateOrUpdateBgpConfig(global)
}

func (b *Bgp) CreateOrUpdateBgpConfig(bgpConf *bgpapi.BgpConf) error {
	if _, err := os.Stat(b.v.ConfigFileUsed()); errors.Is(err, os.ErrNotExist) {
		if _, err := os.Create(b.v.ConfigFileUsed()); err != nil {
			return err
		}
	}
	viper := viper.New()
	if err := viper.MergeConfigMap(GetBgpConfigGlobalMap(bgpConf)); err != nil {
		return err
	}
	if err := b.v.MergeConfigMap(viper.AllSettings()); err != nil {
		return err
	}
	if err := b.v.WriteConfigAs(b.v.ConfigFileUsed()); err != nil {
		return err
	}
	if err := b.bgpServer.StopBgp(context.Background(), nil); err != nil {
		return err
	}
	bcs, err := config.ReadConfigFile(b.v.ConfigFileUsed(), "toml")
	if err != nil {
		return err
	}
	_, err = config.InitialConfig(context.Background(), b.bgpServer, bcs, false)
	return err
}

func GetBgpConfigGlobalMap(bgpConf *bgpapi.BgpConf) map[string]interface{} {
	// _, err = config.InitialConfig(context.Background(), b.bgpServer, bcs, false)
	return map[string]interface{}{
		"global.config.as":                 bgpConf.Spec.As,
		"global.config.router-id":          bgpConf.Spec.RouterId,
		"global.config.port":               bgpConf.Spec.ListenPort,
		"global.config.local-address-list": bgpConf.Spec.ListenAddresses,
	}
}

func GetBgpConfigNeighborMap(bgpConf *bgpapi.BgpConf) map[string]interface{} {
	return map[string]interface{}{
		"global.config.as":                 bgpConf.Spec.As,
		"global.config.router-id":          bgpConf.Spec.RouterId,
		"global.config.port":               bgpConf.Spec.ListenPort,
		"global.config.local-address-list": bgpConf.Spec.ListenAddresses,
	}
}
