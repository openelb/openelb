package machinery

import (
	"time"

	corev1 "k8s.io/api/core/v1"
)

const (
	RequeueBaseTime = 10
	RequeueStep     = 10
	RequeueMaximum  = 180
)

var requeueTable map[string]int

func init() {
	requeueTable = make(map[string]int)
}

func GetRequeueTime(service *corev1.Service) time.Duration {
	key := service.Namespace + "/" + service.Name
	t := getRequeueTime(key)
	requeueTable[key] = t
	return time.Duration(t) * time.Second
}

func getRequeueTime(key string) int {
	if v, ok := requeueTable[key]; ok {
		cacuTime := v + RequeueStep
		if cacuTime > RequeueMaximum {
			return RequeueMaximum
		}
		return cacuTime
	}
	return RequeueBaseTime
}
