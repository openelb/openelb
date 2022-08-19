package bgp

import (
	"context"
	"errors"
	"os"

	bgpapi "github.com/openelb/openelb/api/v1alpha2"
	api "github.com/osrg/gobgp/api"
	"github.com/osrg/gobgp/pkg/config"
	"github.com/spf13/viper"
)

func (b *Bgp) HandleBgpGlobalConfig(bgpConf *bgpapi.BgpConf, rack string, delete bool) error {
	b.rack = rack
	if delete {
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
	bcs, err := config.ReadConfigFile(b.v.ConfigFileUsed(), "toml")
	if err != nil {
		return err
	}
	_, err = config.InitialConfig(context.Background(), b.bgpServer, bcs, gracefulRestart)
	return err
}

func GetBgpConfigGlobalMap(apiGlobal *api.Global) map[string]interface{} {
	return map[string]interface{}{
		// config
		"global.config.as":                 apiGlobal.GetAs(),
		"global.config.router-id":          apiGlobal.GetRouterId(),
		"global.config.port":               apiGlobal.GetListenPort(),
		"global.config.local-address-list": apiGlobal.GetListenAddresses(),
		// // use-multiple-paths
		// "globa.use-multiple-paths.config.enabled": apiGlobal.GetUseMultiplePaths(),
		// // graceful-restart
		// "global.graceful-restart.config.enabled":              apiGlobal.GetGracefulRestart().GetEnabled(),
		// "global.graceful-restart.config.restart-time":         apiGlobal.GetGracefulRestart().GetRestartTime(),
		// "global.graceful-restart.config.stale-routes-time":    apiGlobal.GetGracefulRestart().GetStaleRoutesTime(),
		// "global.graceful-restart.config.helper-only":          apiGlobal.GetGracefulRestart().GetHelperOnly(),
		// "global.graceful-restart.config.peer-restart-time":    apiGlobal.GetGracefulRestart().GetPeerRestartTime(),
		// "global.graceful-restart.config.peer-restarting":      apiGlobal.GetGracefulRestart().GetPeerRestarting(),
		// "global.graceful-restart.config.local-restarting":     apiGlobal.GetGracefulRestart().GetLocalRestarting(),
		// "global.graceful-restart.config.mode":                 apiGlobal.GetGracefulRestart().GetMode(),
		// "global.graceful-restart.config.deferral-time":        apiGlobal.GetGracefulRestart().GetDeferralTime(),
		// "global.graceful-restart.config.notification-enabled": apiGlobal.GetGracefulRestart().GetNotificationEnabled(),
		// "global.graceful-restart.config.long-lived-enabled":   apiGlobal.GetGracefulRestart().GetLonglivedEnabled(),
	}
}
