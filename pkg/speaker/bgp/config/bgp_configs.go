package config

import (
	"fmt"
)

type StdRegexp string
type BgpCommunityRegexpType StdRegexp
type BgpExtCommunityType string
type BgpStdCommunityType string
type BgpOriginAttrType string

const (
	BGP_ORIGIN_ATTR_TYPE_IGP        BgpOriginAttrType = "igp"
	BGP_ORIGIN_ATTR_TYPE_EGP        BgpOriginAttrType = "egp"
	BGP_ORIGIN_ATTR_TYPE_INCOMPLETE BgpOriginAttrType = "incomplete"
)

var BgpOriginAttrTypeToIntMap = map[BgpOriginAttrType]int{
	BGP_ORIGIN_ATTR_TYPE_IGP:        0,
	BGP_ORIGIN_ATTR_TYPE_EGP:        1,
	BGP_ORIGIN_ATTR_TYPE_INCOMPLETE: 2,
}

var IntToBgpOriginAttrTypeMap = map[int]BgpOriginAttrType{
	0: BGP_ORIGIN_ATTR_TYPE_IGP,
	1: BGP_ORIGIN_ATTR_TYPE_EGP,
	2: BGP_ORIGIN_ATTR_TYPE_INCOMPLETE,
}

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

type BgpWellKnownStdCommunity string

const (
	BGP_WELL_KNOWN_STD_COMMUNITY_NO_EXPORT           BgpWellKnownStdCommunity = "no_export"
	BGP_WELL_KNOWN_STD_COMMUNITY_NO_ADVERTISE        BgpWellKnownStdCommunity = "no_advertise"
	BGP_WELL_KNOWN_STD_COMMUNITY_NO_EXPORT_SUBCONFED BgpWellKnownStdCommunity = "no_export_subconfed"
	BGP_WELL_KNOWN_STD_COMMUNITY_NOPEER              BgpWellKnownStdCommunity = "nopeer"
)

var BgpWellKnownStdCommunityToIntMap = map[BgpWellKnownStdCommunity]int{
	BGP_WELL_KNOWN_STD_COMMUNITY_NO_EXPORT:           0,
	BGP_WELL_KNOWN_STD_COMMUNITY_NO_ADVERTISE:        1,
	BGP_WELL_KNOWN_STD_COMMUNITY_NO_EXPORT_SUBCONFED: 2,
	BGP_WELL_KNOWN_STD_COMMUNITY_NOPEER:              3,
}

var IntToBgpWellKnownStdCommunityMap = map[int]BgpWellKnownStdCommunity{
	0: BGP_WELL_KNOWN_STD_COMMUNITY_NO_EXPORT,
	1: BGP_WELL_KNOWN_STD_COMMUNITY_NO_ADVERTISE,
	2: BGP_WELL_KNOWN_STD_COMMUNITY_NO_EXPORT_SUBCONFED,
	3: BGP_WELL_KNOWN_STD_COMMUNITY_NOPEER,
}

func (v BgpWellKnownStdCommunity) Validate() error {
	if _, ok := BgpWellKnownStdCommunityToIntMap[v]; !ok {
		return fmt.Errorf("invalid BgpWellKnownStdCommunity: %s", v)
	}
	return nil
}

func (v BgpWellKnownStdCommunity) ToInt() int {
	i, ok := BgpWellKnownStdCommunityToIntMap[v]
	if !ok {
		return -1
	}
	return i
}

type MatchSetOptionsRestrictedType string

const (
	MATCH_SET_OPTIONS_RESTRICTED_TYPE_ANY    MatchSetOptionsRestrictedType = "any"
	MATCH_SET_OPTIONS_RESTRICTED_TYPE_INVERT MatchSetOptionsRestrictedType = "invert"
)

var MatchSetOptionsRestrictedTypeToIntMap = map[MatchSetOptionsRestrictedType]int{
	MATCH_SET_OPTIONS_RESTRICTED_TYPE_ANY:    0,
	MATCH_SET_OPTIONS_RESTRICTED_TYPE_INVERT: 1,
}

