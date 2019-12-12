/*
Copyright 2019 The Kubesphere Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// BgpPeerSpec defines the desired state of BgpPeer
type BgpPeerSpec struct {
	Conf            PeerConf         `json:"conf,omitempty"`
	TimersConfig    *TimersConfig    `json:"timersConfig,omitempty"`
	Transport       *Transport       `json:"transport,omitempty"`
	GracefulRestart *GracefulRestart `json:"gracefulRestart,omitempty"`
}

//PeerConf define the config of neighbour
type PeerConf struct {
	AuthPassword      string `json:"authPassword,omitempty"`
	Description       string `json:"description,omitempty"`
	NeighborAddress   string `json:"neighborAddress,omitempty"`
	PeerAs            uint32 `json:"peerAs,omitempty"`
	PeerType          uint32 `json:"peerType,omitempty"`
	SendCommunity     uint32 `json:"sendCommunity,omitempty"`
	NeighborInterface string `json:"neighborInterface,omitempty"`
	AdminDown         bool   `json:"adminDown,omitempty"`
}

// Transport define the connection config  between peers
type Transport struct {
	LocalAddress string `json:"localAddress,omitempty"`
	LocalPort    uint32 `json:"localPort,omitempty"`
	MtuDiscovery bool   `json:"mtuDiscovery,omitempty"`
	PassiveMode  bool   `json:"passiveMode,omitempty"`
}

type TimersConfig struct {
	ConnectRetry      uint64 `json:"connectRetry,omitempty"`
	HoldTime          uint64 `json:"holdTime,omitempty"`
	KeepaliveInterval uint64 `json:"keepaliveInterval,omitempty"`
}

type GracefulRestart struct {
	Enabled          bool   `json:"enabled,omitempty"`
	RestartTime      uint32 `json:"restart_time,omitempty"`
	DeferralTime     uint32 `json:"deferral_time,omitempty"`
	LonglivedEnabled bool   `json:"longlived_enabled,omitempty"`
	StaleRoutesTime  uint32 `json:"stale_routes_time,omitempty"`
	PeerRestartTime  uint32 `json:"peer_restart_time,omitempty"`
	Mode             string `json:"mode,omitempty"`
}

//SessionState define current state of connection between peers
type SessionState int32

const (
	PeerState_UNKNOWN     SessionState = 0
	PeerState_IDLE        SessionState = 1
	PeerState_CONNECT     SessionState = 2
	PeerState_ACTIVE      SessionState = 3
	PeerState_OPENSENT    SessionState = 4
	PeerState_OPENCONFIRM SessionState = 5
	PeerState_ESTABLISHED SessionState = 6
)

// BgpPeerStatus defines the observed state of BgpPeer
type BgpPeerStatus struct {
	SessionState SessionState `json:"sessionState,omitempty"`
	Uptime       metav1.Time  `json:"uptime,omitempty"`
	Downtime     metav1.Time  `json:"downtime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories=ksnet

// BgpPeer is the Schema for the bgppeers API
type BgpPeer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BgpPeerSpec   `json:"spec,omitempty"`
	Status BgpPeerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BgpPeerList contains a list of BgpPeer
type BgpPeerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BgpPeer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BgpPeer{}, &BgpPeerList{})
}
