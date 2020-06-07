package ipam

import (
	"net"

	"github.com/kubesphere/porter/pkg/constant"
	portererror "github.com/kubesphere/porter/pkg/errors"
)

func (d *DataStore) protocolForEIP(eip string) (string, string) {
	d.lock.Lock()
	defer d.lock.Unlock()
	for _, p := range d.IPPool {
		if p.Contains(net.ParseIP(eip)) {
			return p.EIPRefName, p.Protocol
		}
	}
	return "", ""
}

func (d *DataStore) SetBalancer(ip string, nexthops []string) error {

	name, protocol := d.protocolForEIP(ip)
	switch protocol {
	case constant.PorterProtocolBGP:
		return d.bgpServer.SetBalancer(ip, nexthops)
	case constant.PorterProtocolLayer2:
		if len(nexthops) <= 0 {
			return d.DelBalancer(ip)
		}
		if d.responders[name] != nil {
			return d.responders[name].Gratuitous(net.ParseIP(ip), net.ParseIP(nexthops[0]))
		}
	default:
		return portererror.PorterError{Code: portererror.ParaInvalidError}
	}

	return nil
}

func (d *DataStore) DelBalancer(ip string) error {
	name, protocol := d.protocolForEIP(ip)
	switch protocol {
	case constant.PorterProtocolBGP:
		return d.bgpServer.DelBalancer(ip)
	case constant.PorterProtocolLayer2:
		if d.responders[name] != nil {
			d.responders[name].DeleteIP(ip)
		}
	default:
		return portererror.PorterError{Code: portererror.ParaInvalidError}
	}

	return nil
}