var IntToMatchSetOptionsRestrictedTypeMap = map[int]MatchSetOptionsRestrictedType{
	0: MATCH_SET_OPTIONS_RESTRICTED_TYPE_ANY,
	1: MATCH_SET_OPTIONS_RESTRICTED_TYPE_INVERT,
}

func (v MatchSetOptionsRestrictedType) Validate() error {
	if _, ok := MatchSetOptionsRestrictedTypeToIntMap[v]; !ok {
		return fmt.Errorf("invalid MatchSetOptionsRestrictedType: %s", v)
	}
	return nil
}

func (v MatchSetOptionsRestrictedType) Default() MatchSetOptionsRestrictedType {
	return MATCH_SET_OPTIONS_RESTRICTED_TYPE_ANY
}

func (v MatchSetOptionsRestrictedType) DefaultAsNeeded() MatchSetOptionsRestrictedType {
	if string(v) == "" {
		return v.Default()
	}
	return v
}
func (v MatchSetOptionsRestrictedType) ToInt() int {
	_v := v.DefaultAsNeeded()
	i, ok := MatchSetOptionsRestrictedTypeToIntMap[_v]
	if !ok {
		return -1
	}
	return i
}

type MatchSetOptionsType string

const (
	MATCH_SET_OPTIONS_TYPE_ANY    MatchSetOptionsType = "any"
	MATCH_SET_OPTIONS_TYPE_ALL    MatchSetOptionsType = "all"
	MATCH_SET_OPTIONS_TYPE_INVERT MatchSetOptionsType = "invert"
)

var MatchSetOptionsTypeToIntMap = map[MatchSetOptionsType]int{
	MATCH_SET_OPTIONS_TYPE_ANY:    0,
	MATCH_SET_OPTIONS_TYPE_ALL:    1,
	MATCH_SET_OPTIONS_TYPE_INVERT: 2,
}

var IntToMatchSetOptionsTypeMap = map[int]MatchSetOptionsType{
	0: MATCH_SET_OPTIONS_TYPE_ANY,
	1: MATCH_SET_OPTIONS_TYPE_ALL,
	2: MATCH_SET_OPTIONS_TYPE_INVERT,
}

func (v MatchSetOptionsType) Validate() error {
	if _, ok := MatchSetOptionsTypeToIntMap[v]; !ok {
		return fmt.Errorf("invalid MatchSetOptionsType: %s", v)
	}
	return nil
}

func (v MatchSetOptionsType) Default() MatchSetOptionsType {
	return MATCH_SET_OPTIONS_TYPE_ANY
}

func (v MatchSetOptionsType) DefaultAsNeeded() MatchSetOptionsType {
	if string(v) == "" {
		return v.Default()
	}
	return v
}
func (v MatchSetOptionsType) ToInt() int {
	_v := v.DefaultAsNeeded()
	i, ok := MatchSetOptionsTypeToIntMap[_v]
	if !ok {
		return -1
	}
	return i
}

type TagType string

type InstallProtocolType string

const (
	INSTALL_PROTOCOL_TYPE_BGP                InstallProtocolType = "bgp"
	INSTALL_PROTOCOL_TYPE_ISIS               InstallProtocolType = "isis"
	INSTALL_PROTOCOL_TYPE_OSPF               InstallProtocolType = "ospf"
	INSTALL_PROTOCOL_TYPE_OSPF3              InstallProtocolType = "ospf3"
	INSTALL_PROTOCOL_TYPE_STATIC             InstallProtocolType = "static"
	INSTALL_PROTOCOL_TYPE_DIRECTLY_CONNECTED InstallProtocolType = "directly-connected"
	INSTALL_PROTOCOL_TYPE_LOCAL_AGGREGATE    InstallProtocolType = "local-aggregate"
)

var InstallProtocolTypeToIntMap = map[InstallProtocolType]int{
	INSTALL_PROTOCOL_TYPE_BGP:                0,
	INSTALL_PROTOCOL_TYPE_ISIS:               1,
	INSTALL_PROTOCOL_TYPE_OSPF:               2,
	INSTALL_PROTOCOL_TYPE_OSPF3:              3,
	INSTALL_PROTOCOL_TYPE_STATIC:             4,
	INSTALL_PROTOCOL_TYPE_DIRECTLY_CONNECTED: 5,
	INSTALL_PROTOCOL_TYPE_LOCAL_AGGREGATE:    6,
}

