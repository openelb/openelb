package framework

import (
	"fmt"
	"os/exec"

	"k8s.io/kubernetes/test/e2e/framework"
)

const (
	NetworkFailure = 4
	retryLimit     = 4
)

func Do(address string) error {
	var (
		code     int
		err      error
		out      []byte
		retrycnt = 0
	)

	// Retry loop to handle wget NetworkFailure errors
	for {
		out, err = exec.Command("wget", "-O-", "-q", address, "-T", "60", "-t", "1").CombinedOutput()
		if exitErr, ok := err.(*exec.ExitError); err != nil && ok {
			code = exitErr.ExitCode()
		} else {
			break
		}
		if retrycnt < retryLimit && code == NetworkFailure {
			framework.Logf("wget failed with code %d, err %s retrycnt %d\n", code, err, retrycnt)
			retrycnt++
		} else {
			break
		}
	}
	if err != nil {
		return fmt.Errorf("%w: %s", err, string(out))
	}
	return nil
}
