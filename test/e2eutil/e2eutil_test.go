package e2eutil_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/kubesphere/porter/test/e2eutil"
)

var _ = Describe("E2eutil", func() {
	It("Should be able to parse log", func() {
		cases := []string{"{\"level\":\"info\",\"ts\":1553612596.483035,\"logger\":\"entrypoint\",\"msg\":\"Setting up readiness probe\"}",
			"{\"level\":\"error\",\"ts\":1553612596.4830754,\"logger\":\"entrypoint\",\"msg\":\"Starting the Cmd.\"}",
			"{\"level\":\"error\",\"ts\":1553612596.4830754,\"logger\":\"entrypoint\",\"msg\":\"Starting the Cmd.\",\"service\":\"Test\"}",
			"unknow",
		}
		expected := []*LogInfo{
			&LogInfo{Level: "info", TS: 1553612596.483035, Logger: "entrypoint", Msg: "Setting up readiness probe"},
			&LogInfo{Level: "error", TS: 1553612596.4830754, Logger: "entrypoint", Msg: "Starting the Cmd."},
			&LogInfo{Level: "error", TS: 1553612596.4830754, Logger: "entrypoint", Msg: "Starting the Cmd."},
			nil,
		}
		for index, ca := range cases {
			Expect(ParseLog([]byte(ca))).To(Equal(expected[index]))
		}
	})
})