var IntToInstallProtocolTypeMap = map[int]InstallProtocolType{
	0: INSTALL_PROTOCOL_TYPE_BGP,
	1: INSTALL_PROTOCOL_TYPE_ISIS,
	2: INSTALL_PROTOCOL_TYPE_OSPF,
	3: INSTALL_PROTOCOL_TYPE_OSPF3,
	4: INSTALL_PROTOCOL_TYPE_STATIC,
	5: INSTALL_PROTOCOL_TYPE_DIRECTLY_CONNECTED,
	6: INSTALL_PROTOCOL_TYPE_LOCAL_AGGREGATE,
}

func (v InstallProtocolType) Validate() error {
	if _, ok := InstallProtocolTypeToIntMap[v]; !ok {
		return fmt.Errorf("invalid InstallProtocolType: %s", v)
	}
	return nil
}

func (v InstallProtocolType) ToInt() int {
	i, ok := InstallProtocolTypeToIntMap[v]
	if !ok {
		return -1
	}
	return i
}

// typedef for identity ptypes:attribute-comparison.
// base type for supported comparison operators on route
// attributes.
type AttributeComparison string

const (
	ATTRIBUTE_COMPARISON_ATTRIBUTE_EQ AttributeComparison = "attribute-eq"
	ATTRIBUTE_COMPARISON_ATTRIBUTE_GE AttributeComparison = "attribute-ge"
	ATTRIBUTE_COMPARISON_ATTRIBUTE_LE AttributeComparison = "attribute-le"
	ATTRIBUTE_COMPARISON_EQ           AttributeComparison = "eq"
	ATTRIBUTE_COMPARISON_GE           AttributeComparison = "ge"
	ATTRIBUTE_COMPARISON_LE           AttributeComparison = "le"
)

var AttributeComparisonToIntMap = map[AttributeComparison]int{
	ATTRIBUTE_COMPARISON_ATTRIBUTE_EQ: 0,
	ATTRIBUTE_COMPARISON_ATTRIBUTE_GE: 1,
	ATTRIBUTE_COMPARISON_ATTRIBUTE_LE: 2,
	ATTRIBUTE_COMPARISON_EQ:           3,
	ATTRIBUTE_COMPARISON_GE:           4,
	ATTRIBUTE_COMPARISON_LE:           5,
}

var IntToAttributeComparisonMap = map[int]AttributeComparison{
	0: ATTRIBUTE_COMPARISON_ATTRIBUTE_EQ,
	1: ATTRIBUTE_COMPARISON_ATTRIBUTE_GE,
	2: ATTRIBUTE_COMPARISON_ATTRIBUTE_LE,
	3: ATTRIBUTE_COMPARISON_EQ,
	4: ATTRIBUTE_COMPARISON_GE,
	5: ATTRIBUTE_COMPARISON_LE,
}

func (v AttributeComparison) Validate() error {
	if _, ok := AttributeComparisonToIntMap[v]; !ok {
		return fmt.Errorf("invalid AttributeComparison: %s", v)
	}
	return nil
}

func (v AttributeComparison) ToInt() int {
	i, ok := AttributeComparisonToIntMap[v]
	if !ok {
		return -1
	}
	return i
}

// typedef for identity rpol:route-disposition.
// Select the final disposition for the route, either
// accept or reject.
type RouteDisposition string

const (
	ROUTE_DISPOSITION_NONE         RouteDisposition = "none"
	ROUTE_DISPOSITION_ACCEPT_ROUTE RouteDisposition = "accept-route"
	ROUTE_DISPOSITION_REJECT_ROUTE RouteDisposition = "reject-route"
)

var RouteDispositionToIntMap = map[RouteDisposition]int{
	ROUTE_DISPOSITION_NONE:         0,
	ROUTE_DISPOSITION_ACCEPT_ROUTE: 1,
	ROUTE_DISPOSITION_REJECT_ROUTE: 2,
}

