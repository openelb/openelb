package layer2

type Announcer interface {
	AddAnnouncedIP(ip string) error
	DelAnnouncedIP(ip string) error
	Start(stopCh <-chan struct{}) error
	ContainsIP(ip string) bool
}

type announcer struct {
	Announcer
	eips   map[string]struct{}
	stopCh chan struct{}
}
