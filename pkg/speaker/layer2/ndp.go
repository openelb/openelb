package layer2

import (
	"fmt"
	"github.com/go-logr/logr"
	"github.com/kubesphere/porterlb/pkg/speaker"
	"github.com/mdlayher/ndp"
	"io"
	corev1 "k8s.io/api/core/v1"
	"net"
	ctrl "sigs.k8s.io/controller-runtime"
	"sync"
)

/*
NDP 实现
因为是工作在 2层链路,所以本次实现,只实现 NDP 的NS,NA 宣告过程
不实现 NDP 的RS RA.

*/

var _ speaker.Speaker = &ndpSpeaker{}

type ndpSpeaker struct {
	logger              logr.Logger
	intf                *net.Interface
	hardwareAddr        net.HardwareAddr
	conn                *ndp.Conn
	lock                sync.Mutex
	solicitedNodeGroups map[string]int64
}

func newNDPSpeaker(ifi *net.Interface) (*ndpSpeaker, error) {
	ctrl.Log.Info("进度ndp")
	conn, _, err := ndp.Listen(ifi, ndp.LinkLocal)
	if err != nil {
		return nil, fmt.Errorf("creating NDP Speaker for %q: %s", ifi.Name, err)
	}
	ns := &ndpSpeaker{
		logger:              ctrl.Log.WithName("ndpSpeaker"),
		intf:                ifi,              //ipv6 接口
		hardwareAddr:        ifi.HardwareAddr, // mac地址
		conn:                conn,             // 原始连接
		lock:                sync.Mutex{},
		solicitedNodeGroups: map[string]int64{}, // 缓存加入的组
	}

	return ns, err
}

func (n *ndpSpeaker) gratuitous(ip net.IP) error {
	err := n.advertise(net.IPv6linklocalallnodes, ip, true)
	return err
}

func (n *ndpSpeaker) advertise(dst, target net.IP, gratuitous bool) error {
	//https://datatracker.ietf.org/doc/html/rfc4861#section-4.4
	m := &ndp.NeighborAdvertisement{
		Solicited:     !gratuitous,
		Override:      gratuitous,
		TargetAddress: target,
		Options: []ndp.Option{
			&ndp.LinkLayerAddress{
				Direction: ndp.Target,
				Addr:      n.hardwareAddr, //发送本机的mac地址
			},
		},
	}
	return n.conn.WriteTo(m, nil, dst)
}

func (n *ndpSpeaker) run(stopCh <-chan struct{}) {
	for {
		err := n.processRequest()

		if err == dropReasonClosed {
			return
		} else if err == dropReasonError {
			select {
			case <-stopCh:
				return
			default:
			}
		}
	}
}

func (n *ndpSpeaker) processRequest() dropReason {
	msg, _, src, err := n.conn.ReadFrom()
	if err != nil {
		if err == io.EOF {
			return dropReasonClosed
		}
		return dropReasonError
	}

	ns, ok := msg.(*ndp.NeighborSolicitation)
	if !ok {
		return dropReasonMessageTypeError
	}

	var nsLLAddr net.HardwareAddr
	for _, o := range ns.Options {
		lla, ok := o.(*ndp.LinkLayerAddress)
		if !ok {
			continue
		}
		if lla.Direction != ndp.Source {
			continue
		}
		nsLLAddr = lla.Addr
		break
	}
	if nsLLAddr == nil {
		return dropReasonNoSourceHardwareAddr
	}
	if err := n.advertise(src, ns.TargetAddress, false); err != nil {
		n.logger.Error(err, "failed to send NDP reply")
	}
	return dropReasonNone
}

func (n *ndpSpeaker) SetBalancer(ip string, nexthops []corev1.Node) error {
	// 当所在的节点down掉时,主动发出NA消息，告知邻居本节点的变化
	ctrl.Log.Info(fmt.Sprintf("SetBalancer %s %+v ", ip, nexthops))
	// TODO
	// 配置双栈后,nexthops 目前无法获取到节点的ipv6地址
	n.lock.Lock()
	n.lock.Unlock()
	return fmt.Errorf("DelBalancer")

}

func (n *ndpSpeaker) DelBalancer(ip string) error {
	return fmt.Errorf("DelBalancer")
}

func (n *ndpSpeaker) Start(stopCh <-chan struct{}) error {
	// 监听NDP NS 消息,并发送NDP NA消息,NDP NA 消息的link layer 使用linklocal
	go n.run(stopCh)
	go func() {
		<-stopCh
		n.conn.Close()
	}()
	return nil
}
