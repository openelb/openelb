package e2eutil_test

import (
	"fmt"
	"os"

	. "github.com/kubesphere/porter/test/e2eutil"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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
			Expect(ParseLog(ca)).To(Equal(expected[index]))
		}
	})
	It("Should be able to copy file to remote", func() {
		remote := os.Getenv("TEST_REMOTE_HOST")
		if remote == "" {
			fmt.Fprintln(GinkgoWriter, "Skipping testing scp")
			return
		}
		source := "/tmp/scp.file"
		str := "HelloWorld"
		f, err := os.Create(source)
		Expect(err).ShouldNot(HaveOccurred(), "Error in create test file")
		defer func() {
			f.Close()
			os.Remove(source)
		}()
		_, err = f.WriteString(str)
		f.Sync()
		Expect(ScpFileToRemote(source, source, remote)).ShouldNot(HaveOccurred())
		output, err := QuickConnectAndRun(remote, "cat "+source)
		Expect(err).ShouldNot(HaveOccurred(), "Error in cat remote file")
		Expect(output).To(Equal([]byte(str)))
	})
})
