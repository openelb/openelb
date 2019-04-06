package main

import (
	"fmt"

	"github.com/kubesphere/porter/test/e2eutil"
)

type S struct {
	name string
}

func main() {
	s := S{name: "Hello"}
	s.A()
}

func (s *S) Print() {
	fmt.Println(s.name)
}
func (s *S) A() string {
	id, err := e2eutil.RunGoBGPContainer("/root/bgp/test.toml")
	if err != nil {
		panic(err)
	}

	fmt.Println(id)
	s.name = id
	return id
}
