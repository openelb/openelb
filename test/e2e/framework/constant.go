package framework

import "time"

const (
	PauseImage = "kubesphere/pause:3.6"

	// PodStartTimeout is how long to wait for the pod to be started.
	PodStartTimeout = 5 * time.Minute

	timeout = 2 * time.Minute

	// pollInterval defines the interval time for a poll operation.
	pollInterval = 5 * time.Second
	// pollTimeout defines the time after which the poll operation times out.
	pollTimeout = 400 * time.Second
)
