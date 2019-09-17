package route

type Advertiser interface {
	ReconcileRoutes(ip string, prefix uint32, nexthops []string) (toAdd []string, toDelete []string, err error)
	AddMultiRoutes(ip string, prefix uint32, nexthops []string) error
	DeleteMultiRoutes(ip string, prefix uint32, nexthops []string) error
	DeleteAllRoutesOfIP(ip string) error
}