var IntToRouteDispositionMap = map[int]RouteDisposition{
	0: ROUTE_DISPOSITION_NONE,
	1: ROUTE_DISPOSITION_ACCEPT_ROUTE,
	2: ROUTE_DISPOSITION_REJECT_ROUTE,
}

func (v RouteDisposition) Validate() error {
	if _, ok := RouteDispositionToIntMap[v]; !ok {
		return fmt.Errorf("invalid RouteDisposition: %s", v)
	}
	return nil
}

func (v RouteDisposition) ToInt() int {
	i, ok := RouteDispositionToIntMap[v]
	if !ok {
		return -1
	}
	return i
}

type RouteType string

const (
	ROUTE_TYPE_NONE     RouteType = "none"
	ROUTE_TYPE_INTERNAL RouteType = "internal"
	ROUTE_TYPE_EXTERNAL RouteType = "external"
	ROUTE_TYPE_LOCAL    RouteType = "local"
)

var RouteTypeToIntMap = map[RouteType]int{
	ROUTE_TYPE_NONE:     0,
	ROUTE_TYPE_INTERNAL: 1,
	ROUTE_TYPE_EXTERNAL: 2,
	ROUTE_TYPE_LOCAL:    3,
}

var IntToRouteTypeMap = map[int]RouteType{
	0: ROUTE_TYPE_NONE,
	1: ROUTE_TYPE_INTERNAL,
	2: ROUTE_TYPE_EXTERNAL,
	3: ROUTE_TYPE_LOCAL,
}

func (v RouteType) Validate() error {
	if _, ok := RouteTypeToIntMap[v]; !ok {
		return fmt.Errorf("invalid RouteType: %s", v)
	}
	return nil
}

func (v RouteType) ToInt() int {
	i, ok := RouteTypeToIntMap[v]
	if !ok {
		return -1
	}
	return i
}

type DefaultPolicyType string

const (
	DEFAULT_POLICY_TYPE_ACCEPT_ROUTE DefaultPolicyType = "accept-route"
	DEFAULT_POLICY_TYPE_REJECT_ROUTE DefaultPolicyType = "reject-route"
)

var DefaultPolicyTypeToIntMap = map[DefaultPolicyType]int{
	DEFAULT_POLICY_TYPE_ACCEPT_ROUTE: 0,
	DEFAULT_POLICY_TYPE_REJECT_ROUTE: 1,
}

var IntToDefaultPolicyTypeMap = map[int]DefaultPolicyType{
	0: DEFAULT_POLICY_TYPE_ACCEPT_ROUTE,
	1: DEFAULT_POLICY_TYPE_REJECT_ROUTE,
}

type BgpNextHopType string
type BgpAsPathPrependRepeat uint8
type BgpSetMedType string
type BgpSetCommunityOptionType string

const (
	BGP_SET_COMMUNITY_OPTION_TYPE_ADD     BgpSetCommunityOptionType = "add"
	BGP_SET_COMMUNITY_OPTION_TYPE_REMOVE  BgpSetCommunityOptionType = "remove"
	BGP_SET_COMMUNITY_OPTION_TYPE_REPLACE BgpSetCommunityOptionType = "replace"
)

var BgpSetCommunityOptionTypeToIntMap = map[BgpSetCommunityOptionType]int{
	BGP_SET_COMMUNITY_OPTION_TYPE_ADD:     0,
	BGP_SET_COMMUNITY_OPTION_TYPE_REMOVE:  1,
	BGP_SET_COMMUNITY_OPTION_TYPE_REPLACE: 2,
}

type ApplyPolicyConfig struct {
	ImportPolicyList    []string          `mapstructure:"import-policy-list" json:"import-policy-list,omitempty"`
	DefaultImportPolicy DefaultPolicyType `mapstructure:"default-import-policy" json:"default-import-policy,omitempty"`
	ExportPolicyList    []string          `mapstructure:"export-policy-list" json:"export-policy-list,omitempty"`
	DefaultExportPolicy DefaultPolicyType `mapstructure:"default-export-policy" json:"default-export-policy,omitempty"`
	InPolicyList        []string          `mapstructure:"in-policy-list" json:"in-policy-list,omitempty"`
	DefaultInPolicy     DefaultPolicyType `mapstructure:"default-in-policy" json:"default-in-policy,omitempty"`
}

