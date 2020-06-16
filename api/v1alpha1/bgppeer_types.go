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

// BgpPeerStatus defines the observed state of BgpPeer
type BgpPeerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:resource:scope=Cluster

// BgpPeer is the Schema for the bgppeers API
type BgpPeer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BgpPeerSpec   `json:"spec,omitempty"`
	Status BgpPeerStatus `json:"status,omitempty"`
}

type BgpPeerSpec struct {
	// original -> bgp:neighbor-address
	// original -> bgp:neighbor-config
	// Configuration parameters relating to the BGP neighbor or
	// group.
	Config NeighborConfig `mapstructure:"config" json:"config,omitempty"`

	// original -> bgp:add-paths
	// Parameters relating to the advertisement and receipt of
	// multiple paths for a single NLRI (add-paths).
	AddPaths AddPaths `mapstructure:"add-paths" json:"addPaths,omitempty"`

	// original -> bgp:transport
	// Transport session parameters for the BGP neighbor or group.
	Transport Transport `mapstructure:"transport" json:"transport,omitempty"`

	UsingPortForward bool `json:"usingPortForward,omitempty" mapstructure:"using-port-forward"`
}

// struct for container bgp:transport.
// Transport session parameters for the BGP neighbor or group.
type Transport struct {
	// original -> bgp:passive-mode
	// bgp:passive-mode's original type is boolean.
	// Wait for peers to issue requests to open a BGP session,
	// rather than initiating sessions from the local router.
	PassiveMode bool `mapstructure:"passive-mode" json:"passiveMode,omitempty"`

	// original -> gobgp:remote-port
	// gobgp:remote-port's original type is inet:port-number.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	RemotePort uint16 `mapstructure:"remote-port" json:"remotePort,omitempty"`
}

// struct for container bgp:add-paths.
// Parameters relating to the advertisement and receipt of
// multiple paths for a single NLRI (add-paths).
type AddPaths struct {
	// original -> bgp:send-max
	// The maximum number of paths to advertise to neighbors
	// for a single NLRI.
	SendMax uint8 `mapstructure:"send-max" json:"sendMax,omitempty"`
}

// struct for container bgp:config.
// Configuration parameters relating to the BGP neighbor or
// group.
type NeighborConfig struct {
	// original -> bgp:peer-as
	// bgp:peer-as's original type is inet:as-number.
	// AS number of the peer.
	PeerAs uint32 `mapstructure:"peer-as" json:"peerAs,required"`

	// original -> bgp:neighbor-address
	// bgp:neighbor-address's original type is inet:ip-address.
	// Address of the BGP peer, either in IPv4 or IPv6.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^([0-9]{1,3}\.){3}[0-9]{1,3}$`
	NeighborAddress string `mapstructure:"neighbor-address" json:"neighborAddress,required"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BgpPeerList contains a list of BgpPeer
type BgpPeerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BgpPeer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BgpPeer{}, &BgpPeerList{})
}
