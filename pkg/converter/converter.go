package converter

import (
	networkv1alpha1 "github.com/kubesphere/porter/api/v1alpha1"
	bgpapi "github.com/osrg/gobgp/api"
)

func ConvertPeerFromGoBGP(peer *bgpapi.Peer) *networkv1alpha1.BgpPeer {
	return nil
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

	transport := new(bgpapi.Transport)
	transport.LocalAddress = peer.Spec.Transport.LocalAddress
	transport.LocalPort = peer.Spec.Transport.LocalPort
	transport.PassiveMode = peer.Spec.Transport.PassiveMode
	transport.RemoteAddress = peer.Spec.Transport.RemoteAddress
	transport.RemotePort = peer.Spec.Transport.RemotePort
	transport.MtuDiscovery = peer.Spec.Transport.MtuDiscovery
	output.Transport = transport

	timer := new(bgpapi.Timers)
	timer.Config = new(bgpapi.TimersConfig)
	timer.Config.ConnectRetry = peer.Spec.TimersConfig.ConnectRetry
	timer.Config.HoldTime = peer.Spec.TimersConfig.HoldTime
	timer.Config.KeepaliveInterval = peer.Spec.TimersConfig.KeepaliveInterval
	output.Timers = timer

	gr := new(bgpapi.GracefulRestart)
	gr.DeferralTime = peer.Spec.GracefulRestart.DeferralTime
	gr.Enabled = peer.Spec.GracefulRestart.Enabled
	gr.RestartTime = peer.Spec.GracefulRestart.RestartTime
	gr.HelperOnly = peer.Spec.GracefulRestart.HelperOnly
	gr.NotificationEnabled = peer.Spec.GracefulRestart.NotificationEnabled
	gr.StaleRoutesTime = peer.Spec.GracefulRestart.StaleRoutesTime
	gr.LonglivedEnabled = peer.Spec.GracefulRestart.LonglivedEnabled
	gr.Mode = peer.Spec.GracefulRestart.Mode
	output.GracefulRestart = gr
	return output
}
