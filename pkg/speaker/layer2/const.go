package layer2

// dropReason is the reason why a layer2 protocol packet was not
// responded to.
type dropReason int

// Various reasons why a packet was dropped.
const (
	dropReasonNone dropReason = iota
	dropReasonClosed
	dropReasonError
	dropReasonARPReply
	dropReasonUnknowTargetIP
	dropReasonLeader
	dropReasonNotNeighborSolicitation
	dropReasonNotSourceDirection
)
