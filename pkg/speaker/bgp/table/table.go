package table

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	api "github.com/osrg/gobgp/api"
	"github.com/osrg/gobgp/pkg/packet/bgp"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/openelb/openelb/pkg/speaker/bgp/config"
)

const (
	GLOBAL_RIB_NAME = "global"
)

type RouteType int

const (
	ROUTE_TYPE_NONE RouteType = iota
	ROUTE_TYPE_ACCEPT
	ROUTE_TYPE_REJECT
)

type PolicyDirection int

const (
	POLICY_DIRECTION_NONE PolicyDirection = iota
	POLICY_DIRECTION_IMPORT
	POLICY_DIRECTION_EXPORT
)

type MatchOption int

const (
	MATCH_OPTION_ANY MatchOption = iota
	MATCH_OPTION_ALL
	MATCH_OPTION_INVERT
)

func newMatchOption(c interface{}) (MatchOption, error) {
	switch t := c.(type) {
	case config.MatchSetOptionsType:
		t = t.DefaultAsNeeded()
		switch t {
		case config.MATCH_SET_OPTIONS_TYPE_ANY:
			return MATCH_OPTION_ANY, nil
		case config.MATCH_SET_OPTIONS_TYPE_ALL:
			return MATCH_OPTION_ALL, nil
		case config.MATCH_SET_OPTIONS_TYPE_INVERT:
			return MATCH_OPTION_INVERT, nil
		}
	case config.MatchSetOptionsRestrictedType:
		t = t.DefaultAsNeeded()
		switch t {
		case config.MATCH_SET_OPTIONS_RESTRICTED_TYPE_ANY:
			return MATCH_OPTION_ANY, nil
		case config.MATCH_SET_OPTIONS_RESTRICTED_TYPE_INVERT:
			return MATCH_OPTION_INVERT, nil
		}
	}
	return MATCH_OPTION_ANY, fmt.Errorf("invalid argument to create match option: %v", c)
}

type Policy struct {
	Name string
}

func (p *Policy) toConfig() *config.PolicyDefinition {
	return &config.PolicyDefinition{
		Name: p.Name,
	}
}

type PolicyAssignment struct {
	Name     string
	Type     PolicyDirection
	Policies []*Policy
	Default  RouteType
}

var _regexpMedActionType = regexp.MustCompile(`([+-]?)(\d+)`)

