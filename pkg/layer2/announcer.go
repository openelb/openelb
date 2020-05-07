package layer2

import (
	"github.com/go-logr/logr"
	portererror "github.com/kubesphere/porter/pkg/errors"
	"github.com/kubesphere/porter/pkg/kubeutil"
	"net"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"time"
)

// Announce is used to "announce" new IPs mapped to the node's MAC address.
type Announce struct {
	logger logr.Logger
	client client.Client

	arp *arpResponder
}

// New returns an initialized Announce.
func New(log logr.Logger, client client.Client) *Announce {
	ret := &Announce{
		logger: log.WithName("Announcer"),
		client: client,
	}

	go ret.createResponder()

	return ret
}

func (a *Announce) createResponder() {
	var (
		ifi   net.Interface
		nodes map[string]string
		err   error
	)

	for true {
		//Until k8s client is available
		time.Sleep(1 * time.Second)
		nodes, err = kubeutil.GetNodeIPMap(a.client)
		if err != nil {
			a.logger.Error(err, "couldn't get nodeip")
			continue
		}
		if len(nodes) != 0 {
			break
		}
	}

	ifs, err := net.Interfaces()
	if err != nil {
		a.logger.Error(err, "couldn't list interfaces")
		goto exit
	}

	for _, intf := range ifs {
		ifi = intf
		for _, nodeIP := range nodes {
			addrs, err := ifi.Addrs()
			if err != nil {
				a.logger.Error(err, "couldn't list address")
				goto exit
			}

			for _, addr := range addrs {
				if strings.Index(addr.String(), nodeIP) >= 0 {
					a.logger.Info("found interface:", "name", ifi.Name)
					goto found
				}
			}
		}
	}

exit:
	os.Exit(1)

found:
	resp, err := newARPResponder(a.logger, &ifi)
	if err != nil {
		a.logger.Error(err, "couldn't new arpResponder")
		goto exit
	}
	a.arp = resp
	//TODO support ndp
}

// SetBalancer adds ip to the set of announced addresses.
func (a *Announce) SetBalancer(ip, nodeIP string) error {
	a.logger.Info("set layer2 balancer", "ip", ip, "nodeIP", nodeIP)
	if net.ParseIP(ip).To4() != nil && net.ParseIP(nodeIP).To4() != nil {
		if a.arp == nil {
			return portererror.NewLayer2AnnouncerNotReadyError()
		}
		if err := a.arp.Gratuitous(net.ParseIP(ip), net.ParseIP(nodeIP)); err != nil {
			return err
		}

		return nil
	}

	if net.ParseIP(ip).To16() != nil && net.ParseIP(nodeIP).To16() != nil {
		//TODO support ndp
	}

	return nil
}

// DeleteBalancer deletes an address from the set of addresses we should announce.
func (a *Announce) DeleteBalancer(ip string) error {
	if a.arp != nil {
		a.arp.DeleteIP(ip)
	}
	//TODO support ndp

	return nil
}

// dropReason is the reason why a layer2 protocol packet was not
// responded to.
type dropReason int

// Various reasons why a packet was dropped.
const (
	dropReasonNone dropReason = iota
	dropReasonClosed
	dropReasonError
	dropReasonARPReply
	dropReasonMessageType
	dropReasonNoSourceLL
	dropReasonEthernetDestination
	dropReasonAnnounceIP
)