type ApplyPolicy struct {
	Config ApplyPolicyConfig `mapstructure:"config" json:"config,omitempty"`
}

type Global struct {
	ApplyPolicy ApplyPolicy `mapstructure:"apply-policy" json:"apply-policy,omitempty"`
}

type SetLargeCommunityMethod struct {
	CommunitiesList []string `mapstructure:"communities-list" json:"communities-list,omitempty"`
}
type SetLargeCommunity struct {
	SetLargeCommunityMethod SetLargeCommunityMethod   `mapstructure:"set-large-community-method" json:"set-large-community-method,omitempty"`
	Options                 BgpSetCommunityOptionType `mapstructure:"options" json:"options,omitempty"`
}

type SetExtCommunityMethod struct {
	CommunitiesList    []string `mapstructure:"communities-list" json:"communities-list,omitempty"`
	ExtCommunitySetRef string   `mapstructure:"ext-community-set-ref" json:"ext-community-set-ref,omitempty"`
}

type SetExtCommunity struct {
	SetExtCommunityMethod SetExtCommunityMethod `mapstructure:"set-ext-community-method" json:"set-ext-community-method,omitempty"`
	Options               string                `mapstructure:"options" json:"options,omitempty"`
}

type SetCommunityMethod struct {
	CommunitiesList []string `mapstructure:"communities-list" json:"communities-list,omitempty"`
	CommunitySetRef string   `mapstructure:"community-set-ref" json:"community-set-ref,omitempty"`
}

type SetCommunity struct {
	SetCommunityMethod SetCommunityMethod `mapstructure:"set-community-method" json:"set-community-method,omitempty"`
	Options            string             `mapstructure:"options" json:"options,omitempty"`
}

type SetAsPathPrepend struct {
	RepeatN uint8  `mapstructure:"repeat-n" json:"repeat-n,omitempty"`
	As      string `mapstructure:"as" json:"as,omitempty"`
}

type BgpActions struct {
	SetAsPathPrepend  SetAsPathPrepend  `mapstructure:"set-as-path-prepend" json:"set-as-path-prepend,omitempty"`
	SetCommunity      SetCommunity      `mapstructure:"set-community" json:"set-community,omitempty"`
	SetExtCommunity   SetExtCommunity   `mapstructure:"set-ext-community" json:"set-ext-community,omitempty"`
	SetRouteOrigin    BgpOriginAttrType `mapstructure:"set-route-origin" json:"set-route-origin,omitempty"`
	SetLocalPref      uint32            `mapstructure:"set-local-pref" json:"set-local-pref,omitempty"`
	SetNextHop        BgpNextHopType    `mapstructure:"set-next-hop" json:"set-next-hop,omitempty"`
	SetMed            BgpSetMedType     `mapstructure:"set-med" json:"set-med,omitempty"`
	SetLargeCommunity SetLargeCommunity `mapstructure:"set-large-community" json:"set-large-community,omitempty"`
}

type IgpActions struct {
	SetTag TagType `mapstructure:"set-tag" json:"set-tag,omitempty"`
}

type Actions struct {
	RouteDisposition RouteDisposition `mapstructure:"route-disposition" json:"route-disposition,omitempty"`
	IgpActions       IgpActions       `mapstructure:"igp-actions" json:"igp-actions,omitempty"`
	BgpActions       BgpActions       `mapstructure:"bgp-actions" json:"bgp-actions,omitempty"`
}

type MatchLargeCommunitySet struct {
	LargeCommunitySet string              `mapstructure:"large-community-set" json:"large-community-set,omitempty"`
	MatchSetOptions   MatchSetOptionsType `mapstructure:"match-set-options" json:"match-set-options,omitempty"`
}

type AsPathLength struct {
	Operator AttributeComparison `mapstructure:"operator" json:"operator,omitempty"`
	Value    uint32              `mapstructure:"value" json:"value,omitempty"`
}

