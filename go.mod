module github.com/openelb/openelb

go 1.16

require (
	github.com/coreos/go-iptables v0.4.2
	github.com/dgryski/go-farm v0.0.0-20190423205320-6a90982ecee2 // indirect
	github.com/go-chi/chi/v5 v5.0.7
	github.com/go-chi/cors v1.2.1
	github.com/go-logr/logr v0.1.0
	github.com/golang/protobuf v1.5.2
	github.com/hashicorp/golang-lru v0.5.3 // indirect
	github.com/j-keck/arping v1.0.1
	github.com/magiconair/properties v1.8.1 // indirect
	github.com/mdlayher/arp v0.0.0-20191213142603-f72070a231fc
	github.com/mdlayher/ethernet v0.0.0-20190606142754-0394541c37b7
	github.com/mdlayher/raw v0.0.0-20191009151244-50f2db8cc065
	github.com/nxadm/tail v1.4.6 // indirect
	github.com/onsi/ginkgo v1.14.2
	github.com/onsi/gomega v1.10.1
	github.com/osrg/gobgp v0.0.0-20210101133947-496b372f7b8d
	github.com/pelletier/go-toml v1.4.0 // indirect
	github.com/projectcalico/libcalico-go v1.7.2-0.20191104213956-8f81e1e344ce
	github.com/prometheus/client_golang v1.12.0
	github.com/spf13/cobra v1.0.0
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5
	github.com/vishvananda/netlink v1.0.0
	github.com/vishvananda/netns v0.0.0-20190625233234-7109fa855b0f // indirect
	go.uber.org/zap v1.10.0
	golang.org/x/net v0.0.0-20210525063256-abc453219eb5
	google.golang.org/grpc v1.31.0
	k8s.io/api v0.18.2
	k8s.io/apimachinery v0.18.2
	k8s.io/apiserver v0.18.2
	k8s.io/client-go v0.18.2
	k8s.io/component-base v0.18.2
	sigs.k8s.io/controller-runtime v0.6.0
)
