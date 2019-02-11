package main

import (
	"fmt"
	"os"

	"github.com/kubesphere/porter/pkg/controller/eip/nettool"
	"github.com/vishvananda/netlink"
)

func main() {
	link, err := netlink.LinkByName(os.Args[1])
	if err != nil {
		panic(err)
	}
	routes, _ := netlink.RouteList(link, netlink.FAMILY_V4)
	for _, item := range routes {
		fmt.Println(item.String())
	}

	rule := nettool.NewEIPRule("139.198.121.228", 32)
	if ok, err := rule.IsExist(); err == nil {
		if ok {
			fmt.Println(rule.Delete())
		} else {
			fmt.Println("Rule do not exist")
		}
	} else {
		panic(err)
	}
}
