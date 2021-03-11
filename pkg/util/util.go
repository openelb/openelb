package util

import (
	"github.com/kubesphere/porterlb/pkg/constant"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"net"
	"os"
)

func ContainsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func RemoveString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}

// IsDeletionCandidate checks if object is candidate to be deleted
func IsDeletionCandidate(obj v1.Object, finalizer string) bool {
	return obj.GetDeletionTimestamp() != nil && ContainsString(obj.GetFinalizers(), finalizer)
}

// NeedToAddFinalizer checks if need to add finalizer to object
func NeedToAddFinalizer(obj v1.Object, finalizer string) bool {
	return obj.GetDeletionTimestamp() == nil && !ContainsString(obj.GetFinalizers(), finalizer)
}

// Find node first NodeInternalIP, should check result
func GetNodeIP(node corev1.Node) net.IP {
	for _, address := range node.Status.Addresses {
		if address.Type == corev1.NodeInternalIP {
			return net.ParseIP(address.Address)
		}
	}

	return nil
}

func GetNodeName() string {
	return os.Getenv(constant.EnvNodeName)
}