func toStatementApi(s *config.Statement) *api.Statement {
	cs := &api.Conditions{}
	if s.Conditions.MatchPrefixSet.PrefixSet != "" {
		o, _ := newMatchOption(s.Conditions.MatchPrefixSet.MatchSetOptions)
		cs.PrefixSet = &api.MatchSet{
			MatchType: api.MatchType(o),
			Name:      s.Conditions.MatchPrefixSet.PrefixSet,
		}
	}
	if s.Conditions.MatchNeighborSet.NeighborSet != "" {
		o, _ := newMatchOption(s.Conditions.MatchNeighborSet.MatchSetOptions)
		cs.NeighborSet = &api.MatchSet{
			MatchType: api.MatchType(o),
			Name:      s.Conditions.MatchNeighborSet.NeighborSet,
		}
	}
	if s.Conditions.BgpConditions.AsPathLength.Operator != "" {
		cs.AsPathLength = &api.AsPathLength{
			Length:     s.Conditions.BgpConditions.AsPathLength.Value,
			LengthType: api.AsPathLengthType(s.Conditions.BgpConditions.AsPathLength.Operator.ToInt()),
		}
	}
	if s.Conditions.BgpConditions.MatchAsPathSet.AsPathSet != "" {
		cs.AsPathSet = &api.MatchSet{
			MatchType: api.MatchType(s.Conditions.BgpConditions.MatchAsPathSet.MatchSetOptions.ToInt()),
			Name:      s.Conditions.BgpConditions.MatchAsPathSet.AsPathSet,
		}
	}
	if s.Conditions.BgpConditions.MatchCommunitySet.CommunitySet != "" {
		cs.CommunitySet = &api.MatchSet{
			MatchType: api.MatchType(s.Conditions.BgpConditions.MatchCommunitySet.MatchSetOptions.ToInt()),
			Name:      s.Conditions.BgpConditions.MatchCommunitySet.CommunitySet,
		}
	}
	if s.Conditions.BgpConditions.MatchExtCommunitySet.ExtCommunitySet != "" {
		cs.ExtCommunitySet = &api.MatchSet{
			MatchType: api.MatchType(s.Conditions.BgpConditions.MatchExtCommunitySet.MatchSetOptions.ToInt()),
			Name:      s.Conditions.BgpConditions.MatchExtCommunitySet.ExtCommunitySet,
		}
	}
	if s.Conditions.BgpConditions.MatchLargeCommunitySet.LargeCommunitySet != "" {
		cs.LargeCommunitySet = &api.MatchSet{
			MatchType: api.MatchType(s.Conditions.BgpConditions.MatchLargeCommunitySet.MatchSetOptions.ToInt()),
			Name:      s.Conditions.BgpConditions.MatchLargeCommunitySet.LargeCommunitySet,
		}
	}
	if s.Conditions.BgpConditions.RouteType != "" {
		cs.RouteType = api.Conditions_RouteType(s.Conditions.BgpConditions.RouteType.ToInt())
	}
	if len(s.Conditions.BgpConditions.NextHopInList) > 0 {
		cs.NextHopInList = s.Conditions.BgpConditions.NextHopInList
	}
	if s.Conditions.BgpConditions.AfiSafiInList != nil {
		afiSafiIn := make([]*api.Family, 0)
		for _, afiSafiType := range s.Conditions.BgpConditions.AfiSafiInList {
			if mapped, ok := bgp.AddressFamilyValueMap[string(afiSafiType)]; ok {
				afi, safi := bgp.RouteFamilyToAfiSafi(mapped)
				afiSafiIn = append(afiSafiIn, &api.Family{Afi: api.Family_Afi(afi), Safi: api.Family_Safi(safi)})
			}
		}
		cs.AfiSafiIn = afiSafiIn
	}
	as := &api.Actions{
		RouteAction: func() api.RouteAction {
			switch s.Actions.RouteDisposition {
			case config.ROUTE_DISPOSITION_ACCEPT_ROUTE:
				return api.RouteAction_ACCEPT
			case config.ROUTE_DISPOSITION_REJECT_ROUTE:
				return api.RouteAction_REJECT
			}
			return api.RouteAction_NONE
		}(),
		Community: func() *api.CommunityAction {
			if len(s.Actions.BgpActions.SetCommunity.SetCommunityMethod.CommunitiesList) == 0 {
				return nil
			}
			return &api.CommunityAction{
				ActionType: api.CommunityActionType(
					config.BgpSetCommunityOptionTypeToIntMap[config.BgpSetCommunityOptionType(s.Actions.BgpActions.SetCommunity.Options)],
				),
				Communities: s.Actions.BgpActions.SetCommunity.SetCommunityMethod.CommunitiesList}
		}(),
		Med: func() *api.MedAction {
			medStr := strings.TrimSpace(string(s.Actions.BgpActions.SetMed))
			if len(medStr) == 0 {
				return nil
			}
			matches := _regexpMedActionType.FindStringSubmatch(medStr)
			if len(matches) < 3 {
				return nil
			}
			action := api.MedActionType_MED_REPLACE
			switch matches[1] {
			case "+", "-":
				action = api.MedActionType_MED_MOD
			}
			value, err := strconv.ParseInt(matches[1]+matches[2], 10, 64)
			if err != nil {
				return nil
			}
			return &api.MedAction{
				Value:      value,
				ActionType: action,
			}
		}(),
		AsPrepend: func() *api.AsPrependAction {
			if len(s.Actions.BgpActions.SetAsPathPrepend.As) == 0 {
				return nil
			}
			var asn uint64
			useleft := false
			if s.Actions.BgpActions.SetAsPathPrepend.As != "last-as" {
				asn, _ = strconv.ParseUint(s.Actions.BgpActions.SetAsPathPrepend.As, 10, 32)
			} else {
				useleft = true
			}
			return &api.AsPrependAction{
				Asn:         uint32(asn),
				Repeat:      uint32(s.Actions.BgpActions.SetAsPathPrepend.RepeatN),
				UseLeftMost: useleft,
			}
		}(),
		ExtCommunity: func() *api.CommunityAction {
			if len(s.Actions.BgpActions.SetExtCommunity.SetExtCommunityMethod.CommunitiesList) == 0 {
				return nil
			}
			return &api.CommunityAction{
				ActionType: api.CommunityActionType(
					config.BgpSetCommunityOptionTypeToIntMap[config.BgpSetCommunityOptionType(s.Actions.BgpActions.SetExtCommunity.Options)],
				),
				Communities: s.Actions.BgpActions.SetExtCommunity.SetExtCommunityMethod.CommunitiesList,
			}
		}(),
		LargeCommunity: func() *api.CommunityAction {
			if len(s.Actions.BgpActions.SetLargeCommunity.SetLargeCommunityMethod.CommunitiesList) == 0 {
				return nil
			}
			return &api.CommunityAction{
				ActionType: api.CommunityActionType(
					config.BgpSetCommunityOptionTypeToIntMap[config.BgpSetCommunityOptionType(s.Actions.BgpActions.SetLargeCommunity.Options)],
				),
				Communities: s.Actions.BgpActions.SetLargeCommunity.SetLargeCommunityMethod.CommunitiesList,
			}
		}(),
		Nexthop: func() *api.NexthopAction {
			if len(string(s.Actions.BgpActions.SetNextHop)) == 0 {
				return nil
			}

			if string(s.Actions.BgpActions.SetNextHop) == "self" {
				return &api.NexthopAction{
					Self: true,
				}
			}
			return &api.NexthopAction{
				Address: string(s.Actions.BgpActions.SetNextHop),
			}
		}(),
		LocalPref: func() *api.LocalPrefAction {
			if s.Actions.BgpActions.SetLocalPref == 0 {
				return nil
			}
			return &api.LocalPrefAction{Value: s.Actions.BgpActions.SetLocalPref}
		}(),
	}
	return &api.Statement{
		Name:       s.Name,
		Conditions: cs,
		Actions:    as,
	}
}

