package config

type AfiSafiType string

const (
	AFI_SAFI_TYPE_IPV4_UNICAST          AfiSafiType = "ipv4-unicast"
	AFI_SAFI_TYPE_IPV6_UNICAST          AfiSafiType = "ipv6-unicast"
	AFI_SAFI_TYPE_IPV4_LABELLED_UNICAST AfiSafiType = "ipv4-labelled-unicast"
	AFI_SAFI_TYPE_IPV6_LABELLED_UNICAST AfiSafiType = "ipv6-labelled-unicast"
	AFI_SAFI_TYPE_L3VPN_IPV4_UNICAST    AfiSafiType = "l3vpn-ipv4-unicast"
	AFI_SAFI_TYPE_L3VPN_IPV6_UNICAST    AfiSafiType = "l3vpn-ipv6-unicast"
	AFI_SAFI_TYPE_L3VPN_IPV4_MULTICAST  AfiSafiType = "l3vpn-ipv4-multicast"
	AFI_SAFI_TYPE_L3VPN_IPV6_MULTICAST  AfiSafiType = "l3vpn-ipv6-multicast"
	AFI_SAFI_TYPE_L2VPN_VPLS            AfiSafiType = "l2vpn-vpls"
	AFI_SAFI_TYPE_L2VPN_EVPN            AfiSafiType = "l2vpn-evpn"
	AFI_SAFI_TYPE_IPV4_MULTICAST        AfiSafiType = "ipv4-multicast"
	AFI_SAFI_TYPE_IPV6_MULTICAST        AfiSafiType = "ipv6-multicast"
	AFI_SAFI_TYPE_RTC                   AfiSafiType = "rtc"
	AFI_SAFI_TYPE_IPV4_ENCAP            AfiSafiType = "ipv4-encap"
	AFI_SAFI_TYPE_IPV6_ENCAP            AfiSafiType = "ipv6-encap"
	AFI_SAFI_TYPE_IPV4_FLOWSPEC         AfiSafiType = "ipv4-flowspec"
	AFI_SAFI_TYPE_L3VPN_IPV4_FLOWSPEC   AfiSafiType = "l3vpn-ipv4-flowspec"
	AFI_SAFI_TYPE_IPV6_FLOWSPEC         AfiSafiType = "ipv6-flowspec"
	AFI_SAFI_TYPE_L3VPN_IPV6_FLOWSPEC   AfiSafiType = "l3vpn-ipv6-flowspec"
	AFI_SAFI_TYPE_L2VPN_FLOWSPEC        AfiSafiType = "l2vpn-flowspec"
	AFI_SAFI_TYPE_IPV4_SRPOLICY         AfiSafiType = "ipv4-srpolicy"
	AFI_SAFI_TYPE_IPV6_SRPOLICY         AfiSafiType = "ipv6-srpolicy"
	AFI_SAFI_TYPE_OPAQUE                AfiSafiType = "opaque"
	AFI_SAFI_TYPE_LS                    AfiSafiType = "ls"
)

var IntToAfiSafiTypeMap = map[int]AfiSafiType{
	0:  AFI_SAFI_TYPE_IPV4_UNICAST,
	1:  AFI_SAFI_TYPE_IPV6_UNICAST,
	2:  AFI_SAFI_TYPE_IPV4_LABELLED_UNICAST,
	3:  AFI_SAFI_TYPE_IPV6_LABELLED_UNICAST,
	4:  AFI_SAFI_TYPE_L3VPN_IPV4_UNICAST,
	5:  AFI_SAFI_TYPE_L3VPN_IPV6_UNICAST,
	6:  AFI_SAFI_TYPE_L3VPN_IPV4_MULTICAST,
	7:  AFI_SAFI_TYPE_L3VPN_IPV6_MULTICAST,
	8:  AFI_SAFI_TYPE_L2VPN_VPLS,
	9:  AFI_SAFI_TYPE_L2VPN_EVPN,
	10: AFI_SAFI_TYPE_IPV4_MULTICAST,
	11: AFI_SAFI_TYPE_IPV6_MULTICAST,
	12: AFI_SAFI_TYPE_RTC,
	13: AFI_SAFI_TYPE_IPV4_ENCAP,
	14: AFI_SAFI_TYPE_IPV6_ENCAP,
	15: AFI_SAFI_TYPE_IPV4_FLOWSPEC,
	16: AFI_SAFI_TYPE_L3VPN_IPV4_FLOWSPEC,
	17: AFI_SAFI_TYPE_IPV6_FLOWSPEC,
	18: AFI_SAFI_TYPE_L3VPN_IPV6_FLOWSPEC,
	19: AFI_SAFI_TYPE_L2VPN_FLOWSPEC,
	20: AFI_SAFI_TYPE_IPV4_SRPOLICY,
	21: AFI_SAFI_TYPE_IPV6_SRPOLICY,
	22: AFI_SAFI_TYPE_OPAQUE,
	23: AFI_SAFI_TYPE_LS,
}
