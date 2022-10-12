package config

import (
	"fmt"
	"net"
	"regexp"
	"strconv"

	api "github.com/osrg/gobgp/api"
)

func NewAPIDefinedSetsFromConfigStruct(t *DefinedSets) ([]*api.DefinedSet, error) {
	definedSets := make([]*api.DefinedSet, 0)

	for _, ps := range t.PrefixSets {
		prefixes := make([]*api.Prefix, 0)
		for _, p := range ps.PrefixList {
			ap, err := newAPIPrefixFromConfigStruct(p)
			if err != nil {
				return nil, err
			}
			prefixes = append(prefixes, ap)
		}
		definedSets = append(definedSets, &api.DefinedSet{
			DefinedType: api.DefinedType_PREFIX,
			Name:        ps.PrefixSetName,
			Prefixes:    prefixes,
		})
	}

	for _, ns := range t.NeighborSets {
		definedSets = append(definedSets, &api.DefinedSet{
			DefinedType: api.DefinedType_NEIGHBOR,
			Name:        ns.NeighborSetName,
			List:        ns.NeighborInfoList,
		})
	}

	bs := t.BgpDefinedSets
	for _, cs := range bs.CommunitySets {
		definedSets = append(definedSets, &api.DefinedSet{
			DefinedType: api.DefinedType_COMMUNITY,
			Name:        cs.CommunitySetName,
			List:        cs.CommunityList,
		})
	}

	for _, es := range bs.ExtCommunitySets {
		definedSets = append(definedSets, &api.DefinedSet{
			DefinedType: api.DefinedType_EXT_COMMUNITY,
			Name:        es.ExtCommunitySetName,
			List:        es.ExtCommunityList,
		})
	}

	for _, ls := range bs.LargeCommunitySets {
		definedSets = append(definedSets, &api.DefinedSet{
			DefinedType: api.DefinedType_LARGE_COMMUNITY,
			Name:        ls.LargeCommunitySetName,
			List:        ls.LargeCommunityList,
		})
	}

	for _, as := range bs.AsPathSets {
		definedSets = append(definedSets, &api.DefinedSet{
			DefinedType: api.DefinedType_AS_PATH,
			Name:        as.AsPathSetName,
			List:        as.AsPathList,
		})
	}

	return definedSets, nil
}

func newAPIPrefixFromConfigStruct(c Prefix) (*api.Prefix, error) {
	min, max, err := parseMaskLength(c.IpPrefix, c.MasklengthRange)
	if err != nil {
		return nil, err
	}
	return &api.Prefix{
		IpPrefix:      c.IpPrefix,
		MaskLengthMin: uint32(min),
		MaskLengthMax: uint32(max),
	}, nil
}

var _regexpPrefixMaskLengthRange = regexp.MustCompile(`(\d+)\.\.(\d+)`)

func parseMaskLength(prefix, mask string) (int, int, error) {
	_, ipNet, err := net.ParseCIDR(prefix)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid prefix: %s", prefix)
	}
	if mask == "" {
		l, _ := ipNet.Mask.Size()
		return l, l, nil
	}
	elems := _regexpPrefixMaskLengthRange.FindStringSubmatch(mask)
	if len(elems) != 3 {
		return 0, 0, fmt.Errorf("invalid mask length range: %s", mask)
	}
	// we've already checked the range is sane by regexp
	min, _ := strconv.ParseUint(elems[1], 10, 8)
	max, _ := strconv.ParseUint(elems[2], 10, 8)
	if min > max {
		return 0, 0, fmt.Errorf("invalid mask length range: %s", mask)
	}
	if ipv4 := ipNet.IP.To4(); ipv4 != nil {
		f := func(i uint64) bool {
			return i <= 32
		}
		if !f(min) || !f(max) {
			return 0, 0, fmt.Errorf("ipv4 mask length range outside scope :%s", mask)
		}
	} else {
		f := func(i uint64) bool {
			return i <= 128
		}
		if !f(min) || !f(max) {
			return 0, 0, fmt.Errorf("ipv6 mask length range outside scope :%s", mask)
		}
	}
	return int(min), int(max), nil
}