type CommunityCount struct {
	Operator AttributeComparison `mapstructure:"operator" json:"operator,omitempty"`
	Value    uint32              `mapstructure:"value" json:"value,omitempty"`
}

type MatchAsPathSet struct {
	AsPathSet       string              `mapstructure:"as-path-set" json:"as-path-set,omitempty"`
	MatchSetOptions MatchSetOptionsType `mapstructure:"match-set-options" json:"match-set-options,omitempty"`
}

type MatchExtCommunitySet struct {
	ExtCommunitySet string              `mapstructure:"ext-community-set" json:"ext-community-set,omitempty"`
	MatchSetOptions MatchSetOptionsType `mapstructure:"match-set-options" json:"match-set-options,omitempty"`
}

type MatchCommunitySet struct {
	CommunitySet    string              `mapstructure:"community-set" json:"community-set,omitempty"`
	MatchSetOptions MatchSetOptionsType `mapstructure:"match-set-options" json:"match-set-options,omitempty"`
}

type BgpConditions struct {
	MatchCommunitySet      MatchCommunitySet      `mapstructure:"match-community-set" json:"match-community-set,omitempty"`
	MatchExtCommunitySet   MatchExtCommunitySet   `mapstructure:"match-ext-community-set" json:"match-ext-community-set,omitempty"`
	MatchAsPathSet         MatchAsPathSet         `mapstructure:"match-as-path-set" json:"match-as-path-set,omitempty"`
	MedEq                  uint32                 `mapstructure:"med-eq" json:"med-eq,omitempty"`
	OriginEq               BgpOriginAttrType      `mapstructure:"origin-eq" json:"origin-eq,omitempty"`
	NextHopInList          []string               `mapstructure:"next-hop-in-list" json:"next-hop-in-list,omitempty"`
	AfiSafiInList          []AfiSafiType          `mapstructure:"afi-safi-in-list" json:"afi-safi-in-list,omitempty"`
	LocalPrefEq            uint32                 `mapstructure:"local-pref-eq" json:"local-pref-eq,omitempty"`
	CommunityCount         CommunityCount         `mapstructure:"community-count" json:"community-count,omitempty"`
	AsPathLength           AsPathLength           `mapstructure:"as-path-length" json:"as-path-length,omitempty"`
	RouteType              RouteType              `mapstructure:"route-type" json:"route-type,omitempty"`
	MatchLargeCommunitySet MatchLargeCommunitySet `mapstructure:"match-large-community-set" json:"match-large-community-set,omitempty"`
}
type IgpConditions struct {
}

type MatchTagSet struct {
	TagSet          string                        `mapstructure:"tag-set" json:"tag-set,omitempty"`
	MatchSetOptions MatchSetOptionsRestrictedType `mapstructure:"match-set-options" json:"match-set-options,omitempty"`
}

type MatchNeighborSet struct {
	NeighborSet     string                        `mapstructure:"neighbor-set" json:"neighbor-set,omitempty"`
	MatchSetOptions MatchSetOptionsRestrictedType `mapstructure:"match-set-options" json:"match-set-options,omitempty"`
}

type MatchPrefixSet struct {
	PrefixSet       string                        `mapstructure:"prefix-set" json:"prefix-set,omitempty"`
	MatchSetOptions MatchSetOptionsRestrictedType `mapstructure:"match-set-options" json:"match-set-options,omitempty"`
}

type Conditions struct {
	CallPolicy        string              `mapstructure:"call-policy" json:"call-policy,omitempty"`
	MatchPrefixSet    MatchPrefixSet      `mapstructure:"match-prefix-set" json:"match-prefix-set,omitempty"`
	MatchNeighborSet  MatchNeighborSet    `mapstructure:"match-neighbor-set" json:"match-neighbor-set,omitempty"`
	MatchTagSet       MatchTagSet         `mapstructure:"match-tag-set" json:"match-tag-set,omitempty"`
	InstallProtocolEq InstallProtocolType `mapstructure:"install-protocol-eq" json:"install-protocol-eq,omitempty"`
	IgpConditions     IgpConditions       `mapstructure:"igp-conditions" json:"igp-conditions,omitempty"`
	BgpConditions     BgpConditions       `mapstructure:"bgp-conditions" json:"bgp-conditions,omitempty"`
}

