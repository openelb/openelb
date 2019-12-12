package converter

import (
	networkv1alpha1 "github.com/kubesphere/porter/api/v1alpha1"
	bgpapi "github.com/osrg/gobgp/api"
)

func ConvertPeerFromGoBGP(peer *bgpapi.Peer) *networkv1alpha1.BgpPeer {
	output := new(networkv1alpha1.BgpPeer)
	if peer.Conf != nil {
		output.Spec.Conf.AuthPassword = peer.Conf.AuthPassword
		output.Spec.Conf.Description = peer.Conf.Description
		output.Spec.Conf.NeighborAddress = peer.Conf.NeighborAddress
		output.Spec.Conf.NeighborInterface = peer.Conf.NeighborInterface
		output.Spec.Conf.PeerAs = peer.Conf.PeerAs
		output.Spec.Conf.PeerType = peer.Conf.PeerType
		output.Spec.Conf.AdminDown = peer.Conf.AdminDown
		output.Spec.Conf.SendCommunity = peer.Conf.SendCommunity
	}
	if peer.Transport != nil {
		output.Spec.Transport = &networkv1alpha1.Transport{}
		output.Spec.Transport.LocalAddress = peer.Transport.LocalAddress
		output.Spec.Transport.LocalPort = peer.Transport.LocalPort
		output.Spec.Transport.PassiveMode = peer.Transport.PassiveMode
		output.Spec.Transport.MtuDiscovery = peer.Transport.MtuDiscovery
	}
	if peer.Timers != nil {
		output.Spec.TimersConfig = &networkv1alpha1.TimersConfig{}
		output.Spec.TimersConfig.ConnectRetry = peer.Timers.Config.ConnectRetry
		output.Spec.TimersConfig.HoldTime = peer.Timers.Config.HoldTime
		output.Spec.TimersConfig.KeepaliveInterval = peer.Timers.Config.KeepaliveInterval
	}
	if peer.GracefulRestart != nil {
		output.Spec.GracefulRestart = &networkv1alpha1.GracefulRestart{}
		output.Spec.GracefulRestart.DeferralTime = peer.GracefulRestart.DeferralTime
		output.Spec.GracefulRestart.Enabled = peer.GracefulRestart.Enabled
		output.Spec.GracefulRestart.RestartTime = peer.GracefulRestart.RestartTime
		output.Spec.GracefulRestart.StaleRoutesTime = peer.GracefulRestart.StaleRoutesTime
		output.Spec.GracefulRestart.LonglivedEnabled = peer.GracefulRestart.LonglivedEnabled
		output.Spec.GracefulRestart.Mode = peer.GracefulRestart.Mode
	}
	return output
}

func ConvertPeerToGoBGP(peer *networkv1alpha1.BgpPeer) *bgpapi.Peer {
	output := new(bgpapi.Peer)
	conf := new(bgpapi.PeerConf)
	conf.AuthPassword = peer.Spec.Conf.AuthPassword
	conf.Description = peer.Spec.Conf.Description
	conf.NeighborAddress = peer.Spec.Conf.NeighborAddress
	conf.NeighborInterface = peer.Spec.Conf.NeighborInterface
	conf.PeerAs = peer.Spec.Conf.PeerAs
	conf.PeerType = peer.Spec.Conf.PeerType
	conf.AdminDown = peer.Spec.Conf.AdminDown
	conf.SendCommunity = peer.Spec.Conf.SendCommunity
	output.Conf = conf

	if peer.Spec.Transport != nil {
		transport := new(bgpapi.Transport)
		transport.LocalAddress = peer.Spec.Transport.LocalAddress
		transport.LocalPort = peer.Spec.Transport.LocalPort
		transport.PassiveMode = peer.Spec.Transport.PassiveMode
		transport.MtuDiscovery = peer.Spec.Transport.MtuDiscovery
		output.Transport = transport
	}
	if peer.Spec.TimersConfig != nil {
		timer := new(bgpapi.Timers)
		timer.Config = new(bgpapi.TimersConfig)
		timer.Config.ConnectRetry = peer.Spec.TimersConfig.ConnectRetry
		timer.Config.HoldTime = peer.Spec.TimersConfig.HoldTime
		timer.Config.KeepaliveInterval = peer.Spec.TimersConfig.KeepaliveInterval
		output.Timers = timer
	}
	if peer.Spec.GracefulRestart != nil {
		gr := new(bgpapi.GracefulRestart)
		gr.DeferralTime = peer.Spec.GracefulRestart.DeferralTime
		gr.Enabled = peer.Spec.GracefulRestart.Enabled
		gr.RestartTime = peer.Spec.GracefulRestart.RestartTime
		gr.StaleRoutesTime = peer.Spec.GracefulRestart.StaleRoutesTime
		gr.LonglivedEnabled = peer.Spec.GracefulRestart.LonglivedEnabled
		gr.Mode = peer.Spec.GracefulRestart.Mode
		output.GracefulRestart = gr
	}
	return output
}
