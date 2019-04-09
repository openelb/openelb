package main

import (
	"fmt"

	"github.com/kubesphere/porter/test/e2eutil"
)

func main() {
	s, err := e2eutil.QuickConnectAndRun("192.168.98.8", "echo $PATH")
	fmt.Println(string(s), err)
}