func newAPIPolicyFromTableStruct(p *Policy) *api.Policy {
	return toPolicyApi(p.toConfig())
}

func toPolicyApi(p *config.PolicyDefinition) *api.Policy {
	return &api.Policy{
		Name: p.Name,
		Statements: func() []*api.Statement {
			l := make([]*api.Statement, 0)
			for _, s := range p.Statements {
				l = append(l, toStatementApi(&s))
			}
			return l
		}(),
	}
}

func NewAPIPolicyAssignmentFromTableStruct(t *PolicyAssignment) *api.PolicyAssignment {
	return &api.PolicyAssignment{
		Direction: func() api.PolicyDirection {
			switch t.Type {
			case POLICY_DIRECTION_IMPORT:
				return api.PolicyDirection_IMPORT
			case POLICY_DIRECTION_EXPORT:
				return api.PolicyDirection_EXPORT
			}
			ctrl.Log.Error(fmt.Errorf("invalid policy-type: %d", t.Type), "error while converting policy assignment")
			return api.PolicyDirection_UNKNOWN
		}(),
		DefaultAction: func() api.RouteAction {
			switch t.Default {
			case ROUTE_TYPE_ACCEPT:
				return api.RouteAction_ACCEPT
			case ROUTE_TYPE_REJECT:
				return api.RouteAction_REJECT
			}
			return api.RouteAction_NONE
		}(),
		Name: t.Name,
		Policies: func() []*api.Policy {
			l := make([]*api.Policy, 0)
			for _, p := range t.Policies {
				l = append(l, newAPIPolicyFromTableStruct(p))
			}
			return l
		}(),
	}
}

func NewAPIRoutingPolicyFromConfigStruct(c *config.RoutingPolicy) (*api.RoutingPolicy, error) {
	definedSets, err := config.NewAPIDefinedSetsFromConfigStruct(&c.DefinedSets)
	if err != nil {
		return nil, err
	}
	policies := make([]*api.Policy, 0, len(c.PolicyDefinitions))
	for _, policy := range c.PolicyDefinitions {
		policies = append(policies, toPolicyApi(&policy))
	}

	return &api.RoutingPolicy{
		DefinedSets: definedSets,
		Policies:    policies,
	}, nil
}
