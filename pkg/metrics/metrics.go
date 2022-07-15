package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	// eip
	addressesTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "addresses_total",
			Help: "The eip contains the total number of ips",
		},
		[]string{
			"eipName",
		})
	addressesInUseTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "addresses_in_use_total",
			Help: "The eip has been assigned the number of ips",
		},
		[]string{
			"eipName",
		})
	servicesAllocatedTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "services_allocated_total",
			Help: "Number of services to assign IP addresses from eip",
		},
		[]string{
			"eipName",
		})

	// ARP / NDP
	requestsReceived = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "requests_received",
			Help: "The number of arp requests received.",
		},
		[]string{
			"ip",
		})
	responsesSent = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "responses_sent",
			Help: "The number of arp responses sent.",
		},
		[]string{
			"ip",
		})
	gratuitousSent = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gratuitous_sent",
			Help: "The number of sent GARPs.",
		},
		[]string{
			"ip",
		})

	// BGP
	sessionUp = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "session_up",
			Help: "The status of BGP Sessions.",
		},
		[]string{
			"peerIP",
			"nodeName",
		})
	updatesTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "updates_total",
			Help: "The total number of update packets received.",
		},
		[]string{
			"peerIP",
			"nodeName",
		})
	announcedPrefixesTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "announced_prefixes_total",
			Help: "The total number of prefixes waiting to be announced.",
		},
		[]string{
			"peerIP",
			"nodeName",
		})
	pendingPrefixesTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "pending_prefixes_total",
			Help: "The total number of prefixes waiting to be deleted.",
		},
		[]string{
			"peerIP",
			"nodeName",
		})
)

func init() {
	// eip
	metrics.Registry.MustRegister(addressesTotal)
	metrics.Registry.MustRegister(addressesInUseTotal)
	metrics.Registry.MustRegister(servicesAllocatedTotal)

	// ARP/NDP
	metrics.Registry.MustRegister(requestsReceived)
	metrics.Registry.MustRegister(responsesSent)
	metrics.Registry.MustRegister(gratuitousSent)

	// BGP
	metrics.Registry.MustRegister(sessionUp)
	metrics.Registry.MustRegister(updatesTotal)
	metrics.Registry.MustRegister(announcedPrefixesTotal)
	metrics.Registry.MustRegister(pendingPrefixesTotal)
}

func UpdateEipMetrics(eipName string, total, used, svcCount float64) {
	addressesTotal.WithLabelValues(eipName).Set(total)
	addressesInUseTotal.WithLabelValues(eipName).Set(used)
	servicesAllocatedTotal.WithLabelValues(eipName).Set(svcCount)
}

func DeleteEipMetrics(eip string) {
	gratuitousSent.DeleteLabelValues(eip)
	responsesSent.DeleteLabelValues(eip)
	requestsReceived.DeleteLabelValues(eip)
}

func InitLayer2Metrics(ip string) {
	gratuitousSent.WithLabelValues(ip).Add(0)
	responsesSent.WithLabelValues(ip).Add(0)
	requestsReceived.WithLabelValues(ip).Add(0)
}

func UpdateGratuitousSentMetrics(ip string) {
	gratuitousSent.WithLabelValues(ip).Inc()
}

func UpdateResponsesSentMetrics(ip string) {
	responsesSent.WithLabelValues(ip).Inc()
}
func UpdateRequestsReceivedMetrics(ip string) {
	requestsReceived.WithLabelValues(ip).Inc()
}

func DeleteLayer2Metrics(ip string) {
	gratuitousSent.DeleteLabelValues(ip)
	responsesSent.DeleteLabelValues(ip)
	requestsReceived.DeleteLabelValues(ip)
}

func InitBGPPeerMetrics(peerIP, node string) {
	sessionUp.WithLabelValues(peerIP, node).Add(0)
	updatesTotal.WithLabelValues(peerIP, node).Add(0)
	announcedPrefixesTotal.WithLabelValues(peerIP, node).Add(0)
	pendingPrefixesTotal.WithLabelValues(peerIP, node).Add(0)
}

func UpdateBGPSessionMetrics(peerIP, node string, state, updateTotal float64) {
	sessionUp.WithLabelValues(peerIP, node).Set(state)
	updatesTotal.WithLabelValues(peerIP, node).Set(updateTotal)
}

func UpdateBGPPathMetrics(peerIP, node string, announcedTotal, pendingTotal float64) {
	announcedPrefixesTotal.WithLabelValues(peerIP, node).Add(announcedTotal)
	pendingPrefixesTotal.WithLabelValues(peerIP, node).Add(pendingTotal)
}

func DeleteBGPPeerMetrics(peerIP, node string) {
	sessionUp.DeleteLabelValues(peerIP, node)
	updatesTotal.DeleteLabelValues(peerIP, node)
	announcedPrefixesTotal.DeleteLabelValues(peerIP, node)
	pendingPrefixesTotal.DeleteLabelValues(peerIP, node)
}
