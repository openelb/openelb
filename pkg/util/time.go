package util

import (
	"github.com/golang/protobuf/ptypes"
	prototime "github.com/golang/protobuf/ptypes/timestamp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ParseProtoTime convert proto time to k8s time
func ParseProtoTime(t *prototime.Timestamp) (*metav1.Time, error) {
	gotime, err := ptypes.Timestamp(t)
	if err != nil {
		return nil, err
	}
	return &metav1.Time{
		Time: gotime,
	}, nil
}
