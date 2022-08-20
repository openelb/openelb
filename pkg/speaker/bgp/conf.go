package bgp

import (
	"context"
	"errors"
	"os"

	bgpapi "github.com/openelb/openelb/api/v1alpha2"
	"github.com/openelb/openelb/pkg/speaker/bgp/config"
	gobgpapi "github.com/osrg/gobgp/api"
	gobgpconfig "github.com/osrg/gobgp/pkg/config"
	"github.com/spf13/viper"
)

func (b *Bgp) HandleBgpGlobalConfig(bgpConf *bgpapi.BgpConf, rack string, delete bool) error {
	b.rack = rack
	if delete {
		if err := os.Remove(b.v.ConfigFileUsed()); err != nil {
			return err
		}
		return b.bgpServer.StopBgp(context.Background(), nil)
	}
	apiGlobal, err := bgpConf.Spec.ToGoBgpGlobalConf()
	if err != nil {
		return err
	}
	globalMap := GetBgpConfigGlobalMap(apiGlobal)
	return b.CreateOrUpdateBgpConfig(globalMap, apiGlobal.GetGracefulRestart().GetEnabled())
}

func (b *Bgp) CreateOrUpdateBgpConfig(globalMap map[string]interface{}, gracefulRestart bool) error {
	if _, err := os.Stat(b.v.ConfigFileUsed()); errors.Is(err, os.ErrNotExist) {
		if _, err := os.Create(b.v.ConfigFileUsed()); err != nil {
			return err
		}
	}
	viper := viper.New()
	if err := viper.MergeConfigMap(globalMap); err != nil {
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
	bcs, err := gobgpconfig.ReadConfigFile(b.v.ConfigFileUsed(), "toml")
	if err != nil {
		return err
	}
	_, err = gobgpconfig.InitialConfig(context.Background(), b.bgpServer, bcs, gracefulRestart)
	return err
}

func GetBgpConfigGlobalMap(apiGlobal *gobgpapi.Global) map[string]interface{} {
	globalMap := map[string]interface{}{
		// config
		"global.config.as":                 apiGlobal.GetAs(),
		"global.config.router-id":          apiGlobal.GetRouterId(),
		"global.config.port":               apiGlobal.GetListenPort(),
		"global.config.local-address-list": apiGlobal.GetListenAddresses(),
		// use-multiple-paths
		"global.use-multiple-paths.config.enabled": apiGlobal.GetUseMultiplePaths(),
		// graceful-restart
		"global.graceful-restart.config.enabled":              apiGlobal.GetGracefulRestart().GetEnabled(),
		"global.graceful-restart.config.restart-time":         uint16(apiGlobal.GetGracefulRestart().GetRestartTime()),
		"global.graceful-restart.config.stale-routes-time":    float64(apiGlobal.GetGracefulRestart().GetStaleRoutesTime()),
		"global.graceful-restart.config.helper-only":          apiGlobal.GetGracefulRestart().GetHelperOnly(),
		"global.graceful-restart.config.deferral-time":        uint16(apiGlobal.GetGracefulRestart().GetDeferralTime()),
		"global.graceful-restart.config.notification-enabled": apiGlobal.GetGracefulRestart().GetNotificationEnabled(),
		"global.graceful-restart.config.long-lived-enabled":   apiGlobal.GetGracefulRestart().GetLonglivedEnabled(),
	}
	// families
	for _, f := range apiGlobal.GetFamilies() {
		name := config.IntToAfiSafiTypeMap[int(f)]
		globalMap["global.afi-safi.config.afi-safi-name"] = name
		globalMap["global.afi-safi.config.enabled"] = true
	}

	return globalMap
}
