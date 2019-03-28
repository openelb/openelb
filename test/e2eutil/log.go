package e2eutil

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type LogInfo struct {
	Level  string  `json:"level,omitempty"`
	TS     float64 `json:"ts,omitempty"`
	Logger string  `json:"logger,omitempty"`
	Msg    string  `json:"msg,omitempty"`
}

func ParseLog(log string) *LogInfo {
	logInfo := new(LogInfo)
	err := json.Unmarshal([]byte(log), logInfo)
	if err != nil {
		return nil
	}
	return logInfo
}

func checkLog(namespace, name, containerName string) (string, error) {
	params := []string{"logs", "-n", namespace, name}
	if containerName != "" {
		params = append(params, "-c", containerName)
	}
	logcmd := exec.Command("kubectl", params...)
	log, _ := logcmd.CombinedOutput()
	lines := strings.Split(string(log), "\n")
	for _, line := range lines {
		if l := ParseLog(line); l != nil {
			if l.Level == "error" {
				return string(log), fmt.Errorf(line)
			}
		} else {
			if strings.Contains(line, "error") {
				return string(log), fmt.Errorf(line)
			}
		}
	}
	return string(log), nil
}
func CheckManagerLog(namespace, name string) (string, error) {
	return checkLog(namespace, name, "manager")
}

func CheckAgentLog(namespace, name string, dynclient client.Client) (string, error) {
	podlist := &corev1.PodList{}
	label := make(map[string]string)
	label["app"] = name
	err := dynclient.List(context.TODO(), client.MatchingLabels(label).InNamespace(namespace), podlist)
	if err != nil {
		return "Failed to get podlist of agent", err
	}
	for _, pod := range podlist.Items {
		log, err := checkLog(namespace, pod.Name, "")
		if err != nil {
			return pod.Spec.NodeName + "\n" + log, err
		}
	}
	return "", nil
}