type Statement struct {
	Name       string     `mapstructure:"name" json:"name,omitempty"`
	Conditions Conditions `mapstructure:"conditions" json:"conditions,omitempty"`
	Actions    Actions    `mapstructure:"actions" json:"actions,omitempty"`
}

type PolicyDefinition struct {
	Name       string      `mapstructure:"name" json:"name,omitempty"`
	Statements []Statement `mapstructure:"statements" json:"statements,omitempty"`
}
type LargeCommunitySet struct {
	LargeCommunitySetName string   `mapstructure:"large-community-set-name" json:"large-community-set-name,omitempty"`
	LargeCommunityList    []string `mapstructure:"large-community-list" json:"large-community-list,omitempty"`
}

type AsPathSet struct {
	AsPathSetName string   `mapstructure:"as-path-set-name" json:"as-path-set-name,omitempty"`
	AsPathList    []string `mapstructure:"as-path-list" json:"as-path-list,omitempty"`
}

type ExtCommunitySet struct {
	ExtCommunitySetName string   `mapstructure:"ext-community-set-name" json:"ext-community-set-name,omitempty"`
	ExtCommunityList    []string `mapstructure:"ext-community-list" json:"ext-community-list,omitempty"`
}

type CommunitySet struct {
	CommunitySetName string   `mapstructure:"community-set-name" json:"community-set-name,omitempty"`
	CommunityList    []string `mapstructure:"community-list" json:"community-list,omitempty"`
}

type BgpDefinedSets struct {
	CommunitySets      []CommunitySet      `mapstructure:"community-sets" json:"community-sets,omitempty"`
	ExtCommunitySets   []ExtCommunitySet   `mapstructure:"ext-community-sets" json:"ext-community-sets,omitempty"`
	AsPathSets         []AsPathSet         `mapstructure:"as-path-sets" json:"as-path-sets,omitempty"`
	LargeCommunitySets []LargeCommunitySet `mapstructure:"large-community-sets" json:"large-community-sets,omitempty"`
}

type Tag struct {
	Value TagType `mapstructure:"value" json:"value,omitempty"`
}

type TagSet struct {
	TagSetName string `mapstructure:"tag-set-name" json:"tag-set-name,omitempty"`
	TagList    []Tag  `mapstructure:"tag-list" json:"tag-list,omitempty"`
}

type NeighborSet struct {
	NeighborSetName  string   `mapstructure:"neighbor-set-name" json:"neighbor-set-name,omitempty"`
	NeighborInfoList []string `mapstructure:"neighbor-info-list" json:"neighbor-info-list,omitempty"`
}

type Prefix struct {
	IpPrefix        string `mapstructure:"ip-prefix" json:"ip-prefix,omitempty"`
	MasklengthRange string `mapstructure:"masklength-range" json:"masklength-range,omitempty"`
}
type PrefixSet struct {
	PrefixSetName string   `mapstructure:"prefix-set-name" json:"prefix-set-name,omitempty"`
	PrefixList    []Prefix `mapstructure:"prefix-list" json:"prefix-list,omitempty"`
}
type DefinedSets struct {
	PrefixSets     []PrefixSet    `mapstructure:"prefix-sets" json:"prefix-sets,omitempty"`
	NeighborSets   []NeighborSet  `mapstructure:"neighbor-sets" json:"neighbor-sets,omitempty"`
	TagSets        []TagSet       `mapstructure:"tag-sets" json:"tag-sets,omitempty"`
	BgpDefinedSets BgpDefinedSets `mapstructure:"bgp-defined-sets" json:"bgp-defined-sets,omitempty"`
}

type RoutingPolicy struct {
	DefinedSets       DefinedSets        `mapstructure:"defined-sets" json:"defined-sets,omitempty"`
	PolicyDefinitions []PolicyDefinition `mapstructure:"policy-definitions" json:"policy-definitions,omitempty"`
}
